package keyicon

import (
	"image"
	"testing"
)

var sizes = []int{16, 20, 24, 32, 40, 48, 64, 256}

func TestRenderSize(t *testing.T) {
	for _, size := range sizes {
		for _, enabled := range []bool{true, false} {
			if b := Render(size, enabled).Bounds(); b != image.Rect(0, 0, size, size) {
				t.Errorf("Render(%d, %v) is %v", size, enabled, b)
			}
		}
	}
}

func TestKeycapShape(t *testing.T) {
	for _, size := range sizes {
		img := Render(size, true)
		g := layout(size)
		mid := size / 2
		// A pixel fully inside the top border, and one fully inside the
		// thicker bottom edge.
		top := int(g.top + g.stroke/2)
		bottom := int(g.bottom - g.base/2)
		tests := []struct {
			name      string
			x, y      int
			wantAlpha func(uint8) bool
		}{
			{"the corner outside the rounded keycap is clear", 0, 0, isClear},
			{"the middle of the top border is solid", mid, top, isSolid},
			{"the middle of the bottom edge is solid", mid, bottom, isSolid},
			{"just inside the left border, beside the letter, is clear", int(g.left + g.stroke), mid, isClear},
		}
		for _, tt := range tests {
			if a := img.NRGBAAt(tt.x, tt.y).A; !tt.wantAlpha(a) {
				t.Errorf("%dpx: %s: alpha at (%d, %d) is %d", size, tt.name, tt.x, tt.y, a)
			}
		}
	}
}

// TestLetterIsOnTheKeycap checks the letter is drawn at every size, fills
// a good share of the keycap, stays centered, and never touches the
// border.
func TestLetterIsOnTheKeycap(t *testing.T) {
	for _, size := range sizes {
		frame, letter := masks(size)
		ink := letter.Bounds()
		minX, minY, maxX, maxY := size, size, -1, -1
		for y := ink.Min.Y; y < ink.Max.Y; y++ {
			for x := ink.Min.X; x < ink.Max.X; x++ {
				if letter.AlphaAt(x, y).A == 0 {
					continue
				}
				if frame.AlphaAt(x, y).A != 0 {
					t.Fatalf("%dpx: the letter overlaps the keycap border at (%d, %d)", size, x, y)
				}
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x), max(maxY, y)
			}
		}
		if maxX < 0 {
			t.Fatalf("%dpx: no letter was drawn", size)
		}
		if h := maxY - minY + 1; h < size/2 {
			t.Errorf("%dpx: the letter is only %dpx tall", size, h)
		}
		g := layout(size)
		il, _, ir, _, _ := g.inner()
		center := float64(minX+maxX+1) / 2
		if keycapCenter := (il + ir) / 2; center < keycapCenter-1.5 || center > keycapCenter+1.5 {
			t.Errorf("%dpx: the letter is centered at x=%.1f, the keycap at x=%.1f", size, center, keycapCenter)
		}
	}
}

// TestEdgesAreSmooth is what separates this from pixel art: the outlines
// and the letter must have partly covered pixels along their edges.
func TestEdgesAreSmooth(t *testing.T) {
	for _, size := range []int{32, 48, 256} {
		img := Render(size, true)
		partial := 0
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if a := img.NRGBAAt(x, y).A; a > 0 && a < 255 {
					partial++
				}
			}
		}
		if partial < size {
			t.Errorf("%dpx: only %d anti-aliased pixels", size, partial)
		}
	}
}

func TestDisabledLook(t *testing.T) {
	for _, size := range sizes {
		on, off := Render(size, true), Render(size, false)
		// The strike runs through the diagonal, including the corner that
		// is otherwise transparent.
		if c := off.NRGBAAt(0, 0); c.A < 128 || c.R < 150 || c.G > 80 {
			t.Errorf("%dpx: the top-left corner of the disabled icon is %v, want the red strike", size, c)
		}
		// Off the diagonal, the border is grey instead of black.
		g := layout(size)
		x, y := size/2+1, int(g.top+g.stroke/2)
		if c := on.NRGBAAt(x, y); c != enabledInk {
			t.Errorf("%dpx: enabled border at (%d, %d) is %v, want %v", size, x, y, c, enabledInk)
		}
		if c := off.NRGBAAt(x, y); c != disabledInk {
			t.Errorf("%dpx: disabled border at (%d, %d) is %v, want %v", size, x, y, c, disabledInk)
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	a, b := Render(32, true), Render(32, true)
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			t.Fatal("two renders of the same size differ")
		}
	}
}

func isClear(a uint8) bool { return a == 0 }
func isSolid(a uint8) bool { return a == 255 }
