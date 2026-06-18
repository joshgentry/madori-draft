package durablewindows

import (
	_ "embed"
)

// Embedded icon files. These are loaded by internal/tray.
var (
	//go:embed resources/pwIcon.ico
	IdleIcoData []byte

	//go:embed resources/pwIconBusy.png
	BusyPngData []byte

	//go:embed resources/pwIconUpdate.png
	UpdatePngData []byte
)
