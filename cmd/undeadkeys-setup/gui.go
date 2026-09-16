//go:build windows

package main

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"windows-us-international-keyboard-without-dead-keys/internal/keyicon"
	"windows-us-international-keyboard-without-dead-keys/internal/setup"
	"windows-us-international-keyboard-without-dead-keys/internal/setupflow"
	"windows-us-international-keyboard-without-dead-keys/internal/win32"
)

// The window is plain Win32: a white content area with the icon, title
// and status, over a grey footer holding the buttons - the look of a
// Windows task dialog. The manifest embedded next to this file turns on
// the modern control styles and per-monitor DPI awareness.

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modGdi32    = windows.NewLazySystemDLL("gdi32.dll")
	modComctl32 = windows.NewLazySystemDLL("comctl32.dll")

	procSendMessageW               = modUser32.NewProc("SendMessageW")
	procSetWindowTextW             = modUser32.NewProc("SetWindowTextW")
	procEnableWindow               = modUser32.NewProc("EnableWindow")
	procShowWindow                 = modUser32.NewProc("ShowWindow")
	procUpdateWindow               = modUser32.NewProc("UpdateWindow")
	procSetWindowPos               = modUser32.NewProc("SetWindowPos")
	procMoveWindow                 = modUser32.NewProc("MoveWindow")
	procGetClientRect              = modUser32.NewProc("GetClientRect")
	procInvalidateRect             = modUser32.NewProc("InvalidateRect")
	procBeginPaint                 = modUser32.NewProc("BeginPaint")
	procEndPaint                   = modUser32.NewProc("EndPaint")
	procFillRect                   = modUser32.NewProc("FillRect")
	procGetSysColorBrush           = modUser32.NewProc("GetSysColorBrush")
	procGetSysColor                = modUser32.NewProc("GetSysColor")
	procDrawIconEx                 = modUser32.NewProc("DrawIconEx")
	procSetTimer                   = modUser32.NewProc("SetTimer")
	procKillTimer                  = modUser32.NewProc("KillTimer")
	procIsDialogMessageW           = modUser32.NewProc("IsDialogMessageW")
	procSetFocus                   = modUser32.NewProc("SetFocus")
	procGetDpiForWindow            = modUser32.NewProc("GetDpiForWindow")
	procGetDpiForSystem            = modUser32.NewProc("GetDpiForSystem")
	procAdjustWindowRectExForDpi   = modUser32.NewProc("AdjustWindowRectExForDpi")
	procSystemParametersInfoForDpi = modUser32.NewProc("SystemParametersInfoForDpi")
	procGetSystemMetricsForDpi     = modUser32.NewProc("GetSystemMetricsForDpi")
	procGetCursorPos               = modUser32.NewProc("GetCursorPos")
	procMonitorFromPoint           = modUser32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW            = modUser32.NewProc("GetMonitorInfoW")
	procSetTextColor               = modGdi32.NewProc("SetTextColor")
	procSetBkMode                  = modGdi32.NewProc("SetBkMode")
	procCreateFontIndirectW        = modGdi32.NewProc("CreateFontIndirectW")
	procCreateSolidBrush           = modGdi32.NewProc("CreateSolidBrush")
	procInitCommonControlsEx       = modComctl32.NewProc("InitCommonControlsEx")
)

