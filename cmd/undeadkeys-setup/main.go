//go:build windows

// Command undeadkeys-setup is the single entry point for installing,
// updating, and uninstalling UndeadKeys.exe. Run it with no arguments
// (e.g. by double-clicking Setup_UndeadKeys.exe) and it opens a small
// window offering Install/Update and Uninstall - installing on its own if
// nobody touches the window within a few seconds (see
// internal/setupflow). Pass -mode to skip the window for scripted use;
// progress then goes to the console it was started from.
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

	"windows-us-international-keyboard-without-dead-keys/internal/setup"
)

func main() {
	mode := flag.String("mode", "", "skip the window and run this action directly: install or uninstall")
	installDir := flag.String("install-dir", "", "directory to install into/remove from (default: %LOCALAPPDATA%\\UndeadKeys)")
	githubToken := flag.String("github-token", "", "optional GitHub token, to avoid the unauthenticated API rate limit (install only)")
	noLaunch := flag.Bool("no-launch", false, "install/update and register autostart, but don't start it now (install only)")
	noAutostart := flag.Bool("no-autostart", false, "don't register (or update) the autostart shortcut (install only)")
	keepFiles := flag.Bool("keep-files", false, "remove autostart and stop the process, but don't delete the installed files (uninstall only)")

	hasConsole := useParentConsole()
	flag.CommandLine.SetOutput(os.Stderr)
	flag.Parse()

	install := setup.InstallOptions{
		InstallDir:  *installDir,
		GitHubToken: *githubToken,
		NoLaunch:    *noLaunch,
		NoAutostart: *noAutostart,
	}
	uninstall := setup.UninstallOptions{
		InstallDir: *installDir,
		KeepFiles:  *keepFiles,
	}

	if *mode == "" {
		if err := runWindow(install, uninstall); err != nil {
			fail(hasConsole, err)
		}
		return
	}

	printStep := func(step string) { fmt.Println(step) }
	install.Progress, uninstall.Progress = printStep, printStep

	var err error
	switch strings.ToLower(*mode) {
	case "install":
		var version string
		if version, err = setup.Install(install); err == nil {
			fmt.Printf("UndeadKeys %s is installed.\n", version)
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

// fail reports err where the user can see it - the console if there is
// one, otherwise a message box - and exits with status 1.
func fail(hasConsole bool, err error) {
	if hasConsole {
		fmt.Fprintln(os.Stderr, "error:", err)
	} else {
		showError(err)
	}
	os.Exit(1)
}
