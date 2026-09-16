//go:build windows

package win32

import (
	"fmt"
	"image"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modGdi32 = windows.NewLazySystemDLL("gdi32.dll")

	procCreateDIBSection   = modGdi32.NewProc("CreateDIBSection")
	procCreateBitmap       = modGdi32.NewProc("CreateBitmap")
	procDeleteObject       = modGdi32.NewProc("DeleteObject")
	procCreateIconIndirect = modUser32.NewProc("CreateIconIndirect")
	procDestroyIcon        = modUser32.NewProc("DestroyIcon")
)

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  uintptr
	hbmColor uintptr
}

// pixel is BGRA order (what a 32bpp Windows DIB section expects).
type pixel struct{ B, G, R, A byte }

// NewIcon turns a square image with straight (non-premultiplied) alpha -
// what icon bitmaps take - into an alpha-blended HICON. Free it with
// DestroyIcon.
func NewIcon(img *image.NRGBA) (uintptr, error) {
	size := img.Bounds().Dx()
	if size <= 0 || img.Bounds().Dy() != size {
		return 0, fmt.Errorf("icon image must be square, got %v", img.Bounds())
	}

	var bi bitmapInfoHeader
	bi.biSize = uint32(unsafe.Sizeof(bi))
	bi.biWidth = int32(size)
	bi.biHeight = -int32(size) // negative = top-down DIB, same row order as the image
	bi.biPlanes = 1
	bi.biBitCount = 32
	bi.biCompression = 0 // BI_RGB

	var bitsPtr uintptr
	hColor, _, e := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bitsPtr)), 0, 0)
	if hColor == 0 {
		return 0, fmt.Errorf("CreateDIBSection: %w", e)
	}
	pixels := unsafe.Slice((*pixel)(unsafe.Pointer(bitsPtr)), size*size)
	min := img.Bounds().Min
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := img.NRGBAAt(min.X+x, min.Y+y)
			pixels[y*size+x] = pixel{B: c.B, G: c.G, R: c.R, A: c.A}
		}
	}

	// AND mask: all zero bits means "always use the color bitmap's own
	// alpha", the standard approach for a modern alpha-blended icon. Each
	// row is padded to a 16-bit boundary.
	maskBytes := make([]byte, ((size+15)/16*2)*size)
	hMask, _, e := procCreateBitmap.Call(uintptr(size), uintptr(size), 1, 1, uintptr(unsafe.Pointer(&maskBytes[0])))
	if hMask == 0 {
		procDeleteObject.Call(hColor)
		return 0, fmt.Errorf("CreateBitmap: %w", e)
	}

	ii := iconInfo{fIcon: 1, hbmMask: hMask, hbmColor: hColor}
	hIcon, _, e := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))

	// CreateIconIndirect copies the bitmap data internally; the source
	// bitmaps are ours to delete right away regardless of its outcome.
	procDeleteObject.Call(hColor)
	procDeleteObject.Call(hMask)

	if hIcon == 0 {
		return 0, fmt.Errorf("CreateIconIndirect: %w", e)
	}
	return hIcon, nil
}

// DestroyIcon frees an HICON from NewIcon. Safe to call on 0.
func DestroyIcon(hIcon uintptr) {
	if hIcon != 0 {
		procDestroyIcon.Call(hIcon)
	}
}

// DeleteObject frees a GDI object (font, brush, bitmap). Safe to call on 0.
func DeleteObject(h uintptr) {
	if h != 0 {
		procDeleteObject.Call(h)
	}
}
