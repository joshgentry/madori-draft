package storage

import (
	"syscall"
)

// isProcessRunning checks whether a process with the given PID is still running
// on Windows. os.FindProcess always succeeds on Windows (it just stores the PID),
// so we must call OpenProcess directly to get a real handle.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	// PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	const processQueryLimitedInfo = 0x1000
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	openProcess := kernel32.NewProc("OpenProcess")
	closeHandle := kernel32.NewProc("CloseHandle")

	handle, _, _ := openProcess.Call(
		uintptr(processQueryLimitedInfo),
		0, // bInheritHandle = FALSE
		uintptr(pid),
	)
	if handle == 0 {
		return false // OpenProcess failed — process doesn't exist
	}
	closeHandle.Call(handle)
	return true
}