const (
	className   = "UndeadKeysSetupWindow"
	windowTitle = "UndeadKeys Setup"

	// Control IDs. IDOK and IDCANCEL are what Enter and Escape send, so the
	// install button and the close button take those.
	idPrimary   = 1 // IDOK
	idClose     = 2 // IDCANCEL
	idUninstall = 3

	tickTimer = 1

	wsOverlapped   = 0x00000000
	wsCaption      = 0x00C00000
	wsSysMenu      = 0x00080000
	wsMinimizeBox  = 0x00020000
	wsChild        = 0x40000000
	wsVisible      = 0x10000000
	wsTabStop      = 0x00010000
	wsExControlPar = 0x00010000

	bsPushButton    = 0x0
	bsDefPushButton = 0x1
	ssNoPrefix      = 0x80
	pbsMarquee      = 0x08

	wmDestroy        = 0x0002
	wmPaint          = 0x000F
	wmClose          = 0x0010
	wmSetFont        = 0x0030
	wmSetIcon        = 0x0080
	wmKeyDown        = 0x0100
	wmSysKeyDown     = 0x0104
	wmCommand        = 0x0111
	wmTimer          = 0x0113
	wmCtlColorBtn    = 0x0135
	wmCtlColorStatic = 0x0138
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmRButtonDown    = 0x0204
	wmMouseWheel     = 0x020A
	wmNCLButtonDown  = 0x00A1
	wmPrintClient    = 0x0318
	wmDpiChanged     = 0x02E0
	dmGetDefID       = 0x0400 // WM_USER
	bmSetStyle       = 0x00F4
	pbmSetMarquee    = 0x0400 + 10

	dcHasDefID = 0x534B

	swHide = 0
	swShow = 5

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010

	colorWindow   = 5
	colorBtnFace  = 15
	colorGrayText = 17

	transparent = 1

	iconSmall = 0
	iconBig   = 1

	smCxIcon   = 11
	smCxSmIcon = 49

	spiGetNonClientMetrics = 0x0029

	iccProgressClass = 0x00000020

	monitorDefaultToNearest = 2
)

type rect struct{ Left, Top, Right, Bottom int32 }

type point struct{ X, Y int32 }

type paintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     rect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type logFont struct {
	Height, Width, Escapement, Orientation, Weight int32
	Italic, Underline, StrikeOut, CharSet          byte
	OutPrecision, ClipPrecision, Quality           byte
	PitchAndFamily                                 byte
	FaceName                                       [32]uint16
}

type nonClientMetrics struct {
	cbSize                       uint32
	iBorderWidth, iScrollWidth   int32
	iScrollHeight, iCaptionWidth int32
	iCaptionHeight               int32
	lfCaptionFont                logFont
	iSmCaptionWidth              int32
	iSmCaptionHeight             int32
	lfSmCaptionFont              logFont
	iMenuWidth, iMenuHeight      int32
	lfMenuFont, lfStatusFont     logFont
	lfMessageFont                logFont
	iPaddedBorderWidth           int32
}

type monitorInfo struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

type initCommonControlsEx struct {
	dwSize, dwICC uint32
}

// Layout, in pixels at 96 DPI (100% scaling); everything is scaled by the
// window's actual DPI.
const (
	clientW   = 440
	clientH   = 226
	margin    = 24
	iconPx    = 40
	footerH   = 56
	buttonH   = 30
	buttonW   = 104
	primaryW  = 118
	buttonGap = 8
)

// resources are the window's DPI-dependent GDI objects and icons.
type resources struct {
	bodyFont, titleFont            uintptr
	headerIcon, smallIcon, bigIcon uintptr
	footerBrush, lineBrush         uintptr
}

// free releases them all. Safe on zero handles.
func (r resources) free() {
	for _, h := range []uintptr{r.bodyFont, r.titleFont, r.footerBrush, r.lineBrush} {
		win32.DeleteObject(h)
	}
	for _, h := range []uintptr{r.headerIcon, r.smallIcon, r.bigIcon} {
		win32.DestroyIcon(h)
	}
}

// update is one message from the worker goroutine to the window.
type update struct {
	step    string
	done    bool
	version string
	err     error
}

type window struct {
	hwnd                             uintptr
	title, heading, status, progress uintptr
	primary, uninstallBtn, closeBtn  uintptr
	dpi                              uint32
	resources
	defaultID int

	model     setupflow.Model
	view      setupflow.View
	install   setup.InstallOptions
	uninstall setup.UninstallOptions
	startPt   point

	mu      sync.Mutex
	pending []update
}

// theWindow is the one setup window; the window procedure, a plain Win32
// callback, has no other way to reach it.
var theWindow *window

