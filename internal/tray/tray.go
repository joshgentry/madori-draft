package tray

import (
	"runtime"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	"durablewindows/internal/engine"
	"durablewindows/internal/logger"
	"durablewindows/internal/winapi"
)

// TrayApp manages the system tray icon and the hidden message window.
type TrayApp struct {
	mu sync.Mutex

	hwnd       uintptr
	notifyIcon winapi.NOTIFYICONDATA
	menu       uintptr

	processor *engine.Processor

	// State
	running      bool
	iconBusy     bool
	silent       bool
	notification bool
	gui          bool

	// singleClickPending: a timer is armed; will fire restore on expiry.
	singleClickPending bool
	// suppressClick: set by WM_LBUTTONDBLCLK to ignore the spurious
	// WM_LBUTTONUP that Shell_NotifyIcon sends for the second click.
	suppressClick bool
	// clickDownTime: timestamp of last WM_LBUTTONDOWN, used to
	// distinguish a quick click (<200ms) from a click-and-hold/drag.
	clickDownTime time.Time

	// Icons
	idleIcon   uintptr
	busyIcon   uintptr
	updateIcon uintptr
}

// Window class name
const windowClassName = "DurableWindowsMsgWindow"

// Timer IDs for the hidden window
const (
	TimerSingleClick       = 1
	TimerClearNotification = 2
)

// Menu command IDs
const (
	CmdCaptureSnapshot  = 1003
	CmdRestoreSnapshot  = 1004
	CmdPauseResume      = 1005
	CmdRestoreAllParked = 1006
	CmdExit             = 1007

	// Tray icon UIDs: 1 = main icon, 100+ = parked windows.
	FirstParkedIconUID = 100
)

// NewTrayApp creates a new system tray application.
func NewTrayApp(proc *engine.Processor) *TrayApp {
	return &TrayApp{
		processor: proc,
		gui:       true,
	}
}

func (t *TrayApp) SetSilent(s bool) { t.silent = s }
func (t *TrayApp) SetGUI(g bool)    { t.gui = g }

func (t *TrayApp) SetNotification(n bool) { t.notification = n }

// startEngine defers processor initialization until the message pump is active.
// SetWinEventHook requires an active message loop on the calling thread.
func startEngine() {
	t := globalTrayApp
	if t == nil || t.processor == nil {
		return
	}
	t.processor.Start(false, false)
	t.processor.SetCallbacks(engine.Callbacks{
		ShowRestoreTip:            t.ShowRestoreTip,
		HideRestoreTip:            t.HideRestoreTip,
		ShowSnapshotCaptureTip:    t.ShowSnapshotCaptureTip,
		ShowSnapshotRestoreTip:    t.ShowSnapshotRestoreTip,
		EnableRestoreMenu:         t.EnableRestoreMenu,
		EnableRestoreSnapshotMenu: t.EnableRestoreSnapshotMenu,
		ChangeIconText:            t.ChangeIconText,
	})
}

// windowProc is the main window procedure.
func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case winapi.WM_APP_START:
		logger.Tray("", "WM_APP_START received, starting engine...")
		startEngine()
		if t := globalTrayApp; t != nil {
			logger.Parking("", "Starting minimize-to-tray...")
			t.processor.StartMinimizeToTray()
		}
		return 0
	case winapi.WM_APP_PARKED:
		if t := globalTrayApp; t != nil {
			t.processor.AddParkedTrayIcon(uintptr(wParam))
		}
		return 0
	case winapi.WM_APP_SHUTDOWN:
		logger.Tray("", "WM_APP_SHUTDOWN received, initiating graceful shutdown...")
		if t := globalTrayApp; t != nil {
			t.Quit()
		}
		return 0
	case winapi.WM_TRAYICON:
		return handleTrayMessage(wParam, lParam)
	case winapi.WM_DISPLAYCHANGE:
		onDisplayChange()
		return 0
	case winapi.WM_WTSSESSION_CHANGE:
		onSessionChange(uint32(wParam))
		return 0
	case winapi.WM_POWERBROADCAST:
		if wParam == winapi.PBT_APMSUSPEND {
			onPowerSuspend()
		} else if wParam == winapi.PBT_APMRESUMEAUTOMATIC || wParam == winapi.PBT_APMRESUMESUSPEND {
			onPowerResume()
		}
		return 1
	case winapi.WM_TIMER:
		if wParam == TimerSingleClick {
			logger.Tray("", "WM_TIMER(TimerSingleClick) fired — handleSingleClick")
			winapi.KillTimer(hwnd, TimerSingleClick)
			handleSingleClick()
		} else if wParam == TimerClearNotification {
			winapi.KillTimer(hwnd, TimerClearNotification)
			if t := globalTrayApp; t != nil {
				t.clearNotificationBalloon()
			}
		}
		return 0
	case winapi.WM_DESTROY:
		winapi.PostQuitMessage(0)
		return 0
	case winapi.WM_COMMAND:
		cmdID := uint32(wParam & 0xFFFF)
		onMenuCommand(cmdID)
		return 0
	case winapi.WM_CLOSE:
		winapi.DestroyWindow(hwnd)
		return 0
	}
	return winapi.DefWindowProc(hwnd, msg, wParam, lParam)
}

