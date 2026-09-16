//go:build windows

// Command undeadkeys is a background program for Windows that adds AltGr
// (Right Alt) shortcuts for accented characters and cancels the dead-key
// wait, without installing anything system-wide and without administrator
// rights. Hold Right Alt and press a mapped key to get the accented
// character immediately; press the apostrophe, backtick or 6 key and get
// that plain character immediately, instead of the layout's wait for a
// second keystroke to combine into an accent.
//
// By default it only acts while the focused window's keyboard layout is
// Windows' built-in "United States-International" - under German, plain
// US or anything else it is a no-op, so normal typing there is completely
// unaffected. Which layout (if any) it restricts itself to, both features
// above, and whether it starts with Windows are settings in config.yaml in
// %LOCALAPPDATA%\UndeadKeys, each overridable per-run with a flag.
//
// A tray icon (keycap with an "Á" = enabled, the same glyph greyed out
// with a diagonal red strike = disabled) lets the user pause and resume
// without stopping the process: left-click toggles it, right-click opens
// an Enable/Disable/Configure/Exit menu. Configure opens config.yaml in
// whatever application Windows has associated with .yaml files.
//
// The program does not exit on its own. To stop it: the tray menu's Exit,
// Task Manager, taskkill, or the setup program (which does this
// automatically when updating).
//
// Logging is OFF by default. Pass -enable-logging to write diagnostics to
// UndeadKeys.log next to the exe, for troubleshooting only.
//
// Built with -ldflags "-H=windowsgui" so it never shows a console window;
// everything below runs inside a top-level recover() that never lets a
// panic surface as a crash dialog - diagnostics go only to the log file.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sys/windows"

	"windows-us-international-keyboard-without-dead-keys/internal/config"
	"windows-us-international-keyboard-without-dead-keys/internal/hook"
	"windows-us-international-keyboard-without-dead-keys/internal/shortcut"
	"windows-us-international-keyboard-without-dead-keys/internal/tray"
	"windows-us-international-keyboard-without-dead-keys/internal/win32"
)

// version is stamped in at build time via -ldflags "-X main.version=...";
// left as "dev" for local/manual builds.
var version = "dev"

func main() {
	// A low-level keyboard hook and the message queue that feeds it are
	// bound to the OS thread that created them; the Go runtime must never
	// migrate this goroutine to a different one mid-run.
	runtime.LockOSThread()

	cfg, cfgErr := config.Load()

	startEnabled := flag.Bool("start-enabled", cfg.StartEnabled, "intercept keys from the start, rather than waiting to be enabled from the tray (overrides config.yaml)")
	altGrShortcuts := flag.Bool("altgr-shortcuts", cfg.AltGrShortcuts, "type accented characters on AltGr combinations (overrides config.yaml)")
	undeadKeys := flag.Bool("undead-keys", cfg.UndeadKeys, "type ' ` and 6 immediately instead of waiting as dead keys (overrides config.yaml)")
	restrictToLayout := flag.String("restrict-to-layout", cfg.RestrictToLayout, "only intercept under this 8-hex-digit keyboard layout ID, or \"\" for every layout (overrides config.yaml)")
	autostartOn := flag.Bool("autostart", cfg.Autostart, "start at sign-in through a Startup-folder shortcut; the installed copy applies this every time it starts (overrides config.yaml)")
	enableLogging := flag.Bool("enable-logging", false, "write diagnostics to UndeadKeys.log next to the exe")
	flag.Parse()

	logf, logPath := newLog("UndeadKeys.log", *enableLogging)
	if cfgErr != nil {
		logf("WARNING loading config.yaml (falling back to defaults): %v", cfgErr)
	}
	if !config.ValidLayoutID(*restrictToLayout) {
		logf("WARNING -restrict-to-layout=%q is not a layout ID; using %s", *restrictToLayout, config.DefaultRestrictToLayout)
		*restrictToLayout = config.DefaultRestrictToLayout
	}

	release, alreadyRunning, err := acquireSingleInstance(`UndeadKeys_SingleInstance`)
	if err != nil {
		logf("EXCEPTION acquiring single-instance mutex: %v", err)
		return
	}
	if alreadyRunning {
		logf("Another instance is already running - exiting.")
		return
	}
	defer release()

	syncAutostart(*autostartOn, logf)

	a := &app{logf: logf, opts: hook.Options{
		AltGrShortcuts: *altGrShortcuts,
		UndeadKeys:     *undeadKeys,
		LayoutID:       *restrictToLayout,
		StartEnabled:   *startEnabled,
	}}
	defer func() {
		a.cleanup()
		logf("Cleanup done. Log at: %s", logPath)
	}()
	defer func() {
		if r := recover(); r != nil {
			logf("PANIC: %v", r)
		}
	}()

	// Before any window or icon exists, so the tray icon is drawn at the
	// size the taskbar really shows.
	win32.EnableDPIAwareness()

	if err := a.start(); err != nil {
		logf("EXCEPTION %v", err)
		return
	}
	logf("Entering message loop (altgr=%v, undead=%v, layout=%q, enabled=%v, autostart=%v)",
		*altGrShortcuts, *undeadKeys, *restrictToLayout, *startEnabled, *autostartOn)
	a.run()
	logf("Message loop returned (Exit or unexpected shutdown).")
}

