//go:build windows

// Command undeadkeys-setup installs, updates and uninstalls UndeadKeys.
// Opened normally it shows a small Windows dialog with Install/Update,
// Uninstall and Close (see dialog.go), and runs Install/Update on its own
// if nothing is chosen within a few seconds; -mode runs an action directly
// for scripts, printing its steps to the console it was started from.
//
// Everything it touches is inside the current user's profile
// (%LOCALAPPDATA%\UndeadKeys and the user's own Startup folder), so it
// never needs administrator rights.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"

	"windows-us-international-keyboard-without-dead-keys/internal/setup"
)

// version is stamped in at build time via -ldflags "-X main.version=...";
// left as "dev" for local/manual builds.
var version = "dev"

// options are the flags that shape what an action does.
type options struct {
	installDir                       string
	noLaunch, noAutostart, keepFiles bool
}

func main() {
	mode := flag.String("mode", "", "run without the dialog: install or uninstall")
	installDir := flag.String("install-dir", "", "directory to install into/remove from (default: %LOCALAPPDATA%\\UndeadKeys)")
	noLaunch := flag.Bool("no-launch", false, "install/update without starting it now (install only)")
	noAutostart := flag.Bool("no-autostart", false, "leave the Startup shortcut as it is instead of applying config.yaml's autostart setting (install only)")
	keepFiles := flag.Bool("keep-files", false, "remove the shortcut and stop the program, but don't delete the installed files (uninstall only)")
	flag.Parse()

	o := options{*installDir, *noLaunch, *noAutostart, *keepFiles}
	if *mode == "" {
		os.Exit(runDialog(o))
	}

	attachConsole()
	tag, err := run(strings.ToLower(*mode), o, func(step string) { fmt.Println(step) })
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if tag != "" {
		fmt.Printf("UndeadKeys %s is installed.\n", tag)
	} else {
		fmt.Println("UndeadKeys has been removed.")
	}
}

// run performs one Setup action, reporting each step to progress. For an
// install it returns the release tag it put in place.
func run(action string, o options, progress func(string)) (string, error) {
	switch action {
	case "install":
		return setup.Install(setup.InstallOptions{
			InstallDir:  o.installDir,
			NoLaunch:    o.noLaunch,
			NoAutostart: o.noAutostart,
			Progress:    progress,
		})
	case "uninstall":
		return "", setup.Uninstall(setup.UninstallOptions{
			InstallDir: o.installDir,
			KeepFiles:  o.keepFiles,
			Progress:   progress,
		})
	}
	return "", fmt.Errorf("unknown -mode %q (want install or uninstall)", action)
}

var procAttachConsole = windows.NewLazySystemDLL("kernel32.dll").NewProc("AttachConsole")

// attachConsole makes -mode's output visible. Setup is a GUI program, so
// Windows gives it no console of its own; unless its output is already
// going to a pipe or file, it borrows the console of whatever started it.
func attachConsole() {
	if h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && h != 0 && h != windows.InvalidHandle {
		return
	}
	const attachParentProcess = uintptr(^uint32(0))
	if r, _, _ := procAttachConsole.Call(attachParentProcess); r == 0 {
		return
	}
	if out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout, os.Stderr = out, out
	}
}
