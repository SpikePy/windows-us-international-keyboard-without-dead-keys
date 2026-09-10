// Command windows-us-international-keyboard-without-dead-keys is a small
// background program for Windows that adds
// AltGr (Right Alt) shortcuts for accented characters, without installing
// anything system-wide - no admin rights needed. Hold the physical Right
// Alt key and press a mapped letter/digit/symbol to get the accented
// character immediately (no dead keys - nothing waits for a second
// keystroke).
//
// It implements a "US International - AltGr - No Dead Keys" character set:
// AltGr+letter/digit/symbol produces the accented character directly. This
// only activates while the active Windows keyboard layout for the
// focused window is actually "United States-International" - switching to
// German, plain US, or any other layout makes this program a no-op, so
// normal typing under those layouts is completely unaffected.
//
// Technique: a low-level keyboard hook (WH_KEYBOARD_LL) watches for the
// Right Alt key plus a mapped key; when both are held, it swallows that
// keystroke and injects the target Unicode character via SendInput
// instead. Everything else passes through untouched.
package main

import (
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	whKeyboardLL = 13

	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105

	inputKeyboard    = 1
	keyEventFKeyUp   = 0x0002
	keyEventFUnicode = 0x0004

	vkShift = 0x10
	vkRMenu = 0xA5 // Right Alt / AltGr

	// Standard, stable Win32 virtual-key codes for the US OEM punctuation
	// keys (letters and digits use their own ASCII codes as VK codes).
	vkOemMinus = 0xBD // -
	vkOemPlus  = 0xBB // =
	vkOem3     = 0xC0 // ` (grave/backtick)
	vkOem4     = 0xDB // [
	vkOem6     = 0xDD // ]
	vkOem1     = 0xBA // ;
	vkOem7     = 0xDE // '
	vkOem5     = 0xDC // backslash
	vkOemComma = 0xBC // ,
	vkOem2     = 0xBF // /
)

// usInternationalKLID is the 8-hex-digit keyboard layout identifier for
// Windows' built-in "United States-International" layout, as reported by
// GetKeyboardLayoutNameW.
const usInternationalKLID = "00020409"

// undeadMap covers keys that are dead keys under some active Windows
// layouts (notably the built-in "United States-International") even
// without AltGr - e.g. apostrophe and backtick normally wait for a second
// keystroke to combine into an accented letter. These are always forced to
// produce their plain character immediately, regardless of AltGr state,
// bypassing whatever the active layout would otherwise do.
var undeadMap = map[uint32][2]rune{
	vkOem7: {'\'', '"'},
	vkOem3: {'`', '~'},
	'6':    {'6', '^'},
}

// altGrMap maps a virtual-key code to its {base, Shift+AltGr} characters.
var altGrMap = map[uint32][2]rune{
	'1':        {0x00A1, 0x00B9},
	'2':        {0x00B2, 0x00B2},
	'3':        {0x00B3, 0x00B3},
	'4':        {0x00A4, 0x00A3},
	'5':        {0x20AC, 0x20AC},
	'6':        {0x00BC, 0x00BC},
	'7':        {0x00BD, 0x00BD},
	'8':        {0x00BE, 0x00BE},
	'9':        {0x2018, 0x2018},
	'0':        {0x2019, 0x2019},
	vkOemMinus: {0x00A5, 0x00A5},
	vkOemPlus:  {0x00D7, 0x00F7},
	'Q':        {0x00E4, 0x00C4},
	'W':        {0x00E5, 0x00C5},
	'E':        {0x00E9, 0x00C9},
	'R':        {0x00AE, 0x00AE},
	'T':        {0x00FE, 0x00DE},
	'Y':        {0x00FC, 0x00DC},
	'U':        {0x00FA, 0x00DA},
	'I':        {0x00ED, 0x00CD},
	'O':        {0x00F3, 0x00D3},
	'P':        {0x00F6, 0x00D6},
	vkOem4:     {0x00AB, 0x00AB},
	vkOem6:     {0x00BB, 0x00BB},
	'A':        {0x00E1, 0x00C1},
	'S':        {0x00DF, 0x00A7},
	'D':        {0x00F0, 0x00D0},
	'L':        {0x00F8, 0x00D8},
	vkOem1:     {0x00B6, 0x00B0},
	vkOem7:     {0x00B4, 0x00A8},
	vkOem5:     {0x00AC, 0x00A6},
	'Z':        {0x00E6, 0x00C6},
	'C':        {0x00A9, 0x00A2},
	'N':        {0x00F1, 0x00D1},
	'M':        {0x00B5, 0x00B5},
	vkOemComma: {0x00E7, 0x00C7},
	vkOem2:     {0x00BF, 0x00BF},
}

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

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procSetWindowsHookExW    = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx       = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx  = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procGetKeyState          = user32.NewProc("GetKeyState")
	procSendInput            = user32.NewProc("SendInput")
	procGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
	procGetForegroundWindow  = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadPID   = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput    = user32.NewProc("AttachThreadInput")
	procGetKeyboardLayoutNam = user32.NewProc("GetKeyboardLayoutNameW")
	procGetCurrentThreadId   = kernel32.NewProc("GetCurrentThreadId")

	mu       sync.Mutex
	raltDown bool

	layoutMu       sync.Mutex
	layoutHwnd     uintptr
	layoutIsUSIntl bool
)

