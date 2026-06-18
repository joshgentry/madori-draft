package engine

import "golang.org/x/sys/windows"

var normalPriority uint32 = windows.NORMAL_PRIORITY_CLASS

// boostPriority raises the process priority during restore operations.
func boostPriority() {
	current := windows.CurrentProcess()
	var err error
	normalPriority, err = windows.GetPriorityClass(current)
	if err != nil {
		normalPriority = windows.NORMAL_PRIORITY_CLASS
	}
	windows.SetPriorityClass(current, windows.HIGH_PRIORITY_CLASS)
}

// restorePriority returns the process to its normal priority.
func restorePriority() {
	windows.SetPriorityClass(windows.CurrentProcess(), normalPriority)
}
