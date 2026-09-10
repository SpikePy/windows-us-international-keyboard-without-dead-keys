package main

// baseEntry is one scancode's default (VK expression, extra flags) before
// any .klc LAYOUT row override is applied. Flags is a literal C suffix like
// " | KBDEXT | KBDMULTIVK", or "" for none. This mirrors Microsoft's own
// kbdus.c reference sample (the standard US keyboard), which is the
// physical basis for every "main block" QWERTY-based layout: scancode-to-
// virtual-key assignments and special-key flags (KBDEXT, KBDNUMPAD, ...)
// don't change between layouts - only which characters a key produces does.
type baseEntry struct {
	VK    string
	Flags string
}

// ausVKBase covers scancodes 0x00-0x7E. A .klc LAYOUT row for a given
// scancode replaces the VK here (with no extra flags - none of our rows
// target a flagged scancode); scancodes with no row keep this default.
var ausVKBase = map[byte]baseEntry{
	0x00: {"0", ""},
	0x01: {"VK_ESCAPE", ""},
	0x02: {"'1'", ""}, 0x03: {"'2'", ""}, 0x04: {"'3'", ""}, 0x05: {"'4'", ""},
	0x06: {"'5'", ""}, 0x07: {"'6'", ""}, 0x08: {"'7'", ""}, 0x09: {"'8'", ""},
	0x0A: {"'9'", ""}, 0x0B: {"'0'", ""},
	0x0C: {"VK_OEM_MINUS", ""}, 0x0D: {"VK_OEM_PLUS", ""},
	0x0E: {"VK_BACK", ""}, 0x0F: {"VK_TAB", ""},
	0x10: {"'Q'", ""}, 0x11: {"'W'", ""}, 0x12: {"'E'", ""}, 0x13: {"'R'", ""},
	0x14: {"'T'", ""}, 0x15: {"'Y'", ""}, 0x16: {"'U'", ""}, 0x17: {"'I'", ""},
	0x18: {"'O'", ""}, 0x19: {"'P'", ""},
	0x1A: {"VK_OEM_4", ""}, 0x1B: {"VK_OEM_6", ""},
	0x1C: {"VK_RETURN", ""}, 0x1D: {"VK_LCONTROL", ""},
	0x1E: {"'A'", ""}, 0x1F: {"'S'", ""}, 0x20: {"'D'", ""}, 0x21: {"'F'", ""},
	0x22: {"'G'", ""}, 0x23: {"'H'", ""}, 0x24: {"'J'", ""}, 0x25: {"'K'", ""}, 0x26: {"'L'", ""},
	0x27: {"VK_OEM_1", ""}, 0x28: {"VK_OEM_7", ""}, 0x29: {"VK_OEM_3", ""},
	0x2A: {"VK_LSHIFT", ""}, 0x2B: {"VK_OEM_5", ""},
	0x2C: {"'Z'", ""}, 0x2D: {"'X'", ""}, 0x2E: {"'C'", ""}, 0x2F: {"'V'", ""},
	0x30: {"'B'", ""}, 0x31: {"'N'", ""}, 0x32: {"'M'", ""},
	0x33: {"VK_OEM_COMMA", ""}, 0x34: {"VK_OEM_PERIOD", ""}, 0x35: {"VK_OEM_2", ""},
	0x36: {"VK_RSHIFT", " | KBDEXT"},
	0x37: {"VK_MULTIPLY", " | KBDMULTIVK"},
	0x38: {"VK_LMENU", ""}, 0x39: {"VK_SPACE", ""}, 0x3A: {"VK_CAPITAL", ""},
	0x3B: {"VK_F1", ""}, 0x3C: {"VK_F2", ""}, 0x3D: {"VK_F3", ""}, 0x3E: {"VK_F4", ""},
	0x3F: {"VK_F5", ""}, 0x40: {"VK_F6", ""}, 0x41: {"VK_F7", ""}, 0x42: {"VK_F8", ""},
	0x43: {"VK_F9", ""}, 0x44: {"VK_F10", ""},
	0x45: {"VK_NUMLOCK", " | KBDEXT | KBDMULTIVK"},
	0x46: {"VK_SCROLL", " | KBDMULTIVK"},
	0x47: {"VK_HOME", " | KBDNUMPAD | KBDSPECIAL"},
	0x48: {"VK_UP", " | KBDNUMPAD | KBDSPECIAL"},
	0x49: {"VK_PRIOR", " | KBDNUMPAD | KBDSPECIAL"},
	0x4A: {"VK_SUBTRACT", ""},
	0x4B: {"VK_LEFT", " | KBDNUMPAD | KBDSPECIAL"},
	0x4C: {"VK_CLEAR", " | KBDNUMPAD | KBDSPECIAL"},
	0x4D: {"VK_RIGHT", " | KBDNUMPAD | KBDSPECIAL"},
	0x4E: {"VK_ADD", ""},
	0x4F: {"VK_END", " | KBDNUMPAD | KBDSPECIAL"},
	0x50: {"VK_DOWN", " | KBDNUMPAD | KBDSPECIAL"},
	0x51: {"VK_NEXT", " | KBDNUMPAD | KBDSPECIAL"},
	0x52: {"VK_INSERT", " | KBDNUMPAD | KBDSPECIAL"},
	0x53: {"VK_DELETE", " | KBDNUMPAD | KBDSPECIAL"},
	0x54: {"VK_SNAPSHOT", ""},
	0x55: {"0", ""},
	0x56: {"VK_OEM_102", ""},
	0x57: {"VK_F11", ""}, 0x58: {"VK_F12", ""},
	0x59: {"0", ""}, 0x5A: {"0", ""}, 0x5B: {"0", ""}, 0x5C: {"0", ""}, 0x5D: {"0", ""},
	0x5E: {"0", ""}, 0x5F: {"0", ""}, 0x60: {"0", ""}, 0x61: {"0", ""}, 0x62: {"0", ""}, 0x63: {"0", ""},
	0x64: {"VK_F13", ""}, 0x65: {"VK_F14", ""}, 0x66: {"VK_F15", ""}, 0x67: {"VK_F16", ""},
	0x68: {"VK_F17", ""}, 0x69: {"VK_F18", ""}, 0x6A: {"VK_F19", ""}, 0x6B: {"VK_F20", ""},
	0x6C: {"VK_F21", ""}, 0x6D: {"VK_F22", ""}, 0x6E: {"VK_F23", ""},
	0x6F: {"0", ""}, 0x70: {"0", ""}, 0x71: {"0", ""}, 0x72: {"0", ""}, 0x73: {"0", ""},
	0x74: {"0", ""}, 0x75: {"0", ""},
	0x76: {"VK_F24", ""},
	0x77: {"0", ""}, 0x78: {"0", ""}, 0x79: {"0", ""}, 0x7A: {"0", ""}, 0x7B: {"0", ""},
	0x7C: {"0", ""}, 0x7D: {"0", ""}, 0x7E: {"0", ""},
}