var wndProc = syscall.NewCallback(func(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if w := theWindow; w != nil {
		if r, handled := w.handle(hwnd, msg, wParam, lParam); handled {
			return r
		}
	}
	return win32.DefWindowProc(hwnd, msg, wParam, lParam)
})

// runWindow shows the setup window and returns once it is closed.
func runWindow(install setup.InstallOptions, uninstall setup.UninstallOptions) error {
	// Window, timers and messages belong to the thread that created them.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	icc := initCommonControlsEx{dwICC: iccProgressClass}
	icc.dwSize = uint32(unsafe.Sizeof(icc))
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))

	w := &window{
		install:   install,
		uninstall: uninstall,
		model:     setupflow.New(setup.IsInstalled(install.InstallDir)),
	}
	w.install.Progress = func(step string) { w.post(update{step: step}) }
	w.uninstall.Progress = w.install.Progress
	theWindow = w
	defer func() { theWindow = nil }()

	if err := win32.RegisterClass(className, wndProc, windowBackground()); err != nil {
		return err
	}
	if err := w.create(); err != nil {
		return err
	}
	defer func() { w.resources.free() }()

	procGetCursorPos.Call(uintptr(unsafe.Pointer(&w.startPt)))
	procSetTimer.Call(w.hwnd, tickTimer, 1000, 0)

	for {
		m, ok := win32.GetMessage()
		if !ok {
			return nil
		}
		w.noticeUser(m)
		if r, _, _ := procIsDialogMessageW.Call(w.hwnd, uintptr(unsafe.Pointer(&m))); r != 0 {
			continue
		}
		win32.Dispatch(m)
	}
}

// windowBackground is the brush the whole client area is cleared to: the
// white of a window. The footer is painted over it.
func windowBackground() syscall.Handle {
	r, _, _ := procGetSysColorBrush.Call(colorWindow)
	return syscall.Handle(r)
}

// create makes the window and its controls, sized for the DPI of the
// monitor under the cursor, and centers it on that monitor.
func (w *window) create() error {
	dpi, _, _ := procGetDpiForSystem.Call()
	w.dpi = uint32(dpi)
	if w.dpi == 0 {
		w.dpi = 96
	}

	style := uint32(wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox)
	hwnd, err := win32.CreateWindow(wsExControlPar, style, className, windowTitle,
		win32.CWUseDefault, win32.CWUseDefault, 100, 100)
	if err != nil {
		return err
	}
	w.hwnd = hwnd
	if d, _, _ := procGetDpiForWindow.Call(hwnd); d != 0 {
		w.dpi = uint32(d)
	}

	child := func(class, text string, style uint32, id int) uintptr {
		h, err := win32.CreateChild(class, text, wsChild|style, hwnd, id)
		if err != nil {
			return 0
		}
		return h
	}
	w.title = child("STATIC", "UndeadKeys", wsVisible|ssNoPrefix, 0)
	w.heading = child("STATIC", "", wsVisible|ssNoPrefix, 0)
	w.status = child("STATIC", "", wsVisible|ssNoPrefix, 0)
	w.progress = child("msctls_progress32", "", pbsMarquee, 0)
	w.uninstallBtn = child("BUTTON", "Uninstall", wsTabStop|bsPushButton, idUninstall)
	w.primary = child("BUTTON", "Install", wsVisible|wsTabStop|bsDefPushButton, idPrimary)
	w.closeBtn = child("BUTTON", "Close", wsVisible|wsTabStop|bsPushButton, idClose)
	if w.title == 0 || w.status == 0 || w.primary == 0 || w.closeBtn == 0 || w.uninstallBtn == 0 {
		return fmt.Errorf("creating the setup window's controls failed")
	}
	procSendMessageW.Call(w.progress, pbmSetMarquee, 1, 30)

	w.applyDPI(w.dpi)
	w.placeOnScreen()
	w.render()
	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)
	procSetFocus.Call(w.primary)
	return nil
}

// scale converts a 96-DPI length to the window's DPI.
func (w *window) scale(v int) int32 { return int32(v * int(w.dpi) / 96) }

