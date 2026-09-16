//go:build windows

// Package win32 holds the raw Win32 declarations and helpers the rest of
// the program needs - window-class registration, window creation, the
// message loop, opening a file in its default application - plus every
// private message number this program defines, kept in one place so they
// can never collide.
package win32

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modUser32   = windows.NewLazySystemDLL("user32.dll")

	modShell32 = windows.NewLazySystemDLL("shell32.dll")

	procGetModuleHandleW    = modKernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW    = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW     = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW      = modUser32.NewProc("DefWindowProcW")
	procDestroyWindow       = modUser32.NewProc("DestroyWindow")
	procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
	procPostMessageW        = modUser32.NewProc("PostMessageW")
	procGetMessageW         = modUser32.NewProc("GetMessageW")
	procTranslateMessage    = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW    = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage     = modUser32.NewProc("PostQuitMessage")

	procShellExecuteW = modShell32.NewProc("ShellExecuteW")

	procSetProcessDpiAwarenessContext = modUser32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDPIAware            = modUser32.NewProc("SetProcessDPIAware")
)

// Message numbers from WM_APP upward are free for application use.
const (
	wmApp = 0x8000

	// WMTrayCallback is sent to the tray icon's window when the icon is
	// clicked.
	WMTrayCallback = wmApp + 1

	// WMSetupUpdate is posted to the setup window by its worker goroutine
	// whenever it has progress or a result to show.
	WMSetupUpdate = wmApp + 2
)

// CWUseDefault is CW_USEDEFAULT, for CreateWindow's position and size.
const CWUseDefault int32 = -0x80000000

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

// ModuleHandle returns this exe's HINSTANCE.
func ModuleHandle() syscall.Handle {
	r, _, _ := procGetModuleHandleW.Call(0)
	return syscall.Handle(r)
}

// UTF16Ptr converts s to a NUL-terminated UTF-16 string for a Win32 call.
// It panics if s contains a NUL byte, so it's only for this program's own
// fixed strings, never for user input.
func UTF16Ptr(s string) *uint16 {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		panic(err)
	}
	return p
}

// RegisterClass registers a window class with the given window procedure
// (from syscall.NewCallback) and background brush (0 for none).
func RegisterClass(name string, wndProc uintptr, background syscall.Handle) error {
	wc := wndClassExW{
		lpfnWndProc:   wndProc,
		hInstance:     ModuleHandle(),
		hbrBackground: background,
		lpszClassName: UTF16Ptr(name),
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("RegisterClassExW: %w", e)
	}
	return nil
}

// CreateWindow creates, but doesn't show, a window of a class registered
// with RegisterClass.
func CreateWindow(exStyle, style uint32, class, title string, x, y, width, height int32) (uintptr, error) {
	hwnd, _, e := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(UTF16Ptr(class))),
		uintptr(unsafe.Pointer(UTF16Ptr(title))),
		uintptr(style),
		intArg(x), intArg(y), intArg(width), intArg(height),
		0, 0, uintptr(ModuleHandle()), 0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW: %w", e)
	}
	return hwnd, nil
}

// CreateChild creates a visible-or-not (per style) child control of parent
// with the given control ID.
func CreateChild(class, text string, style uint32, parent uintptr, id int) (uintptr, error) {
	hwnd, _, e := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(UTF16Ptr(class))),
		uintptr(unsafe.Pointer(UTF16Ptr(text))),
		uintptr(style),
		0, 0, 0, 0,
		parent, uintptr(id), uintptr(ModuleHandle()), 0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW(%s): %w", class, e)
	}
	return hwnd, nil
}

// intArg passes a possibly negative int32 (a monitor left of the primary
// one, or CWUseDefault) in a syscall argument slot, going via int64 so
// the sign survives regardless of pointer width.
func intArg(v int32) uintptr { return uintptr(int64(v)) }

// DestroyWindow destroys hwnd. Safe to call on 0.
func DestroyWindow(hwnd uintptr) {
	if hwnd != 0 {
		procDestroyWindow.Call(hwnd)
	}
}

// SetForegroundWindow brings hwnd to the foreground.
func SetForegroundWindow(hwnd uintptr) { procSetForegroundWindow.Call(hwnd) }

// DefWindowProc is the default window procedure, for every message a
// window procedure doesn't handle itself.
func DefWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

// PostMessage posts msg to hwnd's queue, or to the calling thread's own
// queue if hwnd is 0.
func PostMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) {
	procPostMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
}

// Msg is a Win32 MSG: one message taken off the calling thread's queue.
type Msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// GetMessage blocks until the next message for this thread arrives and
// returns it, or reports ok == false once WM_QUIT (from PostQuitMessage)
// ends the loop.
func GetMessage() (m Msg, ok bool) {
	r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
	// GetMessageW returns 0 for WM_QUIT and -1 for an error; either way
	// the loop is over.
	return m, int32(r) > 0
}

// Dispatch translates and dispatches a message to its window procedure.
func Dispatch(m Msg) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
}

// PostQuitMessage asks the calling thread's message loop to end.
func PostQuitMessage() { procPostQuitMessage.Call(0) }

// OpenFile opens path in whatever application Windows has associated with
// its file type, exactly as double-clicking it in Explorer would.
func OpenFile(path string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	// ShellExecuteW reports success as a fake HINSTANCE greater than 32.
	if r, _, e := procShellExecuteW.Call(0,
		uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, uintptr(swShowNormal)); r <= 32 {
		return fmt.Errorf("ShellExecuteW(%s): %w", path, e)
	}
	return nil
}

const swShowNormal = 1

// dpiAwarenessContextPerMonitorAwareV2 is (DPI_AWARENESS_CONTEXT)-4,
// written this way so it is correct whatever uintptr's width.
const dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3)

// EnableDPIAwareness tells Windows this program handles display scaling
// itself. Without it, Windows reports every size as if at 100% and then
// stretches whatever the program draws - which is what makes a tray icon
// blurry at 125% or 150%. It picks the best mode the running Windows
// supports and silently does nothing on failure.
func EnableDPIAwareness() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		if r, _, _ := procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2); r != 0 {
			return
		}
	}
	if procSetProcessDPIAware.Find() == nil {
		procSetProcessDPIAware.Call()
	}
}

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
