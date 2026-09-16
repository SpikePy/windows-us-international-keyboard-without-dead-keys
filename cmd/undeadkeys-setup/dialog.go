//go:build windows

package main

// Setup's window is a Windows task dialog (TaskDialogIndirect): the
// system's own dialog with buttons, a progress bar and pages, so Setup
// needs no GUI toolkit and draws nothing itself but the tool's icon. Task
// dialogs live in version 6 of the common controls, which setup.manifest -
// embedded through rsrc_windows_amd64.syso - asks for.

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"windows-us-international-keyboard-without-dead-keys/internal/config"
	"windows-us-international-keyboard-without-dead-keys/internal/keyicon"
	"windows-us-international-keyboard-without-dead-keys/internal/setup"
	"windows-us-international-keyboard-without-dead-keys/internal/win32"
)

// comctl32 is loaded by bare name, not from an explicit System32 path, so
// the manifest can redirect it to version 6 - the one with task dialogs.
var (
	modComctl32            = windows.NewLazyDLL("comctl32.dll")
	procTaskDialogIndirect = modComctl32.NewProc("TaskDialogIndirect")
	procGetSystemMetrics   = windows.NewLazySystemDLL("user32.dll").NewProc("GetSystemMetrics")
)

const (
	wmUser                   = 0x0400
	tdmNavigatePage          = wmUser + 101
	tdmSetProgressBarMarquee = wmUser + 107
	tdmSetElementText        = wmUser + 108
	tdmClickButton           = wmUser + 102
	tdmEnableButton          = wmUser + 111

	tdnNavigated     = 1
	tdnButtonClicked = 2
	tdnTimer         = 4

	tdeContent = 0

	tdfUseHIconMain            = 0x0002
	tdfAllowDialogCancellation = 0x0008
	tdfShowMarqueeProgressBar  = 0x0400
	tdfCallbackTimer           = 0x0800

	tdErrorIcon = 0xFFFE // MAKEINTRESOURCE(-2)

	sOK    = 0
	sFalse = 1

	smCxIcon = 11

	// The three buttons every Setup has. Close is IDCANCEL, so Escape and
	// the title bar's X do exactly what it does.
	idClose     = 2 // IDCANCEL
	idInstall   = 101
	idUninstall = 102

	// Which page is showing, passed to the callback as its reference data.
	pageChoose   = 0
	pageProgress = 1
	pageDone     = 2
	// pageClosing is the result of a successful action, which closes
	// itself after setup.AutoCloseSeconds.
	pageClosing = 3
)

const title = "UndeadKeys Setup"

var (
	// Set before the dialog opens and only read afterwards. The callback
	// is created in runDialog rather than here because dialogProc leads
	// back to it (through pack), which Go rejects as an init cycle.
	dialogCallback uintptr
	dialogOpts     options
	appIcon        uintptr

	// failed is set by the worker and read once the dialog has closed.
	failed atomic.Bool

	// firstContent is page one's text above the countdown line, and
	// shownCountdown the line currently under whichever page counts down.
	// Both are only touched on the dialog's thread, as is timerRestart:
	// set when the closing page appears, so its countdown starts from
	// there rather than from when the dialog opened.
	firstContent   string
	shownCountdown string
	timerRestart   bool

	// closingContent is the closing page's text above its countdown line.
	// The worker sets it just before navigating there, and the dialog's
	// thread reads it afterwards.
	closingContent atomic.Value

	// kept holds every page handed to Windows, so the memory its raw
	// pointers refer to stays alive while the dialog may still use it.
	keptMu sync.Mutex
	kept   []*packedPage
)

// page describes one page of the dialog.
type page struct {
	instruction, content string
	flags                uint32
	buttons              []button
	defaultButton        int32
	errorIcon            bool
	kind                 uintptr
}

type button struct {
	id   int32
	text string
}

// closeOnly is the button row of the progress and result pages.
var closeOnly = []button{{idClose, "Close"}}

// packedPage is a TASKDIALOGCONFIG as Windows reads it, plus everything
// its raw pointers refer to.
type packedPage struct {
	buf  []byte
	refs [][]uint16
	btns []byte
}

// str keeps s alive in p and returns its address as UTF-16, or 0 for "".
func (p *packedPage) str(s string) uint64 {
	if s == "" {
		return 0
	}
	u, err := windows.UTF16FromString(s)
	if err != nil {
		u, _ = windows.UTF16FromString("?")
	}
	p.refs = append(p.refs, u)
	return uint64(uintptr(unsafe.Pointer(&u[0])))
}

