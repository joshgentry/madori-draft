# shouldTrackWindow and the Capture Timer System

## Architecture

DurableWindows uses a two-tier filtering system for deciding which windows
to capture and restore.

### Tier 1: Event Handler Pre-checks (lightweight)

When a WinEvent arrives, the event handler decides whether to arm the capture
timer. These handlers use fast, cheap checks:

| Handler | Checks |
|---|---|
| `onWindowCreate` | `sessionActive`, `IsWindowVisible`, `IsTopLevelWindow`, `isFullScreen` |
| `onWindowShow` | same as `onWindowCreate` |
| `onForegroundTimer` | `hwnd != 0`, `sessionActive`, `IsWindow`, `IsWindowVisible`, `IsTopLevelWindow` |

These checks are intentionally minimal — a few `USER32` calls each — because
event handlers fire frequently. A single app launch can produce dozens of
`OBJECT_CREATE` events for child windows, tooltips, and internal frames.

If the pre-checks pass, the handler calls `startCaptureTimer(delay)`.

### Tier 2: `shouldTrackWindow` (comprehensive)

When the capture timer fires, `onCaptureTimer` calls
`BatchCaptureApplicationsOnCurrentDisplays` → `CaptureWindowsOfInterest`, which
enumerates **every top-level window** on the system via a `GetWindow(hwnd,
GW_HWNDNEXT)` Z-order walk. For each window, `CaptureWindow` calls
`shouldTrackWindow`, which runs 14 checks:

1. `IsWindow` — valid handle
2. `IsTopLevelWindow` — not a child
3. Not in `noRestoreWindows`
4. `IsWindowVisible`
5. Not DWM-cloaked (app or inherited)
6. Non-zero size, area ≥ 3000px²
7. Not `WS_EX_TOOLWINDOW` (unless `captureFloatingWindow`)
8. Not a known system class (IME, GDI+, menu, button, etc.)
9. Not the taskbar
10. Not an overlay title (NVIDIA, AMD, Discord, Steam, Xbox Game Bar, etc.)
11. Looks like a real user window (title, caption, or sizable)
12. Not a dormant UWP ghost frame
13. In `careProcess` list (if set)
14. Not in `ignoreProcess` list

Several of these are expensive: process name resolution (`getProcessName`
opens a process handle), DWM cloak queries (`DwmGetWindowAttribute`), and
string matching against overlay/taskbar/system-class lists.

### The Capture Timer

The capture timer is a **debounce mechanism**, not a targeted capture:

```
WinEvent → startCaptureTimer(delay) → [debounce] → timer fires →
  onCaptureTimer → BatchCaptureApplicationsOnCurrentDisplays →
    CaptureWindowsOfInterest → for every top-level window:
      CaptureWindow → shouldTrackWindow → [capture or skip]
```

Key properties:
- The timer is one-shot (`time.AfterFunc`), not periodic.
- `startCaptureTimer` will not reset an already-pending timer — it returns
  early if `captureTimerStarted > 0`. This prevents noisy events from
  endlessly postponing the capture cycle.
- `resetCaptureTimer` (used by `onMoveSizeEnd` and `onMinimizeEnd`)
  unconditionally stops and re-arms the timer with a shorter delay
  (`UserMoveLatency`, 1s).
- When the timer fires, **every top-level window is enumerated** — not just
  the window that triggered the event. The event only determines *when* the
  sweep happens, not *which* windows are captured.

## Discussion: Should Event Handlers Call `shouldTrackWindow`?

**Observation:** The event handlers do not call `shouldTrackWindow`. They use
their own lighter pre-checks. Since each unnecessary timer fire triggers a
full window enumeration, preventing unnecessary fires would save work.

**Trade-off against adding `shouldTrackWindow` to events:**

- `shouldTrackWindow` is too expensive for the event hot path. Opening
  process handles, querying DWM cloak state, and matching overlay title
  strings on every `OBJECT_CREATE`/`OBJECT_SHOW` would slow down event
  processing significantly.
- The current pre-checks (`IsTopLevelWindow`, `IsWindowVisible`) catch the
  biggest noise sources (child windows, hidden windows) with trivial cost.

**Alternative considered: add a subset of cheap sub-checks to event handlers.**

Several `shouldTrackWindow` sub-checks are nearly free because they only
inspect data already available:

| Check | Cost |
|---|---|
| `WS_EX_TOOLWINDOW` | One style bit test |
| `isSystemClass` | String prefix comparisons on class name |
| `isTaskBarWindow` | String comparison on class name |
| `isRealUserWindow` | Style bits + title check |

These could catch additional noise (tool windows, system utility windows)
without adding meaningful overhead. However, `IsTopLevelWindow` already
filters out child windows — the biggest noise source — and it's unclear how
many top-level tool windows or system-class windows generate WinEvents that
would reach the handlers.

## Decision

**No changes for now.** The current design works correctly: the capture system
is guarded by `isFullScreen` checks at three layers (event handler, timer
start, timer fire), and `shouldTrackWindow` filters comprehensively during
the actual capture pass. The only cost of an "unnecessary" timer fire is a
full window sweep, which is bounded by the timer-not-pending guard (at most
one sweep per debounce window).

If profiling or log analysis shows significant unnecessary capture cycles,
the cheapest next step would be adding a `WS_EX_TOOLWINDOW` check to the
event handlers — a single style bit test that would catch floating tool
palettes and utility popups before they arm the timer.
