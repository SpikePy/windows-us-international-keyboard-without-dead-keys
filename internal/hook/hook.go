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
// character they should produce instead, immediately. Unlike AltGr,
// these apply whether or not any modifier is held.
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
		if ch, ok := Lookup(AltGr, vk, s.ShiftHeld); ok {
			return handled(keyDown, ch)
		}
	}
	if s.UndeadKeys {
		if ch, ok := Lookup(Undead, vk, s.ShiftHeld); ok {
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
