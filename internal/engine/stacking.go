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
		hwnd uintptr
		rank int
	}
	var windows []entry
	for hwnd, metricsList := range apps {
		if len(metricsList) == 0 {
			continue
		}
		// During snapshot restore, find the metrics tagged with the snapshot
		// ID — same logic as findBestRestoreMetrics. Otherwise use the last
		// entry. This is critical: foreground changes trigger auto-captures
		// that append entries without StackingRank, which would shadow the
		// snapshot entry if we always used the last.
		m := p.findStackingMetrics(metricsList)
		// NeedRestoreStacking is a one-shot flag cleared after the first
		// restore. During snapshot restore the snapshot bit on the entry
		// is signal enough — don't gate on the consumed flag.
		if m == nil {
			continue
		}
		if !p.restoringSnapshot && !m.NeedRestoreStacking && p.FixStacking != 2 {
			continue
		}
		windows = append(windows, entry{hwnd, m.StackingRank})
	}
	if len(windows) == 0 {
		logger.WindowEvent(logger.LevelDebug, "stacking restore", "skipped: no windows with NeedRestoreStacking (FixStacking=%d)", p.FixStacking)
		return
	}

	// Sort by StackingRank ascending — topmost (rank 0) first.
	sort.Slice(windows, func(i, j int) bool { return windows[i].rank < windows[j].rank })

	// Prune dead HWNDs and check whether the stacking is already correct.
	// Untracked system windows sit between our tracked windows in the
	// z-order chain, so we can't compare HWND predecessors directly —
	// instead walk the current chain and check relative ordering of
	// tracked windows.
	{
		alive := windows[:0]
		for _, w := range windows {
			if winapi.IsWindow(w.hwnd) && winapi.IsWindowVisible(w.hwnd) {
				alive = append(alive, w)
			}
		}
		windows = alive
	}
	if len(windows) == 0 {
		return
	}

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

	logger.WindowEvent(logger.LevelInfo, "stacking restore", "rebuilding stacking for %d windows (FixStacking=%d)", len(windows), p.FixStacking)
	hDWP := winapi.BeginDeferWindowPos(int32(len(windows)))
	if hDWP == 0 {
		logger.Error(logger.LevelError, "", "BeginDeferWindowPos failed for %d windows", len(windows))
		return
	}
	deferred := 0
	var prevHWND uintptr
	for _, w := range windows {
		prevDWP := hDWP
		hDWP = winapi.DeferWindowPos(hDWP, w.hwnd, prevHWND,
			0, 0, 0, 0,
			winapi.SWP_NOMOVE|winapi.SWP_NOSIZE|winapi.SWP_NOACTIVATE)
		if hDWP == 0 {
			logger.Error(logger.LevelError, "", "DeferWindowPos failed for %s", p.WindowDesc(w.hwnd))
			winapi.EndDeferWindowPos(prevDWP)
			return
		}
		prevHWND = w.hwnd
		deferred++
	}
	if !winapi.EndDeferWindowPos(hDWP) {
		logger.Error(logger.LevelError, "", "EndDeferWindowPos failed")
		return
	}
	logger.WindowEvent(logger.LevelInfo, "stacking restore", "placed %d windows", deferred)
}

// CaptureStackingAll walks the full stacking chain from top to bottom and
// assigns each tracked window a StackingRank (0 = topmost, 1 = next, …).
// Call this once per snapshot to capture the absolute stacking position
// of every window.
func (p *Processor) CaptureStackingAll(displayKey string) {
	apps, ok := p.monitorApplications[displayKey]
	if !ok || len(apps) == 0 {
		return
	}

	// Walk the z-order chain from the topmost window down.
	// GetTopWindow(0) = topmost; GetWindow(h, GW_HWNDNEXT) = next window below.
	hwnd := winapi.GetTopWindow(0)
	rank := 0
	assigned := 0
	for hwnd != 0 {
		if mList, ok := apps[hwnd]; ok && len(mList) > 0 {
			last := mList[len(mList)-1]
			last.StackingRank = rank
			last.NeedRestoreStacking = true
			rank++
			assigned++
		}
		hwnd = winapi.GetWindow(hwnd, winapi.GW_HWNDNEXT)
	}
	logger.WindowEvent(logger.LevelDebug, "stacking capture all", "assigned ranks 0-%d to %d tracked windows (displayKey=%s)", rank-1, assigned, displayKey)
}

// findStackingMetrics returns the metrics entry to use for stacking restore.
// During snapshot restore it searches for the snapshot-tagged entry (same
// logic as findBestRestoreMetrics); otherwise it returns the last entry.
func (p *Processor) findStackingMetrics(metricsList []*models.WindowMetrics) *models.WindowMetrics {
	if len(metricsList) == 0 {
		return nil
	}
	if p.restoringSnapshot {
		for i := len(metricsList) - 1; i >= 0; i-- {
			if metricsList[i].HasSnapshotID(p.snapshotId) {
				return metricsList[i]
			}
		}
		// Fallback to any snapshot
		for i := len(metricsList) - 1; i >= 0; i-- {
			if metricsList[i].HasSnapshot() {
				return metricsList[i]
			}
		}
	}
	return metricsList[len(metricsList)-1]
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
	logger.AutoCapture(logger.LevelInfo, "", "Auto-saving %d windows to database", total)
	dbPruned := 0
	for dk, apps := range p.monitorApplications {
		n, _ := p.store.SaveWindowMetrics("live_"+dk, apps)
		dbPruned += n
		p.store.SaveDisplayKeyTimestamp(dk, time.Now())
	}
	for dk, dead := range p.deadApps {
		n, _ := p.store.SaveWindowMetrics("dead_"+dk, dead)
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
