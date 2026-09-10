package main

import (
	"fmt"
	"strings"
)

// vkNames maps a normalized .klc symbolic VK name (see normalizeVKName) to
// the C expression used to refer to it - either a real VK_xxx constant from
// <winuser.h>, or a bare character literal for keys whose virtual-key value
// equals their ASCII code (VK_A..VK_Z and VK_0..VK_9 aren't named
// constants - you use 'A'..'Z' and '0'..'9' directly, per Win32
// convention). Real-world .klc files spell these names inconsistently
// (e.g. "OemMinus" vs "OEM_MINUS", "Space" vs "SPACE"); normalizeVKName
// strips underscores and lowercases before this lookup so either form
// resolves to the same entry.
var vkNames = map[string]string{
	"escape": "VK_ESCAPE", "esc": "VK_ESCAPE",
	"return": "VK_RETURN", "enter": "VK_RETURN",
	"space": "VK_SPACE",
	"back":  "VK_BACK", "backspace": "VK_BACK",
	"tab":    "VK_TAB",
	"cancel": "VK_CANCEL",

	"oemminus": "VK_OEM_MINUS", "oemplus": "VK_OEM_PLUS",
	"oemcomma": "VK_OEM_COMMA", "oemperiod": "VK_OEM_PERIOD",
	"oem1": "VK_OEM_1", "oem2": "VK_OEM_2", "oem3": "VK_OEM_3",
	"oem4": "VK_OEM_4", "oem5": "VK_OEM_5", "oem6": "VK_OEM_6", "oem7": "VK_OEM_7",
	"oem8": "VK_OEM_8", "oem102": "VK_OEM_102",
	"oemopenbrackets": "VK_OEM_4", "oemclosebrackets": "VK_OEM_6",
	"oembackslash": "VK_OEM_5", "oemtilde": "VK_OEM_3",
	"oemquotes": "VK_OEM_7", "oemsemicolon": "VK_OEM_1",
	"oemquestion": "VK_OEM_2", "oempipe": "VK_OEM_5",

	"decimal": "VK_DECIMAL", "divide": "VK_DIVIDE",
	"multiply": "VK_MULTIPLY", "subtract": "VK_SUBTRACT", "add": "VK_ADD",

	"f1": "VK_F1", "f2": "VK_F2", "f3": "VK_F3", "f4": "VK_F4",
	"f5": "VK_F5", "f6": "VK_F6", "f7": "VK_F7", "f8": "VK_F8",
	"f9": "VK_F9", "f10": "VK_F10", "f11": "VK_F11", "f12": "VK_F12",
	"f13": "VK_F13", "f14": "VK_F14", "f15": "VK_F15", "f16": "VK_F16",
	"f17": "VK_F17", "f18": "VK_F18", "f19": "VK_F19", "f20": "VK_F20",
	"f21": "VK_F21", "f22": "VK_F22", "f23": "VK_F23", "f24": "VK_F24",
}

func normalizeVKName(name string) string {
	name = strings.ReplaceAll(name, "_", "")
	return strings.ToLower(name)
}

// vkExprFor resolves a .klc symbolic VK name to a C expression suitable for
// use both as a table key (e.g. in aVkToWch2[]) and, unmodified, in ausVK[].
func vkExprFor(name string) (string, error) {
	if len(name) == 1 {
		c := name[0]
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return fmt.Sprintf("'%c'", c), nil
		}
	}
	if expr, ok := vkNames[normalizeVKName(name)]; ok {
		return expr, nil
	}
	return "", fmt.Errorf("unrecognized VK name %q (add it to vkNames in vk.go)", name)
}