// pack lays the page out as a TASKDIALOGCONFIG. The Windows headers
// declare it, and TASKDIALOG_BUTTON, with 1-byte packing, which a Go
// struct can't express, so the fields go in at their packed 64-bit
// offsets instead.
func (pg page) pack() *packedPage {
	p := &packedPage{buf: make([]byte, 160)}
	le := binary.LittleEndian
	flags := pg.flags

	le.PutUint32(p.buf[0:], 160)                           // cbSize
	le.PutUint64(p.buf[12:], uint64(win32.ModuleHandle())) // hInstance
	le.PutUint64(p.buf[28:], p.str(title))                 // pszWindowTitle
	switch {
	case pg.errorIcon:
		le.PutUint64(p.buf[36:], tdErrorIcon) // pszMainIcon
	case appIcon != 0:
		flags |= tdfUseHIconMain
		le.PutUint64(p.buf[36:], uint64(appIcon)) // hMainIcon
	}
	le.PutUint32(p.buf[20:], flags)                 // dwFlags
	le.PutUint64(p.buf[44:], p.str(pg.instruction)) // pszMainInstruction
	le.PutUint64(p.buf[52:], p.str(pg.content))     // pszContent
	if n := len(pg.buttons); n > 0 {
		p.btns = make([]byte, 12*n) // TASKDIALOG_BUTTON: int id, then the text pointer
		for i, b := range pg.buttons {
			le.PutUint32(p.btns[12*i:], uint32(b.id))
			le.PutUint64(p.btns[12*i+4:], p.str(b.text))
		}
		le.PutUint32(p.buf[60:], uint32(n))                                   // cButtons
		le.PutUint64(p.buf[64:], uint64(uintptr(unsafe.Pointer(&p.btns[0])))) // pButtons
	}
	le.PutUint32(p.buf[72:], uint32(pg.defaultButton))                                                                                  // nDefaultButton
	le.PutUint64(p.buf[132:], p.str(fmt.Sprintf("Setup %s - installs for your account only, no administrator rights needed", version))) // pszFooter
	le.PutUint64(p.buf[140:], uint64(dialogCallback))                                                                                   // pfCallback
	le.PutUint64(p.buf[148:], uint64(pg.kind))                                                                                          // lpCallbackData

	keptMu.Lock()
	kept = append(kept, p)
	keptMu.Unlock()
	return p
}

func (p *packedPage) addr() uintptr { return uintptr(unsafe.Pointer(&p.buf[0])) }

// runDialog shows Setup's window and returns the process exit code.
func runDialog(o options) int {
	runtime.LockOSThread()
	if err := procTaskDialogIndirect.Find(); err != nil {
		messageBox("Setup can't show its window on this version of Windows:\n\n" + err.Error())
		return 1
	}

	dialogOpts = o
	dialogCallback = syscall.NewCallback(dialogProc)
	// The same glyph as the tray icon, at the size Windows uses for large
	// icons at this display scaling.
	size := 32
	if n, _, _ := procGetSystemMetrics.Call(smCxIcon); n > 0 {
		size = int(n)
	}
	if icon, err := win32.NewIcon(keyicon.Render(size, true)); err == nil {
		appIcon = icon
		defer win32.DestroyIcon(icon)
	}

	firstContent = "UndeadKeys lets you type accented characters with AltGr - AltGr+E for é, AltGr+N for ñ - " +
		"and types ', ` and 6 straight away instead of waiting as dead keys, " +
		"while your keyboard layout is United States-International.\n\n" +
		"Already installed? Install/Update replaces it with the latest version.\n\n"
	shownCountdown, _ = setup.Countdown(0)
	first := page{
		instruction:   "Install or update UndeadKeys?",
		content:       firstContent + shownCountdown,
		flags:         tdfAllowDialogCancellation | tdfCallbackTimer,
		buttons:       []button{{idInstall, "Install/Update"}, {idUninstall, "Uninstall"}, {idClose, "Close"}},
		defaultButton: idInstall,
		kind:          pageChoose,
	}.pack()

	if hr, _, _ := procTaskDialogIndirect.Call(first.addr(), 0, 0, 0); hr != 0 {
		messageBox(fmt.Sprintf("Setup couldn't open its window (error 0x%08X).", hr))
		return 1
	}
	if failed.Load() {
		return 1
	}
	return 0
}

