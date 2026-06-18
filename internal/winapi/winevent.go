package winapi

import "sync"

// WinEventHook manages a single WinEvent hook.
type WinEventHook struct {
	handle uintptr
}

// WinEventManager manages multiple WinEvent hooks for the application.
type WinEventManager struct {
	mu    sync.Mutex
	hooks []WinEventHook
}

// NewWinEventManager creates a new WinEvent hook manager.
func NewWinEventManager() *WinEventManager {
	return &WinEventManager{}
}

// AddHook registers a new WinEvent hook with the given parameters.
// The callback is invoked on an arbitrary thread by the Windows event system.
func (m *WinEventManager) AddHook(
	eventMin, eventMax uint32,
	callback WinEventDelegate,
	flags uint32,
) error {
	handle := SetWinEventHook(
		eventMin, eventMax,
		0, // hmodWinEventProc — must be 0 for out-of-context hooks
		callback,
		0, // idProcess — 0 = all processes
		0, // idThread — 0 = all threads
		flags,
	)
	m.mu.Lock()
	m.hooks = append(m.hooks, WinEventHook{handle: handle})
	m.mu.Unlock()
	return nil
}

// SetupStandardHooks registers all the standard WinEvent hooks used by DurableWindows.
// The provided callback receives all window events for processing.
func (m *WinEventManager) SetupStandardHooks(callback WinEventDelegate) {
	// Foreground window changes
	m.AddHook(
		EVENT_SYSTEM_FOREGROUND,
		EVENT_SYSTEM_FOREGROUND,
		callback,
		WINEVENT_OUTOFCONTEXT,
	)

	// User dragging/resizing (start and end)
	m.AddHook(
		EVENT_SYSTEM_MOVESIZESTART,
		EVENT_SYSTEM_MOVESIZEEND,
		callback,
		WINEVENT_OUTOFCONTEXT,
	)

	// Minimize/restore
	m.AddHook(
		EVENT_SYSTEM_MINIMIZESTART,
		EVENT_SYSTEM_MINIMIZEEND,
		callback,
		WINEVENT_OUTOFCONTEXT,
	)

	// Window location changes — needed for modern window frameworks
	// (ConHost, UWP, CASCADIA_HOSTING_WINDOW_CLASS) that don't fire
	// EVENT_SYSTEM_MOVESIZEEND. onLocationChange filters to tracked
	// windows only, so mouse-move accessibility noise is ignored.
	/*
		m.AddHook(
			EVENT_OBJECT_LOCATIONCHANGE,
			EVENT_OBJECT_LOCATIONCHANGE,
			callback,
			WINEVENT_OUTOFCONTEXT,
		)
	*/

	// Window create/close
	m.AddHook(
		EVENT_OBJECT_CREATE,
		EVENT_OBJECT_DESTROY,
		callback,
		WINEVENT_OUTOFCONTEXT,
	)

	// Window show/hide — fires when a window's visibility changes.
	// Show catches windows created invisible (e.g. Chrome Ctrl+N) that
	// EVENT_OBJECT_CREATE missed. Hide catches windows that become
	// invisible without being destroyed.
	m.AddHook(
		EVENT_OBJECT_SHOW,
		EVENT_OBJECT_HIDE,
		callback,
		WINEVENT_OUTOFCONTEXT,
	)
}

// UnhookAll removes all registered WinEvent hooks.
func (m *WinEventManager) UnhookAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, hook := range m.hooks {
		if hook.handle != 0 {
			UnhookWinEvent(hook.handle)
		}
	}
	m.hooks = nil
}
