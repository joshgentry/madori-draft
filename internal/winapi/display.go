package winapi

import (
	"fmt"
	"sort"
	"strings"
)

// Display holds information about a single monitor.
type DisplayInfo struct {
	Position RECT
	Flags    uint32
}

// GetDisplays enumerates all monitors and returns them sorted by position.
func GetDisplays() []DisplayInfo {
	var displays []DisplayInfo

	callback := func(hMonitor uintptr, hdcMonitor uintptr, lprcMonitor *RECT, dwData uintptr) uintptr {
		var mi MONITORINFO
		if GetMonitorInfo(hMonitor, &mi) {
			displays = append(displays, DisplayInfo{
				Position: mi.RCMonitor,
				Flags:    mi.DwFlags,
			})
		}
		return 1
	}

	EnumDisplayMonitors(0, nil, callback, 0)

	// Sort by position: Left asc, then Top asc, then Width asc, then Height asc
	sort.Slice(displays, func(i, j int) bool {
		if displays[i].Position.Left != displays[j].Position.Left {
			return displays[i].Position.Left < displays[j].Position.Left
		}
		if displays[i].Position.Top != displays[j].Position.Top {
			return displays[i].Position.Top < displays[j].Position.Top
		}
		if displays[i].Position.Width() != displays[j].Position.Width() {
			return displays[i].Position.Width() < displays[j].Position.Width()
		}
		return displays[i].Position.Height() < displays[j].Position.Height()
	})

	return displays
}

// GetDisplayKey computes a unique string key for the current display configuration.
// Format: "Display0_LocXxY_ResWxH__Display1_LocXxY_ResWxH"
// The key replaces '-' with 'M' for compatibility with storage backends that
// don't accept minus characters in keys.
func GetDisplayKey() string {
	displays := GetDisplays()
	segments := make([]string, 0, len(displays))

	for i, d := range displays {
		segments = append(segments, fmt.Sprintf(
			"Display%d_Loc%dx%d_Res%dx%d",
			i, d.Position.Left, d.Position.Top,
			d.Position.Width(), d.Position.Height(),
		))
	}

	key := strings.Join(segments, "__")
	key = strings.ReplaceAll(key, "-", "M")
	return key
}
