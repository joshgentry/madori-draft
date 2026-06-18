//go:build !windows

package storage

import (
	"os"
	"syscall"
)

// isProcessRunning checks whether a process with the given PID is still running.
// On Unix, FindProcess always succeeds, so we send signal 0 (null signal)
// to test whether the process exists.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 performs existence check without sending an actual signal
	return process.Signal(syscall.Signal(0)) == nil
}
