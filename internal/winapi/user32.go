package winapi

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32 = windows.NewLazySystemDLL("user32.dll")

	// User32 functions
	procSetWinEventHook                 = modUser32.NewProc("SetWinEventHook")
	procUnhookWinEvent                  = modUser32.NewProc("UnhookWinEvent")
	procEnumDisplayMonitors             = modUser32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW                 = modUser32.NewProc("GetMonitorInfoW")
	procMonitorFromPoint                = modUser32.NewProc("MonitorFromPoint")
	procWindowFromPoint                 = modUser32.NewProc("WindowFromPoint")
	procGetClassName                    = modUser32.NewProc("GetClassNameW")
	procGetWindowTextLengthW            = modUser32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW                  = modUser32.NewProc("GetWindowTextW")
	procInternalGetWindowText           = modUser32.NewProc("InternalGetWindowText")
	procGetWindowPlacement              = modUser32.NewProc("GetWindowPlacement")
	procGetWindowRect                   = modUser32.NewProc("GetWindowRect")
	procGetClientRect                   = modUser32.NewProc("GetClientRect")
	procIntersectRect                   = modUser32.NewProc("IntersectRect")
	procPtInRect                        = modUser32.NewProc("PtInRect")
	procSetFocus                        = modUser32.NewProc("SetFocus")
	procIsWindowVisible                 = modUser32.NewProc("IsWindowVisible")
	procIsIconic                        = modUser32.NewProc("IsIconic")
	procIsHungAppWindow                 = modUser32.NewProc("IsHungAppWindow")
	procGetDesktopWindow                = modUser32.NewProc("GetDesktopWindow")
	procGetTopWindow                    = modUser32.NewProc("GetTopWindow")
	procGetWindowThreadProcessId        = modUser32.NewProc("GetWindowThreadProcessId")
	procMoveWindow                      = modUser32.NewProc("MoveWindow")
	procMapWindowPoints                 = modUser32.NewProc("MapWindowPoints")
	procSetWindowPlacement              = modUser32.NewProc("SetWindowPlacement")
	procSetWindowPos                    = modUser32.NewProc("SetWindowPos")
	procGetWindow                       = modUser32.NewProc("GetWindow")
	procGetParent                       = modUser32.NewProc("GetParent")
	procBeginDeferWindowPos             = modUser32.NewProc("BeginDeferWindowPos")
	procDeferWindowPos                  = modUser32.NewProc("DeferWindowPos")
	procEndDeferWindowPos               = modUser32.NewProc("EndDeferWindowPos")
	procRedrawWindow                    = modUser32.NewProc("RedrawWindow")
	procIsWindow                        = modUser32.NewProc("IsWindow")
	procIsZoomed                        = modUser32.NewProc("IsZoomed")
	procShowWindow                      = modUser32.NewProc("ShowWindow")
	procShowWindowAsync                 = modUser32.NewProc("ShowWindowAsync")
	procUpdateWindow                    = modUser32.NewProc("UpdateWindow")
	procIsTopLevelWindow                = modUser32.NewProc("IsTopLevelWindow")
	procFindWindowW                     = modUser32.NewProc("FindWindowW")
	procFindWindowExW                   = modUser32.NewProc("FindWindowExW")
	procMouseEvent                      = modUser32.NewProc("mouse_event")
	procSetCursor                       = modUser32.NewProc("SetCursor")
	procGetCursorPos                    = modUser32.NewProc("GetCursorPos")
	procSetCursorPos                    = modUser32.NewProc("SetCursorPos")
	procGetCursorInfo                   = modUser32.NewProc("GetCursorInfo")
	procGetKeyState                     = modUser32.NewProc("GetKeyState")
	procGetAsyncKeyState                = modUser32.NewProc("GetAsyncKeyState")
	procSetForegroundWindow             = modUser32.NewProc("SetForegroundWindow")
	procGetForegroundWindow             = modUser32.NewProc("GetForegroundWindow")
	procSetActiveWindow                 = modUser32.NewProc("SetActiveWindow")
	procGetWindowLongW                  = modUser32.NewProc("GetWindowLongW")
	procSetWindowLongW                  = modUser32.NewProc("SetWindowLongW")
	procGetClassLongPtrW                = modUser32.NewProc("GetClassLongPtrW")
	procLoadIconW                       = modUser32.NewProc("LoadIconW")
	procGetAncestor                     = modUser32.NewProc("GetAncestor")
	procSendMessageW                    = modUser32.NewProc("SendMessageW")
	procSendMessageTimeoutW             = modUser32.NewProc("SendMessageTimeoutW")
	procRegisterHotKey                  = modUser32.NewProc("RegisterHotKey")
	procUnregisterHotKey                = modUser32.NewProc("UnregisterHotKey")
	procSetWindowsHookExW               = modUser32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx             = modUser32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx                  = modUser32.NewProc("CallNextHookEx")
	procDestroyIcon                     = modUser32.NewProc("DestroyIcon")
	procGetDpiForWindow                 = modUser32.NewProc("GetDpiForWindow")
	procIsValidDpiAwarenessContext      = modUser32.NewProc("IsValidDpiAwarenessContext")
	procSetProcessDpiAwarenessContext   = modUser32.NewProc("SetProcessDpiAwarenessContext")
	procSetThreadDpiAwarenessContext    = modUser32.NewProc("SetThreadDpiAwarenessContext")
	procCreateWindowExW                 = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW                  = modUser32.NewProc("DefWindowProcW")
	procDestroyWindow                   = modUser32.NewProc("DestroyWindow")
	procCreateIconFromResourceEx        = modUser32.NewProc("CreateIconFromResourceEx")
	procCreateIcon                      = modUser32.NewProc("CreateIcon")
	procLoadImageW                      = modUser32.NewProc("LoadImageW")
	procRegisterClassExW                = modUser32.NewProc("RegisterClassExW")
	procGetMessageW                     = modUser32.NewProc("GetMessageW")
	procTranslateMessage                = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW                = modUser32.NewProc("DispatchMessageW")
	procPostMessageW                    = modUser32.NewProc("PostMessageW")
	procPostQuitMessage                 = modUser32.NewProc("PostQuitMessage")
	procSetTimer                        = modUser32.NewProc("SetTimer")
	procKillTimer                       = modUser32.NewProc("KillTimer")
	procMessageBoxW                     = modUser32.NewProc("MessageBoxW")
	procCreatePopupMenu                 = modUser32.NewProc("CreatePopupMenu")
	procAppendMenuW                     = modUser32.NewProc("AppendMenuW")
	procTrackPopupMenu                  = modUser32.NewProc("TrackPopupMenu")
	procDestroyMenu                     = modUser32.NewProc("DestroyMenu")
	procSetMenuDefaultItem              = modUser32.NewProc("SetMenuDefaultItem")
	procCheckMenuItem                   = modUser32.NewProc("CheckMenuItem")
	procEnableMenuItem                  = modUser32.NewProc("EnableMenuItem")
	procGetSystemMetrics                = modUser32.NewProc("GetSystemMetrics")
	procIsWindowOnCurrentVirtualDesktop = modUser32.NewProc("IsWindowOnCurrentVirtualDesktop")
)

