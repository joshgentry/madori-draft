package engine

import (
	"time"

	"madori/internal/logger"
	"madori/internal/models"
	"madori/internal/winapi"
)

// restoreResult describes what restoreSingleWindow did (or didn't do).
type restoreResult int

const (
	restoreRestored          restoreResult = iota // window position/size/placement was changed
	restoreAlreadyPositioned                      // window already at target position — no work needed
	restoreFiltered                               // window excluded from restore (dead, no-restore list, wrong desktop)
)

// BatchRestoreApplicationsOnCurrentDisplays performs multi-pass window restore.
func (p *Processor) BatchRestoreApplicationsOnCurrentDisplays() {
	if p.PauseAutoRestore {
		return
	}

	p.iconBusy = true
	if !p.restoringSnapshot && p.callbacks.ShowRestoreTip != nil {
		p.callbacks.ShowRestoreTip()
	} else if p.restoringSnapshot && p.callbacks.ShowSnapshotRestoreTip != nil {
		p.callbacks.ShowSnapshotRestoreTip(p.snapshotId)
	}

	// Boost priority during restore
	boostPriority()
	defer restorePriority()

	p.sessionActive = false

	// Ensure this goroutine's OS thread is per-monitor DPI-aware.
	winapi.SetThreadDpiAwarenessContext(winapi.DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)

	displayKey := p.curDisplayKey

	// Show desktop first if configured
	if p.ShowDesktop {
		p.showDesktop()
	}

	maxPasses := MaxRestoreTimes
	if p.FastRestore {
		maxPasses = MinRestoreTimes
	}

	for pass := 0; pass < maxPasses; pass++ {
		p.restoreTimes = pass + 1
		p.RestoreApplicationsOnCurrentDisplays(displayKey, 0, time.Now())

		// Always restore stacking
		p.RestoreStacking(displayKey)
	}

	// Redraw desktop after restore if configured
	if p.RedrawDesktop {
		winapi.RedrawWindow(0, nil, 0, winapi.RDW_INVALIDATE|winapi.RDW_ALLCHILDREN)
	}

	// Start the finish timer
	p.StartRestoreFinishedTimer(MaxRestoreLatency)
}

// RestoreApplicationsOnCurrentDisplays restores windows for a single display config.
func (p *Processor) RestoreApplicationsOnCurrentDisplays(displayKey string, targetHwnd uintptr, restoreTime time.Time) {
	source := p.monitorApplications[displayKey]
	if source == nil {
		return
	}

	for hwnd, metricsList := range source {
		if targetHwnd != 0 && hwnd != targetHwnd {
			continue
		}
		if len(metricsList) == 0 {
			continue
		}

		// Skip windows that no longer exist or shouldn't be tracked
		// (cloaked ghost frames, dead processes, etc.).
		if !p.shouldTrackWindow(hwnd) {
			continue
		}

		// Find the best metrics to restore
		metrics := p.findBestRestoreMetrics(hwnd, metricsList, restoreTime)
		if metrics == nil {
			continue
		}

		result := p.restoreSingleWindow(hwnd, metrics)
		p.restoredWindows[hwnd] = true
		// Multi-pass restore runs up to 5 times; only log on the
		// first pass to avoid confusing double-output.
		if p.restoreTimes == 1 {
			switch result {
			case restoreRestored:
				logger.Snapshot("window restored", "%s", p.WindowDesc(hwnd))
			case restoreAlreadyPositioned:
				logger.Snapshot("window restore skipped", "in correct position - %s", p.WindowDesc(hwnd))
			}
			// restoreFiltered: no log — window was excluded from restore
		}
	}
}

func (p *Processor) findBestRestoreMetrics(hwnd uintptr, metricsList []*models.WindowMetrics, restoreTime time.Time) *models.WindowMetrics {
	if len(metricsList) == 0 {
		return nil
	}

	// If we have a snapshot, find the one matching the snapshot.
	// Search newest-to-oldest so a fresh snapshot wins over a stale
	// entry from a previous session that still carries the same bit.
	if p.restoringSnapshot {
		for i := len(metricsList) - 1; i >= 0; i-- {
			if metricsList[i].HasSnapshotID(p.snapshotId) {
				return metricsList[i]
			}
		}
		// Fallback to last capture with any snapshot
		for i := len(metricsList) - 1; i >= 0; i-- {
			if metricsList[i].HasSnapshot() {
				return metricsList[i]
			}
		}
	}

	// Default: use the last capture
	return metricsList[len(metricsList)-1]
}

