//go:build windows

package tray

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"windows-us-international-keyboard-without-dead-keys/internal/keyicon"
)

var (
	modGdi32 = windows.NewLazySystemDLL("gdi32.dll")

	procCreateDIBSection   = modGdi32.NewProc("CreateDIBSection")
	procCreateBitmap       = modGdi32.NewProc("CreateBitmap")
	procDeleteObject       = modGdi32.NewProc("DeleteObject")
	procCreateIconIndirect = modUser32.NewProc("CreateIconIndirect")
	procDestroyIcon        = modUser32.NewProc("DestroyIcon")
	procGetSystemMetrics   = modUser32.NewProc("GetSystemMetrics")
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

// smallIconMetric is SM_CXSMICON: the width Windows draws tray icons at,
// which grows with the display scaling (16 at 100%, 24 at 150%, ...).
const smallIconMetric = 49

// iconSize is the size to render the tray icon at: exactly what the
// taskbar will show, so Windows never has to rescale it. It needs the
// process to be DPI aware (see win32.EnableDPIAwareness) to see the real
// value rather than the 100% one.
func iconSize() int {
	if n, _, _ := procGetSystemMetrics.Call(smallIconMetric); n > 0 {
		return int(n)
	}
	return 16
}

// pixel is BGRA order (what a 32bpp Windows DIB section expects).
type pixel struct{ B, G, R, A byte }

// buildKeyIcon renders keyicon's glyph - a keycap with an "Á" on it - as an
// alpha-blended HICON at the taskbar's icon size, in its enabled
// (black) or disabled (grey with a red strike) look.
func buildKeyIcon(enabled bool) (uintptr, error) {
	size := iconSize()
	glyph := keyicon.Render(size, enabled)

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
	// Icon bitmaps take straight (non-premultiplied) alpha, which is what
	// Render produces.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := glyph.NRGBAAt(x, y)
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

// EnabledIcon returns the keycap glyph as a black HICON (interception
// active).
func EnabledIcon() (uintptr, error) { return buildKeyIcon(true) }

// DisabledIcon returns the same keycap glyph as EnabledIcon, greyed out
// with a diagonal red strike across it (interception paused).
func DisabledIcon() (uintptr, error) { return buildKeyIcon(false) }

// DestroyIconHandle frees an HICON returned by EnabledIcon/DisabledIcon.
// Safe to call on a zero handle.
func DestroyIconHandle(hIcon uintptr) {
	if hIcon != 0 {
		procDestroyIcon.Call(hIcon)
	}
}