// isUSInternationalActive reports whether the keyboard layout of the
// currently focused window is Windows' built-in "United States-
// International" layout. The result is cached per foreground window,
// since querying it requires briefly attaching to that window's input
// thread.
func isUSInternationalActive() bool {
	hwnd, _, _ := procGetForegroundWindow.Call()

	layoutMu.Lock()
	if hwnd == layoutHwnd {
		result := layoutIsUSIntl
		layoutMu.Unlock()
		return result
	}
	layoutMu.Unlock()

	isUSIntl := false
	if hwnd != 0 {
		targetThreadID, _, _ := procGetWindowThreadPID.Call(hwnd, 0)
		currentThreadID, _, _ := procGetCurrentThreadId.Call()

		attached := false
		if targetThreadID != 0 && targetThreadID != currentThreadID {
			ret, _, _ := procAttachThreadInput.Call(currentThreadID, targetThreadID, 1)
			attached = ret != 0
		}

		var name [9]uint16 // KL_NAMELENGTH
		procGetKeyboardLayoutNam.Call(uintptr(unsafe.Pointer(&name[0])))
		isUSIntl = strings.EqualFold(syscall.UTF16ToString(name[:]), usInternationalKLID)

		if attached {
			procAttachThreadInput.Call(currentThreadID, targetThreadID, 0)
		}
	}

	layoutMu.Lock()
	layoutHwnd = hwnd
	layoutIsUSIntl = isUSIntl
	layoutMu.Unlock()

	return isUSIntl
}

func hookProc(nCode int32, wParam, lParam uintptr) uintptr {
	if nCode >= 0 {
		kb := (*kbdllhookstruct)(unsafe.Pointer(lParam))
		vk := kb.VkCode

		if vk == vkRMenu {
			mu.Lock()
			switch wParam {
			case wmKeyDown, wmSysKeyDown:
				raltDown = true
			case wmKeyUp, wmSysKeyUp:
				raltDown = false
			}
			mu.Unlock()
		} else if isUSInternationalActive() {
			mu.Lock()
			down := raltDown
			mu.Unlock()

			if down {
				if chars, ok := altGrMap[vk]; ok {
					if wParam == wmKeyDown || wParam == wmSysKeyDown {
						sendUnicodeChar(pickChar(chars))
					}
					// Swallow both keydown and keyup for mapped keys
					// while AltGr is held, so the underlying app never
					// sees the raw, unmapped key.
					return 1
				}
			}

			// Not an AltGr combo (or AltGr isn't held): still force
			// undead keys to their plain character immediately, instead
			// of the dead-key wait this layout would otherwise apply.
			if chars, ok := undeadMap[vk]; ok {
				if wParam == wmKeyDown || wParam == wmSysKeyDown {
					sendUnicodeChar(pickChar(chars))
				}
				return 1
			}
		}
	}
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

func pickChar(chars [2]rune) rune {
	state, _, _ := procGetKeyState.Call(uintptr(vkShift))
	if int16(state) < 0 {
		return chars[1]
	}
	return chars[0]
}

func sendUnicodeChar(ch rune) {
	var in input
	in.Type = inputKeyboard
	in.Ki.WScan = uint16(ch)
	in.Ki.DwFlags = keyEventFUnicode
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))

	in.Ki.DwFlags = keyEventFUnicode | keyEventFKeyUp
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func main() {
	hInstance, _, _ := procGetModuleHandleW.Call(0)

	hook, _, _ := procSetWindowsHookExW.Call(
		uintptr(whKeyboardLL),
		syscall.NewCallback(hookProc),
		hInstance,
		0,
	)
	if hook == 0 {
		return
	}
	defer procUnhookWindowsHookEx.Call(hook)

	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
