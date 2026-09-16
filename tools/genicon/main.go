// Command genicon renders the keyicon glyph (the same picture the
// runtime tray icon shows) as a multi-resolution .ico file, for embedding
// as the .exe file icon of every program in cmd/. It has no OS
// dependency and runs on any platform.
//
// Usage:
//
//	go run ./tools/genicon undeadkeys.ico
//
// The resulting .ico is then embedded as a Windows resource with
// akavel/rsrc, once per cmd directory that should carry it, e.g.:
//
//	go run github.com/akavel/rsrc@latest -ico undeadkeys.ico -arch amd64 -o cmd/undeadkeys/rsrc_windows_amd64.syso
//
// `go build` picks up a *_windows_amd64.syso file automatically, no
// other wiring needed.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"

	"windows-us-international-keyboard-without-dead-keys/internal/keyicon"
)

// sizes are the frames baked into the .ico, covering everything from a
// taskbar-scale icon at 100% and the scaled variants Windows asks for, up
// to Explorer's "extra large" thumbnail view. Each is rendered at its own
// size rather than scaled from another, so every one is sharp.
var sizes = []int{16, 20, 24, 32, 40, 48, 64, 256}

// iconDir and iconDirEntry are the ICO container's header and per-frame
// directory records; encoding/binary writes their fields packed, in
// order, exactly as the format lays them out.
type iconDir struct {
	Reserved uint16
	Type     uint16 // 1 = icon
	Count    uint16
}

type iconDirEntry struct {
	Width, Height byte // 0 means 256
	ColorCount    byte
	Reserved      byte
	Planes        uint16
	BitCount      uint16
	BytesInRes    uint32
	ImageOffset   uint32
}

func render(size int) *image.NRGBA {
	return keyicon.Render(size, true)
}

// encodeICO packs the given square images, each a distinct size, as a
// Vista+-style ICO with PNG-compressed frames.
func encodeICO(imgs []*image.NRGBA) ([]byte, error) {
	var frames [][]byte
	for _, img := range imgs {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
		frames = append(frames, buf.Bytes())
	}

	// Writes to a bytes.Buffer never fail, so binary.Write's error is
	// safe to ignore below.
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, iconDir{Type: 1, Count: uint16(len(imgs))})

	offset := uint32(binary.Size(iconDir{}) + binary.Size(iconDirEntry{})*len(imgs))
	for i, img := range imgs {
		side := img.Bounds().Dx()
		dim := byte(side)
		if side >= 256 {
			dim = 0
		}
		binary.Write(&out, binary.LittleEndian, iconDirEntry{
			Width:       dim,
			Height:      dim,
			Planes:      1,
			BitCount:    32,
			BytesInRes:  uint32(len(frames[i])),
			ImageOffset: offset,
		})
		offset += uint32(len(frames[i]))
	}
	for _, f := range frames {
		out.Write(f)
	}
	return out.Bytes(), nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: genicon <output.ico>")
		os.Exit(1)
	}

	var imgs []*image.NRGBA
	for _, s := range sizes {
		imgs = append(imgs, render(s))
	}

	data, err := encodeICO(imgs)
	if err == nil {
		err = os.WriteFile(os.Args[1], data, 0o666)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