// WinEventDelegate is the callback type for SetWinEventHook.
type WinEventDelegate func(
	hWinEventHook uintptr,
	eventType uint32,
	hwnd uintptr,
	idObject int32,
	idChild int32,
	dwEventThread uint32,
	dwmsEventTime uint32,
)

// MonitorEnumDelegate is the callback type for EnumDisplayMonitors.
type MonitorEnumDelegate func(
	hMonitor uintptr,
	hdcMonitor uintptr,
	lprcMonitor *RECT,
	dwData uintptr,
) uintptr

// HookHandler is the callback type for SetWindowsHookEx (WH_KEYBOARD_LL, WH_MOUSE_LL, etc.).
// All parameters are uintptr to match syscall.NewCallback requirements.
type HookHandler func(nCode, wParam, lParam uintptr) uintptr

// --- User32 Functions ---

func SetWinEventHook(eventMin, eventMax uint32, hmodWinEventProc uintptr,
	lpfnWinEventProc WinEventDelegate, idProcess, idThread, dwFlags uint32) uintptr {
	ret, _, _ := procSetWinEventHook.Call(
		uintptr(eventMin),
		uintptr(eventMax),
		hmodWinEventProc,
		syscall.NewCallback(func(hWinEventHook uintptr, eventType uint32, hwnd uintptr, idObject int32, idChild int32, dwEventThread uint32, dwmsEventTime uint32) uintptr {
			lpfnWinEventProc(hWinEventHook, eventType, hwnd, idObject, idChild, dwEventThread, dwmsEventTime)
			return 0
		}),
		uintptr(idProcess),
		uintptr(idThread),
		uintptr(dwFlags),
	)
	return ret
}

func UnhookWinEvent(hWinEventHook uintptr) bool {
	ret, _, _ := procUnhookWinEvent.Call(hWinEventHook)
	return ret != 0
}

// enumMonitorCallback is used by EnumDisplayMonitors. It receives uintptr from the
// Windows callback and converts to the typed delegate. The unsafe.Pointer conversion
// is necessary for Win32 interop and is safe because Windows passes a valid RECT pointer.
type enumMonitorCallbackCtx struct {
	fn       MonitorEnumDelegate
	userData uintptr
}

