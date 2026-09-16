// Package keymap holds the two character tables this tool types on the
// user's behalf: the AltGr accent shortcuts, and the "undead" keys whose
// dead-key wait is cancelled. It is just data plus lookup, with no OS
// dependency, so it builds and is tested on any platform.
package keymap

// Virtual-key codes. Letters and digits are their own ASCII code, so only
// the US OEM punctuation keys need naming; these values are fixed by
// Win32 and never change.
const (
	VKOem1     = 0xBA // ;
	VKOemPlus  = 0xBB // =
	VKOemComma = 0xBC // ,
	VKOemMinus = 0xBD // -
	VKOem2     = 0xBF // /
	VKOem3     = 0xC0 // ` (grave/backtick)
	VKOem4     = 0xDB // [
	VKOem5     = 0xDC // backslash
	VKOem6     = 0xDD // ]
	VKOem7     = 0xDE // '
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
	VKOemMinus: {'¥', '¥'},
	VKOemPlus:  {'×', '÷'},
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
	VKOem4:     {'«', '«'},
	VKOem6:     {'»', '»'},
	'A':        {'á', 'Á'},
	'S':        {'ß', '§'},
	'D':        {'ð', 'Ð'},
	'L':        {'ø', 'Ø'},
	VKOem1:     {'¶', '°'},
	VKOem7:     {'´', '¨'},
	VKOem5:     {'¬', '¦'},
	'Z':        {'æ', 'Æ'},
	'C':        {'©', '¢'},
	'N':        {'ñ', 'Ñ'},
	'M':        {'µ', 'µ'},
	VKOemComma: {'ç', 'Ç'},
	VKOem2:     {'¿', '¿'},
}

// Undead maps the keys that the "United States-International" layout
// treats as dead keys - they normally swallow the keystroke and wait for
// a second one to combine into an accented letter - to the plain
// character they should produce instead, immediately. Unlike AltGr,
// these apply whether or not any modifier is held.
var Undead = map[uint32]Chars{
	VKOem7: {'\'', '"'},
	VKOem3: {'`', '~'},
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

// Label is the key's printed name - what is on the keycap under a US
// layout - used by the documentation tables and their tests. It returns
// "" for a code no table uses.
func Label(vk uint32) string {
	switch {
	case vk >= '0' && vk <= '9', vk >= 'A' && vk <= 'Z':
		return string(rune(vk))
	}
	switch vk {
	case VKOem1:
		return ";"
	case VKOemPlus:
		return "="
	case VKOemComma:
		return ","
	case VKOemMinus:
		return "-"
	case VKOem2:
		return "/"
	case VKOem3:
		return "`"
	case VKOem4:
		return "["
	case VKOem5:
		return `\`
	case VKOem6:
		return "]"
	case VKOem7:
		return "'"
	}
	return ""
}
