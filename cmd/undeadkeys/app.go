//go:build windows

package main

import (
	"fmt"

	"windows-us-international-keyboard-without-dead-keys/internal/config"
	"windows-us-international-keyboard-without-dead-keys/internal/hook"
	"windows-us-international-keyboard-without-dead-keys/internal/tray"
	"windows-us-international-keyboard-without-dead-keys/internal/win32"
)

const (
	menuIDEnable    = 1
	menuIDDisable   = 2
	menuIDConfigure = 3
	menuIDExit      = 4
)

// app ties the keyboard hook to the tray icon. All its methods run on the
// main thread: either directly from run's message loop, or from the tray
// window's click callbacks, which that same loop dispatches.
type app struct {
	logf func(format string, args ...any)
	opts hook.Options

	hook     *hook.Hook
	trayHwnd uintptr
	trayIcon uintptr
}

// start installs the keyboard hook and shows the tray icon.
func (a *app) start() error {
	var err error
	if a.trayHwnd, err = tray.NewWindow(a.toggle, a.showMenu); err != nil {
		return fmt.Errorf("creating tray window: %w", err)
	}
	if a.hook, err = hook.Install(a.opts); err != nil {
		return err
	}
	a.applyTrayIcon()
	return nil
}

// run pumps messages until Exit is chosen from the tray menu (or the
// thread otherwise receives WM_QUIT). The hook callback is delivered on
// this thread too, in between messages.
func (a *app) run() {
	for {
		m, ok := win32.GetMessage()
		if !ok {
			return
		}
		win32.Dispatch(m)
	}
}

// toggle is the tray icon's left-click action.
func (a *app) toggle() { a.setEnabled(!a.hook.Enabled()) }

func (a *app) setEnabled(v bool) {
	if a.hook.Enabled() == v {
		return
	}
	a.hook.SetEnabled(v)
	if v {
		a.logf("Enabled via tray")
	} else {
		a.logf("Disabled via tray")
	}
	a.applyTrayIcon()
}

// showMenu is the tray icon's right-click action.
func (a *app) showMenu() {
	enabled := a.hook.Enabled()
	id := tray.ShowMenu(a.trayHwnd, []tray.MenuItem{
		{ID: menuIDEnable, Label: "Enable", Checked: enabled},
		{ID: menuIDDisable, Label: "Disable", Checked: !enabled},
		{},
		{ID: menuIDConfigure, Label: "Configure"},
		{},
		{ID: menuIDExit, Label: "Exit"},
	})
	switch id {
	case menuIDEnable:
		a.setEnabled(true)
	case menuIDDisable:
		a.setEnabled(false)
	case menuIDConfigure:
		a.openConfigFile()
	case menuIDExit:
		win32.PostQuitMessage()
	}
}

// openConfigFile opens config.yaml in whatever application Windows has
// associated with .yaml files. Changes take effect the next time the
// program starts.
func (a *app) openConfigFile() {
	path, err := config.Path()
	if err != nil {
		a.logf("EXCEPTION resolving config.yaml path: %v", err)
		return
	}
	if _, err := config.Load(); err != nil {
		a.logf("WARNING creating config.yaml before opening it: %v", err)
	}
	if err := win32.OpenFile(path); err != nil {
		a.logf("EXCEPTION opening config.yaml: %v", err)
	}
}

// applyTrayIcon shows (or updates) the tray icon and tooltip for the
// current enabled state.
func (a *app) applyTrayIcon() {
	build, state := tray.EnabledIcon, "intercepting"
	if !a.hook.Enabled() {
		build, state = tray.DisabledIcon, "paused"
	}
	newIcon, err := build()
	if err != nil {
		a.logf("EXCEPTION building tray icon: %v", err)
		return
	}
	tooltip := fmt.Sprintf("UndeadKeys %s - %s", version, state)
	if err := tray.SetIcon(a.trayHwnd, newIcon, tooltip); err != nil {
		a.logf("EXCEPTION showing tray icon: %v", err)
	}
	old := a.trayIcon
	a.trayIcon = newIcon
	tray.DestroyIconHandle(old)
}

// cleanup undoes everything start set up; it runs as the program exits,
// whatever state it's in.
func (a *app) cleanup() {
	a.hook.Uninstall()
	tray.RemoveIcon(a.trayHwnd)
	tray.DestroyIconHandle(a.trayIcon)
	tray.DestroyWindow(a.trayHwnd)
}
