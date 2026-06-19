package engine

import (
	"fmt"
	"time"
	"unsafe"

	"durablewindows/internal/logger"
	"durablewindows/internal/models"
	"durablewindows/internal/winapi"
)

// BatchCaptureApplicationsOnCurrentDisplays captures all windows on the current display config.
func (p *Processor) BatchCaptureApplicationsOnCurrentDisplays() {
	// Ensure this goroutine's OS thread is per-monitor DPI-aware.
	winapi.SetThreadDpiAwarenessContext(winapi.DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)

	displayKey := p.curDisplayKey

	p.CaptureNewDisplayConfig(displayKey)
	p.CaptureWindowsOfInterest(displayKey)
	p.PersistToDB()
}

// CaptureNewDisplayConfig ensures the display key exists in our databases.
func (p *Processor) CaptureNewDisplayConfig(displayKey string) {
	if _, ok := p.monitorApplications[displayKey]; !ok {
		p.monitorApplications[displayKey] = make(map[uintptr][]*models.WindowMetrics)
	}
	if _, ok := p.deadApps[displayKey]; !ok {
		p.deadApps[displayKey] = make(map[uintptr][]*models.WindowMetrics)
	}
}

// CaptureWindowsOfInterest enumerates all top-level visible windows and captures their state.
func (p *Processor) CaptureWindowsOfInterest(displayKey string) {
	desktop := winapi.GetDesktopWindow()
	hwnd := winapi.GetWindow(desktop, winapi.GW_CHILD)
	count := 0
	changed := 0
	for hwnd != 0 {
		count++
		if p.CaptureWindow(hwnd, 0, time.Now(), displayKey) {
			changed++
		}
		hwnd = winapi.GetWindow(hwnd, winapi.GW_HWNDNEXT)
	}
	logger.AutoCapture("", "Capture cycle: %d windows scanned, %d changed", count, changed)
}

// CaptureWindow captures the state of a single window. Returns true if the
// window was actually captured (position changed from previous state).
func (p *Processor) CaptureWindow(hwnd uintptr, eventType uint32, captureTime time.Time, displayKey string) bool {
	// Skip windows we shouldn't track
	if !p.shouldTrackWindow(hwnd) {
		return false
	}

	metrics := p.captureWindowCore(hwnd, displayKey)
	if metrics == nil {
		return false
	}

	metrics.CaptureTime = captureTime

	// Append to monitor applications
	p.monitorApplications[displayKey][hwnd] = append(
		p.monitorApplications[displayKey][hwnd], metrics,
	)

	minimized := ""
	if metrics.IsMinimized {
		minimized = "(minimized)"
	}
	cloakedStr := ""
	var cloaked uint32
	if winapi.DwmGetWindowAttribute(hwnd, winapi.DWMWA_CLOAKED, unsafe.Pointer(&cloaked), 4) && cloaked != 0 {
		cloakedStr = " cloaked="
		switch cloaked {
		case 1:
			cloakedStr += "APP"
		case 2:
			cloakedStr += "SHELL"
		case 4:
			cloakedStr += "INHERITED"
		default:
			cloakedStr += fmt.Sprintf("%d", cloaked)
		}
	}
	logger.WindowEvent("window captured", "%s%s %s pos=(%d,%d %dx%d)",
		FormatWindowDesc(metrics.ProcessName, metrics.Title, metrics.ClassName),
		cloakedStr, minimized,
		metrics.ScreenPosition.Left, metrics.ScreenPosition.Top,
		metrics.ScreenPosition.Width(), metrics.ScreenPosition.Height())

	// Keep history bounded
	if len(p.monitorApplications[displayKey][hwnd]) > MaxHistoryQueueLength {
		p.monitorApplications[displayKey][hwnd] =
			p.monitorApplications[displayKey][hwnd][len(p.monitorApplications[displayKey][hwnd])-MaxHistoryQueueLength:]
	}
	return true
}

