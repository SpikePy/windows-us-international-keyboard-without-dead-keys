package hook

import (
	"testing"

	"windows-us-international-keyboard-without-dead-keys/internal/keymap"
)

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
			vk:       keymap.VKOem7,
			keyDown:  true,
			want:     Type,
			wantChar: '\'',
		},
		{
			name:     "AltGr wins over the undead table on a key in both",
			state:    with(allOn, func(s *State) { s.AltGrHeld = true }),
			vk:       keymap.VKOem7,
			keyDown:  true,
			want:     Type,
			wantChar: '´',
		},
		{
			name:     "with AltGr shortcuts off, AltGr+' still types a plain apostrophe",
			state:    with(allOn, func(s *State) { s.AltGrHeld, s.AltGrShortcuts = true, false }),
			vk:       keymap.VKOem7,
			keyDown:  true,
			want:     Type,
			wantChar: '\'',
		},
		{
			name:    "with undead keys off, the apostrophe is the layout's business again",
			state:   with(allOn, func(s *State) { s.UndeadKeys = false }),
			vk:      keymap.VKOem7,
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
	for vk, chars := range keymap.AltGr {
		if got, ch := Decide(state, vk, true); got != Type || ch != chars.Plain {
			t.Errorf("AltGr+%q = %v, %q; want Type, %q", keymap.Label(vk), got, ch, chars.Plain)
		}
		if got, _ := Decide(state, vk, false); got != Swallow {
			t.Errorf("releasing AltGr+%q = %v; want Swallow", keymap.Label(vk), got)
		}
	}
}