// dialogProc is the task dialog's callback; Windows calls it on the
// dialog's own thread. Which page is showing comes in as refData, so
// nothing here is shared with the worker goroutine.
func dialogProc(hwnd, msg, wParam, lParam, refData uintptr) uintptr {
	switch msg {
	case tdnNavigated:
		switch refData {
		case pageProgress:
			win32.SendMessage(hwnd, tdmSetProgressBarMarquee, 1, 0)
			win32.SendMessage(hwnd, tdmEnableButton, idClose, 0)
		case pageClosing:
			timerRestart = true
			shownCountdown, _ = setup.CloseCountdown(0)
		}
	case tdnTimer:
		// wParam is the time since the dialog opened, or since a timer
		// notification last returned 1 (which restarts it).
		switch refData {
		case pageChoose:
			// Run Install/Update once nobody has chosen in time.
			text, due := setup.Countdown(uint32(wParam))
			if due {
				start(hwnd, idInstall)
				break
			}
			updateCountdown(hwnd, firstContent, text)
		case pageClosing:
			if timerRestart {
				timerRestart = false
				return 1
			}
			text, due := setup.CloseCountdown(uint32(wParam))
			if due {
				win32.SendMessage(hwnd, tdmClickButton, idClose, 0)
				break
			}
			base, _ := closingContent.Load().(string)
			updateCountdown(hwnd, base, text)
		}
	case tdnButtonClicked:
		switch wParam {
		case idInstall, idUninstall:
			start(hwnd, int(wParam))
			return sFalse // keep the dialog open
		case idClose:
			if refData == pageProgress {
				return sFalse // Escape while files are being replaced
			}
		}
	}
	return sOK
}

// updateCountdown shows text as the countdown line under base, if it
// changed.
func updateCountdown(hwnd uintptr, base, text string) {
	if text == shownCountdown {
		return
	}
	shownCountdown = text
	setContent(hwnd, base+text)
}

// start switches to the progress page and runs the chosen action on a
// separate goroutine, so the dialog keeps painting meanwhile. The worker
// only talks to the dialog through SendMessage, which Windows hands to the
// dialog's thread.
func start(hwnd uintptr, choice int) {
	verb, action := "Installing", "install"
	if choice == idUninstall {
		verb, action = "Removing", "uninstall"
	}
	navigate(hwnd, page{
		instruction: verb + " UndeadKeys...",
		content:     "Getting started...",
		flags:       tdfShowMarqueeProgressBar,
		buttons:     closeOnly,
		kind:        pageProgress,
	})

	go func() {
		// The Startup shortcut is written through COM, which must be set up
		// and torn down on one OS thread.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		tag, err := run(action, dialogOpts, func(step string) { setContent(hwnd, step) })
		failed.Store(err != nil)
		pg := resultPage(choice, tag, err)
		if err == nil {
			// Done: count down and close. A failure stays open so the
			// error can be read.
			closingContent.Store(pg.content + "\n\n")
			text, _ := setup.CloseCountdown(0)
			pg.content += "\n\n" + text
			pg.flags |= tdfCallbackTimer
			pg.kind = pageClosing
		}
		navigate(hwnd, pg)
	}()
}

// resultPage is the last page: what happened, and what to do next.
func resultPage(choice int, tag string, err error) page {
	pg := page{flags: tdfAllowDialogCancellation, buttons: closeOnly, defaultButton: idClose, kind: pageDone}
	switch {
	case err != nil:
		pg.errorIcon = true
		pg.instruction = "Setup didn't finish"
		pg.content = err.Error()
	case choice == idInstall:
		if cfg, _ := config.Load(); !cfg.Autostart {
			pg.instruction = fmt.Sprintf("UndeadKeys %s is installed", tag)
			pg.content = "Autostart is off in its settings, so it wasn't started and won't start when you sign in. " +
				"To use it, open UndeadKeys.exe in %LOCALAPPDATA%\\UndeadKeys - or set autostart: true in its config.yaml and run Setup again."
			break
		}
		pg.instruction = fmt.Sprintf("UndeadKeys %s is running", tag)
		pg.content = "Look for the Á key icon in the notification area (it may be behind the ^ arrow). " +
			"Hold AltGr and press a letter to type its accented form.\n\n" +
			"Right-click the icon to pause it, change its settings or exit. It starts again whenever you sign in."
	default:
		pg.instruction = "UndeadKeys has been removed"
		pg.content = "Its program, Startup shortcut and settings are gone."
	}
	return pg
}

func navigate(hwnd uintptr, pg page) {
	win32.SendMessage(hwnd, tdmNavigatePage, 0, pg.pack().addr())
}

func setContent(hwnd uintptr, text string) {
	u, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	win32.SendMessage(hwnd, tdmSetElementText, tdeContent, uintptr(unsafe.Pointer(u)))
	runtime.KeepAlive(u)
}

func messageBox(text string) {
	t, _ := windows.UTF16PtrFromString(text)
	c, _ := windows.UTF16PtrFromString(title)
	windows.MessageBox(0, t, c, windows.MB_OK|windows.MB_ICONERROR)
}
