package engine

import (
	"strconv"
	"time"

	"madori/internal/logger"
	"madori/internal/models"
	"madori/internal/winapi"
)

// TakeSnapshot records the current window positions as a named snapshot.
// Valid snapshot IDs are 0-9 (number keys), 10-35 (a-z), and 36 (`).
// ID 37 is reserved for the internal undo slot.
func (p *Processor) TakeSnapshot(id int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if id < 0 || id > MaxUserSnapshotID {
		return false
	}

	displayKey := p.curDisplayKey

	// Ensure the display key exists
	if _, ok := p.monitorApplications[displayKey]; !ok {
		return false
	}

	logger.Snapshot(logger.LevelInfo, "snapshot capture", "snapshot %d...", id)

	// Show notification before capture
	if p.callbacks.ShowSnapshotCaptureTip != nil {
		p.callbacks.ShowSnapshotCaptureTip(id)
	}

	// Set snapshot bits for each window's latest capture
	for hwnd, m := range p.monitorApplications[displayKey] {
		if m == nil {
			continue
		}
		if hwnd == p.foreGroundWindow {
			// Take a fresh capture first
			p.CaptureWindow(hwnd, 0, time.Now(), displayKey)
		}
	}

	// Walk the full z-order chain and assign ranks so we can rebuild
	// window stacking during restore.
	p.CaptureStackingAll(displayKey)

	// Bulk-copy current state into the snapshot sub-bucket.
	copies := make(map[uintptr]*models.WindowMetrics, len(p.monitorApplications[displayKey]))
	for hwnd, m := range p.monitorApplications[displayKey] {
		if m == nil {
			continue
		}
		c := *m // value copy
		copies[hwnd] = &c
	}
	p.store.SaveWindowMetrics("dk_"+displayKey, snapshotKey(id), copies)

	// Also update the undo slot so "undo" goes back to this snapshot.
	p.store.SaveWindowMetrics("dk_"+displayKey, snapKeyUndo, copies)

	// Record snapshot time
	if p.snapshotTakenTime[displayKey] == nil {
		p.snapshotTakenTime[displayKey] = make(map[int]time.Time)
	}
	p.snapshotTakenTime[displayKey][id] = time.Now()

	// Persist snapshot data to disk so it survives crashes
	p.PersistToDB()

	logger.Snapshot(logger.LevelInfo, "snapshot captured", "%d", id)

	// Enable restore snapshot menu
	if p.callbacks.EnableRestoreSnapshotMenu != nil {
		p.callbacks.EnableRestoreSnapshotMenu(true)
	}

	return true
}

// RestoreSnapshot restores windows to their positions at the time of the snapshot.
func (p *Processor) RestoreSnapshot(id int) {
	// Save current auto-capture state as undo snapshot before replacing it.
	// We do this under our own lock, not via TakeSnapshot (which validates IDs).
	p.mu.Lock()
	displayKey := p.curDisplayKey
	copies := make(map[uintptr]*models.WindowMetrics, len(p.monitorApplications[displayKey]))
	for hwnd, m := range p.monitorApplications[displayKey] {
		if m == nil {
			continue
		}
		c := *m
		copies[hwnd] = &c
	}
	if p.store != nil {
		p.store.SaveWindowMetrics("dk_"+displayKey, snapKeyUndo, copies)
	}

	// Load the requested snapshot into memory, backing up current auto-capture.
	snapshotEntries, err := p.store.LoadWindowMetrics("dk_"+displayKey, snapshotKey(id))
	if err != nil || len(snapshotEntries) == 0 {
		p.mu.Unlock()
		logger.Error(logger.LevelError, "", "Snapshot %d not found", id)
		return
	}
	p.snapshotAutoBackup = p.monitorApplications[displayKey]
	p.monitorApplications[displayKey] = snapshotEntries

	p.restoringSnapshot = true
	p.snapshotId = id
	p.restoringFromMem = true
	p.restoreTimes = 0
	p.restoredWindows = make(map[uintptr]bool)
	logger.Snapshot(logger.LevelInfo, "snapshot restore", "snapshot %d (%d windows)", id, len(snapshotEntries))
	p.mu.Unlock()

	p.BatchRestoreApplicationsOnCurrentDisplays()
}

// Snapshot sub-bucket key constants.
const (
	snapKeyAuto = "snap_auto"
	snapKeyUndo = "snap_undo"
)

// snapshotKey returns the snapshot sub-bucket key for a numeric snapshot ID.
func snapshotKey(id int) string {
	return "snap_" + strconv.Itoa(id)
}

// ParseSnapshotID converts a key character to its snapshot ID.
// "0"-"9" → 0-9, "a"-"z" or "A"-"Z" → 10-35, "`" or "~" → 36.
// Returns -1 for unrecognized input.
func ParseSnapshotID(s string) int {
	if len(s) != 1 {
		return -1
	}
	c := s[0]
	if c == '`' || c == '~' {
		return 36
	}
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'z' {
		return int(c - 'a' + 10)
	}
	if c >= 'A' && c <= 'Z' {
		return int(c - 'A' + 10)
	}
	return -1
}

// SnapshotName returns the display character for a snapshot ID.
func SnapshotName(id int) string {
	switch {
	case id >= 0 && id <= 9:
		return string(rune('0' + id))
	case id >= 10 && id <= 35:
		return string(rune('a' + id - 10))
	case id == 36:
		return "`"
	default:
		return "?"
	}
}

// RestoreSnapshotCmd is the CLI one-shot: loads the snapshot from
// dk_<dk>/snap_<id>, restores windows, and exits.
func (p *Processor) RestoreSnapshotCmd(id int) {
	if id < 0 || id > MaxUserSnapshotID {
		logger.Error(logger.LevelError, "", "Snapshot %d: valid IDs are 0-%d", id, MaxUserSnapshotID)
		return
	}

	displayKey := winapi.GetDisplayKey()
	metrics, err := p.store.LoadWindowMetrics("dk_"+displayKey, snapshotKey(id))
	if err != nil || len(metrics) == 0 {
		logger.Error(logger.LevelError, "", "Snapshot %d not found for display config %s", id, displayKey)
		return
	}

	p.curDisplayKey = displayKey
	p.CaptureNewDisplayConfig(p.curDisplayKey)
	p.monitorApplications[p.curDisplayKey] = metrics

	p.restoringFromMem = true
	p.restoreTimes = 0
	p.restoredWindows = make(map[uintptr]bool)
	logger.Snapshot(logger.LevelInfo, "snapshot restore", "snapshot %d (%d windows)", id, len(metrics))
	p.BatchRestoreApplicationsOnCurrentDisplays()
}

// CaptureSnapshotCmd is the CLI one-shot: enumerates current windows,
// saves them under dk_<dk>/snap_<id>, and exits.
func (p *Processor) CaptureSnapshotCmd(id int) {
	if id < 0 || id > MaxUserSnapshotID {
		logger.Error(logger.LevelError, "", "Snapshot %d: valid IDs are 0-%d", id, MaxUserSnapshotID)
		return
	}

	displayKey := winapi.GetDisplayKey()
	p.curDisplayKey = displayKey
	p.CaptureNewDisplayConfig(p.curDisplayKey)
	p.CaptureWindowsOfInterest(p.curDisplayKey)

	p.store.SaveWindowMetrics("dk_"+displayKey, snapshotKey(id), p.monitorApplications[p.curDisplayKey])
	logger.Snapshot(logger.LevelInfo, "snapshot captured", "snapshot %d (%d windows)", id,
		len(p.monitorApplications[p.curDisplayKey]))
}
