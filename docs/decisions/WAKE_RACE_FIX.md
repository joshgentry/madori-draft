# Deadlock Fix: Recursive Mutex Lock in Restore-Abort Retry Path

## The Crash

After a sleep-wake cycle, the process deadlocked with all goroutines asleep:

```
fatal error: all goroutines are asleep - deadlock!
```

Four goroutines were all blocked trying to acquire `Processor.mu`:

| Goroutine | Location | Call Chain |
|---|---|---|
| 1 (main) | `OnSessionUnlock:941` | `windowProc` → `onSessionChange` → `OnSessionUnlock` |
| 18 | `handleWinEvent:359` | `ProcessEvents` event loop |
| 708 | `onRestoreTimer:700` | `AfterFunc` callback in `onRestoreFinishedTimer:733` |
| 709 | `AfterFunc` callback:731 | Another `AfterFunc` callback in `onRestoreFinishedTimer` |

## Root Cause

### The deadlock chain

The log shows what triggered it:

```
10:21:08 restore started — for Display0_…__Display1_…    (dual monitor)
10:21:11 restore aborted — for Display0_…                  (display changed to single)
10:21:11 restore restarted — for Display0_… after 3000ms
10:21:12 restore started — for Display0_…                  (single monitor)
10:21:14 restore aborted — for Display0_…__Display1_…     (display changed back to dual)
10:21:14 restore restarted — for Display0_…__Display1_… after 3000ms
→ deadlock
```

Each time a display change is detected mid-restore, `onRestoreFinishedTimer`
schedules a retry via `time.AfterFunc`. The callback **acquired `p.mu` and then
called `onRestoreTimer()`, which also acquires `p.mu`**:

```go
// processor.go:730-734 (BEFORE fix)
time.AfterFunc(time.Duration(p.HaltRestore)*time.Millisecond, func() {
    p.mu.Lock()          // ← goroutine acquires the mutex
    defer p.mu.Unlock()
    p.onRestoreTimer()   // ← tries to acquire the SAME mutex (line 700)
})
```

```go
// processor.go:699-701
func (p *Processor) onRestoreTimer() {
    p.mu.Lock()          // ← DEADLOCK: recursive lock on non-reentrant mutex
    defer p.mu.Unlock()
    ...
}
```

### Why Go's `sync.Mutex` doesn't allow this

Go's `sync.Mutex` is **non-reentrant**. When a goroutine calls `Lock()` on a
mutex it already holds, it blocks forever waiting for itself to unlock.
Because goroutine 708 is the one holding the lock (from its `Lock()` at the
callback's line 731), and it's waiting on itself (at `onRestoreTimer` line
700), no other goroutine can ever acquire the lock. The Go runtime detects
that all goroutines are blocked and reports the deadlock.

### Why this doesn't happen on every restore

The normal restore path never hits this code. `onRestoreTimer` is called
directly from `StartRestoreTimer`'s `AfterFunc` — without an extra lock:

```go
// processor.go:617-619 — CORRECT, no extra lock
p.restoreTimer = time.AfterFunc(
    time.Duration(delay)*time.Millisecond,
    p.onRestoreTimer,   // called without holding p.mu
)
```

The bug only triggers when a display change is detected **during** a restore.
The retry path in `onRestoreFinishedTimer` wraps the call in a lock → re-lock
→ deadlock.

On a sleep-wake, the display configuration often toggles briefly (monitors
re-detect, resolutions change, GPU re-initializes), which creates exactly the
conditions for this race: a restore starts, a display change aborts it, the
retry fires, and another display change creates a second retry whose callback
deadlocks.

## The Fix

Removed the redundant lock acquisition. `onRestoreTimer()` manages its own
locking:

```go
// processor.go:730-732 (AFTER fix)
time.AfterFunc(time.Duration(p.HaltRestore)*time.Millisecond, func() {
    p.onRestoreTimer()
})
```

The callback no longer holds `p.mu` when calling `onRestoreTimer`, so
`onRestoreTimer`'s own `Lock()` succeeds normally.

## Verification

- Cross-compiled for `windows/amd64` — builds cleanly.
- All other `AfterFunc` call sites audited — none have the same wrapper-lock-
  and-call pattern. The fix is isolated to this one location.