// applyDPI (re)creates fonts and icons for dpi and lays the controls out.
// The old ones are freed only once nothing uses them any more.
func (w *window) applyDPI(dpi uint32) {
	w.dpi = dpi
	old := w.resources
	defer old.free()

	var ncm nonClientMetrics
	ncm.cbSize = uint32(unsafe.Sizeof(ncm))
	procSystemParametersInfoForDpi.Call(spiGetNonClientMetrics, uintptr(ncm.cbSize), uintptr(unsafe.Pointer(&ncm)), 0, uintptr(dpi))
	body := ncm.lfMessageFont
	if body.Height == 0 {
		body.Height = -w.scale(12)
		copy(body.FaceName[:], windows.StringToUTF16("Segoe UI"))
	}
	w.bodyFont, _, _ = procCreateFontIndirectW.Call(uintptr(unsafe.Pointer(&body)))
	title := body
	title.Height = body.Height * 16 / 9
	title.Weight = 600 // FW_SEMIBOLD: Segoe UI Semibold
	w.titleFont, _, _ = procCreateFontIndirectW.Call(uintptr(unsafe.Pointer(&title)))

	for _, h := range []uintptr{w.heading, w.status, w.primary, w.uninstallBtn, w.closeBtn} {
		procSendMessageW.Call(h, wmSetFont, w.bodyFont, 1)
	}
	procSendMessageW.Call(w.title, wmSetFont, w.titleFont, 1)

	w.headerIcon, _ = win32.NewIcon(keyicon.Render(int(w.scale(iconPx)), true))
	small, _, _ := procGetSystemMetricsForDpi.Call(smCxSmIcon, uintptr(dpi))
	big, _, _ := procGetSystemMetricsForDpi.Call(smCxIcon, uintptr(dpi))
	w.smallIcon, _ = win32.NewIcon(keyicon.Render(max(int(small), 16), true))
	w.bigIcon, _ = win32.NewIcon(keyicon.Render(max(int(big), 32), true))
	procSendMessageW.Call(w.hwnd, wmSetIcon, iconSmall, w.smallIcon)
	procSendMessageW.Call(w.hwnd, wmSetIcon, iconBig, w.bigIcon)

	// Windows 11 dialog footer colors.
	w.footerBrush, _, _ = procCreateSolidBrush.Call(rgb(243, 243, 243))
	w.lineBrush, _, _ = procCreateSolidBrush.Call(rgb(229, 229, 229))

	w.layout()
}

func rgb(r, g, b byte) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }

// layout sizes the window for its content and places every control.
func (w *window) layout() {
	wr := rect{Right: w.scale(clientW), Bottom: w.scale(clientH)}
	style := uintptr(wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox)
	procAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(&wr)), style, 0, wsExControlPar, uintptr(w.dpi))
	procSetWindowPos.Call(w.hwnd, 0, 0, 0, uintptr(wr.Right-wr.Left), uintptr(wr.Bottom-wr.Top), swpNoMove|swpNoZOrder|swpNoActivate)

	s := w.scale
	textX := s(margin + iconPx + 16)
	textW := s(clientW-margin) - textX
	move := func(h uintptr, x, y, width, height int32) {
		procMoveWindow.Call(h, uintptr(x), uintptr(y), uintptr(width), uintptr(height), 1)
	}
	move(w.title, textX, s(margin-2), textW, s(30))
	move(w.heading, textX, s(margin+30), textW, s(20))
	move(w.status, s(margin), s(margin+iconPx+22), s(clientW-2*margin), s(clientH-footerH-margin-iconPx-22-20))
	move(w.progress, s(margin), s(clientH-footerH-14), s(clientW-2*margin), s(4))

	by := s(clientH - footerH + (footerH-buttonH)/2)
	closeX := s(clientW - margin - buttonW)
	move(w.closeBtn, closeX, by, s(buttonW), s(buttonH))
	move(w.primary, closeX-s(buttonGap)-s(primaryW), by, s(primaryW), s(buttonH))
	move(w.uninstallBtn, s(margin), by, s(buttonW), s(buttonH))
	procInvalidateRect.Call(w.hwnd, 0, 1)
}

