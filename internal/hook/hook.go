// Package hook installs the low-level keyboard hook that turns AltGr
// combinations and dead keys into the characters they should produce.
//
// This file is the part with no OS dependency, built and table-tested on
// any platform: the two character tables and the decision of what to do
// with one key event. hook_windows.go is the Win32 hook that feeds it.
package hook

// Virtual-key codes. Letters and digits are their own ASCII code, so only
// the US OEM punctuation keys need naming; these values are fixed by
// Win32 and never change.
const (
	vkOem1     = 0xBA // ;
	vkOemPlus  = 0xBB // =
	vkOemComma = 0xBC // ,
	vkOemMinus = 0xBD // -
	vkOem2     = 0xBF // /
	vkOem3     = 0xC0 // ` (grave/backtick)
	vkOem4     = 0xDB // [
	vkOem5     = 0xDC // backslash
	vkOem6     = 0xDD // ]
	vkOem7     = 0xDE // '

	vkCapital = 0x14 // Caps Lock
)

// Chars is what one key produces: Plain on its own, Shifted while Shift
// is held.
type Chars struct {
	Plain   rune
	Shifted rune
}

// AltGr maps a virtual-key code to the characters it produces while
// AltGr (Right Alt) is held. This is the "US International - AltGr - no
// dead keys" character set: every entry is typed immediately, with
// nothing waiting for a second keystroke.
var AltGr = map[uint32]Chars{
	'1':        {'¡', '¹'},
	'2':        {'²', '²'},
	'3':        {'³', '³'},
	'4':        {'¤', '£'},
	'5':        {'€', '€'},
	'6':        {'¼', '¼'},
	'7':        {'½', '½'},
	'8':        {'¾', '¾'},
	'9':        {'‘', '‘'},
	'0':        {'’', '’'},
	vkOemMinus: {'¥', '¥'},
	vkOemPlus:  {'×', '÷'},
	'Q':        {'ä', 'Ä'},
	'W':        {'å', 'Å'},
	'E':        {'é', 'É'},
	'R':        {'®', '®'},
	'T':        {'þ', 'Þ'},
	'Y':        {'ü', 'Ü'},
	'U':        {'ú', 'Ú'},
	'I':        {'í', 'Í'},
	'O':        {'ó', 'Ó'},
	'P':        {'ö', 'Ö'},
	vkOem4:     {'«', '«'},
	vkOem6:     {'»', '»'},
	'A':        {'á', 'Á'},
	'S':        {'ß', '§'},
	'D':        {'ð', 'Ð'},
	'L':        {'ø', 'Ø'},
	vkOem1:     {'¶', '°'},
	vkOem7:     {'´', '¨'},
	vkOem5:     {'¬', '¦'},
	'Z':        {'æ', 'Æ'},
	'C':        {'©', '¢'},
	'N':        {'ñ', 'Ñ'},
	'M':        {'µ', 'µ'},
	vkOemComma: {'ç', 'Ç'},
	vkOem2:     {'¿', '¿'},
}

// Undead maps the keys that the "United States-International" layout
// treats as dead keys - they normally swallow the keystroke and wait for
// a second one to combine into an accented letter - to the plain
// character they should produce instead, immediately. They apply alone
// and with Shift (or AltGr), but not while a shortcut modifier is held:
// Ctrl+`, Alt+' or Win+6 are shortcuts the application must see as keys.
var Undead = map[uint32]Chars{
	vkOem7: {'\'', '"'},
	vkOem3: {'`', '~'},
	'6':    {'6', '^'},
}

// Lookup returns the character table m produces for the virtual-key code
// vk with or without Shift, and whether vk is mapped at all.
func Lookup(m map[uint32]Chars, vk uint32, shift bool) (rune, bool) {
	c, ok := m[vk]
	if !ok {
		return 0, false
	}
	if shift {
		return c.Shifted, true
	}
	return c.Plain, true
}

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
	// AsShift sends a Shift press or release - matching the key's own -
	// instead of the key, which is consumed. It is how Caps Lock becomes
	// a second Shift key that never locks.
	AsShift
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
	// AltGrShortcuts, UndeadKeys and CapsLockAsShift are the features,
	// each of which the user can switch in config.yaml.
	AltGrShortcuts  bool
	UndeadKeys      bool
	CapsLockAsShift bool
	// AltGrHeld and ShiftHeld are the modifiers held down right now.
	AltGrHeld bool
	ShiftHeld bool
	// ShortcutHeld is true while Ctrl, the left Alt or a Windows key is
	// held - not counting the Ctrl that Windows reports along with AltGr.
	ShortcutHeld bool
	// PressHandled reports, for a release, whether this key's press was
	// handled (typed or turned into Shift) - only then is the release
	// handled too.
	PressHandled bool
}

// Decide reports what to do with a press (keyDown) or release of the key
// with virtual-key code vk, and the character to type for Action Type.
//
// A release follows its press, whatever changed in between: it is
// handled exactly when the press was, so letting go of a modifier first,
// pausing from the tray or switching layouts mid-press never leaves the
// application with half a keystroke - or Shift stuck down.
//
// AltGr wins over the undead keys where a key is in both tables: with
// AltGr held, the apostrophe types the acute accent it maps to there,
// and on its own it types a plain apostrophe.
func Decide(s State, vk uint32, keyDown bool) (Action, rune) {
	if !keyDown {
		switch {
		case !s.PressHandled:
			return PassThrough, 0
		case vk == vkCapital:
			return AsShift, 0
		}
		return Swallow, 0
	}
	if !s.Enabled || !s.LayoutMatches {
		return PassThrough, 0
	}
	if s.CapsLockAsShift && vk == vkCapital {
		return AsShift, 0
	}
	if s.AltGrShortcuts && s.AltGrHeld {
		if ch, ok := Lookup(AltGr, vk, s.ShiftHeld); ok {
			return Type, ch
		}
	}
	if s.UndeadKeys && !s.ShortcutHeld {
		if ch, ok := Lookup(Undead, vk, s.ShiftHeld); ok {
			return Type, ch
		}
	}
	return PassThrough, 0
}
