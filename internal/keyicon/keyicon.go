// Package keyicon defines the glyph shared by the runtime tray icon
// (internal/tray) and the generated .exe file icon (tools/genicon), so
// both always draw the exact same shape: a keycap with an acute accent
// on it. It has no OS dependency - just the glyph's
// geometry - so it builds on any platform.
package keyicon

// GridSize is the logical resolution the glyph is authored at; other
// render sizes sample this grid via nearest-neighbor scaling (see
// AtScaled), which stays crisp because every edge in the glyph is an
// integer-multiple-friendly axis-aligned rectangle.
const GridSize = 32

// Part identifies which part of the glyph a grid cell belongs to, so the
// hollow inside of the keycap can be left clear while its border and the
// accent above it are drawn.
type Part int

const (
	PartNone Part = iota
	PartFrame
	PartCapInterior
	PartAccent
)

// At reports which part of the glyph (x, y) on a GridSize x GridSize
// logical grid belongs to: an outlined keycap with an acute accent stroke
// leaning right across the middle of it.
func At(x, y int) Part {
	switch {
	case y >= 14 && y <= 20 && x >= 13+(20-y) && x <= 16+(20-y):
		return PartAccent
	case x >= 5 && x <= 26 && y >= 8 && y <= 27:
		return PartCapInterior
	case x >= 3 && x <= 28 && y >= 6 && y <= 29:
		return PartFrame
	default:
		return PartNone
	}
}

// AtScaled maps pixel (x, y) on a size x size canvas down to the
// GridSize logical grid via nearest-neighbor scaling and reports its
// part.
func AtScaled(x, y, size int) Part {
	return At(x*GridSize/size, y*GridSize/size)
}
