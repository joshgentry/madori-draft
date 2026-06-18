package engine

import (
	"sync"
	"time"
	"unsafe"

	"durablewindows/internal/logger"
	"durablewindows/internal/winapi"
)

// minimizeToTrayWindows tracks windows that were right-click-minimized to tray.
var minimizeToTrayWindows = make(map[uintptr]bool)

// Tray icon tracking for parked windows.
var (
	parkedIconUID            = make(map[uintptr]uint32) // hwnd -> tray icon UID
	parkedIconRev            = make(map[uint32]uintptr) // UID -> hwnd
	nextParkedIconUID uint32 = 100                      // must match FirstParkedIconUID in tray.go
)

// minButtonCache stores per-hwnd minimize-button boundaries discovered by
// probing via SendMessageTimeout(WM_NCHITTEST). This is populated during
// capture (outside the hook context) and used in the hook for pixel-accurate
// detection even on custom-drawn title bars.
type minButtonRect struct {
	Left  int32
	Right int32
	Dpi   uint32 // DPI at time of probe; invalidated on DPI change
}

var (
	minButtonCache   = make(map[uintptr]minButtonRect)
	minButtonCacheMu sync.Mutex
)

// persistParkedWindows writes the current set of parked HWNDs to BoltDB so
// crashed sessions can recover. No-op if the store hasn't been initialised
// (one-shot CLI commands).
func persistParkedWindows() {
	if store == nil {
		return
	}
	hwnds := make([]uintptr, 0, len(minimizeToTrayWindows))
	for hwnd := range minimizeToTrayWindows {
		hwnds = append(hwnds, hwnd)
	}
	_ = store.SaveParkedWindows(hwnds)
}

// ---------------------------------------------------------------------------
// WH_MOUSE_LL hook (default)
// ---------------------------------------------------------------------------

var (
	mouseHookProcessor   *Processor
	mouseHookHandle      uintptr
	pendingMinButtonHWND uintptr // hwnd whose WM_RBUTTONDOWN on minimize button was eaten
)