func doesWindowPositionMatch(hwnd uintptr, metrics *models.WindowMetrics) bool {
	var currentRect winapi.RECT
	if winapi.GetWindowRect(hwnd, &currentRect) {
		targetRect := metrics.ScreenPosition
		if !metrics.IsMinimized && currentRect.Equals(targetRect) {
			return true
		}
	}
	return false
}

func (p *Processor) restoreSingleWindow(hwnd uintptr, metrics *models.WindowMetrics) restoreResult {
	if !winapi.IsWindow(hwnd) {
		return restoreFiltered
	}
	if p.noRestoreWindows[hwnd] || p.noRestoreWindowsTmp[hwnd] {
		return restoreFiltered
	}
	if IsMinimized(hwnd) {
		// Windows blocks all write APIs (SetWindowPlacement,
		// ShowWindow, MoveWindowToDesktop) for windows on other
		// virtual desktops, so we have no non-disruptive way to
		// restore a minimized other-desktop window.
		if p.vdManager != nil && p.vdManager.Enabled() && !p.vdManager.IsWindowOnCurrentVirtualDesktop(hwnd) {
			// Only log when there was actual work to do — the window
			// was visible at capture but is now minimized elsewhere.
			if !metrics.IsMinimized {
				logger.Snapshot("vd-restore", "%s skipped: visible at snapshot, now minimized on other desktop — cannot restore across desktops",
					p.WindowDesc(hwnd))
			}
			return restoreFiltered
		}

		// If the saved metrics show the window was NOT minimized when
		// captured, unminimize it before restoring position. Otherwise
		// a minimized window stays minimized even when restoring a
		// snapshot taken while it was visible.
		if metrics.WindowPlacement.ShowCmd != winapi.SW_SHOWMINIMIZED &&
			metrics.WindowPlacement.ShowCmd != winapi.SW_MINIMIZE {
			winapi.ShowWindow(hwnd, winapi.SW_RESTORE)
			if doesWindowPositionMatch(hwnd, metrics) {
				return restoreRestored
			}
			// otherwise we pass through to reposition the newly unminimized window
		} else {
			return restoreAlreadyPositioned // minimized when captured, still minimized — correct state
		}
	} else if doesWindowPositionMatch(hwnd, metrics) {
		// Check if the window is already at the target position and size.
		// When a redundant restore fires (e.g. back-to-back display-change
		// events during monitor wake-up), the window may already be exactly
		// where a prior restore left it — skip the expensive SetWindowPlacement
		// and MoveWindow calls in that case. Topmost and off-screen fix are
		// lightweight and defensive, so they still run.
		//
		// Skip this check if we just unminimized the window — the unminimize
		// itself is a real restore action, even if Windows happened to place
		// the window at the same coordinates the metrics recorded.
		return restoreAlreadyPositioned
	}

	// Restore window placement
	if metrics.WindowPlacement.Length > 0 {
		wp := metrics.WindowPlacement
		wp.Length = uint32(winapi.DefaultWINDOWPLACEMENT().Length)

		winapi.SetWindowPlacement(hwnd, &wp)
	}

	// Move to saved screen position.
	// If the window was minimized when captured, GetWindowRect returned the
	// parking position (-32000, -32000) — skip MoveWindow so we don't undo
	// the correct NormalPosition that SetWindowPlacement just restored.
	if !metrics.IsMinimized {
		pos := metrics.ScreenPosition
		winapi.MoveWindow(hwnd, pos.Left, pos.Top, pos.Width(), pos.Height(), true)
	}

	// If the window was minimized at capture time but is visible now,
	// minimize it to match the captured state (mirrors the unminimize
	// logic in the IsMinimized block above).
	if metrics.IsMinimized && !IsMinimized(hwnd) {
		winapi.ShowWindow(hwnd, winapi.SW_MINIMIZE)
	}

	// Fix top-most state
	if metrics.IsTopMost {
		winapi.SetWindowPos(hwnd, ^uintptr(0), 0, 0, 0, 0,
			winapi.SWP_NOMOVE|winapi.SWP_NOSIZE|winapi.SWP_NOACTIVATE)
	} else if metrics.NeedClearTopMost {
		winapi.SetWindowPos(hwnd, 1, 0, 0, 0, 0,
			winapi.SWP_NOMOVE|winapi.SWP_NOSIZE|winapi.SWP_NOACTIVATE)
	}

	// Fix off-screen windows
	if p.EnableOffScreenFix && p.isOffScreen(hwnd) {
		var rect winapi.RECT
		winapi.GetWindowRect(hwnd, &rect)
		logger.Snapshot("off-screen fix", "%s rect=(%d,%d %dx%d) saved=(%d,%d %dx%d) minimized=%v",
			p.WindowDesc(hwnd),
			rect.Left, rect.Top, rect.Width(), rect.Height(),
			metrics.ScreenPosition.Left, metrics.ScreenPosition.Top,
			metrics.ScreenPosition.Width(), metrics.ScreenPosition.Height(),
			metrics.IsMinimized)
		logMonitors()
		p.FixOffScreenWindow(hwnd)
	}

	return restoreRestored
}

