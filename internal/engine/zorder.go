package engine

import (
	"time"

	"durablewindows/internal/logger"
	"durablewindows/internal/models"
	"durablewindows/internal/winapi"
)

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
	if p.store == nil {
		return
	}
	total := 0
	for _, apps := range p.monitorApplications {
		total += len(apps)
	}
	logger.AutoCapture("", "Auto-saving %d windows to database", total)
	for dk, apps := range p.monitorApplications {
		p.store.SaveWindowMetrics("live_"+dk, apps)
		p.store.SaveDisplayKeyTimestamp(dk, time.Now())
	}
	for dk, dead := range p.deadApps {
		p.store.SaveWindowMetrics("dead_"+dk, dead)
		p.store.SaveDisplayKeyTimestamp(dk, time.Now())
	}
	p.store.SaveSnapshotTimes(p.snapshotTakenTime)
}

// LoadFromDB loads the full engine state from BoltDB on startup.
func (p *Processor) LoadFromDB() {
	if p.store == nil {
		return
	}
	keys, _ := p.store.ListDisplayKeys()
	for _, dk := range keys {
		if len(dk) > 5 && dk[:5] == "live_" {
			realKey := dk[5:]
			if metrics, err := p.store.LoadWindowMetrics(dk); err == nil && metrics != nil {
				if p.monitorApplications[realKey] == nil {
					p.monitorApplications[realKey] = make(map[uintptr][]*models.WindowMetrics)
				}
				for hwnd, mList := range metrics {
					p.monitorApplications[realKey][hwnd] = mList
				}
			}
		} else if len(dk) > 5 && dk[:5] == "dead_" {
			realKey := dk[5:]
			if metrics, err := p.store.LoadWindowMetrics(dk); err == nil && metrics != nil {
				p.deadApps[realKey] = metrics
			}
		}
	}
	// Load snapshot times
	if times, err := p.store.LoadSnapshotTimes(); err == nil && times != nil {
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
	p.store.PruneDisplayKeys(25, snapshotDKs)
}