func mouseHookProc(nCode, wParam, lParam uintptr) uintptr {
	// Per MSDN: if nCode < 0, the hook must call CallNextHookEx without
	// further processing and return its result.
	if int32(nCode) < winapi.HC_ACTION {
		return winapi.CallNextHookEx(mouseHookHandle, nCode, wParam, lParam)
	}

	if mouseHookProcessor == nil || !mouseHookProcessor.EnableMinimizeToTray {
		return winapi.CallNextHookEx(mouseHookHandle, nCode, wParam, lParam)
	}

	// Only care about right-button events.
	if wParam != winapi.WM_RBUTTONDOWN && wParam != winapi.WM_RBUTTONUP {
		return winapi.CallNextHookEx(mouseHookHandle, nCode, wParam, lParam)
	}

	hookStruct := (*winapi.MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
	pt := hookStruct.PT

	hwnd := winapi.WindowFromPoint(pt)
	if hwnd == 0 {
		return winapi.CallNextHookEx(mouseHookHandle, nCode, wParam, lParam)
	}

	// XAML island / custom-chrome windows (Windows Terminal, etc.) may
	// have WindowFromPoint return a child window instead of the top-level
	// owner.  Walk up to the root and check both the child and the root
	// against the minimize-target cache.
	if !winapi.IsTopLevelWindow(hwnd) {
		root := winapi.GetAncestor(hwnd, winapi.GA_ROOT)
		if root != 0 && root != hwnd {
			hwnd = root
		}
	}

	if !winapi.IsTopLevelWindow(hwnd) {
		return winapi.CallNextHookEx(mouseHookHandle, nCode, wParam, lParam)
	}

	// WM_RBUTTONDOWN: if on a minimize button, eat it so the system never
	// registers the press. Track the hwnd so we can eat the matching up.
	if wParam == winapi.WM_RBUTTONDOWN {
		if isMinimizeTarget(hwnd, pt) {
			pendingMinButtonHWND = hwnd
			// Eat the down — system never sees the button pressed.
			winapi.CallNextHookEx(mouseHookHandle, nCode, wParam, lParam)
			return 1
		}
		// Down on something else — clear any stale pending state.
		pendingMinButtonHWND = 0
		return winapi.CallNextHookEx(mouseHookHandle, nCode, wParam, lParam)
	}

	// WM_RBUTTONUP: if this up pairs with a down we ate, eat it too so the
	// system sees a balanced (zeroed) pair. Hide the window on up for a
	// more natural feel (the window vanishes when the user releases).
	if pendingMinButtonHWND != 0 && hwnd == pendingMinButtonHWND {
		pendingMinButtonHWND = 0

		// Re-check the minimize button geometry — the user may have moved
		// the mouse before releasing.
		if isMinimizeTarget(hwnd, pt) {
			winapi.ShowWindowAsync(hwnd, winapi.SW_HIDE)
			minimizeToTrayWindows[hwnd] = true
			persistParkedWindows()
			logger.Parking("minimized to tray", "%s", mouseHookProcessor.WindowDesc(hwnd))
			winapi.PostMessage(mouseHookProcessor.trayHWnd, winapi.WM_APP_PARKED, uintptr(hwnd), 0)
		}
		// Eat the up regardless so the system never sees an orphaned up.
		winapi.CallNextHookEx(mouseHookHandle, nCode, wParam, lParam)
		return 1
	}

	// Up on a different window — clear stale pending and pass through.
	pendingMinButtonHWND = 0
	return winapi.CallNextHookEx(mouseHookHandle, nCode, wParam, lParam)
}

// isMinimizeTarget checks whether a window under the cursor is a visible
// top-level window with a minimize button and the point is on that button.
func isMinimizeTarget(hwnd uintptr, pt winapi.POINT) bool {
	// If we have a probed cache entry, trust it over style bits.
	// Custom-chrome windows (Windows Terminal, etc.) may not set the
	// standard WS_MINIMIZEBOX / WS_CAPTION style flags even though
	// they have functional caption buttons.
	minButtonCacheMu.Lock()
	_, cached := minButtonCache[hwnd]
	minButtonCacheMu.Unlock()

	if cached {
		return isOnMinimizeButton(hwnd, pt)
	}

	// No cache — apply coarse pre-filters before falling through to
	// the geometry estimate.
	style := winapi.GetWindowLong(hwnd, winapi.GWL_STYLE)
	if (style&winapi.WS_MINIMIZEBOX) == 0 || (style&winapi.WS_CAPTION) == 0 {
		return false
	}
	if !winapi.IsWindowVisible(hwnd) {
		return false
	}
	return isOnMinimizeButton(hwnd, pt)
}

func (p *Processor) installMouseHook() {
	mouseHookProcessor = p
	mouseHookHandle = winapi.SetWindowsHookExDirect(winapi.WH_MOUSE_LL, mouseHookProc, 0, 0)
	logger.Parking("hook installed", "WH_MOUSE_LL (handle=%d)", mouseHookHandle)
}

func (p *Processor) uninstallMouseHook() {
	if mouseHookHandle != 0 {
		winapi.UnhookWindowsHookEx(mouseHookHandle)
		logger.Parking("hook removed", "")
		mouseHookHandle = 0
	}
	mouseHookProcessor = nil
}

// ---------------------------------------------------------------------------
// Polling fallback (opt-in via -window_parking_method poll)
// ---------------------------------------------------------------------------

var (
	pollProcessor *Processor
	pollWasDown   bool
	pollStopCh    chan struct{}
)

func (p *Processor) startPolling() {
	if !p.EnableMinimizeToTray {
		return
	}
	pollProcessor = p
	pollStopCh = make(chan struct{})
	go pollLoop(pollStopCh)
	logger.Parking("polling started", "minimize-to-tray (30ms interval)")
}

func (p *Processor) stopPolling() {
	if pollStopCh != nil {
		close(pollStopCh)
		pollStopCh = nil
	}
	pollProcessor = nil
	logger.Parking("polling stopped", "")
}

func pollLoop(stop chan struct{}) {
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			pollTick()
		}
	}
}

func pollTick() {
	p := pollProcessor
	if p == nil || !p.EnableMinimizeToTray || p.trayHWnd == 0 {
		return
	}

	state := winapi.GetAsyncKeyState(winapi.VK_RBUTTON)
	isDown := (uint16(state) & 0x8000) != 0

	if isDown && !pollWasDown {
		var pt winapi.POINT
		winapi.GetCursorPos(&pt)
		hwnd := winapi.WindowFromPoint(pt)
		if hwnd != 0 && winapi.IsTopLevelWindow(hwnd) {
			if isMinimizeTarget(hwnd, pt) {
				parkWindow(p, hwnd)
			}
		}
	}

	pollWasDown = isDown
}