// captureWindowCore reads the current state of a window and returns a WindowMetrics.
func (p *Processor) captureWindowCore(hwnd uintptr, displayKey string) *models.WindowMetrics {
	m := &models.WindowMetrics{
		HWnd:    hwnd,
		IsValid: true,
	}

	// Get window rect
	var rect winapi.RECT
	if !winapi.GetWindowRect(hwnd, &rect) {
		return nil
	}
	m.ScreenPosition = rect

	// Get window placement
	wp := winapi.DefaultWINDOWPLACEMENT()
	if winapi.GetWindowPlacement(hwnd, &wp) {
		m.WindowPlacement = wp
	}

	// Get class name
	m.ClassName = GetWindowClassName(hwnd)

	// Get title
	m.Title = GetWindowTitle(hwnd)

	// Get style
	m.Style = winapi.GetWindowLong(hwnd, winapi.GWL_STYLE)
	m.ExtStyle = winapi.GetWindowLong(hwnd, winapi.GWL_EXSTYLE)

	// Window state
	m.IsMinimized = IsMinimized(hwnd)
	m.IsInvisible = !winapi.IsWindowVisible(hwnd)
	m.IsFullScreen = p.isFullScreen(hwnd)
	m.IsResizable = (m.Style & winapi.WS_SIZEBOX) != 0
	m.IsTopMost = (m.ExtStyle & winapi.WS_EX_TOPMOST) != 0

	// Process info
	_, pid := winapi.GetWindowThreadProcessId(hwnd)
	m.ProcessID = pid
	// Use the cached process name from shouldTrackWindow, not whatever
	// m.ProcessName defaults to (which is always "" — it was never set).
	if name, ok := p.windowProcessName[hwnd]; ok {
		m.ProcessName = name
	}

	// Window ID
	m.WindowID = uint32(hwnd & 0xFFFFFFFF)

	// Virtual desktop
	if p.vdManager != nil {
		m.GUID = winapi.GUIDToString(p.vdManager.GetWindowDesktopId(hwnd))
	}

	// Explorer directory tracking
	if m.ClassName == "CabinetWClass" || m.ClassName == "ExploreWClass" {
		m.Dir = GetExplorerFolderPath(hwnd)
	}

	// Process command-line from cache
	if cmdline, ok := p.processCmd[m.ProcessID]; ok {
		m.ProcessExePath = cmdline
	}

	// Compare with previous capture — skip if position hasn't changed
	if existing, ok := p.monitorApplications[displayKey][hwnd]; ok && len(existing) > 0 {
		last := existing[len(existing)-1]
		if last.EqualPlacement(m) {
			last.CaptureTime = time.Now()
			return nil
		}
	}

	return m
}

// shouldTrackWindow determines if a window should be captured.
// Minimized windows are included; hidden system windows, tool windows,
// and zero-size windows are excluded.
func (p *Processor) shouldTrackWindow(hwnd uintptr) bool {
	if !winapi.IsWindow(hwnd) {
		return false
	}
	if !winapi.IsTopLevelWindow(hwnd) {
		return false
	}
	if p.noRestoreWindows[hwnd] {
		return false
	}

	// Skip hidden windows (IME, GDI+, menus, etc.)
	// Minimized windows still pass IsWindowVisible.
	if !winapi.IsWindowVisible(hwnd) {
		return false
	}

	// Skip windows cloaked by the application (e.g. dormant UWP frames,
	// explicitly hidden overlay windows). Windows cloaked by the shell
	// (DWM_CLOAKED_SHELL) are on other virtual desktops — we still want
	// to capture those so their positions are saved.
	var cloaked uint32
	if winapi.DwmGetWindowAttribute(hwnd, winapi.DWMWA_CLOAKED, unsafe.Pointer(&cloaked), 4) {
		if cloaked == winapi.DWM_CLOAKED_APP || cloaked == winapi.DWM_CLOAKED_INHERITED {
			logger.Filtered("window filtered", "%s cloaked (APP/INHERITED)", p.WindowDesc(hwnd))
			return false
		}
	}

	// Skip zero-size or tiny windows
	var rect winapi.RECT
	if winapi.GetWindowRect(hwnd, &rect) {
		if rect.Width() <= 0 || rect.Height() <= 0 || rect.Width()*rect.Height() < 3000 {
			return false
		}
	}

	// Skip tool windows (overlays, floating widgets)
	if !p.captureFloatingWindow {
		extStyle := winapi.GetWindowLong(hwnd, winapi.GWL_EXSTYLE)
		if (extStyle & winapi.WS_EX_TOOLWINDOW) != 0 {
			return false
		}
	}

	// Skip known system classes
	className := GetWindowClassName(hwnd)
	if isSystemClass(className) {
		return false
	}

	// Skip taskbar windows
	if isTaskBarWindow(hwnd) {
		return false
	}

	// Skip known overlay titles
	title := GetWindowTitle(hwnd)
	if isOverlayTitle(title) {
		return false
	}

	// Must look like a real user window (has title, caption, or is resizable)
	if !isRealUserWindow(hwnd, className, title) {
		return false
	}
	// Resolve process name and apply filters
	_, pid := winapi.GetWindowThreadProcessId(hwnd)
	processName := p.resolveProcessName(pid)

	// Skip dormant UWP ghost frames. When a UWP app (like Settings) is
	// closed, Windows keeps an ApplicationFrameWindow that is shell-cloaked
	// but still on the current virtual desktop. Real virtual-desktop windows
	// are shell-cloaked AND on a different desktop. Ghost frames are
	// shell-cloaked AND on the current desktop — they have no visible UI.
	if className == "ApplicationFrameWindow" {
		var cloaked uint32
		if winapi.DwmGetWindowAttribute(hwnd, winapi.DWMWA_CLOAKED, unsafe.Pointer(&cloaked), 4) &&
			cloaked == winapi.DWM_CLOAKED_SHELL &&
			p.vdManager != nil &&
			p.vdManager.IsWindowOnCurrentVirtualDesktop(hwnd) {
			logger.Filtered("window filtered", "%s uwp ghost (shell-cloaked on current desktop)", p.WindowDesc(hwnd))
			return false
		}
	}

	// If care list is set, only track those processes
	if len(p.careProcess) > 0 {
		if !p.careProcess[processName] {
			return false
		}
	}

	// Ignore listed processes
	if p.ignoreProcess[processName] {
		return false
	}

	// Cache process name
	p.windowProcessName[hwnd] = processName

	return true
}