// placeOnScreen centers the window in the work area of the monitor under
// the mouse, where the user just double-clicked.
func (w *window) placeOnScreen() {
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	mon, _, _ := procMonitorFromPoint.Call(uintptr(uint32(pt.X))|uintptr(uint32(pt.Y))<<32, monitorDefaultToNearest)
	var mi monitorInfo
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	if r, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); r == 0 {
		return
	}
	wr := rect{Right: w.scale(clientW), Bottom: w.scale(clientH)}
	style := uintptr(wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox)
	procAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(&wr)), style, 0, wsExControlPar, uintptr(w.dpi))
	width, height := wr.Right-wr.Left, wr.Bottom-wr.Top
	x := mi.rcWork.Left + (mi.rcWork.Right-mi.rcWork.Left-width)/2
	y := mi.rcWork.Top + (mi.rcWork.Bottom-mi.rcWork.Top-height)/2
	procSetWindowPos.Call(w.hwnd, 0, uintptr(x), uintptr(y), uintptr(width), uintptr(height), swpNoZOrder|swpNoActivate)
}

// handle is the window procedure proper.
func (w *window) handle(hwnd uintptr, msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	if hwnd != w.hwnd {
		return 0, false
	}
	switch msg {
	case wmCommand:
		switch int(wParam & 0xFFFF) {
		case idPrimary:
			w.act(w.model.Choose(setupflow.Install))
		case idUninstall:
			w.act(w.model.Choose(setupflow.Uninstall))
		case idClose:
			w.close()
		}
		return 0, true

	case dmGetDefID:
		return uintptr(w.defaultID) | dcHasDefID<<16, true

	case wmTimer:
		if wParam == tickTimer {
			w.act(w.model.Tick())
		}
		return 0, true

	case win32.WMSetupUpdate:
		w.drain()
		return 0, true

	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		w.paint(hdc)
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0, true

	case wmPrintClient:
		// Themed buttons ask their parent to draw what is behind them.
		w.paint(wParam)
		return 0, true

	case wmCtlColorStatic:
		procSetBkMode.Call(wParam, transparent)
		switch {
		case lParam == w.heading:
			c, _, _ := procGetSysColor.Call(colorGrayText)
			procSetTextColor.Call(wParam, c)
		case lParam == w.status && w.view.StatusIsError:
			procSetTextColor.Call(wParam, rgb(196, 43, 28))
		}
		r, _, _ := procGetSysColorBrush.Call(colorWindow)
		return r, true

	case wmCtlColorBtn:
		return w.footerBrush, true

	case wmDpiChanged:
		// Resize for the new DPI, then move to where Windows suggests.
		suggested := (*rect)(unsafe.Pointer(lParam))
		w.applyDPI(uint32(wParam & 0xFFFF))
		procSetWindowPos.Call(hwnd, 0, uintptr(suggested.Left), uintptr(suggested.Top), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
		return 0, true

	case wmClose:
		w.close()
		return 0, true

	case wmDestroy:
		procKillTimer.Call(hwnd, tickTimer)
		win32.PostQuitMessage()
		return 0, true
	}
	return 0, false
}

// paint draws the header icon and the grey footer band.
func (w *window) paint(hdc uintptr) {
	var cr rect
	procGetClientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&cr)))
	footerTop := cr.Bottom - w.scale(footerH)
	footer := rect{Left: 0, Top: footerTop, Right: cr.Right, Bottom: cr.Bottom}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&footer)), w.footerBrush)
	line := rect{Left: 0, Top: footerTop, Right: cr.Right, Bottom: footerTop + max(1, w.scale(1))}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&line)), w.lineBrush)

	const diNormal = 3
	size := w.scale(iconPx)
	procDrawIconEx.Call(hdc, uintptr(w.scale(margin)), uintptr(w.scale(margin)), w.headerIcon, uintptr(size), uintptr(size), 0, 0, diNormal)
}

