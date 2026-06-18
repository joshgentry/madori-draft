package tray

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"durablewindows/internal/winapi"
)

const (
	dlgEditID   = 1001
	dlgOKID     = 1002
	dlgCancelID = 1003
)

var (
	dlgResult    byte
	dlgClassName *uint16
	dlgHwnd      uintptr
	dlgHInstance uintptr
)

// EnterSnapshotName shows a small input dialog asking for a snapshot
// character (0-9, a-z, `). Returns 0 if cancelled.
func EnterSnapshotName() byte {
	dlgResult = 0
	dlgHInstance = winapi.GetModuleHandle(nil)

	// Register a dialog-like window class
	if dlgClassName == nil {
		dlgClassName = windows.StringToUTF16Ptr("DWSnapshotDlg")
		var wc winapi.WNDCLASSEX
		wc.Style = 0 // CS_HREDRAW | CS_VREDRAW
		wc.LpfnWndProc = syscall.NewCallback(snapshotDlgProc)
		wc.HInstance = dlgHInstance
		wc.HCursor = winapi.LoadCursor(0, winapi.IDC_ARROW)
		wc.HbrBackground = uintptr(winapi.COLOR_BTNFACE + 1)
		wc.LpszClassName = dlgClassName
		winapi.RegisterClassEx(&wc)
	}

	// Position the dialog near the center of the screen
	screenW := winapi.GetSystemMetrics(winapi.SM_CXSCREEN)
	screenH := winapi.GetSystemMetrics(winapi.SM_CYSCREEN)
	dlgW := int32(320)
	dlgH := int32(130)
	x := (screenW - dlgW) / 2
	y := (screenH - dlgH) / 2

	dlgHwnd = winapi.CreateWindowEx(
		winapi.WS_EX_DLGMODALFRAME,
		dlgClassName,
		windows.StringToUTF16Ptr("Snapshot Name"),
		winapi.WS_OVERLAPPED|winapi.WS_CAPTION|winapi.WS_SYSMENU|winapi.WS_VISIBLE,
		x, y, dlgW, dlgH,
		0, 0, dlgHInstance,
		nil,
	)
	if dlgHwnd == 0 {
		return 0
	}

	// Bring to foreground
	winapi.SetForegroundWindow(dlgHwnd)

	// Local message loop for the dialog
	var msg winapi.MSG
	for winapi.GetMessage(&msg, 0, 0, 0) > 0 {
		winapi.TranslateMessage(&msg)
		winapi.DispatchMessage(&msg)
		if dlgHwnd == 0 {
			break
		}
	}

	return dlgResult
}

func snapshotDlgProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case winapi.WM_CREATE:
		// Create label
		winapi.CreateWindowEx(0,
			windows.StringToUTF16Ptr("STATIC"),
			windows.StringToUTF16Ptr("Enter snapshot key (0-9, a-z, `):"),
			winapi.WS_CHILD|winapi.WS_VISIBLE,
			10, 10, 300, 22,
			hwnd, 0, dlgHInstance, nil)

		// Create edit control
		edit := winapi.CreateWindowEx(
			winapi.WS_EX_CLIENTEDGE,
			windows.StringToUTF16Ptr("EDIT"),
			nil,
			winapi.WS_CHILD|winapi.WS_VISIBLE|winapi.WS_TABSTOP|winapi.WS_BORDER,
			10, 35, 50, 22,
			hwnd, uintptr(dlgEditID), dlgHInstance, nil)
		// Limit to 1 character
		winapi.SendMessage(edit, 0xC5, 1, 0) // EM_SETLIMITTEXT = 0xC5

		// Focus the edit
		winapi.SetFocus(edit)

		// Create OK button
		winapi.CreateWindowEx(0,
			windows.StringToUTF16Ptr("BUTTON"),
			windows.StringToUTF16Ptr("OK"),
			winapi.WS_CHILD|winapi.WS_VISIBLE|winapi.WS_TABSTOP|winapi.BS_DEFPUSHBUTTON,
			155, 35, 50, 24,
			hwnd, uintptr(dlgOKID), dlgHInstance, nil)

		// Create Cancel button
		winapi.CreateWindowEx(0,
			windows.StringToUTF16Ptr("BUTTON"),
			windows.StringToUTF16Ptr("Cancel"),
			winapi.WS_CHILD|winapi.WS_VISIBLE|winapi.WS_TABSTOP,
			210, 35, 75, 24,
			hwnd, uintptr(dlgCancelID), dlgHInstance, nil)
		return 0

	case winapi.WM_COMMAND:
		cmd := uint32(wParam & 0xFFFF)
		switch cmd {
		case dlgOKID:
			// Read the edit control text
			var buf [2]uint16
			winapi.SendMessage(winapi.GetDlgItem(hwnd, dlgEditID), 0x0D, 2, uintptr(unsafe.Pointer(&buf[0]))) // WM_GETTEXT
			if buf[0] != 0 {
				c := byte(buf[0])
				if isValidSnapshotChar(c) {
					dlgResult = c
					winapi.DestroyWindow(hwnd)
					dlgHwnd = 0
				}
			}
			return 0
		case dlgCancelID:
			winapi.DestroyWindow(hwnd)
			dlgHwnd = 0
			return 0
		}

	case winapi.WM_CLOSE:
		winapi.DestroyWindow(hwnd)
		dlgHwnd = 0
		return 0
	}

	return winapi.DefWindowProc(hwnd, msg, wParam, lParam)
}

func isValidSnapshotChar(c byte) bool {
	if c >= '0' && c <= '9' {
		return true
	}
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c == '`' || c == '~' {
		return true
	}
	return false
}
