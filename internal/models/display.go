package models

import "durablewindows/internal/winapi"

// DesktopDisplayMetrics wraps the display enumeration logic.
// In the Go port this is simpler since we just call winapi.GetDisplayKey().
type DesktopDisplayMetrics struct {
	displays []winapi.DisplayInfo
}

// AcquireMetrics enumerates and sorts all monitors.
func (ddm *DesktopDisplayMetrics) AcquireMetrics() {
	ddm.displays = winapi.GetDisplays()
}

// GetDisplays returns the current sorted list of displays.
func (ddm *DesktopDisplayMetrics) GetDisplays() []winapi.DisplayInfo {
	ddm.AcquireMetrics()
	return ddm.displays
}

// Key returns the display configuration key string.
func (ddm *DesktopDisplayMetrics) Key() string {
	return winapi.GetDisplayKey()
}
