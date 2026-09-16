// Package keyicon draws the glyph shared by the runtime tray icon
// (internal/tray) and the generated .exe file icon (tools/genicon), so
// both always show the exact same picture: a rounded keycap with an "Á"
// on it. Everything is rendered from shapes and a real typeface with
// anti-aliasing, at whatever size is asked for, so the icon stays smooth
// on any display scaling. It has no OS dependency and builds anywhere.
package keyicon

import (
	"image"
	"image/color"
	"math"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Letter is the character shown on the keycap.
const Letter = "Á"

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

var (
	parsedFont     *opentype.Font
	parsedFontErr  error
	parsedFontOnce sync.Once
)

// letterFont returns the typeface the letter is set in: Go Bold, a clean
// humanist sans that is compiled into the program, so the icon looks the
// same on every machine and needs no font files.
func letterFont() (*opentype.Font, error) {
	parsedFontOnce.Do(func() {
		parsedFont, parsedFontErr = opentype.Parse(gobold.TTF)
	})
	return parsedFont, parsedFontErr
}

// letterMask sets Letter as large as fits comfortably inside the keycap
// (left, top, right, bottom) and centers it there by its actual ink
// bounds, accent included. If the embedded font can't be read - which
// would be a build problem, not a runtime one - the keycap is left empty.
func letterMask(size int, left, top, right, bottom float64) *image.Alpha {
	m := image.NewAlpha(image.Rect(0, 0, size, size))
	f, err := letterFont()
	if err != nil {
		return m
	}

	// Leave a small margin to the frame on every side.
	pad := math.Max(1, float64(size)/20)
	// Small sizes get the letter set a little heavier by drawing it twice,
	// shifted sideways by a fraction of a pixel: at 16px even a bold face
	// thins out to grey hairlines otherwise.
	embolden := math.Max(0, 0.6-float64(size)/80)
	boxW, boxH := right-left-2*pad, bottom-top-2*pad

	// Find the largest font size whose ink box fits, starting from one
	// that is certainly too big.
	var face font.Face
	var ink fixed.Rectangle26_6
	for pt := boxH * 1.4; pt > 1; pt *= 0.98 {
		fc, err := opentype.NewFace(f, &opentype.FaceOptions{Size: pt, DPI: 72, Hinting: font.HintingNone})
		if err != nil {
			return m
		}
		b, _ := font.BoundString(fc, Letter)
		w, h := fixedToFloat(b.Max.X-b.Min.X)+embolden, fixedToFloat(b.Max.Y-b.Min.Y)
		if w <= boxW && h <= boxH {
			face, ink = fc, b
			break
		}
		fc.Close()
	}
	if face == nil {
		return m
	}
	defer face.Close()

	// Place the pen so the ink box's center lands on the box's center.
	inkW, inkH := fixedToFloat(ink.Max.X-ink.Min.X)+embolden, fixedToFloat(ink.Max.Y-ink.Min.Y)
	originX := left + pad + (boxW-inkW)/2 - fixedToFloat(ink.Min.X)
	originY := top + pad + (boxH-inkH)/2 - fixedToFloat(ink.Min.Y)

	for _, dx := range []float64{0, embolden} {
		d := font.Drawer{
			Dst:  m,
			Src:  image.Opaque,
			Face: face,
			Dot:  fixed.Point26_6{X: floatToFixed(originX + dx), Y: floatToFixed(originY)},
		}
		d.DrawString(Letter)
		if embolden == 0 {
			break
		}
	}
	return m
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

func fixedToFloat(v fixed.Int26_6) float64 { return float64(v) / 64 }
func floatToFixed(v float64) fixed.Int26_6 { return fixed.Int26_6(math.Round(v * 64)) }
