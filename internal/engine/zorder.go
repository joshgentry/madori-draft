package engine

import (
	"time"

	"durablewindows/internal/logger"
	"durablewindows/internal/models"
	"durablewindows/internal/storage"
	"durablewindows/internal/winapi"
)

// store is the package-level reference to the BoltDB store, set via SetStore.
var store *storage.Store

// SetStore sets the storage backend used for disk persistence.
func SetStore(s *storage.Store) {
	store = s
}

// RestoreZorder restores the z-order (window stacking) for all windows on the given display config.
func (p *Processor) RestoreZorder(displayKey string) {
	if p.FixZorder == 0 {
		return
	}
	apps, ok := p.monitorApplications[displayKey]
	if !ok || len(apps) == 0 {
		return
	}
	count := 0
	for _, metricsList := range apps {
		if len(metricsList) == 0 {
			continue
		}
		last := metricsList[len(metricsList)-1]
		if p.FixZorder == 2 || last.NeedRestoreZorder {
			count++
		}
	}
	if count == 0 {
		return
	}
	hDWP := winapi.BeginDeferWindowPos(int32(count))
	if hDWP == 0 {
		return
	}
	for hwnd, metricsList := range apps {
		if len(metricsList) == 0 {
			continue
		}
		last := metricsList[len(metricsList)-1]
		if p.FixZorder != 2 && !last.NeedRestoreZorder {
			continue
		}
		if !winapi.IsWindow(hwnd) || !winapi.IsWindowVisible(hwnd) {
			continue
		}
		hWndInsertAfter := uintptr(1)
		if last.PrevZorderWindow != 0 && winapi.IsWindow(last.PrevZorderWindow) {
			hWndInsertAfter = last.PrevZorderWindow
		}
		var rect winapi.RECT
		winapi.GetWindowRect(hwnd, &rect)
		hDWP = winapi.DeferWindowPos(hDWP, hwnd, hWndInsertAfter,
			rect.Left, rect.Top, rect.Width(), rect.Height(),
			winapi.DWP_NOACTIVATE|winapi.DWP_NOOWNERZORDER)
		if hDWP == 0 {
			return
		}
		last.NeedRestoreZorder = false
	}
	winapi.EndDeferWindowPos(hDWP)
}

// BatchFixTopMostWindows clears topmost flags where needed after restore.
func (p *Processor) BatchFixTopMostWindows() {
	for displayKey, apps := range p.monitorApplications {
		for hwnd, metricsList := range apps {
			if len(metricsList) == 0 {
				continue
			}
			last := metricsList[len(metricsList)-1]
			if last.NeedClearTopMost && !last.IsTopMost {
				if winapi.IsWindow(hwnd) {
					winapi.SetWindowPos(hwnd, 1, 0, 0, 0, 0,
						winapi.SWP_NOMOVE|winapi.SWP_NOSIZE|winapi.SWP_NOACTIVATE)
				}
				last.NeedClearTopMost = false
			}
			if p.topmostWindowsFixed[hwnd] {
				continue
			}
			if last.IsTopMost {
				extStyle := winapi.GetWindowLong(hwnd, winapi.GWL_EXSTYLE)
				if (extStyle & winapi.WS_EX_TOPMOST) == 0 {
					if winapi.IsWindow(hwnd) {
						winapi.SetWindowPos(hwnd, ^uintptr(0), 0, 0, 0, 0,
							winapi.SWP_NOMOVE|winapi.SWP_NOSIZE|winapi.SWP_NOACTIVATE)
					}
				}
				p.topmostWindowsFixed[hwnd] = true
			}
		}
		_ = displayKey
	}
}

// CaptureZorder records the z-order position of a window relative to its siblings.
func (p *Processor) CaptureZorder(hwnd uintptr, displayKey string) {
	apps, ok := p.monitorApplications[displayKey]
	if !ok {
		return
	}
	metricsList, ok := apps[hwnd]
	if !ok || len(metricsList) == 0 {
		return
	}
	prevWindow := winapi.GetWindow(hwnd, winapi.GW_HWNDPREV)
	if prevWindow != 0 && winapi.IsWindow(prevWindow) {
		last := metricsList[len(metricsList)-1]
		last.PrevZorderWindow = prevWindow
		last.NeedRestoreZorder = true
	}
}

// --- Disk persistence ---

// PersistToDB saves the full engine state (live windows, dead windows, snapshot times)
// to BoltDB so it survives crashes and reboots.
func (p *Processor) PersistToDB() {
	if store == nil {
		return
	}
	total := 0
	for _, apps := range p.monitorApplications {
		total += len(apps)
	}
	logger.AutoCapture("", "Auto-saving %d windows to database", total)
	for dk, apps := range p.monitorApplications {
		store.SaveWindowMetrics("live_"+dk, apps)
		store.SaveDisplayKeyTimestamp(dk, time.Now())
	}
	for dk, dead := range p.deadApps {
		store.SaveWindowMetrics("dead_"+dk, dead)
		store.SaveDisplayKeyTimestamp(dk, time.Now())
	}
	store.SaveSnapshotTimes(p.snapshotTakenTime)
}

// LoadFromDB loads the full engine state from BoltDB on startup.
func (p *Processor) LoadFromDB() {
	if store == nil {
		return
	}
	keys, _ := store.ListDisplayKeys()
	for _, dk := range keys {
		if len(dk) > 5 && dk[:5] == "live_" {
			realKey := dk[5:]
			if metrics, err := store.LoadWindowMetrics(dk); err == nil && metrics != nil {
				if p.monitorApplications[realKey] == nil {
					p.monitorApplications[realKey] = make(map[uintptr][]*models.WindowMetrics)
				}
				for hwnd, mList := range metrics {
					p.monitorApplications[realKey][hwnd] = mList
				}
			}
		} else if len(dk) > 5 && dk[:5] == "dead_" {
			realKey := dk[5:]
			if metrics, err := store.LoadWindowMetrics(dk); err == nil && metrics != nil {
				p.deadApps[realKey] = metrics
			}
		}
	}
	// Load snapshot times
	if times, err := store.LoadSnapshotTimes(); err == nil && times != nil {
		p.snapshotTakenTime = times
	}
	// Prune display-config buckets older than the most-recent N to cap DB growth.
	// Kept display keys are determined by _display_key_meta timestamps (updated on
	// every PersistToDB). Without this, every unique monitor layout ever seen
	// (dock/undock, resolution changes) persists forever.
	//
	// Display keys with snapshot data are preserved — pruning them would silently
	// destroy the snapshot (the metrics with snapshot bits live in live_<dk>).
	snapshotDKs := make(map[string]bool, len(p.snapshotTakenTime))
	for dk := range p.snapshotTakenTime {
		snapshotDKs[dk] = true
	}
	store.PruneDisplayKeys(25, snapshotDKs)
}

// ResetState clears temporary restore state.
func (p *Processor) ResetState() {
	p.restoreTimes = 0
	p.restoreHalted = false
	p.noRestoreWindowsTmp = make(map[uintptr]bool)
}
