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
// unaffected. Which layout (if any) it restricts itself to, and both
// features above, are settings in config.yaml in
// %LOCALAPPDATA%\UndeadKeys, each overridable per-run with a flag.
//
// A tray icon (keycap glyph = intercepting, the same glyph greyed out
// with a diagonal red strike = paused) lets the user pause and resume
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
	"flag"
	"runtime"

	"windows-us-international-keyboard-without-dead-keys/internal/applog"
	"windows-us-international-keyboard-without-dead-keys/internal/config"
	"windows-us-international-keyboard-without-dead-keys/internal/hook"
	"windows-us-international-keyboard-without-dead-keys/internal/singleinstance"
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
	enableLogging := flag.Bool("enable-logging", false, "write diagnostics to UndeadKeys.log next to the exe")
	flag.Parse()

	logf, logPath := applog.New("UndeadKeys.log", *enableLogging)
	if cfgErr != nil {
		logf("WARNING loading config.yaml (falling back to defaults): %v", cfgErr)
	}
	if !config.ValidLayoutID(*restrictToLayout) {
		logf("WARNING -restrict-to-layout=%q is not a layout ID; using %s", *restrictToLayout, config.DefaultRestrictToLayout)
		*restrictToLayout = config.DefaultRestrictToLayout
	}

	release, alreadyRunning, err := singleinstance.Acquire(`UndeadKeys_SingleInstance`)
	if err != nil {
		logf("EXCEPTION acquiring single-instance mutex: %v", err)
		return
	}
	if alreadyRunning {
		logf("Another instance is already running - exiting.")
		return
	}
	defer release()

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

	if err := a.start(); err != nil {
		logf("EXCEPTION %v", err)
		return
	}
	logf("Entering message loop (altgr=%v, undead=%v, layout=%q, enabled=%v)",
		*altGrShortcuts, *undeadKeys, *restrictToLayout, *startEnabled)
	a.run()
	logf("Message loop returned (Exit or unexpected shutdown).")
}