func enumMonitorCallback(hMonitor, hdcMonitor, lprcMonitor, data uintptr) uintptr {
	ctx := (*enumMonitorCallbackCtx)(unsafe.Pointer(data))
	return ctx.fn(hMonitor, hdcMonitor, (*RECT)(unsafe.Pointer(lprcMonitor)), ctx.userData)
}

func EnumDisplayMonitors(hdc uintptr, lprcClip *RECT, lpfnEnum MonitorEnumDelegate, dwData uintptr) bool {
	ctx := &enumMonitorCallbackCtx{fn: lpfnEnum, userData: dwData}

	ret, _, _ := procEnumDisplayMonitors.Call(
		hdc,
		uintptr(unsafe.Pointer(lprcClip)),
		syscall.NewCallback(enumMonitorCallback),
		uintptr(unsafe.Pointer(ctx)),
	)
	return ret != 0
}

type MONITORINFO struct {
	CbSize    uint32
	RCMonitor RECT
	RCWork    RECT
	DwFlags   uint32
}

func GetMonitorInfo(hMonitor uintptr, lpmi *MONITORINFO) bool {
	lpmi.CbSize = uint32(unsafe.Sizeof(MONITORINFO{}))
	ret, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(lpmi)))
	return ret != 0
}

func MonitorFromPoint(pt POINT, dwFlags uint32) uintptr {
	ret, _, _ := procMonitorFromPoint.Call(
		uintptr(pt.X)|(uintptr(pt.Y)<<32),
		uintptr(dwFlags),
	)
	return ret
}

func WindowFromPoint(pt POINT) uintptr {
	ret, _, _ := procWindowFromPoint.Call(uintptr(pt.X) | (uintptr(pt.Y) << 32))
	return ret
}

func GetClassName(hWnd uintptr, lpClassName *[256]uint16, nMaxCount int32) int32 {
	ret, _, _ := procGetClassName.Call(hWnd, uintptr(unsafe.Pointer(lpClassName)), uintptr(nMaxCount))
	return int32(ret)
}

func GetWindowTextLength(hWnd uintptr) int32 {
	ret, _, _ := procGetWindowTextLengthW.Call(hWnd)
	return int32(ret)
}

func GetWindowText(hWnd uintptr, lpString *[512]uint16, nMaxCount int32) int32 {
	ret, _, _ := procGetWindowTextW.Call(hWnd, uintptr(unsafe.Pointer(lpString)), uintptr(nMaxCount))
	return int32(ret)
}

func InternalGetWindowText(hWnd uintptr, lpString *[512]uint16, nMaxCount int32) int32 {
	ret, _, _ := procInternalGetWindowText.Call(hWnd, uintptr(unsafe.Pointer(lpString)), uintptr(nMaxCount))
	return int32(ret)
}

func GetWindowPlacement(hWnd uintptr, lpwndpl *WINDOWPLACEMENT) bool {
	lpwndpl.Length = uint32(unsafe.Sizeof(WINDOWPLACEMENT{}))
	ret, _, _ := procGetWindowPlacement.Call(hWnd, uintptr(unsafe.Pointer(lpwndpl)))
	return ret != 0
}

func GetWindowRect(hWnd uintptr, lpRect *RECT) bool {
	ret, _, _ := procGetWindowRect.Call(hWnd, uintptr(unsafe.Pointer(lpRect)))
	return ret != 0
}

func GetClientRect(hWnd uintptr, lpRect *RECT) bool {
	ret, _, _ := procGetClientRect.Call(hWnd, uintptr(unsafe.Pointer(lpRect)))
	return ret != 0
}

func IntersectRect(lprcDst *RECT, lprcSrc1 *RECT, lprcSrc2 *RECT) bool {
	ret, _, _ := procIntersectRect.Call(
		uintptr(unsafe.Pointer(lprcDst)),
		uintptr(unsafe.Pointer(lprcSrc1)),
		uintptr(unsafe.Pointer(lprcSrc2)),
	)
	return ret != 0
}

func PtInRect(lprc *RECT, pt POINT) bool {
	ret, _, _ := procPtInRect.Call(uintptr(unsafe.Pointer(lprc)), uintptr(pt.X)|(uintptr(pt.Y)<<32))
	return ret != 0
}

func SetFocus(hWnd uintptr) uintptr {
	ret, _, _ := procSetFocus.Call(hWnd)
	return ret
}

func IsWindowVisible(hWnd uintptr) bool {
	ret, _, _ := procIsWindowVisible.Call(hWnd)
	return ret != 0
}

func IsIconic(hWnd uintptr) bool {
	ret, _, _ := procIsIconic.Call(hWnd)
	return ret != 0
}

func IsHungAppWindow(hWnd uintptr) bool {
	ret, _, _ := procIsHungAppWindow.Call(hWnd)
	return ret != 0
}

