# Event Gates

The engine has three boolean flags that gate event handling and capture. They act as a "mode switch" controlling whether the engine is in normal operation, restore, or paused.

## `sessionActive`

**Field:** `Processor.sessionActive` (`processor.go:78`)

**Purpose:** Master gate on window event handlers. When `false`, the engine is in a quiet mode — it ignores window events and defers capture.

**Set to `true`:**
| Location | Trigger |
|---|---|
| `Start()` | Engine initial activation |

**Set to `false`:**
| Location | Trigger |
|---|---|
| `BatchRestoreApplicationsOnCurrentDisplays()` | Beginning of every restore pass — prevents handlers from firing while the engine moves windows |
| `OnSessionLock()` | Workstation locked (Win+L) |
| `OnPowerSuspend()` | System going to sleep |

**Read by (all guard clauses that return early when `false`):**
| Handler | What it skips |
|---|---|
| `onForegroundChange()` | Foreground window tracking |
| `onLocationChange()` | Window move tracking |
| `onWindowCreate()` | New window discovery |
| `onWindowShow()` | Window show tracking (includes parked-window cleanup) |
| `onWindowHide()` | Window hide tracking |
| `onWindowDestroy()` | Window destruction cleanup |
| `onForegroundTimer()` | Deferred foreground checks |
| `onCaptureTimer()` | Capture cycle execution (combined with `restoringFromMem` + `freezeCapture`) |

## `restoringFromMem`

**Field:** `Processor.restoringFromMem` (`processor.go:82`)

**Purpose:** Narrower gate that specifically suppresses the capture system. When `true`, no new capture timers can be armed and no capture cycle can execute. This prevents capture from overwriting the in-memory position data that the restore is actively writing back to windows.

**Set to `true`:**
| Location | Trigger |
|---|---|
| `OnSessionUnlock()` | User unlocks workstation — windows need repositioning |
| `NoteDisplayChange()` | Monitor connect/disconnect or resolution change |
| `OnPowerResume()` | System wakes from sleep (if not locked) |
| `RestoreSnapshot()` | User restores a snapshot from the running session |
| `RestoreSnapshotCmd()` | Snapshot restore via CLI one-shot command |
| `onRestoreFinishedTimer()` | Re-armed for a second restore pass when display changes mid-restore + `HaltRestore` is active |

**Set to `false`:**
| Location | Trigger |
|---|---|
| `onRestoreFinishedTimer()` | End of the restore-finished timer — after `RetryDeferredCommands()` completes |

**Read by:**
| Location | Effect |
|---|---|
| `startCaptureTimer()` | Refuses to arm new capture timers when `true` |
| `onCaptureTimer()` | Bails out of capture cycle when `true` (defense-in-depth alongside `startCaptureTimer` guard) |

## `freezeCapture`

**Field:** `Processor.freezeCapture` (`processor.go:84`)

**Purpose:** User-driven capture pause. Set by external action (e.g., pausing auto-capture from the tray menu). Unlike the other two flags, this is not tied to the restore state machine.

**Read by:** `onCaptureTimer()` — combined check: `!p.sessionActive || p.restoringFromMem || p.freezeCapture`

None of the three gates block `PersistToDB()` — persistence runs on its own cadence independent of event/capture gating.

## State machine

```
                   Start()
                      │
                      ▼
              ┌───────────────┐
              │  NORMAL       │
              │  sessionActive = true
              │  restoringFromMem = false
              └───────┬───────┘
                      │ restore trigger (unlock, display change, power resume, snapshot)
                      ▼
              ┌───────────────┐
              │  RESTORING    │
              │  sessionActive = false  (set in BatchRestore…)
              │  restoringFromMem = true (set before restore begins)
              └───────┬───────┘
                      │ onRestoreFinishedTimer()
                      ▼
              ┌───────────────┐
              │  NORMAL       │  (back to active)
              └───────────────┘

Lock / Suspend branches:
  Normal ──► sessionActive = false ──► unlock/resume sets restoringFromMem = true
                                          │
                                          ▼
                                      RESTORING ──► Normal
```

`restoringFromMem` is set before `sessionActive` is cleared, and cleared after `sessionActive` is restored. They cover slightly different windows: `restoringFromMem = true` outlives `sessionActive = false` in the display-change-during-restore path, where a second restore pass re-arms `restoringFromMem` after `sessionActive` is already back to `true`.