// Run starts the message pump and system tray icon. Blocks until quit.
func (t *TrayApp) Run() error {
	// Pin this goroutine to the current OS thread. Windows message queues are
	// per-thread: CreateWindowEx attaches the queue to the calling thread, and
	// GetMessage retrieves from the calling thread's queue. If Go's runtime
	// reschedules this goroutine to a different thread between those calls,
	// GetMessage blocks forever on an empty queue while the posted messages
	// sit in the original thread's queue. LockOSThread prevents migration.
	runtime.LockOSThread()

	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = true
	t.mu.Unlock()

	winapi.SetProcessDpiAwarenessContext(winapi.DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)

	hInstance := winapi.GetModuleHandle(nil)

	className := windows.StringToUTF16Ptr(windowClassName)
	var wc winapi.WNDCLASSEX
	wc.Style = 0
	wc.LpfnWndProc = syscall.NewCallback(windowProc)
	wc.HInstance = hInstance
	wc.LpszClassName = className

	logger.Tray("", "RegisterClassEx %s...", windowClassName)
	if _, err := winapi.RegisterClassEx(&wc); err != nil {
		logger.Error("", "RegisterClassEx failed: %v", err)
		return err
	}
	logger.Tray("", "RegisterClassEx succeeded")

	logger.Tray("", "CreateWindowEx...")
	hwnd := winapi.CreateWindowEx(
		0,
		className,
		windows.StringToUTF16Ptr("DurableWindows"),
		0,
		0, 0, 0, 0,
		0, 0, hInstance,
		nil,
	)
	if hwnd == 0 {
		logger.Error("", "CreateWindowEx failed: %v", syscall.GetLastError())
		return syscall.GetLastError()
	}
	t.hwnd = hwnd
	t.processor.SetTrayWindow(hwnd)

	// Install console control handler so Ctrl+C triggers graceful
	// shutdown (restore parked windows) instead of instant ExitProcess.
	installConsoleCtrlHandler(hwnd)

	logger.Tray("", "CreateWindowEx succeeded, hwnd=0x%x", hwnd)

	winapi.WTSRegisterSessionNotification(hwnd, 0)

	// Load icons before adding tray icon
	logger.Tray("", "Loading icons...")
	t.initIcons()
	logger.Tray("", "Icons loaded")

	// Add tray icon
	logger.Tray("", "Adding tray icon...")
	t.addTrayIcon()
	logger.Tray("", "Tray icon added")

	// Defer engine start until the message pump is running.
	// SetWinEventHook requires an active message loop on the calling thread.
	// PostMessage queues the request; it dispatches once GetMessage runs.
	logger.Tray("", "Posting WM_APP_START...")
	if !winapi.PostMessage(hwnd, winapi.WM_APP_START, 0, 0) {
		logger.Error("", "PostMessage WM_APP_START failed")
	}
	logger.Tray("", "Entering message pump...")

	// Message pump — blocks until WM_QUIT
	var msg winapi.MSG
	for winapi.GetMessage(&msg, 0, 0, 0) > 0 {
		winapi.TranslateMessage(&msg)
		winapi.DispatchMessage(&msg)
	}

	// Restore any windows that were hidden to tray before we exit.
	// Without this they'd stay invisible forever since our hook is
	// the only thing that can restore them.
	t.processor.RestoreAllParked()

	// Cleanup
	t.processor.StopMinimizeToTray()
	t.removeTrayIcon()
	winapi.WTSUnRegisterSessionNotification(hwnd)
	return nil
}

// Quit posts a quit message to the message pump.
func (t *TrayApp) Quit() {
	if t.hwnd != 0 {
		winapi.PostQuitMessage(0)
	}
}

func (t *TrayApp) addTrayIcon() {
	nid := winapi.NOTIFYICONDATA{
		HWnd:             t.hwnd,
		UID:              1,
		UFlags:           winapi.NIF_MESSAGE | winapi.NIF_ICON | winapi.NIF_TIP,
		UCallbackMessage: winapi.WM_TRAYICON,
		HIcon:            t.idleIcon,
	}
	copy(nid.SzTip[:], windows.StringToUTF16("DurableWindows"))
	winapi.ShellNotifyIcon(winapi.NIM_ADD, &nid)
	t.notifyIcon = nid
}

