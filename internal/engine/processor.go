package engine

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows"

	"madori/internal/logger"
	"madori/internal/models"
	"madori/internal/storage"
	"madori/internal/winapi"
)

// Constants
const (
	RestoreLatency         = 500  // default delay in ms from display change to restore
	SlowRestoreLatency     = 1000 // delay in ms from power resume to restore
	MaxRestoreLatency      = 2000 // max delay in ms from final restore pass to restore finish
	MinRestoreTimes        = 2    // minimum restore passes
	MaxRestoreTimes        = 5    // maximum restore passes
	CaptureLatency         = 3000 // delay in ms from window OS move to capture
	UserMoveLatency        = 1000 // delay from user move to capture, must < CaptureLatency
	ForegroundTimerLatency = UserMoveLatency / 4
	MaxUserMoves           = 4  // max user window moves per capture cycle
	MinWindowOsMoveEvents  = 12 // threshold of window move events by OS per cycle
	MaxSnapshots           = 38 // 0-9, a-z, ` and final one for undo
	MaxUserSnapshotID      = 36 // 0-9 (10), a-z (26), ` (1) = 37 slots, index 0-36
	MinClassNamePrefix     = 8  // allow partial class name matching

	SyncCommandTimeoutMs        = 200
	SyncCommandTaskbarTimeoutMs = 500
	SyncCommandRetryTimeoutMs   = 1500
)

// WindowEvent represents a WinEvent callback invocation, sent to the main goroutine.
type WindowEvent struct {
	EventType uint32
	HWnd      uintptr
	IDObject  int32
	IDChild   int32
	Time      time.Time
}

// Callbacks defines UI callbacks the engine invokes for status changes.
type Callbacks struct {
	ShowRestoreTip            func()
	HideRestoreTip            func(showIcon bool)
	ShowSnapshotCaptureTip    func(id int)
	ShowSnapshotRestoreTip    func(id int)
	EnableRestoreMenu         func(enableDB bool)
	EnableRestoreSnapshotMenu func(enable bool)
	ChangeIconText            func(text string)
}

// --- Sub-structs ---
// Embedded in Processor — field promotion means existing code referencing
// p.FieldName resolves transparently through the embedded struct.

type captureState struct {
	captureTimerStarted  int
	pendingMoveEvents    []uintptr
	lastCaptureRect      map[uintptr]winapi.RECT
	allUserMoveWindows   map[uintptr]bool
	noRestoreWindows     map[uintptr]bool
	noRestoreWindowsTmp  map[uintptr]bool
	lastKillTime         time.Time
	lastUnminimizeTime   time.Time
	lastUnminimizeWindow uintptr
	freezeCapture        bool
	userMove             bool
}

type restoreState struct {
	restoreTimes        int
	restoredWindows     map[uintptr]bool
	topmostWindowsFixed map[uintptr]bool
	unResponsiveWindows map[uintptr]bool
	slowResponseWindows map[uintptr]bool
	restoreHalted       bool
	HaltRestore         int
	deferredCommands    map[uintptr][]deferredCommand
	initCursorPos       winapi.POINT
	restoringFromMem    bool
	restoringSnapshot   bool
}

type featureFlags struct {
	FastRestore                  bool
	FixStacking                  int
	ShowDesktop                  bool
	RedrawDesktop                bool
	EnableOffScreenFix           bool
	EnhancedOffScreenFix         bool
	FixMinimizedRestore          bool
	CopyCornerPreference         bool
	PromptSessionRestore         bool
	AutoRestoreLiveWindowsFromDb bool
	enableDualPosSwitch          bool
	EnableTrayParking            bool
	rejectScaleFactorChange      bool
	captureFloatingWindow        bool
}

type processFilter struct {
	careProcess   map[string]bool
	ignoreProcess map[string]bool
}

type timerSet struct {
	foregroundTimer      *time.Timer
	captureTimer         *time.Timer
	restoreTimer         *time.Timer
	restoreFinishedTimer *time.Timer
}

type sessionState struct {
	sessionActive         bool
	sessionLocked         bool
	PauseAutoRestore      bool
	curDisplayKey         string
	prevDisplayKey        string
	lastDisplayChangeTime time.Time
	normalSessions        map[string]bool
}

