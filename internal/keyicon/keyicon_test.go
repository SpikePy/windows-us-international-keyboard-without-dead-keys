package keyicon

import "testing"

func TestGlyphShape(t *testing.T) {
	tests := []struct {
		name string
		x, y int
		want Part
	}{
		{"top-left corner is empty", 0, 0, PartNone},
		{"bottom of the accent stroke", 14, 20, PartAccent},
		{"top of the accent leans right", 20, 15, PartAccent},
		{"top border of the keycap", 16, 6, PartFrame},
		{"left border of the keycap", 3, 20, PartFrame},
		{"inside the keycap, clear of the accent", 8, 11, PartCapInterior},
		{"bottom border of the keycap", 16, 29, PartFrame},
		{"below the keycap is empty", 16, 30, PartNone},
	}
	for _, tt := range tests {
		if got := At(tt.x, tt.y); got != tt.want {
			t.Errorf("%s: At(%d, %d) = %v, want %v", tt.name, tt.x, tt.y, got, tt.want)
		}
	}
}

// TestAccentSitsInsideTheKeycap keeps the accent clear of the keycap's
// border: touching it would merge the two into a smudge at tray size.
func TestAccentSitsInsideTheKeycap(t *testing.T) {
	for y := 0; y < GridSize; y++ {
		for x := 0; x < GridSize; x++ {
			if At(x, y) != PartAccent {
				continue
			}
			for _, n := range [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
				if At(n[0], n[1]) == PartFrame {
					t.Fatalf("the accent touches the keycap's border at (%d, %d)", x, y)
				}
			}
		}
	}
}

func TestAtScaledMatchesAtAtNativeSize(t *testing.T) {
	for y := 0; y < GridSize; y++ {
		for x := 0; x < GridSize; x++ {
			if a, s := At(x, y), AtScaled(x, y, GridSize); a != s {
				t.Fatalf("(%d, %d): At = %v but AtScaled at GridSize = %v", x, y, a, s)
			}
		}
	}
}