// ---------------------------------------------------------------------------
// Unified API
// ---------------------------------------------------------------------------

// StartMinimizeToTray starts the minimize-to-tray detection mechanism.
// Uses WH_MOUSE_LL by default; pass -window_parking_method poll for the
// polling fallback.
func (p *Processor) StartMinimizeToTray() {
	if p.UsePollParking {
		p.startPolling()
	} else {
		p.installMouseHook()
	}
}

// StopMinimizeToTray stops the detection mechanism.
func (p *Processor) StopMinimizeToTray() {
	if p.UsePollParking {
		p.stopPolling()
	} else {
		p.uninstallMouseHook()
	}
}

// ---------------------------------------------------------------------------
// Shared: geometry and tray icons
// ---------------------------------------------------------------------------

// isOnMinimizeButton checks whether the point (in screen coordinates) falls
// on the window's minimize button. It prefers the probed cache (pixel-accurate
// for custom title bars) and falls back to geometry from system metrics.
func isOnMinimizeButton(hwnd uintptr, pt winapi.POINT) bool {
	// Check the probed cache first.
	minButtonCacheMu.Lock()
	cached, ok := minButtonCache[hwnd]
	minButtonCacheMu.Unlock()

	dpi := winapi.GetDpiForWindow(hwnd)
	if dpi == 0 {
		dpi = 96
	}

	// Get title bar Y range for both cache and geometry paths.
	titleTop, titleBottom := getTitleBarYRange(hwnd, dpi)

	if ok && cached.Dpi == dpi {
		// Use cached boundaries.
		if titleBottom == 0 || (pt.Y >= titleTop && pt.Y <= titleBottom) {
			hit := pt.X >= cached.Left && pt.X <= cached.Right
			if !hit {
				dist := cached.Left - pt.X
				if dist < 0 {
					dist = pt.X - cached.Right
				}
				if dist < 20 {
					logger.Parking("minimize button near-miss", "%s: pt=(%d,%d) cached=(%d..%d) title=(%d..%d) dpi=%d/%d",
						WindowDescLive(hwnd), pt.X, pt.Y, cached.Left, cached.Right, titleTop, titleBottom, cached.Dpi, dpi)
				}
			}
			return hit
		}
		// Y outside title bar — stale cache.
	}

	// Fallback: geometry-based estimate using GetTitleBarInfo for the
	// title bar rect and SM_CXSIZE for button width.
	if !ok {
		logger.Parking("minimize button no-cache", "%s geometry fallback", WindowDescLive(hwnd))
	}

	if titleBottom == 0 || pt.Y < titleTop || pt.Y > titleBottom {
		return false
	}

	// Use the visible window rect for X calculations.
	var rect winapi.RECT
	if !getVisibleWindowRect(hwnd, &rect) {
		return false
	}

	btnW := winapi.GetSystemMetrics(winapi.SM_CXSIZE) * int32(dpi) / 96
	style := winapi.GetWindowLong(hwnd, winapi.GWL_STYLE)
	hasMax := (style & winapi.WS_MAXIMIZEBOX) != 0
	buttonsRight := int32(1) // close
	if hasMax {
		buttonsRight = 2 // close + maximize
	}
	minRight := rect.Right - buttonsRight*btnW
	minLeft := minRight - btnW

	return pt.X >= minLeft && pt.X <= minRight
}

// getTitleBarYRange returns the Y-range of the title bar.  Uses
// GetTitleBarInfo if the window supports it (works for most custom-chrome
// windows), falling back to system metrics for older or non-DWM windows.
func getTitleBarYRange(hwnd uintptr, dpi uint32) (int32, int32) {
	var tbi winapi.TITLEBARINFO
	if winapi.GetTitleBarInfo(hwnd, &tbi) && tbi.RcTitleBar.Top < tbi.RcTitleBar.Bottom {
		return tbi.RcTitleBar.Top, tbi.RcTitleBar.Bottom
	}
	// Fallback: estimate from system metrics + visible rect.
	var rect winapi.RECT
	if getVisibleWindowRect(hwnd, &rect) {
		titleH := (winapi.GetSystemMetrics(winapi.SM_CYCAPTION) +
			winapi.GetSystemMetrics(winapi.SM_CYFRAME) +
			winapi.GetSystemMetrics(winapi.SM_CXPADDEDBORDER)) * int32(dpi) / 96
		return rect.Top, rect.Top + titleH
	}
	return 0, 0
}