// The last base-plane scancode; ausVK[] covers 0x00 through this value.
const maxScanCode = 0x7E

type e0Entry struct {
	ScanCode byte
	VK       string
}

// aE0VscToVkBase is the standard set of E0-prefixed (extended) key
// mappings; not customizable via .klc LAYOUT rows.
var aE0VscToVkBase = []e0Entry{
	{0x1D, "VK_RCONTROL"},
	{0x35, "VK_DIVIDE"},
	{0x37, "VK_SNAPSHOT"},
	{0x38, "VK_RMENU"},
	{0x47, "VK_HOME"},
	{0x48, "VK_UP"},
	{0x49, "VK_PRIOR"},
	{0x4B, "VK_LEFT"},
	{0x4D, "VK_RIGHT"},
	{0x4F, "VK_END"},
	{0x50, "VK_DOWN"},
	{0x51, "VK_NEXT"},
	{0x52, "VK_INSERT"},
	{0x53, "VK_DELETE"},
	{0x5B, "VK_LWIN"},
	{0x5C, "VK_RWIN"},
	{0x5D, "VK_APPS"},
	{0x1C, "VK_RETURN"},
}

// aE1VscToVkBase is the standard E1-prefixed mapping (the Pause key).
var aE1VscToVkBase = []e0Entry{
	{0x1D, "VK_PAUSE"},
}