// noticeUser stops the countdowns as soon as someone uses the keyboard or
// mouse on the window - moving the mouse only counts once it has left
// where it was when the window opened.
func (w *window) noticeUser(m win32.Msg) {
	if w.model.Countdown == 0 && w.model.CloseIn == 0 {
		return
	}
	switch m.Message {
	case wmKeyDown, wmSysKeyDown, wmLButtonDown, wmRButtonDown, wmNCLButtonDown, wmMouseWheel:
	case wmMouseMove:
		if m.Pt.X == w.startPt.X && m.Pt.Y == w.startPt.Y {
			return
		}
	default:
		return
	}
	w.model = w.model.UserActive()
	w.render()
}

// act applies a model transition and carries out its command.
func (w *window) act(m setupflow.Model, c setupflow.Command) {
	w.model = m
	switch c {
	case setupflow.StartInstall:
		w.startWorker(func() update {
			version, err := setup.Install(w.install)
			return update{done: true, version: version, err: err}
		})
	case setupflow.StartUninstall:
		w.startWorker(func() update {
			return update{done: true, err: setup.Uninstall(w.uninstall)}
		})
	case setupflow.Close:
		w.close()
		return
	}
	w.render()
}

// startWorker runs job off the window's thread, so the window keeps
// painting, and posts its result back.
func (w *window) startWorker(job func() update) {
	go func() {
		// The Startup shortcut is written through COM, which must be set
		// up and torn down on one OS thread.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		w.post(job())
	}()
}

// post queues u for the window thread and wakes it up. Safe from any
// goroutine.
func (w *window) post(u update) {
	w.mu.Lock()
	w.pending = append(w.pending, u)
	w.mu.Unlock()
	win32.PostMessage(w.hwnd, win32.WMSetupUpdate, 0, 0)
}

// drain applies everything the worker posted.
func (w *window) drain() {
	w.mu.Lock()
	updates := w.pending
	w.pending = nil
	w.mu.Unlock()
	for _, u := range updates {
		if u.done {
			w.model = w.model.Finished(u.version, u.err, setup.IsInstalled(w.install.InstallDir))
		} else {
			w.model = w.model.Progress(u.step)
		}
	}
	w.render()
}

func (w *window) close() {
	if w.model.CanClose() {
		win32.DestroyWindow(w.hwnd)
	}
}

// render makes the controls match the model's view.
func (w *window) render() {
	prev := w.view
	v := w.model.View()
	w.view = v

	setText(w.heading, v.Heading)
	setText(w.status, v.Status)
	setText(w.primary, v.Primary)
	setText(w.closeBtn, v.Close)
	enable(w.primary, v.PrimaryEnabled)
	enable(w.uninstallBtn, v.UninstallEnabled)
	enable(w.closeBtn, v.CloseEnabled)
	show(w.uninstallBtn, v.UninstallVisible)
	show(w.progress, v.Busy)
	if v.StatusIsError != prev.StatusIsError {
		procInvalidateRect.Call(w.status, 0, 1)
	}

	defaultID, defaultBtn, otherBtn := idPrimary, w.primary, w.closeBtn
	if v.CloseIsDefault {
		defaultID, defaultBtn, otherBtn = idClose, w.closeBtn, w.primary
	}
	if defaultID != w.defaultID {
		w.defaultID = defaultID
		procSendMessageW.Call(defaultBtn, bmSetStyle, bsDefPushButton, 1)
		procSendMessageW.Call(otherBtn, bmSetStyle, bsPushButton, 1)
		if v.CloseIsDefault {
			procSetFocus.Call(w.closeBtn)
		}
	}
}

func setText(h uintptr, s string) {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		return
	}
	procSetWindowTextW.Call(h, uintptr(unsafe.Pointer(p)))
}

func enable(h uintptr, on bool) {
	v := uintptr(0)
	if on {
		v = 1
	}
	procEnableWindow.Call(h, v)
}

func show(h uintptr, on bool) {
	cmd := uintptr(swHide)
	if on {
		cmd = swShow
	}
	procShowWindow.Call(h, cmd)
}
