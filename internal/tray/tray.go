package tray

import (
	"runtime"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	"madori/internal/engine"
	"madori/internal/logger"
	"madori/internal/winapi"
)

// TrayApp manages the system tray icon and the hidden message window.
type TrayApp struct {
	mu sync.Mutex

	hwnd       uintptr
	notifyIcon winapi.NOTIFYICONDATA
	menu       uintptr

	processor *engine.Processor

	// State
	running              bool
	iconBusy             bool
	silent               bool
	notification         bool
	notificationLingerMs int
	notificationShowing  bool // true while a notification balloon is visible

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
const windowClassName = "MadoriMsgWindow"

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
	}
}

func (t *TrayApp) SetSilent(s bool) { t.silent = s }

func (t *TrayApp) SetNotification(n bool) { t.notification = n }

// SetNotificationLinger sets how long notification balloons remain visible (ms).
// Default 3000. Call before Run().
func (t *TrayApp) SetNotificationLinger(durationMs int) {
	if durationMs < 500 {
		durationMs = 500
	}
	t.notificationLingerMs = durationMs
}

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
		logger.Tray(logger.LevelInfo, "", "WM_APP_START received, starting engine...")
		startEngine()
		if t := globalTrayApp; t != nil {
			logger.Parking(logger.LevelInfo, "", "Starting minimize-to-tray...")
			t.processor.StartTrayParking()
		}
		return 0
	case winapi.WM_APP_PARKED:
		if t := globalTrayApp; t != nil {
			t.processor.AddParkedTrayIcon(uintptr(wParam))
		}
		return 0
	case winapi.WM_APP_SHUTDOWN:
		logger.Tray(logger.LevelInfo, "", "WM_APP_SHUTDOWN received, initiating graceful shutdown...")
		if t := globalTrayApp; t != nil {
			t.Quit()
		}
		return 0
	case winapi.WM_TRAYICON:
		return handleTrayMessage(wParam, lParam)
	case winapi.WM_DISPLAYCHANGE:
		if t := globalTrayApp; t != nil {
			t.processor.NoteDisplayChange()
		}
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
			logger.Tray(logger.LevelDebug, "", "WM_TIMER(TimerSingleClick) fired — handleSingleClick")
			winapi.KillTimer(hwnd, TimerSingleClick)
			handleSingleClick()
		} else if wParam == TimerClearNotification {
			winapi.KillTimer(hwnd, TimerClearNotification)
			if t := globalTrayApp; t != nil {
				t.dismissNotificationBalloon()
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

	hInstance := winapi.GetModuleHandle(nil)

	className := windows.StringToUTF16Ptr(windowClassName)
	var wc winapi.WNDCLASSEX
	wc.Style = 0
	wc.LpfnWndProc = syscall.NewCallback(windowProc)
	wc.HInstance = hInstance
	wc.LpszClassName = className

	logger.Tray(logger.LevelInfo, "", "RegisterClassEx %s...", windowClassName)
	if _, err := winapi.RegisterClassEx(&wc); err != nil {
		logger.Error(logger.LevelError, "", "RegisterClassEx failed: %v", err)
		return err
	}
	logger.Tray(logger.LevelInfo, "", "RegisterClassEx succeeded")

	logger.Tray(logger.LevelInfo, "", "CreateWindowEx...")
	hwnd := winapi.CreateWindowEx(
		0,
		className,
		windows.StringToUTF16Ptr("Madori"),
		0,
		0, 0, 0, 0,
		0, 0, hInstance,
		nil,
	)
	if hwnd == 0 {
		logger.Error(logger.LevelError, "", "CreateWindowEx failed: %v", syscall.GetLastError())
		return syscall.GetLastError()
	}
	t.hwnd = hwnd
	t.processor.SetTrayWindow(hwnd)

	// Install console control handler so Ctrl+C triggers graceful
	// shutdown (restore parked windows) instead of instant ExitProcess.
	installConsoleCtrlHandler(hwnd)

	logger.Tray(logger.LevelInfo, "", "CreateWindowEx succeeded, hwnd=0x%x", hwnd)

	winapi.WTSRegisterSessionNotification(hwnd, 0)

	// Load icons before adding tray icon
	logger.Tray(logger.LevelInfo, "", "Loading icons...")
	t.initIcons()
	logger.Tray(logger.LevelInfo, "", "Icons loaded")

	// Add tray icon
	logger.Tray(logger.LevelInfo, "", "Adding tray icon...")
	t.addTrayIcon()
	logger.Tray(logger.LevelInfo, "", "Tray icon added")

	// Defer engine start until the message pump is running.
	// SetWinEventHook requires an active message loop on the calling thread.
	// PostMessage queues the request; it dispatches once GetMessage runs.
	logger.Tray(logger.LevelInfo, "", "Posting WM_APP_START...")
	if !winapi.PostMessage(hwnd, winapi.WM_APP_START, 0, 0) {
		logger.Error(logger.LevelError, "", "PostMessage WM_APP_START failed")
	}
	logger.Tray(logger.LevelInfo, "", "Entering message pump...")

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
	t.processor.StopTrayParking()
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
	copy(nid.SzTip[:], windows.StringToUTF16("Madori"))
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
		t.showNotificationBalloon("Madori", "Restoring window layout...")
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

// showNotificationBalloon shows a notification balloon using a separate
// invisible tray icon (UID 2). This way NIM_DELETE can dismiss the balloon
// without removing the main tray icon.
func (t *TrayApp) showNotificationBalloon(title, msg string) {
	t.dismissNotificationBalloon() // dismiss any existing balloon first
	nid := winapi.NOTIFYICONDATA{
		HWnd:             t.hwnd,
		UID:              2, // separate UID — deleting it won't touch the main icon
		UFlags:           winapi.NIF_INFO | winapi.NIF_STATE,
		DwState:          winapi.NIS_HIDDEN,
		DwStateMask:      winapi.NIS_HIDDEN,
		DwInfoFlags:      winapi.NIIF_INFO,
		UCallbackMessage: winapi.WM_TRAYICON,
	}
	copy(nid.SzInfo[:], windows.StringToUTF16(msg))
	copy(nid.SzInfoTitle[:], windows.StringToUTF16(title))
	winapi.ShellNotifyIcon(winapi.NIM_ADD, &nid)
	t.notificationShowing = true
	t.startNotificationTimer()
}

// dismissNotificationBalloon removes the notification balloon by deleting
// the UID-2 tray icon. Safe — only the balloon icon is removed; the main
// UID-1 tray icon is untouched.
func (t *TrayApp) dismissNotificationBalloon() {
	if !t.notificationShowing {
		return
	}
	nid := winapi.NOTIFYICONDATA{
		HWnd: t.hwnd,
		UID:  2,
	}
	winapi.ShellNotifyIcon(winapi.NIM_DELETE, &nid)
	t.notificationShowing = false
}

// startNotificationTimer arms a one-shot timer that dismisses the balloon
// after notificationLingerMs.
func (t *TrayApp) startNotificationTimer() {
	if t.hwnd != 0 {
		duration := t.notificationLingerMs
		if duration <= 0 {
			duration = 3000
		}
		winapi.SetTimer(t.hwnd, TimerClearNotification, uint32(duration), 0)
	}
}

func (t *TrayApp) ShowSnapshotCaptureTip(id int) {
	if t.silent || !t.notification {
		return
	}
	t.showNotificationBalloon("Madori", "Window layout snapshot "+engine.SnapshotName(id)+" captured...")
}

func (t *TrayApp) ShowSnapshotRestoreTip(id int) {
	if t.silent || !t.notification {
		return
	}
	t.iconBusy = true
	t.showNotificationBalloon("Madori", "Window layout snapshot "+engine.SnapshotName(id)+" restored...")
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
		logger.Tray(logger.LevelDebug, "", "SetConsoleCtrlHandler failed (non-fatal, no console?)")
		ctrlCHwnd = 0
	} else {
		logger.Tray(logger.LevelInfo, "", "Console Ctrl+C handler installed (hwnd=0x%x)", hwnd)
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
				logger.Parking(logger.LevelDebug, "parked icon clicked", "uid=%d, restoring window", uid)
				t.processor.RestoreParkedWindow(hwnd)
			}
		} else if mouseMsg == winapi.WM_RBUTTONUP {
			// Right-click on parked icon: show a simple "Restore" option.
			// For now just restore immediately — same as left-click.
			if hwnd := t.processor.FindParkedWindowByUID(uid); hwnd != 0 {
				logger.Parking(logger.LevelDebug, "parked icon right-clicked", "uid=%d, restoring window", uid)
				t.processor.RestoreParkedWindow(hwnd)
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
			logger.Tray(logger.LevelDebug, "", "Tray WM_LBUTTONUP — held %dms, ignoring (drag/hold)", held.Milliseconds())
			break
		}
		dct := winapi.GetDoubleClickTime()
		logger.Tray(logger.LevelDebug, "", "Tray WM_LBUTTONUP — held %dms, SetTimer(%dms)", held.Milliseconds(), dct)
		winapi.KillTimer(t.hwnd, TimerSingleClick)
		t.singleClickPending = true
		winapi.SetTimer(t.hwnd, TimerSingleClick, dct, 0)
	case winapi.WM_LBUTTONDBLCLK:
		t.suppressClick = true
		t.singleClickPending = false
		killed := winapi.KillTimer(t.hwnd, TimerSingleClick)
		logger.Tray(logger.LevelDebug, "", "Tray WM_LBUTTONDBLCLK — KillTimer=%v, calling TakeSnapshot", killed)
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
	logger.Tray(logger.LevelDebug, "", "Menu command: %d", cmdID)
	switch cmdID {
	case CmdCaptureSnapshot:
		name := EnterSnapshotName()
		if name != 0 {
			id := engine.ParseSnapshotID(string(name))
			if id >= 0 {
				logger.Snapshot(logger.LevelInfo, "snapshot captured", "snapshot %d via menu", id)
				t.processor.TakeSnapshot(id)
			}
		}
	case CmdRestoreSnapshot:
		name := EnterSnapshotName()
		if name != 0 {
			id := engine.ParseSnapshotID(string(name))
			if id >= 0 {
				logger.Snapshot(logger.LevelInfo, "snapshot restored", "snapshot %d via menu", id)
				t.processor.RestoreSnapshot(id)
			}
		}
	case CmdPauseResume:
		t.processor.PauseAutoRestore = !t.processor.PauseAutoRestore
		logger.AutoCapture(logger.LevelInfo, "auto-restore toggled", "%s", map[bool]string{true: "paused", false: "resumed"}[t.processor.PauseAutoRestore])
	case CmdRestoreAllParked:
		logger.Parking(logger.LevelInfo, "restored all parked", "via menu")
		t.processor.RestoreAllParked()
	case CmdExit:
		t.Quit()
	}
}

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
				"Madori", winapi.MB_OK|winapi.MB_ICONINFORMATION)
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
