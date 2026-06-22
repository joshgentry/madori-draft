# SetWindowPos vs DeferWindowPos for Z-Order Restore

## The Problem

During snapshot restore, `RestoreStacking` rebuilds the z-order by placing windows
in rank order: rank 0 at `HWND_TOP`, rank 1 after rank 0, rank 2 after rank 1,
etc.  The original implementation used the batch API:

```go
hDWP := BeginDeferWindowPos(n)
for _, w := range windows {
    hDWP = DeferWindowPos(hDWP, w.hwnd, prevHWND, ...)
    prevHWND = w.hwnd
}
EndDeferWindowPos(hDWP)
```

This worked correctly when all windows were on the current virtual desktop.
However, the global z-order chain (`GetWindow(desktop, GW_CHILD) →
GW_HWNDNEXT`) crosses virtual desktop boundaries — windows on other desktops
(which are DWM-cloaked with `DWM_CLOAKED_SHELL`) appear in the chain after the
current desktop's windows.

When the sorted rank list interleaves current-VD and other-VD windows (because
their ranks are global, assigned in chain-walk order), the `DeferWindowPos` chain
contains `hWndInsertAfter` targets that point to cloaked windows.  `EndDeferWindowPos`
applies all changes simultaneously.  Windows silently ignores z-order changes for
cloaked windows, and — critically — subsequent `hWndInsertAfter` references that
point to those silently-dropped windows become dangling.  Current-VD windows
whose `insertAfter` targets a cloaked window land in the wrong position,
corrupting the current desktop's z-order.

## Why `DeferWindowPos` Fails

The batch API defers all changes and applies them atomically via `EndDeferWindowPos`.
When a cloaked window is in the batch, its z-order change is silently ignored,
but the window handles that depend on it as an `hWndInsertAfter` target still
reference it.  The resolution of those targets happens at `EndDeferWindowPos`
time, where the cloaked window was never actually moved — producing incorrect
placements for every window that references a cloaked predecessor.

Logs confirmed this: with `DeferWindowPos`, 29 of 57 windows in the chain were
on other VDs, and those cloaked windows corrupted the positions of the current-VD
windows that followed them.

## The Fix: Individual `SetWindowPos` Calls

Replaced the batch API with individual `SetWindowPos` calls applied in sorted
rank order:

```go
var prevHWND uintptr
for _, w := range windows {
    SetWindowPos(w.hwnd, prevHWND, 0, 0, 0, 0,
        SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE)
    prevHWND = w.hwnd
}
```

Each call applies immediately.  When `SetWindowPos(B, A, ...)` executes, window A
has **already** been positioned by the previous call.  The `hWndInsertAfter`
target is always a window that was just placed — never a dangling reference to a
cloaked window whose change was silently dropped.

## Trade-off

`SetWindowPos` per-window generates `WM_WINDOWPOSCHANGING`/`WM_WINDOWPOSCHANGED`
messages for each call (N message rounds vs 1).  With `SWP_NOMOVE | SWP_NOSIZE |
SWP_NOACTIVATE` there is no repaint, layout, or activation, so each call is
cheap.  For a typical restore of 30–60 windows this adds perhaps 10–30ms, and
the restore path is not a hot path (fires on display changes and manual snapshot
restores).  The correctness win outweighs the minor overhead.
