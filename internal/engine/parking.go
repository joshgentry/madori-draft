package engine

import (
	"sync"
	"time"
	"unsafe"

	"madori/internal/logger"
	"madori/internal/winapi"
)

// trayParkedWindows (moved to Processor struct — guarded by p.mu).

// shiftState tracks the keyboard hook and Shift-key state.
// It remains a package-level global because keyboardHookProc (a WH_KEYBOARD_LL
// callback) must be a plain function — it cannot be a method or a closure.
type shiftState struct {
	mu            sync.Mutex
	held          bool
	lastUp        time.Time
	gracePeriod   time.Duration
	kbHookHandle  uintptr
	kbHookRunning bool
}

var globalShiftState = shiftState{gracePeriod: 300 * time.Millisecond}

// SetShiftGracePeriod sets how long after Shift is released that a minimize
// is still treated as Shift+minimize (default 300 ms). Call before StartTrayParking.
func SetShiftGracePeriod(d time.Duration) {
	globalShiftState.mu.Lock()
	globalShiftState.gracePeriod = d
	globalShiftState.mu.Unlock()
}

// isShiftDownOrRecent returns true if either Shift key is held right now,
// or was released within the grace period (WinEvent latency compensation).
func isShiftDownOrRecent() bool {
	globalShiftState.mu.Lock()
	defer globalShiftState.mu.Unlock()
	if globalShiftState.held {
		return true
	}
	return time.Since(globalShiftState.lastUp) < globalShiftState.gracePeriod
}

// keyboardHookProc is the WH_KEYBOARD_LL callback. It only tracks Shift key
// state; all other keys are passed through immediately.
func keyboardHookProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) >= winapi.HC_ACTION {
		vkCode := (*winapi.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam)).VkCode
		if vkCode == winapi.VK_LSHIFT || vkCode == winapi.VK_RSHIFT {
			globalShiftState.mu.Lock()
			if wParam == winapi.WM_KEYDOWN || wParam == winapi.WM_SYSKEYDOWN {
				globalShiftState.held = true
			} else {
				globalShiftState.held = false
				globalShiftState.lastUp = time.Now()
			}
			globalShiftState.mu.Unlock()
		}
	}
	return winapi.CallNextHookEx(globalShiftState.kbHookHandle, nCode, wParam, lParam)
}

// parkedIconUID, parkedIconRev, nextParkedIconUID (moved to Processor struct).

// persistParkedWindows writes the current set of parked HWNDs to BoltDB so
// crashed sessions can recover. No-op if the store hasn't been initialised
// (one-shot CLI commands).
func (p *Processor) persistParkedWindows() {
	if p.store == nil {
		return
	}
	hwnds := make([]uintptr, 0, len(p.trayParkedWindows))
	for hwnd := range p.trayParkedWindows {
		hwnds = append(hwnds, hwnd)
	}
	_ = p.store.SaveParkedWindows(hwnds)
}

// StartTrayParking installs the WH_KEYBOARD_LL hook that tracks Shift-key
// state for Shift+minimize parking. Must be called from a thread with a
// message pump (the tray app's main thread).
func (p *Processor) StartTrayParking() {
	if globalShiftState.kbHookRunning {
		return
	}
	globalShiftState.kbHookHandle = winapi.SetWindowsHookExDirect(winapi.WH_KEYBOARD_LL, keyboardHookProc, 0, 0)
	globalShiftState.kbHookRunning = true
	logger.Parking(logger.LevelInfo, "tray parking enabled", "WH_KEYBOARD_LL (handle=%d)", globalShiftState.kbHookHandle)
}

// StopTrayParking removes the keyboard hook.
func (p *Processor) StopTrayParking() {
	if globalShiftState.kbHookHandle != 0 {
		winapi.UnhookWindowsHookEx(globalShiftState.kbHookHandle)
		globalShiftState.kbHookHandle = 0
	}
	globalShiftState.kbHookRunning = false
	logger.Parking(logger.LevelInfo, "tray parking disabled", "")
}