// Processor is the main engine. It mirrors PersistentWindowProcessor in C#.
type Processor struct {
	mu sync.Mutex

	// Embedded sub-structs — fields promoted transparently.
	captureState
	restoreState
	featureFlags
	processFilter
	timerSet
	sessionState

	// In-memory databases
	monitorApplications map[string]map[uintptr]*models.WindowMetrics // snap_auto

	// Configuration
	AppDataFolder string

	// Window tracking
	windowTitle       map[uintptr]string
	windowTitleFast   map[uintptr]string
	windowProcessName map[uintptr]string
	processCmd        map[uint32]string

	// Foreground tracking
	foreGroundWindow     uintptr
	prevForeGroundWindow uintptr

	// Tray message window for parked icons
	trayHWnd uintptr

	// Dual position / special windows
	dualPosSwitchWindows map[uintptr]bool

	// Callbacks to UI
	callbacks Callbacks

	// WinEvent manager
	winEventMgr *winapi.WinEventManager

	// Virtual desktop support
	vdManager *winapi.VirtualDesktopManager

	// Channel for WinEvent → main goroutine
	eventCh chan WindowEvent
	quitCh  chan struct{}

	// User forced latencies
	UserForcedCaptureLatency int
	UserForcedRestoreLatency int

	// Snapshot tracking
	snapshotTakenTime  map[string]map[int]time.Time
	snapshotId         int
	snapshotAutoBackup map[uintptr]*models.WindowMetrics // auto-capture backup during snapshot restore

	// Icon state
	iconBusy bool

	// Database storage
	store *storage.Store

	// Tray parking state.
	// Guarded by mu. All readers and writers must hold the lock.
	// Internal methods (onMinimizeStart, onWindowShow) already hold mu
	// via handleWinEvent. Public methods (IsTrayParked,
	// GetTrayParkedWindows) acquire mu internally.
	trayParkedWindows map[uintptr]bool
	parkedIconUID     map[uintptr]uint32 // hwnd → tray icon UID
	parkedIconRev     map[uint32]uintptr // UID → hwnd
	nextParkedIconUID uint32
}

// A small wrapper around SendMessage params used internally
type deferredCommand struct {
	msg    uint32
	wParam uintptr
}

// New creates a new Processor with default settings.
func New() *Processor {
	return &Processor{
		// Sub-struct initialization
		captureState: captureState{
			lastCaptureRect:     make(map[uintptr]winapi.RECT),
			allUserMoveWindows:  make(map[uintptr]bool),
			noRestoreWindows:    make(map[uintptr]bool),
			noRestoreWindowsTmp: make(map[uintptr]bool),
		},
		restoreState: restoreState{
			restoredWindows:     make(map[uintptr]bool),
			topmostWindowsFixed: make(map[uintptr]bool),
			unResponsiveWindows: make(map[uintptr]bool),
			slowResponseWindows: make(map[uintptr]bool),
			deferredCommands:    make(map[uintptr][]deferredCommand),
			HaltRestore:         3000,
		},
		processFilter: processFilter{
			careProcess:   make(map[string]bool),
			ignoreProcess: make(map[string]bool),
		},
		sessionState: sessionState{
			normalSessions: make(map[string]bool),
		},

		// Top-level fields
		monitorApplications:  make(map[string]map[uintptr]*models.WindowMetrics),
		windowTitle:          make(map[uintptr]string),
		windowTitleFast:      make(map[uintptr]string),
		windowProcessName:    make(map[uintptr]string),
		processCmd:           make(map[uint32]string),
		dualPosSwitchWindows: make(map[uintptr]bool),
		snapshotTakenTime:    make(map[string]map[int]time.Time),
		trayParkedWindows:    make(map[uintptr]bool),
		parkedIconUID:        make(map[uintptr]uint32),
		parkedIconRev:        make(map[uint32]uintptr),
		nextParkedIconUID:    100, // must match FirstParkedIconUID in tray.go

		eventCh:     make(chan WindowEvent, 256),
		quitCh:      make(chan struct{}),
		winEventMgr: winapi.NewWinEventManager(),

		// Default values on sub-structs
		featureFlags: featureFlags{
			FastRestore:                  true,
			FixStacking:                  1,
			EnableOffScreenFix:           true,
			FixMinimizedRestore:          true,
			CopyCornerPreference:         true,
			enableDualPosSwitch:          true,
			EnableTrayParking:            true,
			captureFloatingWindow:        true,
			rejectScaleFactorChange:      true,
			AutoRestoreLiveWindowsFromDb: true,
		},
	}
}

// SetStore sets the database storage reference.
func (p *Processor) SetStore(s *storage.Store) {
	p.store = s
}

