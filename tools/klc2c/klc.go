package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Layout is the parsed contents of a .klc file, enough of it to generate a
// KBDTABLES-based C source file for the "main block" of a keyboard (the
// alphanumeric section handled by the LAYOUT table). Numpad, navigation,
// and other extended keys are not customizable via klc LAYOUT rows and are
// always emitted with their standard mappings.
type Layout struct {
	Name        string // internal KBD name, becomes the DLL base name
	Description string
	Copyright   string
	Company     string
	LocaleName  string
	LocaleID    string
	Version     string

	ShiftStates []int // raw KBDSHIFT|KBDCTRL|KBDALT bitmask values, in declared order

	Rows []Row

	KeyNames    []KeyName
	KeyNamesExt []KeyName
}

type Row struct {
	Line     int
	ScanCode byte
	VKName   string
	Cap      int
	Values   []Value // one per declared shift state
}

// Value is one cell of a LAYOUT row: either "no character" (WCH_NONE) or a
// Unicode code point.
type Value struct {
	None bool
	Rune rune
}

type KeyName struct {
	ScanCode byte
	Name     string
}

// stripComment removes a trailing "//" comment, but only when it starts
// outside of a quoted string.
func stripComment(line string) string {
	inQuotes := false
	for i := 0; i < len(line)-1; i++ {
		switch line[i] {
		case '"':
			inQuotes = !inQuotes
		case '/':
			if !inQuotes && line[i+1] == '/' {
				return line[:i]
			}
		}
	}
	return line
}

// tokenize splits a line on whitespace, keeping "quoted strings" as single
// tokens (with the surrounding quotes stripped).
func tokenize(line string) []string {
	var tokens []string
	var cur strings.Builder
	inQuotes := false
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, r := range line {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case !inQuotes && (r == ' ' || r == '\t'):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return tokens
}

func parseHexByte(tok string) (byte, error) {
	v, err := strconv.ParseUint(tok, 16, 8)
	if err != nil {
		return 0, err
	}
	return byte(v), nil
}

// parseValue interprets one LAYOUT-row character cell. klc allows:
//   - "-1"                 -> no character (WCH_NONE)
//   - a 4-hex-digit code   -> that Unicode code point (e.g. "00e4")
//   - a bare literal char  -> that character (e.g. "q", used by some
//     real-world .klc files instead of the hex form)
//   - a trailing "@" marks a dead key; not supported by this generator.
func parseValue(tok string) (Value, error) {
	if tok == "-1" {
		return Value{None: true}, nil
	}
	if strings.HasSuffix(tok, "@") {
		return Value{}, fmt.Errorf("dead keys are not supported by this generator (value %q)", tok)
	}
	if len(tok) == 4 {
		if v, err := strconv.ParseUint(tok, 16, 32); err == nil {
			return Value{Rune: rune(v)}, nil
		}
	}
	runes := []rune(tok)
	if len(runes) == 1 {
		return Value{Rune: runes[0]}, nil
	}
	return Value{}, fmt.Errorf("unrecognized LAYOUT value %q", tok)
}

func ParseKLC(path string) (*Layout, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	l := &Layout{}
	section := ""

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	first := true
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		if first {
			// Strip a UTF-8 BOM if present.
			raw = strings.TrimPrefix(raw, "\uFEFF")
			first = false
		}
		line := stripComment(raw)
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Section headers and top-level directives are recognized by their
		// first token, case-sensitively, matching the .klc convention.
		fields := tokenize(trimmed)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "KBD":
			if len(fields) < 3 {
				return nil, fmt.Errorf("line %d: malformed KBD line", lineNo)
			}
			l.Name = fields[1]
			l.Description = fields[2]
			section = ""
			continue
		case "COPYRIGHT":
			if len(fields) >= 2 {
				l.Copyright = fields[1]
			}
			continue
		case "COMPANY":
			if len(fields) >= 2 {
				l.Company = fields[1]
			}
			continue
		case "LOCALENAME":
			if len(fields) >= 2 {
				l.LocaleName = fields[1]
			}
			continue
		case "LOCALEID":
			if len(fields) >= 2 {
				l.LocaleID = fields[1]
			}
			continue
		case "VERSION":
			if len(fields) >= 2 {
				l.Version = fields[1]
			}
			continue
		case "SHIFTSTATE":
			section = "SHIFTSTATE"
			continue
		case "LAYOUT":
			section = "LAYOUT"
			continue
		case "KEYNAME":
			section = "KEYNAME"
			continue
		case "KEYNAME_EXT":
			section = "KEYNAME_EXT"
			continue
		case "KEYNAME_DEAD":
			section = "KEYNAME_DEAD"
			continue
		case "DESCRIPTIONS":
			section = "DESCRIPTIONS"
			continue
		case "LANGUAGENAMES":
			section = "LANGUAGENAMES"
			continue
		case "ENDKBD":
			section = ""
			continue
		}

		switch section {
		case "SHIFTSTATE":
			v, err := strconv.Atoi(fields[0])
			if err != nil {
				return nil, fmt.Errorf("line %d: bad SHIFTSTATE value %q: %w", lineNo, fields[0], err)
			}
			l.ShiftStates = append(l.ShiftStates, v)

		case "LAYOUT":
			if len(fields) < 3+len(l.ShiftStates) {
				return nil, fmt.Errorf("line %d: LAYOUT row has %d fields, expected at least %d", lineNo, len(fields), 3+len(l.ShiftStates))
			}
			sc, err := parseHexByte(fields[0])
			if err != nil {
				return nil, fmt.Errorf("line %d: bad scancode %q: %w", lineNo, fields[0], err)
			}
			cap, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: bad Cap value %q: %w", lineNo, fields[2], err)
			}
			row := Row{Line: lineNo, ScanCode: sc, VKName: fields[1], Cap: cap}
			for i := 0; i < len(l.ShiftStates); i++ {
				v, err := parseValue(fields[3+i])
				if err != nil {
					return nil, fmt.Errorf("line %d: %w", lineNo, err)
				}
				row.Values = append(row.Values, v)
			}
			l.Rows = append(l.Rows, row)

		case "KEYNAME", "KEYNAME_EXT":
			if len(fields) < 2 {
				continue
			}
			sc, err := parseHexByte(fields[0])
			if err != nil {
				return nil, fmt.Errorf("line %d: bad scancode %q: %w", lineNo, fields[0], err)
			}
			kn := KeyName{ScanCode: sc, Name: fields[1]}
			if section == "KEYNAME" {
				l.KeyNames = append(l.KeyNames, kn)
			} else {
				l.KeyNamesExt = append(l.KeyNamesExt, kn)
			}

		// DESCRIPTIONS / LANGUAGENAMES / KEYNAME_DEAD are not needed to
		// build the DLL and are intentionally ignored.
		default:
			// Outside any recognized section (e.g. the "//SC VK_ Cap..."
			// header comment inside LAYOUT); nothing to do.
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if l.Name == "" {
		return nil, fmt.Errorf("no KBD line found")
	}
	if len(l.ShiftStates) == 0 {
		return nil, fmt.Errorf("no SHIFTSTATE values found")
	}
	return l, nil
}
