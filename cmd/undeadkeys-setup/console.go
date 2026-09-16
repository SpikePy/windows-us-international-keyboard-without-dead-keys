//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

var (
	modKernel32       = windows.NewLazySystemDLL("kernel32.dll")
	procAttachConsole = modKernel32.NewProc("AttachConsole")
)

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

// showError shows err in a message box, for runs with nowhere to print.
func showError(err error) {
	text, _ := windows.UTF16PtrFromString(err.Error())
	caption, _ := windows.UTF16PtrFromString("UndeadKeys Setup")
	const mbIconError = 0x10
	windows.MessageBox(0, text, caption, mbIconError)
}