// SetCallbacks sets the UI callback functions.
func (p *Processor) SetCallbacks(cb Callbacks) {
	p.callbacks = cb
}

// Start initializes the engine: sets up WinEvent hooks and captures initial state.
func (p *Processor) Start(autoRestoreFromDB, autoRestoreLastCapture bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Init virtual desktop support
	p.vdManager = winapi.InitVirtualDesktopManager()

	p.sessionActive = true

	// Set this thread to per-monitor DPI-aware V2. The process-level
	// SetProcessDpiAwarenessContext call covers most threads, but
	// Go may schedule goroutines onto OS threads created before that
	// call. We never restore — every thread this app touches should
	// use physical coordinates.
	winapi.SetThreadDpiAwarenessContext(winapi.DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)

	p.curDisplayKey = winapi.GetDisplayKey()
	p.prevDisplayKey = p.curDisplayKey
	p.normalSessions[p.curDisplayKey] = true
	logger.Event(logger.LevelInfo, "", "Display config is %s", p.curDisplayKey)

	// Load persisted state from database
	p.LoadFromDB()

	winapi.GetCursorPos(&p.initCursorPos)

	// Setup WinEvent hooks
	p.winEventMgr.SetupStandardHooks(p.winEventCallback)

	// Start event processing goroutine
	go p.ProcessEvents()

	// Do initial capture of all windows
	p.CaptureNewDisplayConfig(p.curDisplayKey)
	p.CaptureWindowsOfInterest(p.curDisplayKey)

	// Persist initial capture to disk so state survives crashes
	p.PersistToDB()

	// Capture process command lines so ProcessExePath is populated
	p.CaptureProcessInfo()

	// Restore any windows that were parked when the process last exited
	// (crash recovery). In a normal shutdown RestoreAllParked clears the
	// DB, so this only finds entries when the previous session ended
	// abnormally.
	p.restoreOrphanedParkedWindows()

	// Start foreground timer
	p.startForegroundTimer()
	// Schedule the first automatic capture cycle with a direct timer.
	// Using time.AfterFunc instead of startCaptureTimer means subsequent
	// WinEvents (which call startCaptureTimer → Stop) cannot reset it.
	time.AfterFunc(time.Duration(CaptureLatency)*time.Millisecond, p.onCaptureTimer)

	logger.Event(logger.LevelInfo, "", "Startup complete, listening for window events")

	return nil
}

// Stop shuts down the engine gracefully.
func (p *Processor) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.quitCh)
	p.winEventMgr.UnhookAll()

	if p.foregroundTimer != nil {
		p.foregroundTimer.Stop()
	}
	if p.captureTimer != nil {
		p.captureTimer.Stop()
	}
	if p.restoreTimer != nil {
		p.restoreTimer.Stop()
	}
	if p.restoreFinishedTimer != nil {
		p.restoreFinishedTimer.Stop()
	}
}

// winEventCallback is the WinEvent hook callback. Runs on an arbitrary thread.
func (p *Processor) winEventCallback(hWinEventHook uintptr, eventType uint32, hwnd uintptr,
	idObject int32, idChild int32, dwEventThread, dwmsEventTime uint32) {

	select {
	case p.eventCh <- WindowEvent{
		EventType: eventType,
		HWnd:      hwnd,
		IDObject:  idObject,
		IDChild:   idChild,
		Time:      time.Now(),
	}:
	default:
	}
}

// ProcessEvents reads from the event channel and dispatches to handlers.
func (p *Processor) ProcessEvents() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(logger.LevelError, "", "Event processing panic: %v", r)
		}
	}()
	eventCount := 0
	for {
		select {
		case evt := <-p.eventCh:
			eventCount++
			if eventCount%1000 == 1 {
				logger.AutoCapture(logger.LevelDebug, "events received", "%d events, last type=%d", eventCount, evt.EventType)
			}
			p.handleWinEvent(evt)
		case <-p.quitCh:
			return
		}
	}
}