func (t *TrayApp) removeTrayIcon() {
	winapi.ShellNotifyIcon(winapi.NIM_DELETE, &t.notifyIcon)
}

func (t *TrayApp) ShowRestoreTip() {
	if t.silent || t.iconBusy {
		return
	}
	t.iconBusy = true
	if t.notification {
		nid := t.notifyIcon
		nid.UFlags = winapi.NIF_INFO
		nid.DwInfoFlags = winapi.NIIF_INFO
		copy(nid.SzInfo[:], windows.StringToUTF16("Restoring window layout..."))
		copy(nid.SzInfoTitle[:], windows.StringToUTF16("DurableWindows"))
		winapi.ShellNotifyIcon(winapi.NIM_MODIFY, &nid)
		t.startNotificationTimer(3000)
	}
	nid := t.notifyIcon
	nid.HIcon = t.busyIcon
	nid.UFlags = winapi.NIF_ICON
	winapi.ShellNotifyIcon(winapi.NIM_MODIFY, &nid)
}

func (t *TrayApp) HideRestoreTip(showIcon bool) {
	t.iconBusy = false
	nid := t.notifyIcon
	nid.HIcon = t.idleIcon
	nid.UFlags = winapi.NIF_ICON
	winapi.ShellNotifyIcon(winapi.NIM_MODIFY, &nid)
}

// clearNotificationBalloon dismisses the notification balloon by sending
// an NIM_MODIFY with an empty SzInfo. Windows clears the balloon immediately.
func (t *TrayApp) clearNotificationBalloon() {
	nid := t.notifyIcon
	nid.UFlags = winapi.NIF_INFO
	nid.DwInfoFlags = winapi.NIIF_NONE
	nid.SzInfo = [256]uint16{}
	nid.SzInfoTitle = [64]uint16{}
	winapi.ShellNotifyIcon(winapi.NIM_MODIFY, &nid)
}

// startNotificationTimer arms a one-shot timer that clears the balloon after
// the given duration. Call this after showing a notification balloon.
func (t *TrayApp) startNotificationTimer(durationMs int) {
	if t.hwnd != 0 {
		winapi.SetTimer(t.hwnd, TimerClearNotification, uint32(durationMs), 0)
	}
}

// snapshotName returns the display character for a snapshot ID.
// IDs 0-9 → "0"..."9", 10-35 → "a"..."z", 36 → "`", 37 → "undo".
func snapshotName(id int) string {
	switch {
	case id >= 0 && id <= 9:
		return string(rune('0' + id))
	case id >= 10 && id <= 35:
		return string(rune('a' + id - 10))
	case id == 36:
		return "`"
	default:
		return "?"
	}
}

func (t *TrayApp) ShowSnapshotCaptureTip(id int) {
	if t.silent || !t.notification {
		return
	}
	nid := t.notifyIcon
	nid.UFlags = winapi.NIF_INFO
	nid.DwInfoFlags = winapi.NIIF_INFO
	msg := "Window layout snapshot " + snapshotName(id) + " captured..."
	copy(nid.SzInfo[:], windows.StringToUTF16(msg))
	copy(nid.SzInfoTitle[:], windows.StringToUTF16("DurableWindows"))
	winapi.ShellNotifyIcon(winapi.NIM_MODIFY, &nid)
	t.startNotificationTimer(3000)
}

func (t *TrayApp) ShowSnapshotRestoreTip(id int) {
	if t.silent || !t.notification {
		return
	}
	t.iconBusy = true
	nid := t.notifyIcon
	nid.UFlags = winapi.NIF_INFO | winapi.NIF_ICON
	nid.DwInfoFlags = winapi.NIIF_INFO
	nid.HIcon = t.busyIcon
	msg := "Window layout snapshot " + snapshotName(id) + " restored..."
	copy(nid.SzInfo[:], windows.StringToUTF16(msg))
	copy(nid.SzInfoTitle[:], windows.StringToUTF16("DurableWindows"))
	winapi.ShellNotifyIcon(winapi.NIM_MODIFY, &nid)
	t.startNotificationTimer(3000)
}

func (t *TrayApp) EnableRestoreMenu(enableDB bool)       {}
func (t *TrayApp) EnableRestoreSnapshotMenu(enable bool) {}