// getVisibleWindowRect fills *rect with the visible bounds of the window,
// excluding DWM's invisible border/shadow. Falls back to GetWindowRect if
// DWM is unavailable (e.g. classic theme or older Windows).
func getVisibleWindowRect(hwnd uintptr, rect *winapi.RECT) bool {
	if winapi.DwmGetWindowAttribute(hwnd, winapi.DWMWA_EXTENDED_FRAME_BOUNDS,
		unsafe.Pointer(rect), uint32(unsafe.Sizeof(*rect))) {
		return true
	}
	return winapi.GetWindowRect(hwnd, rect)
}

// RefreshMinButtonCache queries the title bar button layout via
// WM_GETTITLEBARINFOEX and caches the minimize button rectangle.
// This is a single SendMessageTimeout call — no scanning needed.
func RefreshMinButtonCache(hwnd uintptr) {
	var tbiex winapi.TITLEBARINFOEX
	tbiex.CbSize = uint32(unsafe.Sizeof(tbiex))

	// WM_GETTITLEBARINFOEX returns the screen-coordinate rectangles of
	// every title-bar button.  The window manager handles the query, so
	// it works for custom-chrome windows (Windows Terminal, etc.) that
	// don't respond to per-pixel WM_NCHITTEST probing.
	_, lpdwResult := winapi.SendMessageTimeout(hwnd, winapi.WM_GETTITLEBARINFOEX, 0,
		uintptr(unsafe.Pointer(&tbiex)), winapi.SMTO_ABORTIFHUNG, 200)
	if lpdwResult == 0 {
		logger.Parking("probe minimize button", "%s WM_GETTITLEBARINFOEX failed — geometry fallback only",
			WindowDescLive(hwnd))
		return
	}

	minRect := tbiex.Rgrect[winapi.TitleBarMin]
	if minRect.Left >= minRect.Right || minRect.Top >= minRect.Bottom {
		// Minimize button is hidden or not present.
		return
	}

	dpi := winapi.GetDpiForWindow(hwnd)
	if dpi == 0 {
		dpi = 96
	}

	logger.Parking("probe minimize button", "%s cached minBtn=(%d..%d) dpi=%d",
		WindowDescLive(hwnd), minRect.Left, minRect.Right, dpi)

	minButtonCacheMu.Lock()
	minButtonCache[hwnd] = minButtonRect{
		Left:  minRect.Left,
		Right: minRect.Right,
		Dpi:   dpi,
	}
	minButtonCacheMu.Unlock()
}

// parkWindow hides the window and queues tray icon creation.
func parkWindow(p *Processor, hwnd uintptr) {
	winapi.ShowWindowAsync(hwnd, winapi.SW_HIDE)
	minimizeToTrayWindows[hwnd] = true
	persistParkedWindows()
	logger.Parking("minimized to tray", "%s", p.WindowDesc(hwnd))
	winapi.PostMessage(p.trayHWnd, winapi.WM_APP_PARKED, uintptr(hwnd), 0)
}

// SetTrayWindow stores the tray message window HWND so parked-window tray
// icons can route their callbacks to the correct window.
func (p *Processor) SetTrayWindow(hwnd uintptr) {
	p.trayHWnd = hwnd
}

// AddParkedTrayIcon creates a system-tray icon for a parked window.
func (p *Processor) AddParkedTrayIcon(hwnd uintptr) {
	uid := nextParkedIconUID
	nextParkedIconUID++
	parkedIconUID[hwnd] = uid
	parkedIconRev[uid] = hwnd

	title := GetWindowTitle(hwnd)
	if title == "" {
		title = GetWindowClassName(hwnd)
	}

	var hIcon uintptr
	if winapi.SendMessage(hwnd, winapi.WM_GETICON, winapi.ICON_SMALL, 0) != 0 {
		hIcon = winapi.SendMessage(hwnd, winapi.WM_GETICON, winapi.ICON_SMALL, 0)
	}
	if hIcon == 0 {
		hIcon = winapi.GetClassLongPtr(hwnd, winapi.GCLP_HICONSM)
	}
	if hIcon == 0 {
		hIcon = winapi.GetClassLongPtr(hwnd, winapi.GCLP_HICON)
	}

	nid := winapi.NOTIFYICONDATA{
		HWnd:             p.trayHWnd,
		UID:              uid,
		UFlags:           winapi.NIF_MESSAGE | winapi.NIF_TIP,
		UCallbackMessage: winapi.WM_TRAYICON,
		HIcon:            hIcon,
	}
	if hIcon != 0 {
		nid.UFlags |= winapi.NIF_ICON
	}
	copy16(nid.SzTip[:], title)
	winapi.ShellNotifyIcon(winapi.NIM_ADD, &nid)
	logger.Parking("parked icon added", "%s (uid=%d)", title, uid)
}