func GetDesktopWindow() uintptr {
	ret, _, _ := procGetDesktopWindow.Call()
	return ret
}

func GetTopWindow(hWnd uintptr) uintptr {
	ret, _, _ := procGetTopWindow.Call(hWnd)
	return ret
}

func GetWindowThreadProcessId(hWnd uintptr) (uint32, uint32) {
	var lpdwProcessId uint32
	threadId, _, _ := procGetWindowThreadProcessId.Call(hWnd, uintptr(unsafe.Pointer(&lpdwProcessId)))
	return uint32(threadId), lpdwProcessId
}

func MoveWindow(hWnd uintptr, x, y, width, height int32, repaint bool) bool {
	var bRepaint uintptr
	if repaint {
		bRepaint = 1
	}
	ret, _, _ := procMoveWindow.Call(hWnd, uintptr(x), uintptr(y), uintptr(width), uintptr(height), bRepaint)
	return ret != 0
}

func MapWindowPoints(hWndFrom, hWndTo uintptr, lpPoints *POINT, cPoints uint32) int32 {
	ret, _, _ := procMapWindowPoints.Call(hWndFrom, hWndTo, uintptr(unsafe.Pointer(lpPoints)), uintptr(cPoints))
	return int32(ret)
}

func SetWindowPlacement(hWnd uintptr, lpwndpl *WINDOWPLACEMENT) bool {
	ret, _, _ := procSetWindowPlacement.Call(hWnd, uintptr(unsafe.Pointer(lpwndpl)))
	return ret != 0
}

func SetWindowPos(hWnd, hWndInsertAfter uintptr, x, y, cx, cy int32, uFlags uint32) bool {
	ret, _, _ := procSetWindowPos.Call(
		hWnd, hWndInsertAfter,
		uintptr(x), uintptr(y), uintptr(cx), uintptr(cy),
		uintptr(uFlags),
	)
	return ret != 0
}

func GetWindow(hWnd uintptr, uCmd uint32) uintptr {
	ret, _, _ := procGetWindow.Call(hWnd, uintptr(uCmd))
	return ret
}

func GetParent(hWnd uintptr) uintptr {
	ret, _, _ := procGetParent.Call(hWnd)
	return ret
}

func BeginDeferWindowPos(nNumWindows int32) uintptr {
	ret, _, _ := procBeginDeferWindowPos.Call(uintptr(nNumWindows))
	return ret
}

func DeferWindowPos(hWinPosInfo, hWnd, hWndInsertAfter uintptr, x, y, cx, cy int32, uFlags uint32) uintptr {
	ret, _, _ := procDeferWindowPos.Call(
		hWinPosInfo, hWnd, hWndInsertAfter,
		uintptr(x), uintptr(y), uintptr(cx), uintptr(cy),
		uintptr(uFlags),
	)
	return ret
}

func EndDeferWindowPos(hWinPosInfo uintptr) bool {
	ret, _, _ := procEndDeferWindowPos.Call(hWinPosInfo)
	return ret != 0
}

func RedrawWindow(hWnd uintptr, lprcUpdate *RECT, hrgnUpdate uintptr, flags uint32) bool {
	ret, _, _ := procRedrawWindow.Call(
		hWnd,
		uintptr(unsafe.Pointer(lprcUpdate)),
		hrgnUpdate,
		uintptr(flags),
	)
	return ret != 0
}

func IsWindow(hWnd uintptr) bool {
	ret, _, _ := procIsWindow.Call(hWnd)
	return ret != 0
}

func IsZoomed(hWnd uintptr) bool {
	ret, _, _ := procIsZoomed.Call(hWnd)
	return ret != 0
}

func ShowWindow(hWnd uintptr, nCmdShow int32) bool {
	ret, _, _ := procShowWindow.Call(hWnd, uintptr(nCmdShow))
	return ret != 0
}

func ShowWindowAsync(hWnd uintptr, nCmdShow int32) bool {
	ret, _, _ := procShowWindowAsync.Call(hWnd, uintptr(nCmdShow))
	return ret != 0
}

func UpdateWindow(hWnd uintptr) bool {
	ret, _, _ := procUpdateWindow.Call(hWnd)
	return ret != 0
}

/*
Undocumented, but "well-known in the Win32 community" API to determine if
the window is a child of the Desktop, not a control of another window.

Here's a Chromium-implemented variant:

	bool IsTopLevelWindow(HWND window) {
		LONG style = ::GetWindowLong(window, GWL_STYLE);
		if (!(style & WS_CHILD))
			return true;
		HWND parent = ::GetParent(window);
		return !parent || (parent == ::GetDesktopWindow());
	}

From: https://chromium.googlesource.com/chromium/src/+/299155e5e37a77670b7969771e09e9a16b1f5612/ui/views/win/hwnd_message_handler.cc#231
*/
func IsTopLevelWindow(hWnd uintptr) bool {
	ret, _, _ := procIsTopLevelWindow.Call(hWnd)
	return ret != 0
}

