//go:build windows

// Package win32 holds the raw Win32 declarations and helpers the rest of
// the program needs - window-class registration, window creation, the
// message loop, opening a file in its default application - plus every
// private message number this program defines, kept in one place so they
// can never collide.
package win32

import (
	"fmt"
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