func (p *Processor) handleWinEvent(evt WindowEvent) {
	// Filter out non-window objects (buttons, scrollbars, text, etc.).
	// OBJID_WINDOW=0 means a top-level or child window; idChild!=0 means
	// a sub-element within a window. Without this filter, every internal
	// UI element change (e.g. taskbar clock ticks) resets the capture timer
	// and prevents it from ever firing.
	if evt.IDObject != 0 || evt.IDChild != 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !winapi.IsWindow(evt.HWnd) {
		return
	}

	switch evt.EventType {
	case winapi.EVENT_SYSTEM_FOREGROUND:
		p.onForegroundChange(evt)
	case winapi.EVENT_SYSTEM_MOVESIZESTART:
		p.onMoveSizeStart(evt)
	case winapi.EVENT_SYSTEM_MOVESIZEEND:
		p.onMoveSizeEnd(evt)
	case winapi.EVENT_SYSTEM_MINIMIZESTART:
		p.onMinimizeStart(evt)
	case winapi.EVENT_SYSTEM_MINIMIZEEND:
		p.onMinimizeEnd(evt)
	case winapi.EVENT_OBJECT_LOCATIONCHANGE:
		p.onLocationChange(evt)
	case winapi.EVENT_OBJECT_CREATE:
		p.onWindowCreate(evt)
	case winapi.EVENT_OBJECT_SHOW:
		p.onWindowShow(evt)
	case winapi.EVENT_OBJECT_HIDE:
		p.onWindowHide(evt)
	case winapi.EVENT_OBJECT_DESTROY:
		p.onWindowDestroy(evt)
	}
}

func (p *Processor) onForegroundChange(evt WindowEvent) {
	if !p.sessionActive {
		return
	}
	p.prevForeGroundWindow = p.foreGroundWindow
	p.foreGroundWindow = evt.HWnd

	logger.WindowEvent(logger.LevelDebug, "foreground changed", "to %s", p.WindowDesc(evt.HWnd))
	p.startForegroundTimer()
}

func (p *Processor) onMoveSizeStart(evt WindowEvent) {
	p.allUserMoveWindows[evt.HWnd] = true
	logger.WindowEvent(logger.LevelDebug, "move-size start", "%s", p.WindowDesc(evt.HWnd))
}

func (p *Processor) onMoveSizeEnd(evt WindowEvent) {
	p.userMove = true
	logger.WindowEvent(logger.LevelDebug, "move-size end", "%s", p.WindowDesc(evt.HWnd))
	p.resetCaptureTimer(UserMoveLatency)
}

func (p *Processor) onMinimizeStart(evt WindowEvent) {
	logger.WindowEvent(logger.LevelDebug, "minimize start", "%s", p.WindowDesc(evt.HWnd))

	// Never park programmatic minimizes that happen during restore —
	// they are re-minimizations to match the captured state, not
	// user-initiated minimize actions.
	if p.restoringFromMem {
		return
	}

	// Shift+minimize → park to tray.
	// Let the minimize proceed normally (window shrinks to taskbar), then
	// hide it underneath — the minimize animation covers the flicker.
	//
	// Shift state is tracked by a WH_KEYBOARD_LL hook (installed in
	// StartTrayParking) so we don't race with the out-of-context
	// WinEvent dispatch.  A 300 ms grace period after Shift release
	// handles the case where the user lifts Shift before the callback.
	if p.EnableTrayParking && evt.HWnd != 0 {
		if isShiftDownOrRecent() {
			style := winapi.GetWindowLong(evt.HWnd, winapi.GWL_STYLE)
			if (style&winapi.WS_MINIMIZEBOX) == 0 || !winapi.IsWindowVisible(evt.HWnd) {
				return
			}
			winapi.ShowWindowAsync(evt.HWnd, winapi.SW_HIDE)
			p.trayParkedWindows[evt.HWnd] = true
			p.persistParkedWindows()
			logger.Parking(logger.LevelInfo, "shift-minimized to tray", "%s", p.WindowDesc(evt.HWnd))
			winapi.PostMessage(p.trayHWnd, winapi.WM_APP_PARKED, uintptr(evt.HWnd), 0)
		}
	}
}

func (p *Processor) onMinimizeEnd(evt WindowEvent) {
	p.lastUnminimizeTime = time.Now()
	p.lastUnminimizeWindow = evt.HWnd
	p.userMove = true
	logger.WindowEvent(logger.LevelDebug, "minimize end", "%s", p.WindowDesc(evt.HWnd))
	p.resetCaptureTimer(UserMoveLatency)

	// Treat the unminimized window as foreground so the foreground
	// timer fires ActivateWindow to restore its pre-display-change
	// position (if fixMinimizedRestore is enabled).
	p.foreGroundWindow = evt.HWnd
	p.startForegroundTimer()
}

