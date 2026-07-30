package spleen8x16

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestEmbeddedFont(t *testing.T) {
	if len(data) != 32982 {
		t.Fatalf("data size = %d", len(data))
	}
	if hash := fmt.Sprintf("%x", sha256.Sum256([]byte(data))); hash != "0b78fcb25a1096801e4b90b0415f7bf8ad87b015ff9f9ef8124be7177139d3ba" {
		t.Fatalf("SHA-256 = %s", hash)
	}
	header := Font.Header()
	if string(header.FontID[:]) != "sp16" || string(header.SubsetID[:]) != "full" || header.Region != [2]byte{} || header.GlyphCount != 969 || header.Ascent != 12 || header.Descent != 4 || header.LineGap != 0 || header.MaxWidth != 8 || header.MaxHeight != 16 || header.FileSize != 32982 {
		t.Fatalf("header = %+v", header)
	}
	for _, r := range []rune{' ', 'A', 'M', 'a', '\u00a5', '\u00e9', '\ue0b3'} {
		glyph, ok := Font.Lookup(r)
		if !ok {
			t.Fatalf("missing U+%04X", r)
		}
		if glyph.Width != 8 || glyph.Height != 16 || glyph.AdvanceX != 8 || glyph.BearingX != 0 || glyph.BearingY != 12 || len(glyph.Bitmap) != 16 {
			t.Fatalf("U+%04X glyph=%+v bitmap=%d", r, glyph, len(glyph.Bitmap))
		}
	}
}

func TestAccessAllocations(t *testing.T) {
	tests := []struct {
		name string
		call func()
	}{
		{"lookup hit", func() {
			if _, ok := Font.Lookup('A'); !ok {
				panic("missing")
			}
		}},
		{"lookup miss", func() {
			if _, ok := Font.Lookup('\u3042'); ok {
				panic("unexpected")
			}
		}},
		{"header", func() {
			if Font.Header().GlyphCount != 969 {
				panic("header")
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if allocations := testing.AllocsPerRun(100, test.call); allocations != 0 {
				t.Fatalf("allocations = %v", allocations)
			}
		})
	}
}