// FixOffScreenWindow moves a window that is off-screen back into view.
func (p *Processor) FixOffScreenWindow(hwnd uintptr) {
	var rect winapi.RECT
	if !winapi.GetWindowRect(hwnd, &rect) {
		return
	}

	displays := winapi.GetDisplays()
	if len(displays) == 0 {
		return
	}

	// Check if at least part of the window is on any display
	for _, d := range displays {
		var intersect winapi.RECT
		if winapi.IntersectRect(&intersect, &rect, &d.Position) {
			if intersect.Width() > 10 && intersect.Height() > 10 {
				return // window is visible enough
			}
		}
	}

	// Move to primary display
	primary := displays[0]
	newX := primary.Position.Left + 50
	newY := primary.Position.Top + 50
	newW := rect.Width()
	newH := rect.Height()
	if newW > primary.Position.Width()-100 {
		newW = primary.Position.Width() - 100
	}
	if newH > primary.Position.Height()-100 {
		newH = primary.Position.Height() - 100
	}

	winapi.MoveWindow(hwnd, newX, newY, newW, newH, true)
	winapi.SetForegroundWindow(hwnd)
}

// isOffScreen checks if a window is completely off all displays.
func (p *Processor) isOffScreen(hwnd uintptr) bool {
	if IsMinimized(hwnd) {
		return false
	}
	var rect winapi.RECT
	if !winapi.GetWindowRect(hwnd, &rect) {
		return false
	}
	if rect.Width() <= 10 || rect.Height() <= 10 {
		return false
	}
	return p.isRectOffScreen(rect)
}

func (p *Processor) isRectOffScreen(rect winapi.RECT) bool {
	corners := [][2]int32{
		{rect.Left + 10, rect.Top + 10},
		{rect.Left + rect.Width() - 10, rect.Top + 10},
	}
	if p.EnhancedOffScreenFix {
		corners = append(corners,
			[2]int32{rect.Left + 10, rect.Top + rect.Height() - 10},
			[2]int32{rect.Left + rect.Width() - 10, rect.Top + rect.Height() - 10},
		)
	}
	for _, c := range corners {
		pt := winapi.POINT{X: c[0], Y: c[1]}
		if winapi.MonitorFromPoint(pt, winapi.MONITOR_DEFAULTTONULL) != 0 {
			return false
		}
	}
	return true
}

// logMonitors dumps the current display layout for off-screen diagnostics.
func logMonitors() {
	for i, d := range winapi.GetDisplays() {
		logger.Snapshot("", "  monitor %d: (%d,%d %dx%d)",
			i, d.Position.Left, d.Position.Top, d.Position.Width(), d.Position.Height())
	}
}

// RetryDeferredCommands retries commands that timed out during restore.
func (p *Processor) RetryDeferredCommands() {
	for hwnd, cmds := range p.deferredCommands {
		if !winapi.IsWindow(hwnd) {
			continue
		}
		for _, cmd := range cmds {
			// Longer timeout for retries
			winapi.SendMessageTimeout(hwnd, cmd.msg, cmd.wParam, 0,
				winapi.SMTO_ABORTIFHUNG, SyncCommandRetryTimeoutMs)
		}
	}
	p.deferredCommands = make(map[uintptr][]deferredCommand)
}

// RestoreAllParked restores windows that were right-click-minimized to tray.
func (p *Processor) RestoreAllParked() {
	for _, hwnd := range p.GetTrayParkedWindows() {
		p.RestoreParkedWindow(hwnd)
	}
}

