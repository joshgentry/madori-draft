package winapi

import (
	"syscall"
	"unsafe"
)

var (
	modGdiplus = syscall.NewLazyDLL("gdiplus.dll")
	modOle32   = syscall.NewLazyDLL("ole32.dll")

	procGdiplusStartup             = modGdiplus.NewProc("GdiplusStartup")
	procGdiplusShutdown            = modGdiplus.NewProc("GdiplusShutdown")
	procGdipCreateBitmapFromStream = modGdiplus.NewProc("GdipCreateBitmapFromStream")
	procGdipCreateHICONFromBitmap  = modGdiplus.NewProc("GdipCreateHICONFromBitmap")
	procGdipDisposeImage           = modGdiplus.NewProc("GdipDisposeImage")

	procCreateStreamOnHGlobal = modOle32.NewProc("CreateStreamOnHGlobal")
	procGetHGlobalFromStream  = modOle32.NewProc("GetHGlobalFromStream")
)

// GDI+ types
type GdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}

type GdiplusStartupOutput struct {
	NotificationHook   uintptr
	NotificationUnhook uintptr
}

// IStream COM interface methods we need (simplified — just the vtable offsets)
// IStream inherits from ISequentialStream which inherits from IUnknown

// GDI+ wrappers

func GdiplusStartup() uintptr {
	var input GdiplusStartupInput
	input.GdiplusVersion = 1
	var token uintptr
	ret, _, _ := procGdiplusStartup.Call(
		uintptr(unsafe.Pointer(&token)),
		uintptr(unsafe.Pointer(&input)),
		0,
	)
	if ret != 0 {
		return 0
	}
	return token
}

func GdiplusShutdown(token uintptr) {
	procGdiplusShutdown.Call(token)
}

// Stream helpers using OLE32

func CreateStreamOnHGlobal(hGlobal uintptr, fDeleteOnRelease bool) uintptr {
	var deleteIt uintptr
	if fDeleteOnRelease {
		deleteIt = 1
	}
	var stream uintptr
	ret, _, _ := procCreateStreamOnHGlobal.Call(
		hGlobal,
		deleteIt,
		uintptr(unsafe.Pointer(&stream)),
	)
	if ret != 0 {
		return 0
	}
	return stream
}

func GetHGlobalFromStream(stream uintptr) uintptr {
	var hglobal uintptr
	procGetHGlobalFromStream.Call(stream, uintptr(unsafe.Pointer(&hglobal)))
	return hglobal
}

// ReleaseStream releases the COM stream object.
func ReleaseStream(stream uintptr) {
	// IUnknown::Release is at vtable[2]
	if stream == 0 {
		return
	}
	vtbl := *(*uintptr)(unsafe.Pointer(stream))
	release := *(*uintptr)(unsafe.Pointer(vtbl + 2*unsafe.Sizeof(uintptr(0))))
	syscall.SyscallN(release, stream)
}

// WriteToStream writes data to a COM IStream.
func WriteToStream(stream uintptr, data []byte) bool {
	if stream == 0 || len(data) == 0 {
		return false
	}
	// ISequentialStream::Write is at vtable[4]
	vtbl := *(*uintptr)(unsafe.Pointer(stream))
	write := *(*uintptr)(unsafe.Pointer(vtbl + 4*unsafe.Sizeof(uintptr(0))))
	var written uint32
	ret, _, _ := syscall.SyscallN(write, stream, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), uintptr(unsafe.Pointer(&written)))
	return ret == 0 && written == uint32(len(data))
}

// SeekStream seeks in a COM IStream.
func SeekStream(stream uintptr, offset int64, origin uint32) {
	if stream == 0 {
		return
	}
	// IStream::Seek is at vtable[5]
	vtbl := *(*uintptr)(unsafe.Pointer(stream))
	seek := *(*uintptr)(unsafe.Pointer(vtbl + 5*unsafe.Sizeof(uintptr(0))))
	var newPos uint64
	syscall.SyscallN(seek, stream, uintptr(offset), uintptr(origin), uintptr(unsafe.Pointer(&newPos)))
}

// GDI+ Image functions

func GdipCreateBitmapFromStream(stream uintptr) uintptr {
	var bitmap uintptr
	ret, _, _ := procGdipCreateBitmapFromStream.Call(stream, uintptr(unsafe.Pointer(&bitmap)))
	if ret != 0 {
		return 0
	}
	return bitmap
}

func GdipCreateHICONFromBitmap(bitmap uintptr) uintptr {
	var hIcon uintptr
	ret, _, _ := procGdipCreateHICONFromBitmap.Call(bitmap, uintptr(unsafe.Pointer(&hIcon)))
	if ret != 0 {
		return 0
	}
	return hIcon
}

func GdipDisposeImage(image uintptr) {
	procGdipDisposeImage.Call(image)
}
