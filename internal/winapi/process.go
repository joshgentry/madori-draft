package winapi

import (
	"syscall"
	"unsafe"
)

var (
	modKernel32Extra             = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = modKernel32Extra.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = modKernel32Extra.NewProc("Process32FirstW")
	procProcess32NextW           = modKernel32Extra.NewProc("Process32NextW")
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
)

// PROCESSENTRY32 describes a process from a toolhelp snapshot.
type PROCESSENTRY32 struct {
	DwSize              uint32
	CntUsage            uint32
	Th32ProcessID       uint32
	Th32DefaultHeapID   uintptr
	Th32ModuleID        uint32
	CntThreads          uint32
	Th32ParentProcessID uint32
	PcPriClassBase      int32
	DwFlags             uint32
	SzExeFile           [260]uint16
}

// CreateToolhelp32Snapshot takes a snapshot of processes.
func CreateToolhelp32Snapshot(dwFlags, th32ProcessID uint32) uintptr {
	ret, _, _ := procCreateToolhelp32Snapshot.Call(uintptr(dwFlags), uintptr(th32ProcessID))
	return ret
}

// Process32First retrieves the first process from a snapshot.
func Process32First(hSnapshot uintptr, lppe *PROCESSENTRY32) bool {
	lppe.DwSize = uint32(unsafe.Sizeof(PROCESSENTRY32{}))
	ret, _, _ := procProcess32FirstW.Call(hSnapshot, uintptr(unsafe.Pointer(lppe)))
	return ret != 0
}

// Process32Next retrieves the next process from a snapshot.
func Process32Next(hSnapshot uintptr, lppe *PROCESSENTRY32) bool {
	ret, _, _ := procProcess32NextW.Call(hSnapshot, uintptr(unsafe.Pointer(lppe)))
	return ret != 0
}

// GetProcessName returns the executable name for a process ID.
func GetProcessName(pid uint32) string {
	snapshot := CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		return ""
	}
	defer CloseHandle(snapshot)

	var pe PROCESSENTRY32
	if !Process32First(snapshot, &pe) {
		return ""
	}

	for {
		if pe.Th32ProcessID == pid {
			return syscall.UTF16ToString(pe.SzExeFile[:])
		}
		if !Process32Next(snapshot, &pe) {
			break
		}
	}
	return ""
}