// ActivateWindow handles a window that was just unminimized. If the
// window was previously captured as minimized and a display change has
// occurred since that capture, it restores the window's pre-minimize
// position so it doesn't land somewhere incorrect on the new layout.
func (p *Processor) ActivateWindow(hwnd uintptr) {
	if !winapi.IsWindow(hwnd) {
		return
	}
	if IsMinimized(hwnd) {
		return // defensive: window still minimized, shouldn't happen
	}
	if p.noRestoreWindows[hwnd] {
		return
	}

	metricsList, ok := p.monitorApplications[p.curDisplayKey][hwnd]
	if !ok || len(metricsList) == 0 {
		// Window not tracked — only attempt off-screen fix
		if p.EnableOffScreenFix && p.isOffScreen(hwnd) {
			logger.AutoCapture("off-screen fix", "%s (untracked)",
				p.WindowDesc(hwnd))
			p.FixOffScreenWindow(hwnd)
		}
		return
	}

	// Get the most recent capture for this window.
	prevMetrics := metricsList[len(metricsList)-1]

	// Discard fast captures triggered by the unminimize event itself.
	// A capture taken within 400ms of unminimize may reflect the
	// window mid-animation or at a transient position.
	diff := prevMetrics.CaptureTime.Sub(p.lastUnminimizeTime)
	if diff > 0 && diff < 400*time.Millisecond {
		metricsList = metricsList[:len(metricsList)-1]
		p.monitorApplications[p.curDisplayKey][hwnd] = metricsList
		if len(metricsList) == 0 {
			return
		}
		newLast := metricsList[len(metricsList)-1]
		if !newLast.IsFullScreen {
			// The previous capture already had a valid normal
			// position — no restore needed.
			metricsList = append(metricsList, prevMetrics)
			p.monitorApplications[p.curDisplayKey][hwnd] = metricsList
			return
		}
		prevMetrics = newLast
	}

	// Only act if the previous capture shows the window was minimized.
	// If the capture shows the window was visible (not minimized), its
	// position is already correct on the current display layout.
	if !prevMetrics.IsMinimized {
		return
	}

	// Validate position data. A minimized capture may have garbage
	// ScreenPosition (GetWindowRect returns -32000,-32000 for
	// minimized windows) or a ShowCmd indicating the window was
	// still minimized when captured.
	targetRect := prevMetrics.ScreenPosition
	showCmd := prevMetrics.WindowPlacement.ShowCmd
	if showCmd == winapi.SW_SHOWMINIMIZED || showCmd == winapi.SW_MINIMIZE || targetRect.Left <= -25600 {
		return
	}

	// Skip if the window is already at the target position.
	var currentRect winapi.RECT
	if winapi.GetWindowRect(hwnd, &currentRect) && currentRect.Equals(targetRect) {
		return
	}

	if p.FixMinimizedRestore {
		// Only restore if the capture predates the last display
		// change. Captures taken after the display change already
		// reflect the current layout and don't need correction.
		if !p.lastDisplayChangeTime.IsZero() && !prevMetrics.CaptureTime.Before(p.lastDisplayChangeTime) {
			return
		}

		// Skip borderless windows (no WS_CAPTION style). These
		// are often custom-drawn UI or full-screen overlays.
		style := winapi.GetWindowLong(hwnd, winapi.GWL_STYLE)
		if style&winapi.WS_CAPTION == 0 {
			return
		}

		logger.AutoCapture("unminimize restore", "%s \u2192 (%d,%d %dx%d)",
			p.WindowDesc(hwnd),
			targetRect.Left, targetRect.Top,
			targetRect.Width(), targetRect.Height())

		p.restoreSingleWindow(hwnd, prevMetrics)
		return
	}

	// Fallback: if fixMinimizedRestore is disabled but off-screen
	// fix is enabled, center the window if it's off-screen.
	if p.EnableOffScreenFix && p.isOffScreen(hwnd) {
		logger.AutoCapture("off-screen fix", "%s (fixMinimizedRestore disabled)",
			p.WindowDesc(hwnd))
		p.CenterWindow(hwnd)
	}
}

func (p *Processor) CenterWindow(hwnd uintptr) {
	desktop := winapi.GetDesktopWindow()
	var targetRect winapi.RECT
	winapi.GetWindowRect(desktop, &targetRect)
	winapi.MoveWindow(hwnd,
		targetRect.Left+targetRect.Width()/4,
		targetRect.Top+targetRect.Height()/4,
		targetRect.Width()/2,
		targetRect.Height()/2,
		true,
	)
}

// showDesktop minimizes all top-level windows to show the desktop.
func (p *Processor) showDesktop() {
	desktop := winapi.GetDesktopWindow()
	hwnd := winapi.GetWindow(desktop, winapi.GW_CHILD)
	for hwnd != 0 {
		if winapi.IsWindowVisible(hwnd) && winapi.IsTopLevelWindow(hwnd) && !IsMinimized(hwnd) {
			winapi.ShowWindowAsync(hwnd, winapi.SW_MINIMIZE)
		}
		hwnd = winapi.GetWindow(hwnd, winapi.GW_HWNDNEXT)
	}
}