func (p *Processor) onLocationChange(evt WindowEvent) {
	if !p.sessionActive {
		return
	}
	// Only care about windows we're actually tracking.
	if !p.isTrackedWindow(evt.HWnd) {
		return
	}
	// Filter out spurious LOCATIONCHANGE events (Windows fires these on
	// mouse movement over tracked windows for accessibility). Only start
	// the capture timer if the window actually moved.
	var rect winapi.RECT
	if !winapi.GetWindowRect(evt.HWnd, &rect) {
		return
	}
	if cached, ok := p.lastCaptureRect[evt.HWnd]; ok && cached.Equals(rect) {
		return // position hasn't changed — ignore
	}
	p.lastCaptureRect[evt.HWnd] = rect

	p.pendingMoveEvents = append(p.pendingMoveEvents, evt.HWnd)
	if len(p.pendingMoveEvents) > 100 {
		p.pendingMoveEvents = p.pendingMoveEvents[len(p.pendingMoveEvents)-100:]
	}
	p.startCaptureTimer(CaptureLatency)
}

func (p *Processor) onWindowCreate(evt WindowEvent) {
	if !p.sessionActive {
		return
	}
	// Only react to visible top-level windows — not system popups or tooltips.
	// Otherwise every temporary system window resets the timer endlessly.
	visible := winapi.IsWindowVisible(evt.HWnd)
	topLevel := winapi.IsTopLevelWindow(evt.HWnd)
	if !visible || !topLevel {
		// Log filtered-out windows to help debug cases like Chrome Ctrl+N
		// where the window may be created hidden and only shown later.
		if cls := GetWindowClassName(evt.HWnd); cls != "" || GetWindowTitle(evt.HWnd) != "" {
			logger.Filtered(logger.LevelDebug, "window create filtered", "%s visible=%v topLevel=%v",
				p.WindowDesc(evt.HWnd), visible, topLevel)
		}
		return
	}
	// Don't arm a capture timer for a full-screen window (game launch,
	// browser full-screen video, etc.).
	if p.isFullScreen(evt.HWnd) {
		logger.AutoCapture(logger.LevelDebug, "full-screen window created", "%s — skipping capture timer",
			p.WindowDesc(evt.HWnd))
		return
	}
	logger.WindowEvent(logger.LevelDebug, "window create", "%s starting capture timer", p.WindowDesc(evt.HWnd))
	p.startCaptureTimer(CaptureLatency / 2)
}

// onWindowShow handles windows that become visible after being created hidden.
// EVENT_OBJECT_SHOW (0x8002) fires when visibility transitions from hidden→visible.
// Applications like Chrome create new windows invisible and then show them — those
// windows were filtered out by onWindowCreate's IsWindowVisible check, so we catch
// them here.
func (p *Processor) onWindowShow(evt WindowEvent) {
	if !p.sessionActive {
		return
	}

	// If this window was parked to tray and something else restored it
	// (native app tray icon, Alt+Tab, etc.), clean up our parked state
	// so our tray icon doesn't linger.
	if p.trayParkedWindows[evt.HWnd] {
		delete(p.trayParkedWindows, evt.HWnd)
		p.persistParkedWindows()
		p.removeParkedTrayIcon(evt.HWnd)
		logger.Parking(logger.LevelDebug, "externally restored", "%s — removed parked icon", p.WindowDesc(evt.HWnd))
	}

	// Still need the same filters — EVENT_OBJECT_SHOW also fires for menus,
	// tooltips, and child controls becoming visible.
	if !winapi.IsWindowVisible(evt.HWnd) || !winapi.IsTopLevelWindow(evt.HWnd) {
		return
	}
	// Don't arm a capture timer for a full-screen window (game launch,
	// browser full-screen video, etc.).
	if p.isFullScreen(evt.HWnd) {
		logger.AutoCapture(logger.LevelDebug, "full-screen window shown", "%s — skipping capture timer",
			p.WindowDesc(evt.HWnd))
		return
	}
	logger.WindowEvent(logger.LevelDebug, "window show", "%s starting capture timer", p.WindowDesc(evt.HWnd))
	p.startCaptureTimer(CaptureLatency / 2)
}

