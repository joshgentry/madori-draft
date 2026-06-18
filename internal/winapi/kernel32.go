package winapi

// SetConsoleCtrlHandler registers a console control handler callback.
// handlerRoutine must be created with syscall.NewCallback and have the
// signature: func(dwCtrlType uintptr) uintptr.
// If add is true, the handler is registered; if false, it is removed.
// Returns true on success.
//
// The caller must keep the handlerRoutine callback alive for the
// entire time it is registered — Windows holds a raw pointer to it.
func SetConsoleCtrlHandler(handlerRoutine uintptr, add bool) bool {
	var addVal uintptr
	if add {
		addVal = 1
	}
	ret, _, _ := procSetConsoleCtrlHandler.Call(handlerRoutine, addVal)
	return ret != 0
}