// isSystemClass returns true for known system-owned window class names
// that should never be tracked.
func isSystemClass(className string) bool {
	switch className {
	case "OLEChannelWnd", "CLIPBRDWNDCLASS", "DirectUIHWND",
		"CicMarshalWnd", "CiceroUIWndFrame",
		"Default IME", "MSCTFIME UI", "GDI+ Window",
		"Shell_TrayWnd", "TrayNotifyWnd",
		"Windows.UI.Core.CoreWindow", "Progman",
		"WorkerW", "DV2ControlHost",
		"Button", "Static", "ToolbarWindow32",
		"SysListView32", "SysTreeView32",
		"#32768", "#32769", "#32770", "#32771", "#32772":
		return true
	}
	return false
}

// resolveProcessName gets the executable name for a PID, with caching.
func (p *Processor) resolveProcessName(pid uint32) string {
	if pid == 0 {
		return ""
	}
	return winapi.GetProcessName(pid)
}

// debugWindowEvent logs debug info for windows matching debug process filters.
func (p *Processor) debugWindowEvent(hwnd uintptr, processName, action string) {
	if p.debugProcess[processName] {
		p.debugWindows[hwnd] = true
	}
}

// isFullScreen checks if a window covers an entire display (exclusive
// full-screen, borderless windowed, or browser full-screen). A maximized
// window won't match because GetWindowRect returns the work area (excluding
// the taskbar), which is smaller than the full monitor rect.
func (p *Processor) isFullScreen(hwnd uintptr) bool {
	var rect winapi.RECT
	if !winapi.GetWindowRect(hwnd, &rect) {
		return false
	}
	for _, d := range winapi.GetDisplays() {
		if rect.Equals(d.Position) {
			return true
		}
	}
	return false
}

// EndDisplaySession finalizes capture before a display change.
func (p *Processor) EndDisplaySession() {
	p.CancelCaptureTimer()
	p.BatchCaptureApplicationsOnCurrentDisplays()
}

// UndoCapture rewinds captures after the given time.
func (p *Processor) UndoCapture(after time.Time) {
	for _, apps := range p.monitorApplications {
		for hwnd, metricsList := range apps {
			cutIdx := -1
			for i, m := range metricsList {
				if m.CaptureTime.After(after) {
					cutIdx = i
					break
				}
			}
			if cutIdx >= 0 {
				apps[hwnd] = metricsList[:cutIdx]
			}
		}
	}
}

// isTaskBarWindow returns true if hwnd is a taskbar window.
func isTaskBarWindow(hwnd uintptr) bool {
	className := GetWindowClassName(hwnd)
	if className == "Shell_TrayWnd" || className == "Shell_SecondaryTrayWnd" {
		return true
	}
	// Also check if any ancestor is the taskbar
	ancestor := winapi.GetAncestor(hwnd, winapi.GA_ROOT)
	if ancestor != 0 && GetWindowClassName(ancestor) == "Shell_TrayWnd" {
		return true
	}
	return false
}

var overlayTitlePrefixes = []string{
	"NVIDIA GeForce Overlay", "amd dvr overlay",
	"notificationtoasts",
	"Task Switching",
	"Discord Overlay", "Steam Overlay",
	"Xbox Game Bar", "Game Bar",
}

// isOverlayTitle returns true for known GPU/game overlay window titles.
func isOverlayTitle(title string) bool {
	for _, prefix := range overlayTitlePrefixes {
		if len(title) >= len(prefix) && title[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isRealUserWindow returns true if the window looks like a user application window
// rather than a system artifact, popup, or internal utility window.
func isRealUserWindow(hwnd uintptr, className, title string) bool {
	// Has a meaningful title
	if title != "" {
		return true
	}
	// Has a caption/title bar (standard window chrome)
	style := winapi.GetWindowLong(hwnd, winapi.GWL_STYLE)
	if (style & winapi.WS_CAPTION) != 0 {
		return true
	}
	// Is resizable (user windows typically are)
	if (style&winapi.WS_SIZEBOX) != 0 || (style&winapi.WS_THICKFRAME) != 0 {
		return true
	}
	return false
}
