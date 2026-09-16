package hook

import (
	"os"
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		name  string
		table map[uint32]Chars
		vk    uint32
		shift bool
		want  rune
		ok    bool
	}{
		{"AltGr+E", AltGr, 'E', false, 'é', true},
		{"Shift+AltGr+E", AltGr, 'E', true, 'É', true},
		{"AltGr+5 is the same either way", AltGr, '5', true, '€', true},
		{"AltGr on an OEM key", AltGr, vkOemComma, false, 'ç', true},
		{"unmapped key", AltGr, 'F', false, 0, false},
		{"undead apostrophe", Undead, vkOem7, false, '\'', true},
		{"undead apostrophe with Shift", Undead, vkOem7, true, '"', true},
		{"a key that is only an AltGr key", Undead, 'E', false, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Lookup(tt.table, tt.vk, tt.shift)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Lookup(%#x, shift=%v) = %q, %v; want %q, %v", tt.vk, tt.shift, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestTablesAreWellFormed guards against a typo turning an entry into
// something that would be typed as NUL or a control character, and
// against a key nothing can name appearing in a table.
func TestTablesAreWellFormed(t *testing.T) {
	for name, table := range map[string]map[uint32]Chars{"AltGr": AltGr, "Undead": Undead} {
		for vk, c := range table {
			if label(vk) == "" {
				t.Errorf("%s has an entry for virtual-key %#x, which Label doesn't name", name, vk)
			}
			for _, r := range []rune{c.Plain, c.Shifted} {
				if r < 0x20 || r == 0x7F {
					t.Errorf("%s[%q] produces the non-printable rune %#x", name, label(vk), r)
				}
			}
		}
	}
}

// TestDocumentedTablesMatch keeps DETAILS.md honest: the character tables
// there are the only place a user can look up what a key does, so a map
// changed without the documentation (or the other way round) is a bug.
func TestDocumentedTablesMatch(t *testing.T) {
	data, err := os.ReadFile("../../DETAILS.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)

	tests := []struct {
		header [3]string
		table  map[uint32]Chars
	}{
		{[3]string{"Key", "AltGr", "Shift+AltGr"}, AltGr},
		{[3]string{"Key", "Alone", "With Shift"}, Undead},
	}
	for _, tt := range tests {
		documented := parseTable(t, doc, tt.header)
		for vk, c := range tt.table {
			key := label(vk)
			got, ok := documented[key]
			if !ok {
				t.Errorf("DETAILS.md's %q table has no row for %q", tt.header[1], key)
				continue
			}
			if want := ([2]string{string(c.Plain), string(c.Shifted)}); got != want {
				t.Errorf("DETAILS.md's %q table documents %q as %v, but the code produces %v", tt.header[1], key, got, want)
			}
			delete(documented, key)
		}
		for key := range documented {
			t.Errorf("DETAILS.md's %q table documents %q, which the code doesn't map", tt.header[1], key)
		}
	}
}

// parseTable returns the rows of the Markdown table in doc whose header
// row is header, as key -> the other two cells. Cells are unwrapped from
// the `backticks` the tables use to keep punctuation keys literal.
func parseTable(t *testing.T, doc string, header [3]string) map[string][2]string {
	t.Helper()
	rows := make(map[string][2]string)
	inTable := false
	for _, line := range strings.Split(doc, "\n") {
		cells, ok := tableRow(line)
		if !ok {
			inTable = false
			continue
		}
		if cells == header {
			inTable = true
			continue
		}
		if inTable && !strings.HasPrefix(cells[0], "---") {
			rows[cells[0]] = [2]string{cells[1], cells[2]}
		}
	}
	if len(rows) == 0 {
		t.Fatalf("DETAILS.md has no table with the header %v", header)
	}
	return rows
}

// tableRow splits one Markdown table line into exactly three cells.
func tableRow(line string) (cells [3]string, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return cells, false
	}
	parts := strings.Split(strings.Trim(line, "|"), "|")
	if len(parts) != 3 {
		return cells, false
	}
	for i, p := range parts {
		// Cells are wrapped in a code span - `` ` `` for the backtick key
		// itself - so punctuation renders literally; unwrap it back to the
		// character.
		cells[i] = strings.TrimSpace(strings.Trim(strings.TrimSpace(p), "`"))
	}
	return cells, true
}

// allOn is the state in which everything is switched on and the right
// layout is in front; each test tweaks a copy of it.
var allOn = State{Enabled: true, LayoutMatches: true, AltGrShortcuts: true, UndeadKeys: true}

func with(base State, f func(*State)) State {
	f(&base)
	return base
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		vk       uint32
		keyDown  bool
		want     Action
		wantChar rune
	}{
		{
			name:     "AltGr+E types an e-acute",
			state:    with(allOn, func(s *State) { s.AltGrHeld = true }),
			vk:       'E',
			keyDown:  true,
			want:     Type,
			wantChar: 'é',
		},
		{
			name:    "the release of a handled AltGr key is swallowed",
			state:   with(allOn, func(s *State) { s.AltGrHeld = true }),
			vk:      'E',
			keyDown: false,
			want:    Swallow,
		},
		{
			name:     "Shift+AltGr+E types an E-acute",
			state:    with(allOn, func(s *State) { s.AltGrHeld, s.ShiftHeld = true, true }),
			vk:       'E',
			keyDown:  true,
			want:     Type,
			wantChar: 'É',
		},
		{
			name:    "E on its own passes through",
			state:   allOn,
			vk:      'E',
			keyDown: true,
			want:    PassThrough,
		},
		{
			name:    "AltGr plus an unmapped key passes through",
			state:   with(allOn, func(s *State) { s.AltGrHeld = true }),
			vk:      'X',
			keyDown: true,
			want:    PassThrough,
		},
		{
			name:     "the apostrophe types itself instead of waiting as a dead key",
			state:    allOn,
			vk:       vkOem7,
			keyDown:  true,
			want:     Type,
			wantChar: '\'',
		},
		{
			name:     "AltGr wins over the undead table on a key in both",
			state:    with(allOn, func(s *State) { s.AltGrHeld = true }),
			vk:       vkOem7,
			keyDown:  true,
			want:     Type,
			wantChar: '´',
		},
		{
			name:     "with AltGr shortcuts off, AltGr+' still types a plain apostrophe",
			state:    with(allOn, func(s *State) { s.AltGrHeld, s.AltGrShortcuts = true, false }),
			vk:       vkOem7,
			keyDown:  true,
			want:     Type,
			wantChar: '\'',
		},
		{
			name:    "with undead keys off, the apostrophe is the layout's business again",
			state:   with(allOn, func(s *State) { s.UndeadKeys = false }),
			vk:      vkOem7,
			keyDown: true,
			want:    PassThrough,
		},
		{
			name:    "paused from the tray, nothing is intercepted",
			state:   with(allOn, func(s *State) { s.Enabled, s.AltGrHeld = false, true }),
			vk:      'E',
			keyDown: true,
			want:    PassThrough,
		},
		{
			name:    "under another keyboard layout, nothing is intercepted",
			state:   with(allOn, func(s *State) { s.LayoutMatches, s.AltGrHeld = false, true }),
			vk:      'E',
			keyDown: true,
			want:    PassThrough,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ch := Decide(tt.state, tt.vk, tt.keyDown)
			if got != tt.want || ch != tt.wantChar {
				t.Errorf("Decide(%+v, %#x, keyDown=%v) = %v, %q; want %v, %q",
					tt.state, tt.vk, tt.keyDown, got, ch, tt.want, tt.wantChar)
			}
		})
	}
}

// TestEveryAltGrKeyIsTypedWhileHeld guards the wiring between Decide and
// the tables: every entry must come out as a character, not as a key the
// application gets to see raw.
func TestEveryAltGrKeyIsTypedWhileHeld(t *testing.T) {
	state := with(allOn, func(s *State) { s.AltGrHeld = true })
	for vk, chars := range AltGr {
		if got, ch := Decide(state, vk, true); got != Type || ch != chars.Plain {
			t.Errorf("AltGr+%q = %v, %q; want Type, %q", label(vk), got, ch, chars.Plain)
		}
		if got, _ := Decide(state, vk, false); got != Swallow {
			t.Errorf("releasing AltGr+%q = %v; want Swallow", label(vk), got)
		}
	}
}

// label is the key's printed name - what is on the keycap under a US
// layout - as the documentation tables write it. It returns "" for a code
// no table uses.
func label(vk uint32) string {
	switch {
	case vk >= '0' && vk <= '9', vk >= 'A' && vk <= 'Z':
		return string(rune(vk))
	}
	switch vk {
	case vkOem1:
		return ";"
	case vkOemPlus:
		return "="
	case vkOemComma:
		return ","
	case vkOemMinus:
		return "-"
	case vkOem2:
		return "/"
	case vkOem3:
		return "`"
	case vkOem4:
		return "["
	case vkOem5:
		return `\`
	case vkOem6:
		return "]"
	case vkOem7:
		return "'"
	}
	return ""
}
