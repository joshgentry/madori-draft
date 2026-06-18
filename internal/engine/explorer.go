package engine

import (
	"strings"

	"golang.org/x/sys/windows"

	"durablewindows/internal/winapi"
)

// GetExplorerFolderPath reads the address bar of an Explorer window to get
// the folder path it's viewing. This is stored in WindowMetrics.Dir so that
// auto-restore can re-open Explorer to the correct location.
func GetExplorerFolderPath(hwnd uintptr) string {
	// Walk the child window hierarchy to find the breadcrumb bar
	toolbar := winapi.FindWindowExW(hwnd, 0, windows.StringToUTF16Ptr("WorkerW"), nil)
	if toolbar == 0 {
		// Try direct approach (older Explorer layout)
		toolbar = winapi.FindWindowExW(hwnd, 0, windows.StringToUTF16Ptr("ReBarWindow32"), nil)
	} else {
		toolbar = winapi.FindWindowExW(toolbar, 0, windows.StringToUTF16Ptr("ReBarWindow32"), nil)
	}
	if toolbar == 0 {
		return ""
	}

	toolbar = winapi.FindWindowExW(toolbar, 0, windows.StringToUTF16Ptr("Address Band Root"), nil)
	if toolbar == 0 {
		return ""
	}
	toolbar = winapi.FindWindowExW(toolbar, 0, windows.StringToUTF16Ptr("msctls_progress32"), nil)
	if toolbar == 0 {
		return ""
	}
	toolbar = winapi.FindWindowExW(toolbar, 0, windows.StringToUTF16Ptr("Breadcrumb Parent"), nil)
	if toolbar == 0 {
		return ""
	}
	toolbar = winapi.FindWindowExW(toolbar, 0, windows.StringToUTF16Ptr("ToolbarWindow32"), nil)
	if toolbar == 0 {
		return ""
	}

	// Read the toolbar text (format: "Address: C:\Users\...")
	title := GetWindowTitle(toolbar)
	if strings.HasPrefix(title, "Address: ") {
		return title[9:]
	}
	return title
}