// onWindowHide handles windows that become invisible (hidden but not destroyed).
// EVENT_OBJECT_HIDE (0x8003) fires when visibility transitions from visible→hidden.
// We stop tracking the window — it's no longer meaningful to capture its position.
func (p *Processor) onWindowHide(evt WindowEvent) {
	if !p.sessionActive {
		return
	}
	// Only react to windows we actually track — ignore random system popups
	// becoming invisible.
	if !p.isTrackedWindow(evt.HWnd) {
		return
	}
	logger.WindowEvent(logger.LevelDebug, "window hide", "%s", p.WindowDesc(evt.HWnd))
	for dk := range p.monitorApplications {
		delete(p.monitorApplications[dk], evt.HWnd)
	}
	p.startCaptureTimer(CaptureLatency)
}

func (p *Processor) onWindowDestroy(evt WindowEvent) {
	if !p.sessionActive {
		return
	}
	p.lastKillTime = time.Now()
	delete(p.windowTitle, evt.HWnd)
	delete(p.windowTitleFast, evt.HWnd)
	delete(p.windowProcessName, evt.HWnd)

	// Only react to windows we actually track — don't reset the capture
	// timer for every random system window that gets destroyed.
	if !p.isTrackedWindow(evt.HWnd) {
		return
	}
	logger.WindowEvent(logger.LevelDebug, "window kill", "%s", p.WindowDesc(evt.HWnd))
	for dk := range p.monitorApplications {
		delete(p.monitorApplications[dk], evt.HWnd)
	}
	p.startCaptureTimer(CaptureLatency)
}

func (p *Processor) startForegroundTimer() {
	if p.foregroundTimer != nil {
		p.foregroundTimer.Stop()
	}
	p.foregroundTimer = time.AfterFunc(
		time.Duration(ForegroundTimerLatency)*time.Millisecond,
		p.onForegroundTimer,
	)
}

func (p *Processor) startCaptureTimer(delayMs int) {
	// Pause automatic captures when the foreground window is full-screen
	// (game, video player in exclusive mode, etc.).
	if p.isFullScreen(p.foreGroundWindow) {
		return
	}
	if p.restoringFromMem {
		return
	}
	// If a timer is already pending, don't reset it. Continuous noisy
	// events (LOCATIONCHANGE from cursor blinks, OBJECT_CREATE from
	// tooltips) would otherwise prevent the timer from ever expiring.
	if p.captureTimerStarted > 0 {
		return
	}
	if p.captureTimer != nil {
		p.captureTimer.Stop()
	}
	p.captureTimerStarted++
	actualDelay := delayMs
	if p.UserForcedCaptureLatency > 0 {
		actualDelay = p.UserForcedCaptureLatency
	}
	p.captureTimer = time.AfterFunc(
		time.Duration(actualDelay)*time.Millisecond,
		p.onCaptureTimer,
	)
}

// resetCaptureTimer forces a (possibly shorter) capture timer, replacing
// any pending one. Used only for definitive user actions like move-size-end.
func (p *Processor) resetCaptureTimer(delayMs int) {
	if p.captureTimer != nil {
		p.captureTimer.Stop()
	}
	p.captureTimerStarted = 1
	actualDelay := delayMs
	if p.UserForcedCaptureLatency > 0 {
		actualDelay = p.UserForcedCaptureLatency
	}
	p.captureTimer = time.AfterFunc(
		time.Duration(actualDelay)*time.Millisecond,
		p.onCaptureTimer,
	)
}

// StartRestoreTimer schedules a batch restore.
func (p *Processor) StartRestoreTimer(milliSecond ...int) {
	if p.restoreTimer != nil {
		p.restoreTimer.Stop()
	}
	delay := RestoreLatency
	if len(milliSecond) > 0 {
		delay = milliSecond[0]
	}
	if p.UserForcedRestoreLatency > 0 {
		delay = p.UserForcedRestoreLatency
	}
	p.restoreTimer = time.AfterFunc(
		time.Duration(delay)*time.Millisecond,
		p.onRestoreTimer,
	)
}

// CancelRestoreTimer cancels a pending restore.
func (p *Processor) CancelRestoreTimer() {
	if p.restoreTimer != nil {
		p.restoreTimer.Stop()
	}
}

// CancelCaptureTimer cancels a pending capture.
func (p *Processor) CancelCaptureTimer() {
	if p.captureTimer != nil {
		p.captureTimer.Stop()
	}
	p.captureTimerStarted = 0
}

// StartRestoreFinishedTimer schedules the restore finished callback.
func (p *Processor) StartRestoreFinishedTimer(delayMs int) {
	if p.restoreFinishedTimer != nil {
		p.restoreFinishedTimer.Stop()
	}
	p.restoreFinishedTimer = time.AfterFunc(
		time.Duration(delayMs)*time.Millisecond,
		p.onRestoreFinishedTimer,
	)
}

