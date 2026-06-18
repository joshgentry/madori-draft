package engine

import (
	"strconv"
	"time"

	"durablewindows/internal/logger"
	"durablewindows/internal/models"
	"durablewindows/internal/winapi"
)

// TakeSnapshot records the current window positions as a named snapshot.
// The snapshot ID is 0-9 (number keys), 10-35 (letter keys a-z), 36 (`) or 37 (undo).
func (p *Processor) TakeSnapshot(id int) bool {
	if id < 0 || id >= MaxSnapshots {
		return false
	}

	displayKey := p.curDisplayKey

	// Ensure the display key exists
	if _, ok := p.monitorApplications[displayKey]; !ok {
		return false
	}

	logger.Snapshot("snapshot capture", "snapshot %d...", id)

	// Show notification before capture
	if p.callbacks.ShowSnapshotCaptureTip != nil {
		p.callbacks.ShowSnapshotCaptureTip(id)
	}

	// Set snapshot flag on the last capture for each window
	for hwnd, metricsList := range p.monitorApplications[displayKey] {
		if len(metricsList) == 0 {
			continue
		}
		if hwnd == p.foreGroundWindow {
			// Take a fresh capture first
			p.CaptureWindow(hwnd, 0, time.Now(), displayKey)
		}
		// Set snapshot bit
		last := metricsList[len(metricsList)-1]
		last.SetSnapshotID(id)

		// Also save snapshot to undo slot
		last.SetSnapshotID(MaxSnapshots + 1)
	}

	// Save undo slot — set previous capture snapshot bit on the second-to-last entry
	for _, metricsList := range p.monitorApplications[displayKey] {
		if len(metricsList) >= 2 {
			prev := metricsList[len(metricsList)-2]
			prev.SetSnapshotID(MaxSnapshots + 1)
		}
	}

	// Record snapshot time
	if p.snapshotTakenTime[displayKey] == nil {
		p.snapshotTakenTime[displayKey] = make(map[int]time.Time)
	}
	p.snapshotTakenTime[displayKey][id] = time.Now()

	// Persist snapshot data to disk so it survives crashes
	p.PersistToDB()

	logger.Snapshot("snapshot captured", "%d", id)

	// Enable restore snapshot menu
	if p.callbacks.EnableRestoreSnapshotMenu != nil {
		p.callbacks.EnableRestoreSnapshotMenu(true)
	}

	return true
}

// RestoreSnapshot restores windows to their positions at the time of the snapshot.
func (p *Processor) RestoreSnapshot(id int) {
	// Save current position as undo
	p.TakeSnapshot(MaxSnapshots + 1)

	p.restoringSnapshot = true
	p.snapshotId = id
	p.restoringFromMem = true

	p.restoreTimes = 0
	p.restoredWindows = make(map[uintptr]bool)
	logger.Snapshot("snapshot restore", "snapshot %d", id)
	p.BatchRestoreApplicationsOnCurrentDisplays()
}

// snapshotKey returns the BoltDB bucket key for a one-shot snapshot.
// These are separate from the live_<displayKey> keys used by tray-mode
// auto-capture, so one-shot commands never mix with auto-capture data.
func snapshotKey(id int) string {
	return "snap_" + strconv.Itoa(id)
}

// RestoreSnapshotCmd is the CLI one-shot: loads only snapshot data from
// the snap_<id> key (not live auto-capture data), restores windows, and exits.
func (p *Processor) RestoreSnapshotCmd(id int) {
	metrics, err := store.LoadWindowMetrics(snapshotKey(id))
	if err != nil || len(metrics) == 0 {
		logger.Error("", "Snapshot %d not found", id)
		return
	}

	p.curDisplayKey = winapi.GetDisplayKey()
	p.CaptureNewDisplayConfig(p.curDisplayKey)
	p.monitorApplications[p.curDisplayKey] = metrics

	p.restoringFromMem = true
	p.restoreTimes = 0
	p.restoredWindows = make(map[uintptr]bool)
	logger.Snapshot("snapshot restore", "snapshot %d (%d windows)", id, len(metrics))
	p.BatchRestoreApplicationsOnCurrentDisplays()
}

// CaptureSnapshotCmd is the CLI one-shot: enumerates current windows,
// saves them under the snap_<id> key (not live auto-capture data), and exits.
func (p *Processor) CaptureSnapshotCmd(id int) {
	p.curDisplayKey = winapi.GetDisplayKey()
	p.CaptureNewDisplayConfig(p.curDisplayKey)
	p.CaptureWindowsOfInterest(p.curDisplayKey)

	store.SaveWindowMetrics(snapshotKey(id), p.monitorApplications[p.curDisplayKey])
	logger.Snapshot("snapshot captured", "snapshot %d (%d windows)", id,
		len(p.monitorApplications[p.curDisplayKey]))
}

// findMetricsForSnapshot finds the window metrics matching a snapshot ID.
func (p *Processor) findMetricsForSnapshot(hwnd uintptr, snapshotID int, displayKey string) *models.WindowMetrics {
	apps, ok := p.monitorApplications[displayKey]
	if !ok {
		return nil
	}
	metricsList, ok := apps[hwnd]
	if !ok {
		return nil
	}

	// Search from most recent to oldest for matching snapshot
	for i := len(metricsList) - 1; i >= 0; i-- {
		if metricsList[i].HasSnapshotID(snapshotID) {
			return metricsList[i]
		}
	}

	// If not found, try undo slot
	if snapshotID != MaxSnapshots+1 {
		for i := len(metricsList) - 1; i >= 0; i-- {
			if metricsList[i].HasSnapshotID(MaxSnapshots + 1) {
				return metricsList[i]
			}
		}
	}

	return nil
}
