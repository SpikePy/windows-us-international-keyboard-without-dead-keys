//go:build windows

package tray

import (
	"windows-us-international-keyboard-without-dead-keys/internal/keyicon"
	"windows-us-international-keyboard-without-dead-keys/internal/win32"
)

var procGetSystemMetrics = modUser32.NewProc("GetSystemMetrics")

// smallIconMetric is SM_CXSMICON: the width Windows draws tray icons at,
// which grows with the display scaling (16 at 100%, 24 at 150%, ...).
const smallIconMetric = 49

// iconSize is the size to render the tray icon at: exactly what the
// taskbar will show, so Windows never has to rescale it. It needs the
// process to be DPI aware (see win32.EnableDPIAwareness) to see the real
// value rather than the 100% one.
func iconSize() int {
	if n, _, _ := procGetSystemMetrics.Call(smallIconMetric); n > 0 {
		return int(n)
	}
	return 16
}

// EnabledIcon returns keyicon's glyph - a keycap with an "Á" on it - in
// black, at the taskbar's icon size (interception active).
func EnabledIcon() (uintptr, error) {
	return win32.NewIcon(keyicon.Render(iconSize(), true))
}

// DisabledIcon returns the same glyph greyed out with a diagonal red
// strike across it (interception paused).
func DisabledIcon() (uintptr, error) {
	return win32.NewIcon(keyicon.Render(iconSize(), false))
}

// DestroyIconHandle frees an HICON returned by EnabledIcon/DisabledIcon.
// Safe to call on a zero handle.
func DestroyIconHandle(hIcon uintptr) { win32.DestroyIcon(hIcon) }