func (t *TrayApp) ChangeIconText(text string) {
	if len(text) > 40 {
		text = text[:40]
	}
	nid := t.notifyIcon
	copy(nid.SzTip[:], windows.StringToUTF16(text))
	nid.UFlags = winapi.NIF_TIP
	winapi.ShellNotifyIcon(winapi.NIM_MODIFY, &nid)
}

// --- Global tray app reference ---

var globalTrayApp *TrayApp

// ctrlCHwnd stores the hidden message window HWND for the console
// control handler. Set once during Run(); read from a separate thread
// by the console handler callback.
var ctrlCHwnd uintptr

// consoleCtrlHandler is the PHANDLER_ROUTINE callback registered with
// SetConsoleCtrlHandler. It runs on a separate thread. When Ctrl+C or
// Ctrl+Break is detected, it posts WM_APP_SHUTDOWN to the hidden message
// window, triggering graceful shutdown including RestoreAllParked().
func consoleCtrlHandler(dwCtrlType uintptr) uintptr {
	switch uint32(dwCtrlType) {
	case windows.CTRL_C_EVENT, windows.CTRL_BREAK_EVENT:
		if hwnd := ctrlCHwnd; hwnd != 0 {
			winapi.PostMessage(hwnd, winapi.WM_APP_SHUTDOWN, 0, 0)
		}
		// Return TRUE — we handled the signal; suppress default handler.
		return 1
	}
	// Return FALSE — pass unhandled signals to the next handler.
	return 0
}

// installConsoleCtrlHandler registers the console control handler.
// Must be called after the hidden message window is created.
func installConsoleCtrlHandler(hwnd uintptr) {
	ctrlCHwnd = hwnd
	callback := syscall.NewCallback(consoleCtrlHandler)
	if ok := winapi.SetConsoleCtrlHandler(callback, true); !ok {
		logger.Tray("", "SetConsoleCtrlHandler failed (non-fatal, no console?)")
		ctrlCHwnd = 0
	} else {
		logger.Tray("", "Console Ctrl+C handler installed (hwnd=0x%x)", hwnd)
	}
}

func (t *TrayApp) SetGlobalTrayApp(app *TrayApp) {
	globalTrayApp = app
}

func handleTrayMessage(wParam, lParam uintptr) uintptr {
	t := globalTrayApp
	if t == nil {
		return 0
	}

	// Parked window icons have UID >= FirstParkedIconUID. They respond
	// to a single left-click by restoring the window.
	if uid := uint32(wParam); uid >= FirstParkedIconUID {
		mouseMsg := uint32(lParam & 0xFFFF)
		if mouseMsg == winapi.WM_LBUTTONUP {
			if hwnd := t.processor.FindParkedWindowByUID(uid); hwnd != 0 {
				logger.Parking("parked icon clicked", "uid=%d, restoring window", uid)
				t.processor.RestoreFromTray(hwnd)
			}
		} else if mouseMsg == winapi.WM_RBUTTONUP {
			// Right-click on parked icon: show a simple "Restore" option.
			// For now just restore immediately — same as left-click.
			if hwnd := t.processor.FindParkedWindowByUID(uid); hwnd != 0 {
				logger.Parking("parked icon right-clicked", "uid=%d, restoring window", uid)
				t.processor.RestoreFromTray(hwnd)
			}
		}
		return 0
	}

	// Main tray icon (UID=1) — the existing click/double-click logic.
	switch uint32(lParam & 0xFFFF) {
	case winapi.WM_RBUTTONUP:
		t.showContextMenu()
	case winapi.WM_LBUTTONDOWN:
		t.clickDownTime = time.Now()
	case winapi.WM_LBUTTONUP:
		// Shell_NotifyIcon sends an extra WM_LBUTTONUP for the
		// second click's release, arriving after WM_LBUTTONDBLCLK.
		// suppressClick (set by DBLCLK) filters it out.
		if t.suppressClick {
			t.suppressClick = false
			break
		}
		// If the button was held too long, this wasn't a click —
		// it was a drag or click-and-hold. Don't arm the timer.
		held := time.Since(t.clickDownTime)
		if held > 200*time.Millisecond {
			logger.Tray("", "Tray WM_LBUTTONUP — held %dms, ignoring (drag/hold)", held.Milliseconds())
			break
		}
		dct := winapi.GetDoubleClickTime()
		logger.Tray("", "Tray WM_LBUTTONUP — held %dms, SetTimer(%dms)", held.Milliseconds(), dct)
		winapi.KillTimer(t.hwnd, TimerSingleClick)
		t.singleClickPending = true
		winapi.SetTimer(t.hwnd, TimerSingleClick, dct, 0)
	case winapi.WM_LBUTTONDBLCLK:
		t.suppressClick = true
		t.singleClickPending = false
		killed := winapi.KillTimer(t.hwnd, TimerSingleClick)
		logger.Tray("", "Tray WM_LBUTTONDBLCLK — KillTimer=%v, calling TakeSnapshot", killed)
		t.onTrayDoubleClick()
	}
	return 0
}

