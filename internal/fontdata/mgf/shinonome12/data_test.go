package shinonome12

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestEmbeddedFont(t *testing.T) {
	if len(data) != 288954 {
		t.Fatalf("data size = %d", len(data))
	}
	if hash := fmt.Sprintf("%x", sha256.Sum256([]byte(data))); hash != "3b2ee24462103e1bccbb4fbb1fcc943c61eb74b66dbae7120ff2463d74a0f136" {
		t.Fatalf("SHA-256 = %s", hash)
	}
	header := Font.Header()
	if string(header.FontID[:]) != "sh12" || string(header.SubsetID[:]) != "full" || header.Region != [2]byte{'J', 'P'} || header.GlyphCount != 6879 || header.Ascent != 10 || header.Descent != 2 || header.LineGap != 0 || header.MaxWidth != 12 || header.MaxHeight != 12 || header.FileSize != 288954 {
		t.Fatalf("header = %+v", header)
	}
	for _, r := range []rune{'\\', '\u3005', '\u3042', '\u30a2', '\u65e5', '\uffe5'} {
		glyph, ok := Font.Lookup(r)
		if !ok {
			t.Fatalf("missing U+%04X", r)
		}
		if glyph.Width != 12 || glyph.Height != 12 || glyph.AdvanceX != 12 || glyph.BearingX != 0 || glyph.BearingY != 10 || len(glyph.Bitmap) != 24 {
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
			if _, ok := Font.Lookup('\u3042'); !ok {
				panic("missing")
			}
		}},
		{"lookup miss", func() {
			if _, ok := Font.Lookup('M'); ok {
				panic("unexpected")
			}
		}},
		{"header", func() {
			if Font.Header().GlyphCount != 6879 {
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
