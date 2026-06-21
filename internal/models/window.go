package models

import (
	"time"

	"madori/internal/winapi"
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

	// Window stacking (z-order)
	IsTopMost           bool `json:"is_top_most"`
	NeedClearTopMost    bool `json:"need_clear_top_most"`
	NeedRestoreStacking bool `json:"need_restore_stacking"` // true when StackingRank is valid
	StackingRank        int  `json:"stacking_rank"` // 0=topmost, -1=unset

	// DWM corner preference (Windows 11+)
	// 0=DWMWCP_DEFAULT, 1=DWMWCP_DONOTROUND, 2=DWMWCP_ROUND, 3=DWMWCP_ROUNDSMALL
	WindowCornerPreference uint32 `json:"window_corner_preference"`

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