func FindWindowW(className, windowName *uint16) uintptr {
	var pClass, pName uintptr
	if className != nil {
		pClass = uintptr(unsafe.Pointer(className))
	}
	if windowName != nil {
		pName = uintptr(unsafe.Pointer(windowName))
	}
	ret, _, _ := procFindWindowW.Call(pClass, pName)
	return ret
}

func FindWindowExW(parentHandle, childAfter uintptr, className, windowTitle *uint16) uintptr {
	var pClass, pTitle uintptr
	if className != nil {
		pClass = uintptr(unsafe.Pointer(className))
	}
	if windowTitle != nil {
		pTitle = uintptr(unsafe.Pointer(windowTitle))
	}
	ret, _, _ := procFindWindowExW.Call(parentHandle, childAfter, pClass, pTitle)
	return ret
}

func MouseEvent(dwFlags, dx, dy, dwData uint32, dwExtraInfo uintptr) {
	procMouseEvent.Call(uintptr(dwFlags), uintptr(dx), uintptr(dy), uintptr(dwData), dwExtraInfo)
}

func SetCursor(hCursor uintptr) uintptr {
	ret, _, _ := procSetCursor.Call(hCursor)
	return ret
}

func GetCursorPos(lpPoint *POINT) bool {
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(lpPoint)))
	return ret != 0
}

func SetCursorPos(x, y int32) bool {
	ret, _, _ := procSetCursorPos.Call(uintptr(x), uintptr(y))
	return ret != 0
}

func GetCursorInfo(pci *CURSORINFO) bool {
	pci.CbSize = uint32(unsafe.Sizeof(CURSORINFO{}))
	ret, _, _ := procGetCursorInfo.Call(uintptr(unsafe.Pointer(pci)))
	return ret != 0
}

func GetKeyState(nVirtKey int32) int16 {
	ret, _, _ := procGetKeyState.Call(uintptr(nVirtKey))
	return int16(ret & 0xFFFF)
}

func GetAsyncKeyState(nVirtKey int32) int16 {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(nVirtKey))
	return int16(ret & 0xFFFF)
}

func SetForegroundWindow(hWnd uintptr) bool {
	ret, _, _ := procSetForegroundWindow.Call(hWnd)
	return ret != 0
}

func GetForegroundWindow() uintptr {
	ret, _, _ := procGetForegroundWindow.Call()
	return ret
}

func SetActiveWindow(hWnd uintptr) uintptr {
	ret, _, _ := procSetActiveWindow.Call(hWnd)
	return ret
}

func GetWindowLong(hWnd uintptr, nIndex int32) int64 {
	ret, _, _ := procGetWindowLongW.Call(hWnd, uintptr(nIndex))
	return int64(ret)
}

func SetWindowLong(hWnd uintptr, nIndex int32, dwNewLong uintptr) int32 {
	ret, _, _ := procSetWindowLongW.Call(hWnd, uintptr(nIndex), dwNewLong)
	return int32(ret)
}

func GetClassLongPtr(hWnd uintptr, nIndex int32) uintptr {
	ret, _, _ := procGetClassLongPtrW.Call(hWnd, uintptr(nIndex))
	return ret
}

func LoadIcon(hInstance uintptr, lpIconName uintptr) uintptr {
	ret, _, _ := procLoadIconW.Call(hInstance, lpIconName)
	return ret
}

func GetAncestor(hWnd uintptr, gaFlags uint32) uintptr {
	ret, _, _ := procGetAncestor.Call(hWnd, uintptr(gaFlags))
	return ret
}

func SendMessage(hWnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procSendMessageW.Call(hWnd, uintptr(msg), wParam, lParam)
	return ret
}

func SendMessageTimeout(hWnd uintptr, msg uint32, wParam, lParam uintptr, fuFlags, uTimeout uint32) (uintptr, uint32) {
	var lpdwResult uint32
	ret, _, _ := procSendMessageTimeoutW.Call(
		hWnd, uintptr(msg), wParam, lParam,
		uintptr(fuFlags), uintptr(uTimeout),
		uintptr(unsafe.Pointer(&lpdwResult)),
	)
	return ret, lpdwResult
}

func RegisterHotKey(hWnd uintptr, id int32, fsModifiers, vk uint32) bool {
	ret, _, _ := procRegisterHotKey.Call(hWnd, uintptr(id), uintptr(fsModifiers), uintptr(vk))
	return ret != 0
}

