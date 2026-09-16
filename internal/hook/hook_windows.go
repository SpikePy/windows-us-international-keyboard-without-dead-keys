//go:build windows

package hook

import (
	"fmt"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procSetWindowsHookExW      = modUser32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx    = modUser32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx         = modUser32.NewProc("CallNextHookEx")
	procGetKeyState            = modUser32.NewProc("GetKeyState")
	procSendInput              = modUser32.NewProc("SendInput")
	procGetForegroundWindow    = modUser32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcess = modUser32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput      = modUser32.NewProc("AttachThreadInput")
	procGetKeyboardLayoutNameW = modUser32.NewProc("GetKeyboardLayoutNameW")

	procGetCurrentThreadId = modKernel32.NewProc("GetCurrentThreadId")
)

const (
	whKeyboardLL = 13

	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105

	vkShift = 0x10
	vkRMenu = 0xA5 // Right Alt / AltGr

	inputKeyboard    = 1
	keyEventFKeyUp   = 0x0002
	keyEventFUnicode = 0x0004

	// klNameLength is KL_NAMELENGTH: eight hex digits plus the NUL.
	klNameLength = 9
)

type kbdllhookstruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type keybdinput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// input mirrors the Win32 INPUT struct: a DWORD type tag followed by the
// largest union member (MOUSEINPUT), which is wider than KEYBDINPUT - the
// trailing padding keeps the overall size correct so SendInput reads it
// the same way the real struct would lay out in memory.
type input struct {
	Type uint32
	Ki   keybdinput
	_    [8]byte
}

// Options configures Install. The two feature switches and the layout
// restriction come straight from config.yaml (or the matching flags).
type Options struct {
	AltGrShortcuts bool
	UndeadKeys     bool
	// LayoutID restricts interception to the keyboard layout with this
	// 8-hex-digit ID; empty means every layout.
	LayoutID string
	// StartEnabled is whether interception is active from the start.
	StartEnabled bool
}

// Hook is an installed low-level keyboard hook.
//
// Everything except Enabled/SetEnabled runs on the thread that called
// Install, and Windows delivers the hook callback on that same thread's
// message loop, so the hook's own bookkeeping needs no locking; only the
// enabled flag is atomic, as the tray's callbacks may flip it.
type Hook struct {
	handle  uintptr
	opts    Options
	enabled atomic.Bool

	// altGrHeld tracks Right Alt across events, because a low-level hook
	// sees each key on its own and GetKeyState doesn't reflect keys the
	// hook itself is still deciding about.
	altGrHeld bool

	// layoutCache remembers the layout check for one foreground window:
	// answering it means briefly attaching to that window's input thread,
	// which is far too expensive to redo on every keystroke.
	layoutCache struct {
		hwnd    uintptr
		matches bool
		valid   bool
	}
}

// active is the installed hook. Win32 hook callbacks carry no user data,
// so the callback has to find its way back here; Install allows only one
// at a time.
var active *Hook

var hookProc = syscall.NewCallback(func(nCode int32, wParam, lParam uintptr) uintptr {
	if nCode >= 0 && active != nil {
		if ret, handled := active.handleKey((*kbdllhookstruct)(unsafe.Pointer(lParam)).VkCode, wParam); handled {
			return ret
		}
	}
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
})

// Install puts the keyboard hook in place. It must be called on a thread
// that pumps a message loop for as long as the hook lives (see
// runtime.LockOSThread in main), and Uninstall must be called on that
// same thread.
func Install(opts Options) (*Hook, error) {
	if active != nil {
		return nil, fmt.Errorf("a keyboard hook is already installed")
	}
	h := &Hook{opts: opts}
	h.enabled.Store(opts.StartEnabled)

	active = h
	handle, _, e := procSetWindowsHookExW.Call(whKeyboardLL, hookProc, 0, 0)
	if handle == 0 {
		active = nil
		return nil, fmt.Errorf("SetWindowsHookExW(WH_KEYBOARD_LL): %w", e)
	}
	h.handle = handle
	return h, nil
}