func (p *Processor) onForegroundTimer() {
	p.mu.Lock()
	defer p.mu.Unlock()

	hwnd := p.foreGroundWindow
	if hwnd == 0 || !p.sessionActive || !winapi.IsWindow(hwnd) {
		return
	}

	// Check if in capture database
	tracked := false
	if apps, ok := p.monitorApplications[p.curDisplayKey]; ok {
		if _, ok := apps[hwnd]; ok {
			tracked = true
			p.userMove = true
			p.startCaptureTimer(UserMoveLatency / 2)
		}
	}
	// If the foreground window is a visible top-level window we haven't
	// tracked yet, start a capture cycle. This catches new windows that were
	// created invisible (e.g. Chrome Ctrl+N) and only became visible after
	// the CREATE event fired, so onWindowCreate filtered them out.
	if !tracked && winapi.IsWindowVisible(hwnd) && winapi.IsTopLevelWindow(hwnd) {
		logger.WindowEvent(logger.LevelDebug, "new foreground window", "%s starting capture", p.WindowDesc(hwnd))
		p.startCaptureTimer(CaptureLatency / 2)
	}

	p.ActivateWindow(hwnd)
}

func (p *Processor) onCaptureTimer() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.captureTimerStarted = 0
	if !p.sessionActive || p.restoringFromMem || p.freezeCapture {
		return
	}
	// Suppress capture if the foreground window is now full-screen,
	// even if it wasn't when the timer was started.
	if p.isFullScreen(p.foreGroundWindow) {
		logger.AutoCapture(logger.LevelDebug, "full-screen foreground", "%s — skipping capture cycle",
			p.WindowDesc(p.foreGroundWindow))
		return
	}
	logger.AutoCapture(logger.LevelDebug, "", "Capture timer fired, running capture cycle")
	p.BatchCaptureApplicationsOnCurrentDisplays()
	p.pendingMoveEvents = nil
}

func (p *Processor) onRestoreTimer() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.restoreTimes = 0
	p.restoredWindows = make(map[uintptr]bool)
	logger.AutoCapture(logger.LevelInfo, "restore started", "for %s", p.curDisplayKey)
	p.BatchRestoreApplicationsOnCurrentDisplays()
}

func (p *Processor) onRestoreFinishedTimer() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.RetryDeferredCommands()
	p.slowResponseWindows = make(map[uintptr]bool)
	p.unResponsiveWindows = make(map[uintptr]bool)
	p.restoringFromMem = false
	p.restoringSnapshot = false
	p.noRestoreWindowsTmp = make(map[uintptr]bool)

	// Check if display changed mid-restore — halt and retry after delay
	displayKey := winapi.GetDisplayKey()
	if p.restoreHalted || displayKey != p.curDisplayKey {
		p.restoreHalted = false
		p.lastDisplayChangeTime = time.Now()
		logger.AutoCapture(logger.LevelInfo, "restore aborted", "for %s", displayKey)
		if p.HaltRestore > 0 {
			p.curDisplayKey = displayKey
			logger.AutoCapture(logger.LevelInfo, "restore restarted", "for %s after %dms", displayKey, p.HaltRestore)
			p.restoringFromMem = true
			time.AfterFunc(time.Duration(p.HaltRestore)*time.Millisecond, func() {
				p.onRestoreTimer()
			})
			return
		}
	}

	if p.callbacks.HideRestoreTip != nil {
		p.callbacks.HideRestoreTip(true)
	}
	p.iconBusy = false
	p.sessionActive = true
	if p.snapshotAutoBackup != nil {
		p.monitorApplications[p.curDisplayKey] = p.snapshotAutoBackup
		p.snapshotAutoBackup = nil
	}
}

// isTrackedWindow returns true if the window exists in our capture database.
func (p *Processor) isTrackedWindow(hwnd uintptr) bool {
	for _, apps := range p.monitorApplications {
		if _, ok := apps[hwnd]; ok {
			return true
		}
	}
	return false
}

// SetIgnoreProcess adds process names to ignore list.
func (p *Processor) SetIgnoreProcess(names string) {
	for _, name := range parseProcessList(names) {
		p.ignoreProcess[name] = true
	}
}

// SetCareProcess adds process names to care list.
func (p *Processor) SetCareProcess(names string) {
	for _, name := range parseProcessList(names) {
		p.careProcess[name] = true
	}
}