// SetTrayWindow stores the tray message window HWND so parked-window tray
// icons can route their callbacks to the correct window.
func (p *Processor) SetTrayWindow(hwnd uintptr) {
	p.trayHWnd = hwnd
}

// AddParkedTrayIcon creates a system-tray icon for a parked window.
func (p *Processor) AddParkedTrayIcon(hwnd uintptr) {
	uid := p.nextParkedIconUID
	p.nextParkedIconUID++
	p.parkedIconUID[hwnd] = uid
	p.parkedIconRev[uid] = hwnd

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
	logger.Parking(logger.LevelDebug, "parked icon added", "%s (uid=%d)", title, uid)
}

func (p *Processor) removeParkedTrayIcon(hwnd uintptr) {
	uid, ok := p.parkedIconUID[hwnd]
	if !ok {
		return
	}
	nid := winapi.NOTIFYICONDATA{
		HWnd: p.trayHWnd,
		UID:  uid,
	}
	winapi.ShellNotifyIcon(winapi.NIM_DELETE, &nid)
	delete(p.parkedIconUID, hwnd)
	delete(p.parkedIconRev, uid)
}

// FindParkedWindowByUID returns the HWND for a parked window given its tray icon UID.
func (p *Processor) FindParkedWindowByUID(uid uint32) uintptr {
	return p.parkedIconRev[uid]
}

// RestoreParkedWindow restores a window that was parked to tray.
func (p *Processor) RestoreParkedWindow(hwnd uintptr) {
	if !p.trayParkedWindows[hwnd] {
		return
	}

	p.removeParkedTrayIcon(hwnd)

	winapi.ShowWindow(hwnd, winapi.SW_RESTORE)
	winapi.SetForegroundWindow(hwnd)

	delete(p.trayParkedWindows, hwnd)
	p.persistParkedWindows()

	if metrics, ok := p.monitorApplications[p.curDisplayKey][hwnd]; ok && metrics != nil {
		p.restoreSingleWindow(hwnd, metrics)
	}

	logger.Parking(logger.LevelDebug, "unparked window", "%s", p.WindowDesc(hwnd))
}

// restoreOrphanedParkedWindows loads the parked-window list from BoltDB and
// restores any windows that are still alive. This recovers from a crash where
// RestoreAllParked() never ran. On a clean shutdown the list is empty.
func (p *Processor) restoreOrphanedParkedWindows() {
	if p.store == nil {
		return
	}
	hwnds, err := p.store.LoadParkedWindows()
	if err != nil || len(hwnds) == 0 {
		return
	}
	for _, hwnd := range hwnds {
		if !winapi.IsWindow(hwnd) {
			continue
		}
		// Populate the in-memory map so RestoreParkedWindow's guard passes.
		// It will remove the entry and persist the updated list.
		p.trayParkedWindows[hwnd] = true
		logger.Parking(logger.LevelInfo, "orphaned park restored", "%s (crash recovery)", p.WindowDesc(hwnd))
		p.RestoreParkedWindow(hwnd)
	}
	// Clear the bucket now that we've restored everything.
	_ = p.store.SaveParkedWindows(nil)
}

// RestoreParkedWindowsCmd is the CLI one-shot (-restore_parked_windows):
// loads parked-window HWNDs from the database, restores any that are still
// alive, clears the bucket, and exits. Useful as a manual recovery tool.
func (p *Processor) RestoreParkedWindowsCmd() {
	p.restoreOrphanedParkedWindows()
}

// GetTrayParkedWindows returns all windows currently parked to tray.
func (p *Processor) GetTrayParkedWindows() []uintptr {
	p.mu.Lock()
	defer p.mu.Unlock()
	var result []uintptr
	for hwnd := range p.trayParkedWindows {
		if winapi.IsWindow(hwnd) {
			result = append(result, hwnd)
		} else {
			delete(p.trayParkedWindows, hwnd)
		}
	}
	return result
}

func copy16(dst []uint16, src string) {
	for i := 0; i < len(dst)-1 && i < len(src); i++ {
		dst[i] = uint16(src[i])
	}
}
