package efont16

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestEmbeddedFont(t *testing.T) {
	if len(data) != 1167336 {
		t.Fatalf("data size = %d", len(data))
	}
	if hash := fmt.Sprintf("%x", sha256.Sum256([]byte(data))); hash != "0cbbcc0b0a3845be11d5cd958c2ea092afa6fdd82be9ae82f6d1a87274e9ea16" {
		t.Fatalf("SHA-256 = %s", hash)
	}
	header := Font.Header()
	if string(header.FontID[:]) != "ef16" || string(header.SubsetID[:]) != "full" || header.Region != [2]byte{'J', 'P'} || header.GlyphCount != 24618 || header.Ascent != 14 || header.Descent != 2 || header.LineGap != 0 || header.MaxWidth != 16 || header.MaxHeight != 16 || header.FileSize != 1167336 {
		t.Fatalf("header = %+v", header)
	}
	checkGlyphs(t, []rune{' ', 'A', '\\'}, 8, 16, 8, 14, 16)
	checkGlyphs(t, []rune{'\u3042', '\u30a2', '\u65e5', '\uffe5'}, 16, 16, 16, 14, 32)
}

func checkGlyphs(t *testing.T, runes []rune, width, height, advance, bearingY int, bitmapBytes int) {
	t.Helper()
	for _, r := range runes {
		glyph, ok := Font.Lookup(r)
		if !ok {
			t.Fatalf("missing U+%04X", r)
		}
		if int(glyph.Width) != width || int(glyph.Height) != height || int(glyph.AdvanceX) != advance || glyph.BearingX != 0 || int(glyph.BearingY) != bearingY || len(glyph.Bitmap) != bitmapBytes {
			t.Fatalf("U+%04X glyph=%+v bitmap=%d", r, glyph, len(glyph.Bitmap))
		}
	}
}

func TestAccessAllocations(t *testing.T) {
	tests := []struct {
		name string
		call func()
	}{
		{"lookup hit", func() { _, _ = Font.Lookup('\u3042') }},
		{"lookup miss", func() { _, _ = Font.Lookup('\U0010ffff') }},
		{"header", func() { _ = Font.Header() }},
		{"glyph count", func() { _ = Font.GlyphCount() }},
		{"line height", func() { _ = Font.LineHeight() }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if allocations := testing.AllocsPerRun(100, test.call); allocations != 0 {
				t.Fatalf("allocations = %v", allocations)
			}
		})
	}
}
