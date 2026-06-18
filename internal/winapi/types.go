package winapi

import (
	"math"
	"unsafe"
)

// POINT represents a Windows POINT structure.
type POINT struct {
	X int32
	Y int32
}

func (p POINT) Equals(other POINT) bool {
	return p.X == other.X && p.Y == other.Y
}

// RECT represents a Windows RECT structure.
type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func (r RECT) Width() int32  { return r.Right - r.Left }
func (r RECT) Height() int32 { return r.Bottom - r.Top }

func (r RECT) Equals(other RECT) bool {
	return r.Left == other.Left && r.Top == other.Top &&
		r.Right == other.Right && r.Bottom == other.Bottom
}

// Diff returns the average absolute difference between two RECTs.
func (r RECT) Diff(other RECT) int32 {
	diff := int32(math.Abs(float64(r.Left-other.Left)) +
		math.Abs(float64(r.Right-other.Right)) +
		math.Abs(float64(r.Top-other.Top)) +
		math.Abs(float64(r.Bottom-other.Bottom)))
	return diff / 4
}

// WINDOWPLACEMENT represents the Windows WINDOWPLACEMENT structure.
type WINDOWPLACEMENT struct {
	Length         uint32
	Flags          uint32
	ShowCmd        uint32
	MinPosition    POINT
	MaxPosition    POINT
	NormalPosition RECT
	RectDevice     RECT
}

// DefaultWINDOWPLACEMENT returns a WINDOWPLACEMENT with Length set correctly.
func DefaultWINDOWPLACEMENT() WINDOWPLACEMENT {
	return WINDOWPLACEMENT{
		Length: uint32(unsafe.Sizeof(WINDOWPLACEMENT{})),
	}
}

// APP_BAR_DATA represents the Windows APPBARDATA structure.
type APP_BAR_DATA struct {
	CbSize           uint32
	HWnd             uintptr
	UCallbackMessage int32
	UEdge            int32
	RC               RECT
	LParam           uintptr
}

// CURSORINFO represents the Windows CURSORINFO structure.
type CURSORINFO struct {
	CbSize      uint32
	Flags       uint32
	HCursor     uintptr
	PTScreenPos POINT
}

// Display represents a monitor/display device.
type Display struct {
	DeviceName string
	Position   RECT
	Flags      uint32
}

// MOUSEHOOKSTRUCT represents low-level mouse hook info.
type MOUSEHOOKSTRUCT struct {
	PT           POINT
	HWnd         uintptr
	WHitTestCode uint32
	DWExtraInfo  uintptr
}

// TITLEBARINFO contains title bar information.
// CCHILDREN_TITLEBAR = 5, so arrays have 6 elements.
type TITLEBARINFO struct {
	CbSize     uint32
	RcTitleBar RECT
	Rgstate    [6]uint32
}

// TITLEBARINFOEX extends TITLEBARINFO with per-button rectangles.
type TITLEBARINFOEX struct {
	CbSize     uint32
	RcTitleBar RECT
	Rgstate    [6]uint32
	Rgrect     [6]RECT
}

// Title bar button indices into rgstate / rgrect.
const (
	TitleBarTitle    = 0
	TitleBarReserved = 1
	TitleBarMin      = 2
	TitleBarMax      = 3
	TitleBarHelp     = 4
	TitleBarClose    = 5
)

// MSLLHOOKSTRUCT represents low-level mouse hook info (extended).
type MSLLHOOKSTRUCT struct {
	PT          POINT
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DWExtraInfo uintptr
}

// CWPRETSTRUCT represents the structure passed to WH_CALLWNDPROCRET hook procs.
type CWPRETSTRUCT struct {
	LResult uintptr
	LParam  uintptr
	WParam  uintptr
	Message uint32
	HWnd    uintptr
}