// normalizeProcessName returns a lowercased name with any .exe suffix stripped,
// so user-supplied filter lists and live process names match regardless of
// case or whether .exe is included on either side.
func normalizeProcessName(name string) string {
	n := strings.TrimSpace(name)
	n = strings.TrimSuffix(name, ".exe")
	return strings.ToLower(n)
}

// SetNoinheritProcess adds process names to no-inherit list.
func parseProcessList(input string) []string {
	var result []string
	for _, name := range strings.Split(input, ";") {
		n := normalizeProcessName(name)
		if n != "" {
			result = append(result, n)
		}
	}
	return result
}

func (p *Processor) GetDisplayKey() string {
	return winapi.GetDisplayKey()
}

// GetWindowTitle returns the title of a window.
func GetWindowTitle(hwnd uintptr) string {
	var buf [512]uint16
	n := winapi.InternalGetWindowText(hwnd, &buf, 512)
	if n > 0 {
		return windows.UTF16ToString(buf[:n])
	}
	return ""
}

// GetWindowClassName returns the class name of a window.
func GetWindowClassName(hwnd uintptr) string {
	var buf [256]uint16
	n := winapi.GetClassName(hwnd, &buf, 256)
	if n > 0 {
		return windows.UTF16ToString(buf[:n])
	}
	return ""
}

// FormatWindowDesc returns a consistent string representation of a window
// suitable for log output. Format:
//
//	-- Window Title (ProcessName.exe) --
//	-- [no title, class="ClassName"] (ProcessName.exe) --
func FormatWindowDesc(processName, title, className string) string {
	if title != "" {
		if len(title) > 42 {
			runes := []rune(title)
			if len(runes) > 42 {
				title = string(runes[:41]) + "…"
			}
		}
		return fmt.Sprintf("｢%s (%s)｣", title, processName)
	}
	return fmt.Sprintf("｢[no title, class=%q] (%s)｣", className, processName)
}

// WindowDesc returns a formatted window description for logging, using the
// cached process name when available (falling back to a live PID lookup).
func (p *Processor) WindowDesc(hwnd uintptr) string {
	processName := p.windowProcessName[hwnd]
	if processName == "" {
		var pid uint32
		_, pid = winapi.GetWindowThreadProcessId(hwnd)
		if pid != 0 {
			processName = winapi.GetProcessName(pid)
		}
	}
	if processName == "" {
		processName = "???"
	}
	return FormatWindowDesc(processName, GetWindowTitle(hwnd), GetWindowClassName(hwnd))
}

// IsMinimized checks if a window is minimized.
func IsMinimized(hwnd uintptr) bool {
	if winapi.IsIconic(hwnd) || !winapi.IsWindowVisible(hwnd) {
		return true
	}
	style := winapi.GetWindowLong(hwnd, winapi.GWL_STYLE)
	return (style & winapi.WS_MINIMIZE) != 0
}

// OnSessionLock handles session lock events.
func (p *Processor) OnSessionLock() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessionLocked = true
	p.sessionActive = false
	p.EndDisplaySession()
	logger.AutoCapture(logger.LevelInfo, "session locked", "")
}

// OnSessionUnlock handles session unlock events.
func (p *Processor) OnSessionUnlock() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessionLocked = false
	p.restoringFromMem = true
	p.StartRestoreTimer()
	logger.AutoCapture(logger.LevelInfo, "session unlocked", "restoring windows")
}

// OnPowerSuspend handles system suspend events.
func (p *Processor) OnPowerSuspend() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessionActive = false
	if !p.sessionLocked {
		p.EndDisplaySession()
	}
	logger.AutoCapture(logger.LevelInfo, "system suspending", "")
}

// NoteDisplayChange handles a display configuration change (monitor
// connect/disconnect, resolution change). It records the change time
// (used by ActivateWindow for fixMinimizedRestore) and triggers an
// automatic restore so windows are repositioned for the new layout.
func (p *Processor) NoteDisplayChange() {
	p.mu.Lock()
	p.lastDisplayChangeTime = time.Now()
	p.restoringFromMem = true
	p.mu.Unlock()
	p.StartRestoreTimer()
}

// OnPowerResume handles system resume events.
func (p *Processor) OnPowerResume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.sessionLocked {
		p.restoringFromMem = true
		p.StartRestoreTimer(SlowRestoreLatency)
	}
	logger.AutoCapture(logger.LevelInfo, "system resuming", "")
}
