// Package hook installs the low-level keyboard hook that turns AltGr
// combinations and dead keys into the characters they should produce.
//
// The decision of what to do with a key event (this file) is deliberately
// separate from the Win32 hook that feeds it (hook_windows.go): it is
// pure logic over a snapshot of the keyboard state, so it builds and is
// table-tested on any platform.
package hook

import "windows-us-international-keyboard-without-dead-keys/internal/keymap"

// Action is what should happen to one key event.
type Action int

const (
	// PassThrough lets the key reach the focused application unchanged.
	PassThrough Action = iota
	// Swallow consumes the key without typing anything - the key-up half
	// of a press whose key-down was already turned into a character, so
	// the application never sees the raw, unmapped key.
	Swallow
	// Type sends the character Decide returned, then consumes the key.
	Type
)

// State is the keyboard state a key event arrives in.
type State struct {
	// Enabled is false while the user has paused interception from the
	// tray icon; every key then passes through.
	Enabled bool
	// LayoutMatches reports whether the focused window's keyboard layout
	// is the one this tool is restricted to (always true when it is
	// restricted to none).
	LayoutMatches bool
	// AltGrShortcuts and UndeadKeys are the two features, each of which
	// the user can turn off in config.yaml.
	AltGrShortcuts bool
	UndeadKeys     bool
	// AltGrHeld and ShiftHeld are the modifiers held down right now.
	AltGrHeld bool
	ShiftHeld bool
}

// Decide reports what to do with a press (keyDown) or release of the key
// with virtual-key code vk, and the character to type for Action Type.
//
// AltGr wins over the undead keys where a key is in both tables: with
// AltGr held, the apostrophe types the acute accent it maps to there,
// and on its own it types a plain apostrophe.
func Decide(s State, vk uint32, keyDown bool) (Action, rune) {
	if !s.Enabled || !s.LayoutMatches {
		return PassThrough, 0
	}
	if s.AltGrShortcuts && s.AltGrHeld {
		if ch, ok := keymap.Lookup(keymap.AltGr, vk, s.ShiftHeld); ok {
			return handled(keyDown, ch)
		}
	}
	if s.UndeadKeys {
		if ch, ok := keymap.Lookup(keymap.Undead, vk, s.ShiftHeld); ok {
			return handled(keyDown, ch)
		}
	}
	return PassThrough, 0
}

// handled turns a mapped key into its character on the way down, and
// consumes the matching release so the application never sees half a
// keystroke.
func handled(keyDown bool, ch rune) (Action, rune) {
	if keyDown {
		return Type, ch
	}
	return Swallow, 0
}
