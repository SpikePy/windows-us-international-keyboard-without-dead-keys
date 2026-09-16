// Package keyicon draws the glyph shared by the runtime tray icon
// (internal/tray) and the generated .exe file icon (tools/genicon), so
// both always show the exact same picture: a rounded keycap with an "Á"
// on it. Everything is rendered from outlines with anti-aliasing, at
// whatever size is asked for, so the icon stays smooth on any display
// scaling. It has no OS dependency and builds anywhere.
package keyicon

import (
	"image"
	"image/color"
	"math"
)

// Colors of the glyph in its two states. Disabled is the same glyph in
// grey with a red strike across it - the conventional "off" cue - so it
// stays recognizable.
var (
	enabledInk  = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	disabledInk = color.NRGBA{R: 140, G: 140, B: 140, A: 255}
	strikeInk   = color.NRGBA{R: 200, G: 30, B: 30, A: 255}
)

// supersample is how many sub-samples per pixel side the shapes are
// evaluated at; 8x8 gives 64 coverage levels, plenty for a small icon.
const supersample = 8

// Render draws the glyph at size x size pixels, in its enabled or
// disabled look, with straight (non-premultiplied) alpha - the form both
// Windows icon bitmaps and PNG expect.
func Render(size int, enabled bool) *image.NRGBA {
	frame, letter := masks(size)
	ink := enabledInk
	if !enabled {
		ink = disabledInk
	}

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Frame and letter never overlap, so their coverage adds up.
			a := int(frame.AlphaAt(x, y).A) + int(letter.AlphaAt(x, y).A)
			if a > 255 {
				a = 255
			}
			img.SetNRGBA(x, y, color.NRGBA{R: ink.R, G: ink.G, B: ink.B, A: uint8(a)})
		}
	}
	if !enabled {
		strike := strikeMask(size)
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if a := strike.AlphaAt(x, y).A; a > 0 {
					img.SetNRGBA(x, y, over(strikeInk, a, img.NRGBAAt(x, y)))
				}
			}
		}
	}
	return img
}

// geometry is the keycap's layout at one size, in pixels. The stroke is
// snapped to whole pixels so the outline stays sharp at small sizes; the
// bottom edge is drawn thicker, like the side of a real keycap seen from
// slightly above.
type geometry struct {
	left, top, right, bottom float64 // outer edge of the keycap
	radius                   float64 // outer corner radius
	stroke, base             float64 // side/top and bottom border widths
}

func layout(size int) geometry {
	s := float64(size)
	stroke := math.Max(1, math.Round(s/16))
	inset := math.Max(1, math.Round(s/32))
	return geometry{
		left:   inset,
		top:    inset,
		right:  s - inset,
		bottom: s - inset,
		radius: s * 3 / 16,
		stroke: stroke,
		base:   stroke + math.Max(1, math.Round(s/32)),
	}
}

// inner returns the keycap's inside edge.
func (g geometry) inner() (left, top, right, bottom, radius float64) {
	return g.left + g.stroke, g.top + g.stroke, g.right - g.stroke, g.bottom - g.base,
		math.Max(0, g.radius-g.stroke)
}

// masks renders the keycap outline and the letter as separate coverage
// masks, so tests can check how they relate.
func masks(size int) (frame, letter *image.Alpha) {
	g := layout(size)
	il, it, ir, ib, iradius := g.inner()

	frame = coverage(size, func(x, y float64) bool {
		return inRoundedRect(x, y, g.left, g.top, g.right, g.bottom, g.radius) &&
			!inRoundedRect(x, y, il, it, ir, ib, iradius)
	})
	letter = letterMask(size, il, it, ir, ib)
	return frame, letter
}

// strikeMask is the disabled look's diagonal bar, top-left to
// bottom-right across the whole canvas.
func strikeMask(size int) *image.Alpha {
	half := math.Max(1, float64(size)*3/32) / 2
	return coverage(size, func(x, y float64) bool {
		return math.Abs(x-y)/math.Sqrt2 <= half
	})
}

