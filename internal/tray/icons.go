package tray

import (
	"durablewindows"
	"durablewindows/internal/winapi"
)

// loadIcons loads the embedded icon data and returns HICON handles for each state.
func loadIcons() (idle, busy, update uintptr) {
	idle = loadICO(durablewindows.IdleIcoData)
	busy = loadPNGasIcon(durablewindows.BusyPngData)
	if busy == 0 {
		busy = idle
	}
	update = loadPNGasIcon(durablewindows.UpdatePngData)
	if update == 0 {
		update = idle
	}
	return
}

// loadICO loads a Windows .ico file from raw bytes and returns an HICON.
func loadICO(data []byte) uintptr {
	if len(data) < 22 {
		return 0
	}

	// Parse .ico header: type must be 1 (icon)
	if data[2] != 1 || data[3] != 0 {
		return 0
	}
	count := int(data[4]) | int(data[5])<<8
	if count == 0 {
		return 0
	}

	// Find the largest icon entry by dimensions
	type icoEntry struct {
		offset uint32
	}
	var best icoEntry
	bestSize := uint32(0)

	for i := 0; i < count; i++ {
		entryOff := 6 + i*16
		if entryOff+16 > len(data) {
			break
		}
		w := data[entryOff]
		h := data[entryOff+1]
		offset := uint32(data[entryOff+12]) | uint32(data[entryOff+13])<<8 |
			uint32(data[entryOff+14])<<16 | uint32(data[entryOff+15])<<24

		dim := uint32(w) * uint32(h)
		if w == 0 {
			dim = 256 * 256 // 0 means 256
		}
		if dim > bestSize {
			bestSize = dim
			best = icoEntry{offset: offset}
		}
	}

	if best.offset == 0 || int(best.offset) >= len(data) {
		return 0
	}

	iconData := data[best.offset:]
	return winapi.CreateIconFromResourceEx(
		iconData, uint32(len(iconData)),
		true,
		0x00030000,
		0, 0,
		0, // LR_DEFAULTSIZE
	)
}

// loadPNGasIcon loads a PNG image and converts it to an HICON via GDI+.
func loadPNGasIcon(data []byte) uintptr {
	if len(data) == 0 {
		return 0
	}
	return createIconFromPNG(data)
}

// createIconFromPNG uses GDI+ to decode a PNG and return an HICON.
func createIconFromPNG(pngData []byte) uintptr {
	token := winapi.GdiplusStartup()
	if token == 0 {
		return createFallbackIcon()
	}
	defer winapi.GdiplusShutdown(token)

	stream := winapi.CreateStreamOnHGlobal(0, true)
	if stream == 0 {
		return 0
	}
	defer winapi.ReleaseStream(stream)

	if !winapi.WriteToStream(stream, pngData) {
		return 0
	}
	winapi.SeekStream(stream, 0, 0)

	bitmap := winapi.GdipCreateBitmapFromStream(stream)
	if bitmap == 0 {
		return 0
	}
	defer winapi.GdipDisposeImage(bitmap)

	return winapi.GdipCreateHICONFromBitmap(bitmap)
}

// initIcons loads all icons into the TrayApp.
func (t *TrayApp) initIcons() {
	t.idleIcon, t.busyIcon, t.updateIcon = loadIcons()
}

// createFallbackIcon creates a simple solid-color 32x32 icon without GDI+.
func createFallbackIcon() uintptr {
	// 32x32 icon: AND mask (1bpp, 128 bytes) + XOR mask (32bpp, 4096 bytes)
	andMask := make([]byte, 128)
	xorMask := make([]byte, 32*32*4)

	// Fill XOR mask with a visible color (orange-ish)
	for i := 0; i < len(xorMask); i += 4 {
		xorMask[i] = 0     // Blue
		xorMask[i+1] = 128 // Green
		xorMask[i+2] = 255 // Red
		xorMask[i+3] = 0
	}

	return winapi.CreateIcon(0, 32, 32, 1, 32, andMask, xorMask)
}
