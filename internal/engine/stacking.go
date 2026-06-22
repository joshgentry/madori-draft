package engine

import (
	"sort"
	"time"

	"madori/internal/logger"
	"madori/internal/models"
	"madori/internal/winapi"
)

// RestoreStacking restores the z-order (window stacking) for all windows on the given display config.
func (p *Processor) RestoreStacking(displayKey string) {
	if p.FixStacking == 0 {
		logger.WindowEvent(logger.LevelDebug, "stacking restore", "skipped: FixStacking=0")
		return
	}
	apps, ok := p.monitorApplications[displayKey]
	if !ok || len(apps) == 0 {
		return
	}

	// Collect windows that need z-order restore, with their captured rank.
	type entry struct {
		hwnd        uintptr
		rank        int
		onCurrentVD bool
	}
	var windows []entry
	includedCurrent := 0
	includedOther := 0
	skippedNoRank := 0
	skippedDead := 0
	for hwnd, m := range apps {
		if m == nil {
			continue
		}
		if !winapi.IsWindow(hwnd) {
			skippedDead++
			continue
		}
		onCurrent := p.vdManager == nil || !p.vdManager.Enabled() || p.vdManager.IsWindowOnCurrentVirtualDesktop(hwnd)

		// During snapshot restore, the loaded snapshot entries already have
		// NeedRestoreStacking=true from when the snapshot was taken — don't
		// gate on it. Otherwise, NeedRestoreStacking is a one-shot flag set
		// by CaptureStackingAll that must be consumed here so it doesn't
		// leak into auto-capture persistence.
		hasRank := m.NeedRestoreStacking || p.restoringSnapshot
		if !hasRank && p.FixStacking != 2 {
			skippedNoRank++
			continue
		}
		windows = append(windows, entry{hwnd, m.StackingRank, onCurrent})
		if onCurrent {
			includedCurrent++
		} else {
			includedOther++
		}
		// Clear the flag and rank now that we've read them, so they don't
		// survive through PersistToDB into future auto-restore cycles.
		m.NeedRestoreStacking = false
		m.StackingRank = -1
	}
	if len(windows) == 0 {
		logger.WindowEvent(logger.LevelDebug, "stacking restore",
			"skipped: no windows to restack (included=0, skippedNoRank=%d, skippedDead=%d, FixStacking=%d)",
			skippedNoRank, skippedDead, p.FixStacking)
		return
	}

	logger.WindowEvent(logger.LevelInfo, "stacking restore",
		"collected %d windows (curVD=%d otherVD=%d, skippedNoRank=%d skippedDead=%d, snapshot=%v)",
		len(windows), includedCurrent, includedOther, skippedNoRank, skippedDead, p.restoringSnapshot)

	// Sort by StackingRank ascending — topmost (rank 0) first.
	// Use stable sort so duplicate ranks (from stale data or edge cases)
	// produce deterministic ordering rather than depending on map iteration.
	sort.SliceStable(windows, func(i, j int) bool { return windows[i].rank < windows[j].rank })

	rankSet := make(map[uintptr]int, len(windows))
	for _, w := range windows {
		rankSet[w.hwnd] = w.rank
	}

	// Fast-path: walk the current z-order chain and check whether
	// tracked windows already appear in monotonically increasing
	// rank order. Windows on other desktops aren't in the chain
	// and don't affect the check.
	lastRank := -1
	needRebuild := false
	hwnd := winapi.GetTopWindow(0)
	for hwnd != 0 {
		if rank, ok := rankSet[hwnd]; ok {
			if rank < lastRank {
				needRebuild = true
				break
			}
			lastRank = rank
		}
		hwnd = winapi.GetWindow(hwnd, winapi.GW_HWNDNEXT)
	}
	if !needRebuild {
		logger.WindowEvent(logger.LevelDebug, "stacking restore", "stacking correct for %d windows, no rebuild needed", len(windows))
		return
	}

	logger.WindowEvent(logger.LevelInfo, "stacking restore", "rebuilding stacking for %d windows (curVD=%d otherVD=%d, FixStacking=%d)",
		len(windows), includedCurrent, includedOther, p.FixStacking)

	// Use individual SetWindowPos calls rather than DeferWindowPos /
	// EndDeferWindowPos.  The batch API silently ignores z-order changes
	// for cloaked (other-VD) windows and, worse, leaves subsequent
	// hWndInsertAfter targets dangling — current-VD windows whose
	// insertAfter points to a cloaked window land in the wrong position.
	// Individual SetWindowPos applies each change immediately, so every
	// insertAfter target has already been positioned.
	deferred := 0
	var prevHWND uintptr
	for _, w := range windows {
		if !winapi.SetWindowPos(w.hwnd, prevHWND, 0, 0, 0, 0,
			winapi.SWP_NOMOVE|winapi.SWP_NOSIZE|winapi.SWP_NOACTIVATE) {
			logger.Error(logger.LevelError, "", "SetWindowPos failed for %s (rank=%d, curVD=%v)",
				p.WindowDesc(w.hwnd), w.rank, w.onCurrentVD)
		}
		prevHWND = w.hwnd
		deferred++
	}
	logger.WindowEvent(logger.LevelInfo, "stacking restore", "placed %d windows", deferred)
}