// coverage evaluates inside at supersample x supersample points in every
// pixel and turns the hit ratio into alpha.
func coverage(size int, inside func(x, y float64) bool) *image.Alpha {
	m := image.NewAlpha(image.Rect(0, 0, size, size))
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			hits := 0
			for sy := 0; sy < supersample; sy++ {
				for sx := 0; sx < supersample; sx++ {
					x := float64(px) + (float64(sx)+0.5)/supersample
					y := float64(py) + (float64(sy)+0.5)/supersample
					if inside(x, y) {
						hits++
					}
				}
			}
			m.SetAlpha(px, py, color.Alpha{A: uint8((hits*255 + supersample*supersample/2) / (supersample * supersample))})
		}
	}
	return m
}

// inRoundedRect reports whether (x, y) lies inside the rectangle with
// corners rounded to radius r.
func inRoundedRect(x, y, left, top, right, bottom, r float64) bool {
	if x < left || x > right || y < top || y > bottom {
		return false
	}
	cx := math.Max(left+r, math.Min(x, right-r))
	cy := math.Max(top+r, math.Min(y, bottom-r))
	return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r
}

// point is a position in letter space: x from 0 (left) to 1 (right),
// y from 0 (the top of the A) down to 1 (the baseline), with the accent
// above the A at negative y.
type point struct{ x, y float64 }

// letterOutlines is the "Á" as closed outlines, filled with the even-odd
// rule: a bold, flat-topped sans-serif A, the triangular counter above its
// crossbar cut out of it, and an acute accent leaning right above it. The
// inner edges run parallel to the outer ones, so every stroke has the same
// weight (about a quarter of the letter's width).
var letterOutlines = [][]point{
	{{0, 1}, {0.36, 0}, {0.64, 0}, {1, 1}, {0.76, 1}, {0.68, 0.76}, {0.32, 0.76}, {0.24, 1}},
	{{0.3867, 0.56}, {0.5, 0.22}, {0.6133, 0.56}},
	{{0.40, -0.08}, {0.58, -0.08}, {0.80, -0.30}, {0.62, -0.30}},
}

// letterTop and letterBottom bound the outlines vertically; horizontally
// they span 0 to 1.
const letterTop, letterBottom = -0.30, 1.0

// inLetter reports whether p is inside the letter (even-odd rule).
func inLetter(p point) bool {
	if p.x < 0 || p.x > 1 || p.y < letterTop || p.y > letterBottom {
		return false
	}
	inside := false
	for _, outline := range letterOutlines {
		for i, a := range outline {
			b := outline[(i+1)%len(outline)]
			if (a.y > p.y) != (b.y > p.y) && p.x < a.x+(p.y-a.y)*(b.x-a.x)/(b.y-a.y) {
				inside = !inside
			}
		}
	}
	return inside
}

// letterMask draws the letter as large as fits comfortably inside the
// keycap (left, top, right, bottom), centered there.
func letterMask(size int, left, top, right, bottom float64) *image.Alpha {
	// Leave a small margin to the frame on every side.
	pad := math.Max(1, float64(size)/16)
	boxW, boxH := right-left-2*pad, bottom-top-2*pad
	// Small sizes get the letter set a little heavier, by also filling it
	// shifted sideways by a fraction of a pixel: at 16px its strokes would
	// otherwise thin out to grey.
	embolden := math.Max(0, 0.6-float64(size)/80)

	scale := math.Min((boxW-embolden)/1, boxH/(letterBottom-letterTop))
	originX := left + pad + (boxW-embolden-scale)/2
	originY := top + pad + (boxH-scale*(letterBottom-letterTop))/2 - scale*letterTop

	return coverage(size, func(x, y float64) bool {
		p := point{(x - originX) / scale, (y - originY) / scale}
		if inLetter(p) {
			return true
		}
		return embolden > 0 && inLetter(point{(x - embolden - originX) / scale, p.y})
	})
}

// over blends src with alpha a over dst (both straight alpha).
func over(src color.NRGBA, a uint8, dst color.NRGBA) color.NRGBA {
	sa := float64(a) / 255
	da := float64(dst.A) / 255
	oa := sa + da*(1-sa)
	if oa == 0 {
		return color.NRGBA{}
	}
	mix := func(s, d uint8) uint8 {
		return uint8(math.Round((float64(s)*sa + float64(d)*da*(1-sa)) / oa))
	}
	return color.NRGBA{R: mix(src.R, dst.R), G: mix(src.G, dst.G), B: mix(src.B, dst.B), A: uint8(math.Round(oa * 255))}
}