func (p *Processor) removeParkedTrayIcon(hwnd uintptr) {
	uid, ok := parkedIconUID[hwnd]
	if !ok {
		return
	}
	nid := winapi.NOTIFYICONDATA{
		HWnd: p.trayHWnd,
		UID:  uid,
	}
	winapi.ShellNotifyIcon(winapi.NIM_DELETE, &nid)
	delete(parkedIconUID, hwnd)
	delete(parkedIconRev, uid)
}

// FindParkedWindowByUID returns the HWND for a parked window given its tray icon UID.
func (p *Processor) FindParkedWindowByUID(uid uint32) uintptr {
	return parkedIconRev[uid]
}

// RestoreFromTray restores a window that was minimized to tray.
func (p *Processor) RestoreFromTray(hwnd uintptr) {
	if !minimizeToTrayWindows[hwnd] {
		return
	}

	p.removeParkedTrayIcon(hwnd)

	winapi.ShowWindow(hwnd, winapi.SW_RESTORE)
	winapi.SetForegroundWindow(hwnd)

	delete(minimizeToTrayWindows, hwnd)
	persistParkedWindows()

	if metricsList, ok := p.monitorApplications[p.curDisplayKey][hwnd]; ok && len(metricsList) > 0 {
		p.restoreSingleWindow(hwnd, metricsList[len(metricsList)-1])
	}

	logger.Parking("restored from tray", "%s", p.WindowDesc(hwnd))
}

// restoreOrphanedParkedWindows loads the parked-window list from BoltDB and
// restores any windows that are still alive. This recovers from a crash where
// RestoreAllParked() never ran. On a clean shutdown the list is empty.
func (p *Processor) restoreOrphanedParkedWindows() {
	if store == nil {
		return
	}
	hwnds, err := store.LoadParkedWindows()
	if err != nil || len(hwnds) == 0 {
		return
	}
	for _, hwnd := range hwnds {
		if !winapi.IsWindow(hwnd) {
			continue
		}
		// Populate the in-memory map so RestoreFromTray's guard passes.
		// It will remove the entry and persist the updated list.
		minimizeToTrayWindows[hwnd] = true
		logger.Parking("orphaned park restored", "%s (crash recovery)", p.WindowDesc(hwnd))
		p.RestoreFromTray(hwnd)
	}
	// Clear the bucket now that we've restored everything.
	_ = store.SaveParkedWindows(nil)
}

// RestoreParkedWindowsCmd is the CLI one-shot (-restore_parked_windows):
// loads parked-window HWNDs from the database, restores any that are still
// alive, clears the bucket, and exits. Useful as a manual recovery tool.
func (p *Processor) RestoreParkedWindowsCmd() {
	p.restoreOrphanedParkedWindows()
}

// IsMinimizedToTray returns true if the window was hidden via right-click minimize.
func (p *Processor) IsMinimizedToTray(hwnd uintptr) bool {
	return minimizeToTrayWindows[hwnd]
}

// GetMinimizedToTrayWindows returns all windows currently minimized to tray.
func (p *Processor) GetMinimizedToTrayWindows() []uintptr {
	var result []uintptr
	for hwnd := range minimizeToTrayWindows {
		if winapi.IsWindow(hwnd) {
			result = append(result, hwnd)
		} else {
			delete(minimizeToTrayWindows, hwnd)
		}
	}
	return result
}

// InvalidateMinButtonCache removes a window from the minimize-button cache.
// Call when the window resizes, moves, or its DPI changes.
func InvalidateMinButtonCache(hwnd uintptr) {
	minButtonCacheMu.Lock()
	delete(minButtonCache, hwnd)
	minButtonCacheMu.Unlock()
}

func copy16(dst []uint16, src string) {
	for i := 0; i < len(dst)-1 && i < len(src); i++ {
		dst[i] = uint16(src[i])
	}
}
