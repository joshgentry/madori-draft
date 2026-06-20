package models

import (
	"time"

	"durablewindows/internal/winapi"
)

// WindowMetrics mirrors the C# ApplicationDisplayMetrics.
// It stores a snapshot of one window's position, state, and metadata at a point in time.
type WindowMetrics struct {
	// LiteDB compatibility fields
	ID             int    `json:"id"`
	GUID           string `json:"guid"`
	ProcessID      uint32 `json:"process_id"`
	ProcessExePath string `json:"process_exe_path"`

	// General window info
	HWnd         uintptr `json:"-"`
	WindowID     uint32  `json:"window_id"`
	ClassName    string  `json:"class_name"`
	ProcessName  string  `json:"process_name"`
	Title        string  `json:"title"`
	Dir          string  `json:"dir"`
	IsFullScreen bool    `json:"is_full_screen"`
	IsMinimized  bool    `json:"is_minimized"`
	IsInvisible  bool    `json:"is_invisible"`
	IsResizable  bool    `json:"is_resizable"`
	Style        int64   `json:"style"`
	ExtStyle     int64   `json:"ext_style"`

	// Capture timestamp
	CaptureTime time.Time `json:"capture_time"`

	// Window position
	ScreenPosition            winapi.RECT            `json:"screen_position"`
	WindowPlacement           winapi.WINDOWPLACEMENT `json:"window_placement"`
	NeedUpdateWindowPlacement bool                   `json:"-"` // transient

	// Window z-order
	IsTopMost         bool    `json:"is_top_most"`
	NeedClearTopMost  bool    `json:"need_clear_top_most"`
	PrevZorderWindow  uintptr `json:"prev_zorder_window"`
	NeedRestoreZorder bool    `json:"need_restore_zorder"`

	// Snapshot bitfield (up to 64 snapshot IDs)
	SnapShotFlags uint64 `json:"snapshot_flags"`

	// Validity flag
	IsValid bool `json:"is_valid"`
}

// EqualPlacement returns true if two metrics have the same position, size, and show state.
func (m *WindowMetrics) EqualPlacement(other *WindowMetrics) bool {
	posEqual := m.WindowPlacement.NormalPosition.Equals(other.WindowPlacement.NormalPosition)
	screenEqual := m.ScreenPosition.Equals(other.ScreenPosition)
	minmaxStateEqual := m.WindowPlacement.ShowCmd == other.WindowPlacement.ShowCmd
	isMinimizedEqual := m.IsMinimized == other.IsMinimized
	return posEqual && screenEqual && minmaxStateEqual && isMinimizedEqual
}

// HasSnapshot returns true if any snapshot bit is set.
func (m *WindowMetrics) HasSnapshot() bool {
	return m.SnapShotFlags != 0
}

// HasSnapshotID returns true if the specified snapshot bit is set.
func (m *WindowMetrics) HasSnapshotID(id int) bool {
	return (m.SnapShotFlags & (1 << uint(id))) != 0
}

// SetSnapshotID sets the snapshot bit for the given ID.
func (m *WindowMetrics) SetSnapshotID(id int) {
	m.SnapShotFlags |= (1 << uint(id))
}

