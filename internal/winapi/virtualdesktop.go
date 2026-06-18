package winapi

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modCombaseVD = syscall.NewLazyDLL("combase.dll")
	modOle32VD   = syscall.NewLazyDLL("ole32.dll")

	procCoCreateInstanceVD = modCombaseVD.NewProc("CoCreateInstance")
	procCoInitializeExVD   = modOle32VD.NewProc("CoInitializeEx")
	procStringFromGUID2VD  = modOle32VD.NewProc("StringFromGUID2")
)

const (
	CLSCTX_INPROC_SERVER     = 1
	COINIT_APARTMENTTHREADED = 0x2
)

var (
	CLSID_VirtualDesktopManager = windows.GUID{
		Data1: 0xaa509086, Data2: 0x5ca9, Data3: 0x4c25,
		Data4: [8]byte{0x8f, 0x95, 0x58, 0x9d, 0x3c, 0x07, 0xb4, 0x8a},
	}
	IID_IVirtualDesktopManager = windows.GUID{
		Data1: 0xa5cd92ff, Data2: 0x29be, Data3: 0x454c,
		Data4: [8]byte{0x8d, 0x04, 0xd8, 0x28, 0x79, 0xfb, 0x3f, 0x1b},
	}
)

type VirtualDesktopManager struct {
	iface   uintptr
	enabled bool

	// Cached vtable method pointers
	pIsWindowOnCurrentVirtualDesktop uintptr
	pGetWindowDesktopId              uintptr
	pMoveWindowToDesktop             uintptr
}

var vdManager *VirtualDesktopManager

func InitVirtualDesktopManager() *VirtualDesktopManager {
	if vdManager != nil {
		return vdManager
	}

	procCoInitializeExVD.Call(0, COINIT_APARTMENTTHREADED)

	var iface uintptr
	hr, _, _ := procCoCreateInstanceVD.Call(
		uintptr(unsafe.Pointer(&CLSID_VirtualDesktopManager)),
		0,
		CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(&IID_IVirtualDesktopManager)),
		uintptr(unsafe.Pointer(&iface)),
	)
	if hr != 0 || iface == 0 {
		vdManager = &VirtualDesktopManager{enabled: false}
		return vdManager
	}

	// Extract vtable methods
	vtbl := *(**uintptr)(unsafe.Pointer(iface))
	if vtbl == nil {
		vdManager = &VirtualDesktopManager{enabled: false}
		return vdManager
	}

	getMethod := func(index int) uintptr {
		addr := uintptr(unsafe.Pointer(vtbl)) + uintptr(index)*unsafe.Sizeof(uintptr(0))
		return *(*uintptr)(unsafe.Pointer(addr))
	}

	vdManager = &VirtualDesktopManager{
		iface:                            iface,
		enabled:                          true,
		pIsWindowOnCurrentVirtualDesktop: getMethod(3),
		pGetWindowDesktopId:              getMethod(4),
		pMoveWindowToDesktop:             getMethod(5),
	}

	// Verify at least one method pointer is valid
	if vdManager.pGetWindowDesktopId == 0 || vdManager.pIsWindowOnCurrentVirtualDesktop == 0 {
		vdManager.enabled = false
	}

	return vdManager
}

func (v *VirtualDesktopManager) Enabled() bool {
	return v != nil && v.enabled
}

// callSafely wraps a COM method call with a recover to prevent crashes.
func (v *VirtualDesktopManager) callSafely(method uintptr, args ...uintptr) (hr uintptr) {
	if !v.enabled || method == 0 {
		return 1 // non-zero = failure
	}
	defer func() { recover() }()
	allArgs := append([]uintptr{v.iface}, args...)
	hr, _, _ = syscall.SyscallN(method, allArgs...)
	return
}

func (v *VirtualDesktopManager) IsWindowOnCurrentVirtualDesktop(hwnd uintptr) bool {
	if !v.Enabled() {
		return true
	}
	var result int32
	hr := v.callSafely(v.pIsWindowOnCurrentVirtualDesktop, hwnd, uintptr(unsafe.Pointer(&result)))
	if hr != 0 {
		return true
	}
	return result != 0
}

func (v *VirtualDesktopManager) GetWindowDesktopId(hwnd uintptr) windows.GUID {
	var guid windows.GUID
	if !v.Enabled() {
		return guid
	}
	v.callSafely(v.pGetWindowDesktopId, hwnd, uintptr(unsafe.Pointer(&guid)))
	return guid
}

func (v *VirtualDesktopManager) MoveWindowToDesktop(hwnd uintptr, guid windows.GUID) {
	if !v.Enabled() {
		return
	}
	v.callSafely(v.pMoveWindowToDesktop, hwnd, uintptr(unsafe.Pointer(&guid)))
}

func (v *VirtualDesktopManager) GetCurrentDesktopId() windows.GUID {
	progman := FindWindowExW(0, 0, windows.StringToUTF16Ptr("Progman"), nil)
	if progman == 0 {
		return windows.GUID{}
	}
	return v.GetWindowDesktopId(progman)
}

func GUIDToString(guid windows.GUID) string {
	var buf [40]uint16
	ret, _, _ := procStringFromGUID2VD.Call(
		uintptr(unsafe.Pointer(&guid)),
		uintptr(unsafe.Pointer(&buf)),
		40,
	)
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:])
}