func UnregisterHotKey(hWnd uintptr, id int32) bool {
	ret, _, _ := procUnregisterHotKey.Call(hWnd, uintptr(id))
	return ret != 0
}

// SetWindowsHookExDirect installs a Windows hook with the callback passed
// directly to syscall.NewCallback (no intermediate closure). For LL hooks
// this avoids a layer of Go callback indirection that can interact poorly
// with the message pump's nested dispatch.
func SetWindowsHookExDirect(idHook int32, lpfn HookHandler, hMod uintptr, dwThreadID uint32) uintptr {
	ret, _, _ := procSetWindowsHookExW.Call(
		uintptr(idHook),
		syscall.NewCallback(lpfn),
		hMod,
		uintptr(dwThreadID),
	)
	return ret
}

func UnhookWindowsHookEx(hhk uintptr) bool {
	ret, _, _ := procUnhookWindowsHookEx.Call(hhk)
	return ret != 0
}

func CallNextHookEx(hhk, nCode, wParam, lParam uintptr) uintptr {
	ret, _, _ := procCallNextHookEx.Call(hhk, nCode, wParam, lParam)
	return ret
}

func DestroyIcon(hIcon uintptr) bool {
	ret, _, _ := procDestroyIcon.Call(hIcon)
	return ret != 0
}

// CreateIconFromResourceEx creates an icon or cursor from resource bits.
func CreateIconFromResourceEx(presbits []byte, dwResSize uint32, fIcon bool, dwVer uint32, cxDesired, cyDesired int32, uFlags uint32) uintptr {
	var fIconVal uintptr
	if fIcon {
		fIconVal = 1
	}
	ret, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&presbits[0])),
		uintptr(dwResSize),
		fIconVal,
		uintptr(dwVer),
		uintptr(cxDesired),
		uintptr(cyDesired),
		uintptr(uFlags),
	)
	return ret
}

// CreateIcon creates an icon from bitmap data.
func CreateIcon(hInstance uintptr, nWidth, nHeight int32, cPlanes, cBitsPerPixel byte, lpbANDbits, lpbXORbits []byte) uintptr {
	ret, _, _ := procCreateIcon.Call(
		hInstance,
		uintptr(nWidth),
		uintptr(nHeight),
		uintptr(cPlanes),
		uintptr(cBitsPerPixel),
		uintptr(unsafe.Pointer(&lpbANDbits[0])),
		uintptr(unsafe.Pointer(&lpbXORbits[0])),
	)
	return ret
}

// LoadImage flags
const (
	IMAGE_BITMAP    = 0
	IMAGE_ICON      = 1
	IMAGE_CURSOR    = 2
	LR_LOADFROMFILE = 0x00000010
	LR_DEFAULTSIZE  = 0x00000040
)

// LoadImage loads an icon, cursor, or bitmap.
func LoadImage(hInst uintptr, name uintptr, typ uint32, cx, cy int32, fuLoad uint32) uintptr {
	ret, _, _ := procLoadImageW.Call(hInst, name, uintptr(typ), uintptr(cx), uintptr(cy), uintptr(fuLoad))
	return ret
}

func GetDpiForWindow(hWnd uintptr) uint32 {
	ret, _, _ := procGetDpiForWindow.Call(hWnd)
	return uint32(ret)
}

func IsValidDpiAwarenessContext(value uintptr) bool {
	ret, _, _ := procIsValidDpiAwarenessContext.Call(value)
	return ret != 0
}

func SetProcessDpiAwarenessContext(value int32) bool {
	ret, _, _ := procSetProcessDpiAwarenessContext.Call(uintptr(value))
	return ret != 0
}

func SetThreadDpiAwarenessContext(value int32) uintptr {
	ret, _, _ := procSetThreadDpiAwarenessContext.Call(uintptr(value))
	return ret
}

// SetThreadDpiAwarenessContextSafe wraps SetThreadDpiAwarenessContext
// and should be called before DPI-sensitive window operations on Win10+.
func SetThreadDpiAwarenessContextSafe(value int32) uintptr {
	return SetThreadDpiAwarenessContext(value)
}

// --- Window Creation Helpers ---

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

func RegisterClassEx(wcx *WNDCLASSEX) (uint16, error) {
	wcx.CbSize = uint32(unsafe.Sizeof(WNDCLASSEX{}))
	ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(wcx)))
	if ret == 0 {
		return 0, err
	}
	return uint16(ret), nil
}

func DestroyWindow(hWnd uintptr) bool {
	ret, _, _ := procDestroyWindow.Call(hWnd)
	return ret != 0
}