// CaptureStackingAll walks the global z-order chain from top to bottom and
// assigns each tracked window a StackingRank (0 = global topmost, …).
// The chain includes windows from all virtual desktops — current-desktop
// windows appear first, then other-desktop windows grouped by desktop.
// Ranks are global but RestoreStacking filters to the current desktop, so
// per-desktop relative ordering is preserved.
func (p *Processor) CaptureStackingAll(displayKey string) {
	apps, ok := p.monitorApplications[displayKey]
	if !ok || len(apps) == 0 {
		return
	}

	// Count tracked windows per desktop for diagnostics only —
	// rank assignment below does not depend on this.
	totalTracked := 0
	for hwnd, m := range apps {
		if m == nil || !winapi.IsWindow(hwnd) {
			continue
		}
		totalTracked++
	}

	// Clear stale flags on all tracked windows.
	for _, m := range apps {
		if m == nil {
			continue
		}
		m.NeedRestoreStacking = false
		m.StackingRank = -1
	}

	// Walk the z-order chain starting from the desktop's topmost child.
	desktop := winapi.GetDesktopWindow()
	hwnd := winapi.GetWindow(desktop, winapi.GW_CHILD)
	rank := 0
	assigned := 0
	for hwnd != 0 {
		if m, ok := apps[hwnd]; ok && m != nil {
			m.StackingRank = rank
			m.NeedRestoreStacking = true
			rank++
			assigned++
		}
		hwnd = winapi.GetWindow(hwnd, winapi.GW_HWNDNEXT)
	}
	missed := totalTracked - assigned
	logger.WindowEvent(logger.LevelDebug, "stacking capture all",
		"tracked=%d assigned=%d missed=%d (displayKey=%s)",
		totalTracked, assigned, missed, displayKey)
}

// --- Disk persistence ---

// PersistToDB saves the full engine state to BoltDB so it survives crashes and reboots.
func (p *Processor) PersistToDB() {
	if p.store == nil {
		return
	}
	total := 0
	for _, apps := range p.monitorApplications {
		total += len(apps)
	}
	logger.AutoCapture(logger.LevelInfo, "", "Auto-saving %d windows to database", total)
	dbPruned := 0
	for dk, apps := range p.monitorApplications {
		n, _ := p.store.SaveWindowMetrics("dk_"+dk, snapKeyAuto, apps)
		dbPruned += n
		p.store.SaveDisplayKeyTimestamp(dk, time.Now())
	}
	if dbPruned > 0 {
		logger.AutoCapture(logger.LevelDebug, "", "DB cleanup: removed %d stale entries", dbPruned)
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
		if len(dk) > 3 && dk[:3] == "dk_" {
			realKey := dk[3:]
			if metrics, err := p.store.LoadWindowMetrics(dk, snapKeyAuto); err == nil && metrics != nil {
				if p.monitorApplications[realKey] == nil {
					p.monitorApplications[realKey] = make(map[uintptr]*models.WindowMetrics)
				}
				stale := 0
				for hwnd, m := range metrics {
					if winapi.IsWindow(hwnd) {
						p.monitorApplications[realKey][hwnd] = m
					} else {
						stale++
					}
				}
				if stale > 0 {
					logger.AutoCapture(logger.LevelDebug, "", "Skipped %d stale HWNDs loading %s", stale, dk)
				}
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
	// destroy the snapshot.
	snapshotDKs := make(map[string]bool, len(p.snapshotTakenTime))
	for dk := range p.snapshotTakenTime {
		snapshotDKs[dk] = true
	}
	p.store.PruneDisplayKeys(25, snapshotDKs)
}
