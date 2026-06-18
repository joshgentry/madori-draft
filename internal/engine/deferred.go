package engine

import "durablewindows/internal/winapi"

// TrySendSyncCommand sends a synchronous message to a window with a timeout.
// If the window doesn't respond in time, the command is queued for a final
// retry after the main restore passes finish.
func (p *Processor) TrySendSyncCommand(hwnd uintptr, msg uint32, wParam uintptr, timeoutMs uint32) bool {
	if p.slowResponseWindows[hwnd] {
		p.enqueueDeferredCommand(hwnd, msg, wParam)
		return false
	}

	ret, _ := winapi.SendMessageTimeout(hwnd, msg, wParam, 0,
		winapi.SMTO_ABORTIFHUNG|winapi.SMTO_NORMAL, timeoutMs)

	if ret == 0 {
		if !p.slowResponseWindows[hwnd] {
			p.slowResponseWindows[hwnd] = true
		}
		p.enqueueDeferredCommand(hwnd, msg, wParam)
		return false
	}
	return true
}

func (p *Processor) enqueueDeferredCommand(hwnd uintptr, msg uint32, wParam uintptr) {
	key := command{kind: int(msg), val: int(wParam)}
	for _, existing := range p.deferredCommands[hwnd] {
		if existing == key {
			return // already queued
		}
	}
	p.deferredCommands[hwnd] = append(p.deferredCommands[hwnd], key)
}