// syncAutostart keeps the Startup shortcut in line with the autostart
// setting, so an edited config.yaml takes effect on the next start without
// re-running Setup. Only the installed copy - the one next to config.yaml -
// does this, so running a build from anywhere else never repoints
// autostart at it.
func syncAutostart(enabled bool, logf func(format string, args ...any)) {
	exe, err := os.Executable()
	if err != nil {
		logf("WARNING locating own exe, leaving autostart alone: %v", err)
		return
	}
	cfgPath, err := config.Path()
	if err != nil {
		logf("WARNING locating config.yaml, leaving autostart alone: %v", err)
		return
	}
	if !strings.EqualFold(filepath.Dir(exe), filepath.Dir(cfgPath)) {
		logf("Not the installed copy (%s) - leaving autostart alone", exe)
		return
	}
	if err := shortcut.SyncAutostart(enabled, exe); err != nil {
		logf("EXCEPTION updating autostart: %v", err)
		return
	}
	logf("Startup shortcut matches autostart=%t", enabled)
}

// acquireSingleInstance tries to become the sole running copy, by holding
// the named mutex name (session-local, which is what "don't run twice for
// this user" needs). If another copy holds it, alreadyRunning is true.
// Otherwise release must be called before exiting; a process that dies
// without calling it also gives the mutex up.
func acquireSingleInstance(name string) (release func(), alreadyRunning bool, err error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, err
	}
	h, err := windows.CreateMutex(nil, true, namePtr)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		// CreateMutex still hands back a valid handle to the existing
		// mutex in this case; it's ours to close.
		windows.CloseHandle(h)
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return func() {
		windows.ReleaseMutex(h)
		windows.CloseHandle(h)
	}, false, nil
}

// newLog returns a printf-style logger that appends timestamped lines to
// fileName next to the running exe, plus that file's path. When disabled
// it does nothing, and a failed write is ignored: logging must never be
// the reason the program misbehaves.
func newLog(fileName string, enabled bool) (logf func(format string, args ...any), path string) {
	if exe, err := os.Executable(); err == nil {
		path = filepath.Join(filepath.Dir(exe), fileName)
	}
	logf = func(format string, args ...any) {
		if !enabled || path == "" {
			return
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return
		}
		defer f.Close()
		fmt.Fprintf(f, "%s  %s\n", time.Now().Format("2006-01-02 15:04:05.000"), fmt.Sprintf(format, args...))
	}
	return logf, path
}

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
	build, state := tray.EnabledIcon, "enabled"
	if !a.hook.Enabled() {
		build, state = tray.DisabledIcon, "disabled"
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