func CreateWindowEx(dwExStyle uint32, lpClassName *uint16, lpWindowName *uint16,
	dwStyle uint32, x, y, nWidth, nHeight int32,
	hWndParent, hMenu, hInstance uintptr, lpParam unsafe.Pointer) uintptr {
	ret, _, _ := procCreateWindowExW.Call(
		uintptr(dwExStyle),
		uintptr(unsafe.Pointer(lpClassName)),
		uintptr(unsafe.Pointer(lpWindowName)),
		uintptr(dwStyle),
		uintptr(x), uintptr(y), uintptr(nWidth), uintptr(nHeight),
		hWndParent, hMenu, hInstance,
		uintptr(lpParam),
	)
	return ret
}

func DefWindowProc(hWnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(hWnd, uintptr(msg), wParam, lParam)
	return ret
}

// MSG represents the Windows MSG structure.
type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

func GetMessage(msg *MSG, hWnd uintptr, wMsgFilterMin, wMsgFilterMax uint32) int32 {
	ret, _, _ := procGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)),
		hWnd,
		uintptr(wMsgFilterMin),
		uintptr(wMsgFilterMax),
	)
	return int32(ret)
}

func TranslateMessage(msg *MSG) bool {
	ret, _, _ := procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
	return ret != 0
}

func DispatchMessage(msg *MSG) uintptr {
	ret, _, _ := procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
	return ret
}

func PostMessage(hWnd uintptr, msg uint32, wParam, lParam uintptr) bool {
	ret, _, _ := procPostMessageW.Call(hWnd, uintptr(msg), wParam, lParam)
	return ret != 0
}

func PostQuitMessage(nExitCode int32) {
	procPostQuitMessage.Call(uintptr(nExitCode))
}

func SetTimer(hWnd uintptr, nIDEvent uintptr, uElapse uint32, lpTimerFunc uintptr) uintptr {
	ret, _, _ := procSetTimer.Call(hWnd, nIDEvent, uintptr(uElapse), lpTimerFunc)
	return ret
}

func KillTimer(hWnd uintptr, nIDEvent uintptr) bool {
	ret, _, _ := procKillTimer.Call(hWnd, nIDEvent)
	return ret != 0
}

// --- Menu Functions ---

func CreatePopupMenu() uintptr {
	ret, _, _ := procCreatePopupMenu.Call()
	return ret
}

func AppendMenu(hMenu uintptr, uFlags uint32, uIDNewItem uintptr, lpNewItem *uint16) bool {
	ret, _, _ := procAppendMenuW.Call(hMenu, uintptr(uFlags), uIDNewItem, uintptr(unsafe.Pointer(lpNewItem)))
	return ret != 0
}

// Menu flags
const (
	MF_STRING       = 0x00000000
	MF_SEPARATOR    = 0x00000800
	MF_CHECKED      = 0x00000008
	MF_UNCHECKED    = 0x00000000
	MF_GRAYED       = 0x00000001
	MF_ENABLED      = 0x00000000
	MF_DISABLED     = 0x00000002
	MF_POPUP        = 0x00000010
	MF_MENUBARBREAK = 0x00000020
	MF_MENUBREAK    = 0x00000040
	MF_BYCOMMAND    = 0x00000000
	MF_BYPOSITION   = 0x00000400
)

func TrackPopupMenu(hMenu uintptr, uFlags uint32, x, y int32, nReserved int32, hWnd uintptr, prcRect *RECT) bool {
	ret, _, _ := procTrackPopupMenu.Call(
		hMenu, uintptr(uFlags), uintptr(x), uintptr(y), uintptr(nReserved), hWnd,
		uintptr(unsafe.Pointer(prcRect)),
	)
	return ret != 0
}

// TrackPopupMenu flags
const (
	TPM_LEFTBUTTON   = 0x0000
	TPM_RIGHTBUTTON  = 0x0002
	TPM_LEFTALIGN    = 0x0000
	TPM_CENTERALIGN  = 0x0004
	TPM_RIGHTALIGN   = 0x0008
	TPM_TOPALIGN     = 0x0000
	TPM_VCENTERALIGN = 0x0010
	TPM_BOTTOMALIGN  = 0x0020
	TPM_HORIZONTAL   = 0x0000
	TPM_VERTICAL     = 0x0040
	TPM_NONOTIFY     = 0x0080
	TPM_RETURNCMD    = 0x0100
)

func DestroyMenu(hMenu uintptr) bool {
	ret, _, _ := procDestroyMenu.Call(hMenu)
	return ret != 0
}

func SetMenuDefaultItem(hMenu uintptr, uItem uint32, fByPos uint32) bool {
	ret, _, _ := procSetMenuDefaultItem.Call(hMenu, uintptr(uItem), uintptr(fByPos))
	return ret != 0
}

