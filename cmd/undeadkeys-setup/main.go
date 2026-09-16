//go:build windows

// Command undeadkeys-setup is the single entry point for installing,
// updating, and uninstalling UndeadKeys.exe. Run it with no arguments
// (e.g. by double-clicking Setup_UndeadKeys.exe) and it opens a small
// window offering Install/Update and Uninstall - installing on its own if
// nobody touches the window within a few seconds. Pass -mode to skip the
// window for scripted use; progress then goes to the console it was
// started from.
//
// Everything it touches is inside the current user's profile
// (%LOCALAPPDATA%\UndeadKeys and the user's own Startup folder), so it
// never needs administrator rights.
//
// Built with -ldflags "-H=windowsgui", so double-clicking it never flashes
// a console window.
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

func main() {
	mode := flag.String("mode", "", "skip the window and run this action directly: install or uninstall")
	installDir := flag.String("install-dir", "", "directory to install into/remove from (default: %LOCALAPPDATA%\\UndeadKeys)")
	noLaunch := flag.Bool("no-launch", false, "install/update without starting it now (install only)")
	noAutostart := flag.Bool("no-autostart", false, "leave the Startup shortcut as it is instead of applying config.yaml's autostart setting (install only)")
	keepFiles := flag.Bool("keep-files", false, "remove autostart and stop the process, but don't delete the installed files (uninstall only)")

	hasConsole := useParentConsole()
	flag.CommandLine.SetOutput(os.Stderr)
	flag.Parse()

	install := setup.InstallOptions{InstallDir: *installDir, NoLaunch: *noLaunch, NoAutostart: *noAutostart}
	uninstall := setup.UninstallOptions{InstallDir: *installDir, KeepFiles: *keepFiles}

	if *mode == "" {
		if err := setup.RunWindow(install, uninstall, version); err != nil {
			fail(hasConsole, err)
		}
		return
	}

	printStep := func(step string) { fmt.Println(step) }
	install.Progress, uninstall.Progress = printStep, printStep

	var err error
	switch strings.ToLower(*mode) {
	case "install":
		var tag string
		if tag, err = setup.Install(install); err == nil {
			fmt.Printf("UndeadKeys %s is installed.\n", tag)
		}
	case "uninstall":
		if err = setup.Uninstall(uninstall); err == nil {
			fmt.Println("UndeadKeys has been removed.")
		}
	default:
		err = fmt.Errorf("unknown -mode %q (want install or uninstall)", *mode)
	}
	if err != nil {
		fail(hasConsole, err)
	}
}

var procAttachConsole = windows.NewLazySystemDLL("kernel32.dll").NewProc("AttachConsole")

// attachParentProcess is ATTACH_PARENT_PROCESS, (DWORD)-1.
const attachParentProcess = ^uintptr(0)

// useParentConsole makes fmt output visible for -mode runs. A GUI-subsystem
// program gets no console of its own: if its output is already redirected
// (a pipe, a file) that is used as is; otherwise it borrows the console of
// whatever started it, if any. It reports whether output now goes
// somewhere.
func useParentConsole() bool {
	if h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && h != 0 && h != windows.InvalidHandle {
		return true
	}
	if r, _, _ := procAttachConsole.Call(attachParentProcess); r == 0 {
		return false
	}
	out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	os.Stdout, os.Stderr = out, out
	return true
}

// fail reports err where the user can see it - the console if there is
// one, otherwise a message box - and exits with status 1.
func fail(hasConsole bool, err error) {
	if hasConsole {
		fmt.Fprintln(os.Stderr, "error:", err)
	} else {
		text, _ := windows.UTF16PtrFromString(err.Error())
		caption, _ := windows.UTF16PtrFromString("UndeadKeys Setup")
		const mbIconError = 0x10
		windows.MessageBox(0, text, caption, mbIconError)
	}
	os.Exit(1)
}