// Uninstall removes the hook. Safe to call on a nil Hook, and more than
// once.
func (h *Hook) Uninstall() {
	if h == nil || h.handle == 0 {
		return
	}
	procUnhookWindowsHookEx.Call(h.handle)
	h.handle = 0
	if active == h {
		active = nil
	}
}

// Enabled reports whether keys are being intercepted.
func (h *Hook) Enabled() bool { return h.enabled.Load() }

// SetEnabled pauses (false) or resumes (true) interception without
// removing the hook, so toggling it from the tray never costs the user
// their hook registration.
func (h *Hook) SetEnabled(v bool) { h.enabled.Store(v) }

// handleKey applies Decide to one key event. It reports handled == false
// for anything the hook doesn't act on, which the callback then passes to
// the next hook in the chain.
func (h *Hook) handleKey(vk uint32, wParam uintptr) (ret uintptr, handled bool) {
	var keyDown bool
	switch wParam {
	case wmKeyDown, wmSysKeyDown:
		keyDown = true
	case wmKeyUp, wmSysKeyUp:
	default:
		return 0, false
	}

	// Right Alt is never swallowed: applications must still see it, and
	// the modifier it forms is only known by watching it here.
	if vk == vkRMenu {
		h.altGrHeld = keyDown
		return 0, false
	}

	state := State{
		Enabled:        h.enabled.Load(),
		AltGrShortcuts: h.opts.AltGrShortcuts,
		UndeadKeys:     h.opts.UndeadKeys,
		AltGrHeld:      h.altGrHeld,
		ShiftHeld:      keyHeld(vkShift),
	}
	if state.Enabled {
		state.LayoutMatches = h.layoutMatches()
	}

	switch action, ch := Decide(state, vk, keyDown); action {
	case Type:
		sendUnicodeChar(ch)
		return 1, true
	case Swallow:
		return 1, true
	default:
		return 0, false
	}
}

// layoutMatches reports whether the focused window's keyboard layout is
// the one the hook is restricted to, cached per foreground window.
func (h *Hook) layoutMatches() bool {
	if h.opts.LayoutID == "" {
		return true
	}
	hwnd, _, _ := procGetForegroundWindow.Call()
	if h.layoutCache.valid && h.layoutCache.hwnd == hwnd {
		return h.layoutCache.matches
	}

	matches := false
	if hwnd != 0 {
		matches = strings.EqualFold(foregroundLayoutID(hwnd), h.opts.LayoutID)
	}
	h.layoutCache.hwnd, h.layoutCache.matches, h.layoutCache.valid = hwnd, matches, true
	return matches
}

// foregroundLayoutID returns the layout ID of hwnd's input thread.
// GetKeyboardLayoutNameW only ever reports the calling thread's layout,
// so the call is sandwiched between attaching to and detaching from that
// thread's input queue.
func foregroundLayoutID(hwnd uintptr) string {
	targetThread, _, _ := procGetWindowThreadProcess.Call(hwnd, 0)
	currentThread, _, _ := procGetCurrentThreadId.Call()

	attached := false
	if targetThread != 0 && targetThread != currentThread {
		r, _, _ := procAttachThreadInput.Call(currentThread, targetThread, 1)
		attached = r != 0
	}

	var name [klNameLength]uint16
	procGetKeyboardLayoutNameW.Call(uintptr(unsafe.Pointer(&name[0])))

	if attached {
		procAttachThreadInput.Call(currentThread, targetThread, 0)
	}
	return windows.UTF16ToString(name[:])
}

// keyHeld reports whether the key with virtual-key code vk is currently
// down (GetKeyState's high bit).
func keyHeld(vk uintptr) bool {
	state, _, _ := procGetKeyState.Call(vk)
	return int16(state) < 0
}

// sendUnicodeChar types ch as a Unicode keystroke, independent of the
// active layout: KEYEVENTF_UNICODE events carry the character itself
// rather than a key to be translated. The synthetic events come back
// through this hook with virtual-key code 0, which no table maps, so they
// pass straight through.
func sendUnicodeChar(ch rune) {
	in := input{Type: inputKeyboard}
	in.Ki.WScan = uint16(ch)
	in.Ki.DwFlags = keyEventFUnicode
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))

	in.Ki.DwFlags = keyEventFUnicode | keyEventFKeyUp
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}