func CheckMenuItem(hMenu uintptr, uIDCheckItem uint32, uCheck uint32) uint32 {
	ret, _, _ := procCheckMenuItem.Call(hMenu, uintptr(uIDCheckItem), uintptr(uCheck))
	return uint32(ret)
}

func EnableMenuItem(hMenu uintptr, uIDEnableItem uint32, uEnable uint32) bool {
	ret, _, _ := procEnableMenuItem.Call(hMenu, uintptr(uIDEnableItem), uintptr(uEnable))
	return ret != ^uintptr(0) // -1 means failure
}

func GetSystemMetrics(nIndex int32) int32 {
	ret, _, _ := procGetSystemMetrics.Call(uintptr(nIndex))
	return int32(ret)
}

// System metrics constants
const (
	SM_CXSCREEN       = 0
	SM_CYSCREEN       = 1
	SM_CXVSCROLL      = 2
	SM_CYHSCROLL      = 3
	SM_CYCAPTION      = 4
	SM_CXBORDER       = 5
	SM_CYBORDER       = 6
	SM_CXDLGFRAME     = 7
	SM_CYDLGFRAME     = 8
	SM_CXSIZE         = 30
	SM_CYSIZE         = 31
	SM_CXFRAME        = 32
	SM_CYFRAME        = 33
	SM_CXDOUBLECLK    = 36
	SM_CYDOUBLECLK    = 37
	SM_CXPADDEDBORDER = 92
)

func IsWindowOnCurrentVirtualDesktop(hWnd uintptr) bool {
	ret, _, _ := procIsWindowOnCurrentVirtualDesktop.Call(hWnd)
	return ret != 0
}

// --- Extended User32 Functions ---

var (
	procSetLayeredWindowAttributes = modUser32.NewProc("SetLayeredWindowAttributes")
	procKeybdEvent                 = modUser32.NewProc("keybd_event")
	procLoadCursorW                = modUser32.NewProc("LoadCursorW")
	procGetDC                      = modUser32.NewProc("GetDC")
	procGetDoubleClickTime         = modUser32.NewProc("GetDoubleClickTime")
	procReleaseDC                  = modUser32.NewProc("ReleaseDC")
	procGetDlgItem                 = modUser32.NewProc("GetDlgItem")
)

// SetLayeredWindowAttributes sets the opacity of a layered window.
func SetLayeredWindowAttributes(hwnd uintptr, crKey uint32, bAlpha byte, dwFlags uint32) bool {
	ret, _, _ := procSetLayeredWindowAttributes.Call(hwnd, uintptr(crKey), uintptr(bAlpha), uintptr(dwFlags))
	return ret != 0
}

// LWA flags
const (
	LWA_COLORKEY = 0x00000001
	LWA_ALPHA    = 0x00000002
)

// KeybdEvent synthesizes a keystroke.
func KeybdEvent(bVk byte, bScan byte, dwFlags uint32, dwExtraInfo uintptr) {
	procKeybdEvent.Call(uintptr(bVk), uintptr(bScan), uintptr(dwFlags), dwExtraInfo)
}

// LoadCursor loads a cursor resource.
func LoadCursor(hInstance uintptr, lpCursorName uintptr) uintptr {
	ret, _, _ := procLoadCursorW.Call(hInstance, lpCursorName)
	return ret
}

// GetDC retrieves a device context for a window.
func GetDC(hWnd uintptr) uintptr {
	ret, _, _ := procGetDC.Call(hWnd)
	return ret
}

// ReleaseDC releases a device context.
func ReleaseDC(hWnd uintptr, hDC uintptr) bool {
	ret, _, _ := procReleaseDC.Call(hWnd, hDC)
	return ret != 0
}

// GetDlgItem retrieves a handle to a control in a dialog.
func GetDlgItem(hDlg uintptr, nIDDlgItem int32) uintptr {
	ret, _, _ := procGetDlgItem.Call(hDlg, uintptr(nIDDlgItem))
	return ret
}

// MessageBox flags
const (
	MB_OK              = 0x00000000
	MB_OKCANCEL        = 0x00000001
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030
	MB_ICONERROR       = 0x00000010
)

// MessageBox shows a modal message box.
func MessageBox(hWnd uintptr, text, caption string, uType uint32) int32 {
	txt, _ := syscall.UTF16PtrFromString(text)
	cap, _ := syscall.UTF16PtrFromString(caption)
	ret, _, _ := procMessageBoxW.Call(hWnd, uintptr(unsafe.Pointer(txt)), uintptr(unsafe.Pointer(cap)), uintptr(uType))
	return int32(ret)
}

// Standard cursor IDs
const IDC_ARROW = 32512

// GetDoubleClickTime returns the system double-click time in milliseconds.
func GetDoubleClickTime() uint32 {
	ret, _, _ := procGetDoubleClickTime.Call()
	return uint32(ret)
}
