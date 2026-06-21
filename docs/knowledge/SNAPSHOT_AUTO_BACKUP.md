# Snapshot Auto Backup

`snapshotAutoBackup` on `Processor` is a temporary pointer to the real
auto-capture state, held during snapshot restore so the in-memory map can
be borrowed for the duration of the restore passes.

## Why it exists

`RestoreApplicationsOnCurrentDisplays` reads its source data directly from
`p.monitorApplications[displayKey]`. When restoring a snapshot, we need
that source to be the snapshot entries (so windows get moved to their
snapshot positions). But after the restore finishes, the auto-capture
state must be restored — future capture cycles need the real window data,
not frozen snapshot data.

## Flow

```
RestoreSnapshot(3):
  1. Save current auto-capture state as undo snapshot in DB
  2. Load snap_3 entries from DB → snapshotEntries
  3. snapshotAutoBackup = monitorApplications[dk]  // save real data
  4. monitorApplications[dk] = snapshotEntries     // swap in snapshot data
  5. Run multi-pass restore — windows move to snapshot positions

onRestoreFinishedTimer():
  6. monitorApplications[dk] = snapshotAutoBackup  // swap back real data
  7. snapshotAutoBackup = nil
```

Without the swap-back, the frozen snapshot entries would persist in
`monitorApplications` after the restore, and the next
`CaptureWindowsOfInterest` / `PersistToDB` would operate on stale data
instead of current window state.
