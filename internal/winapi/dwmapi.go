package winapi

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modDwmapi                 = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmGetWindowAttribute = modDwmapi.NewProc("DwmGetWindowAttribute")
	procDwmSetWindowAttribute = modDwmapi.NewProc("DwmSetWindowAttribute")
)

// DwmGetWindowAttribute retrieves a DWM attribute for a window.
// Returns false if the call fails (HRESULT != S_OK).
func DwmGetWindowAttribute(hwnd uintptr, dwAttribute uint32, pvAttribute unsafe.Pointer, cbAttribute uint32) bool {
	ret, _, _ := procDwmGetWindowAttribute.Call(hwnd, uintptr(dwAttribute), uintptr(pvAttribute), uintptr(cbAttribute))
	return ret == 0 // S_OK
}

// DwmSetWindowAttribute sets a DWM attribute for a window.
// On pre-Win11 builds the function doesn't exist in dwmapi.dll and
// LazyProc.Find panics with a DLLError. Callers must recover.
// Returns false if the call fails (HRESULT != S_OK).
func DwmSetWindowAttribute(hwnd uintptr, dwAttribute uint32, pvAttribute unsafe.Pointer, cbAttribute uint32) bool {
	ret, _, _ := procDwmSetWindowAttribute.Call(hwnd, uintptr(dwAttribute), uintptr(pvAttribute), uintptr(cbAttribute))
	return ret == 0 // S_OK
}