func (t *TrayApp) showContextMenu() {
	if t.menu == 0 {
		t.menu = t.createContextMenu()
	}
	var pos winapi.POINT
	winapi.GetCursorPos(&pos)
	winapi.SetForegroundWindow(t.hwnd)
	winapi.TrackPopupMenu(t.menu, winapi.TPM_RIGHTBUTTON|winapi.TPM_BOTTOMALIGN,
		pos.X, pos.Y, 0, t.hwnd, nil)
}

func (t *TrayApp) createContextMenu() uintptr {
	menu := winapi.CreatePopupMenu()
	addMenuItem(menu, CmdCaptureSnapshot, "&Capture snapshot")
	addMenuItem(menu, CmdRestoreSnapshot, "&Restore snapshot")
	winapi.AppendMenu(menu, winapi.MF_SEPARATOR, 0, nil)
	addMenuItem(menu, CmdPauseResume, "Pause auto &restore")
	winapi.AppendMenu(menu, winapi.MF_SEPARATOR, 0, nil)
	addMenuItem(menu, CmdRestoreAllParked, "Restore all &parked windows")
	winapi.AppendMenu(menu, winapi.MF_SEPARATOR, 0, nil)
	addMenuItem(menu, CmdExit, "E&xit")
	return menu
}

func addMenuItem(menu uintptr, id uint32, text string) {
	winapi.AppendMenu(menu, winapi.MF_STRING, uintptr(id), windows.StringToUTF16Ptr(text))
}

func handleSingleClick() {
	t := globalTrayApp
	if t == nil || !t.singleClickPending {
		return
	}
	t.singleClickPending = false
	t.onTrayClick()
}

func (t *TrayApp) onTrayClick()       { t.processor.RestoreSnapshot(0) }
func (t *TrayApp) onTrayDoubleClick() { t.processor.TakeSnapshot(0) }

func onMenuCommand(cmdID uint32) {
	t := globalTrayApp
	if t == nil {
		return
	}
	logger.Tray("", "Menu command: %d", cmdID)
	switch cmdID {
	case CmdCaptureSnapshot:
		name := EnterSnapshotName()
		if name != 0 {
			id := snapshotCharToID(name)
			if id >= 0 {
				logger.Snapshot("snapshot captured", "snapshot %d via menu", id)
				t.processor.TakeSnapshot(id)
			}
		}
	case CmdRestoreSnapshot:
		name := EnterSnapshotName()
		if name != 0 {
			id := snapshotCharToID(name)
			if id >= 0 {
				logger.Snapshot("snapshot restored", "snapshot %d via menu", id)
				t.processor.RestoreSnapshot(id)
			}
		}
	case CmdPauseResume:
		t.processor.PauseAutoRestore = !t.processor.PauseAutoRestore
		logger.AutoCapture("auto-restore toggled", "%s", map[bool]string{true: "paused", false: "resumed"}[t.processor.PauseAutoRestore])
	case CmdRestoreAllParked:
		logger.Parking("restored all parked", "via menu")
		t.processor.RestoreAllParked()
	case CmdExit:
		t.Quit()
	}
}

func snapshotCharToID(c byte) int {
	if c == '`' || c == '~' {
		return 36
	}
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'z' {
		return int(c - 'a' + 10)
	}
	if c >= 'A' && c <= 'Z' {
		return int(c - 'A' + 10)
	}
	return -1
}

func onDisplayChange() {}
func onSessionChange(reason uint32) {
	t := globalTrayApp
	if t == nil {
		return
	}
	switch reason {
	case winapi.WTS_SESSION_LOCK:
		t.processor.OnSessionLock()
	case winapi.WTS_SESSION_UNLOCK:
		if t.processor.PromptSessionRestore {
			winapi.MessageBox(0, "Your session has been unlocked. Restore window layout?",
				"DurableWindows", winapi.MB_OK|winapi.MB_ICONINFORMATION)
		}
		t.processor.OnSessionUnlock()
	}
}

func onPowerSuspend() {
	if globalTrayApp != nil {
		globalTrayApp.processor.OnPowerSuspend()
	}
}

func onPowerResume() {
	if globalTrayApp != nil {
		globalTrayApp.processor.OnPowerResume()
	}
}
