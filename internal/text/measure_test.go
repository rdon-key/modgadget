package text

import (
	"math"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget-fonts/font"
)

func TestMeasureStringEmpty(t *testing.T) {
	got, err := MeasureString(newFace(nil, ""), "")
	if err != nil || got != (Measurement{}) {
		t.Fatalf("measurement=%+v err=%v", got, err)
	}
}

func TestMeasureStringOneGlyph(t *testing.T) {
	face := newFace([]font.GlyphInfo{{Rune: 'a', Width: 3, Height: 5, AdvanceX: 4, BearingX: 1, BearingY: 4}}, strings.Repeat("\x00", 5))
	want := Measurement{Advance: 4, Bounds: Bounds{MinX: 1, MinY: -4, MaxX: 4, MaxY: 1}, HasInk: true}
	got, err := MeasureString(face, "a")
	if err != nil || got != want {
		t.Fatalf("measurement=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureStringVariableGlyphsSpacesAndBearings(t *testing.T) {
	face := measurementFace()
	want := Measurement{Advance: 11, Bounds: Bounds{MinX: 1, MinY: -4, MaxX: 15, MaxY: 4}, HasInk: true}
	got, err := MeasureString(face, "A B C")
	if err != nil || got != want {
		t.Fatalf("measurement=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureStringSpacesHaveAdvanceWithoutInk(t *testing.T) {
	face := newFace([]font.GlyphInfo{{Rune: ' ', AdvanceX: 3}}, "")
	got, err := MeasureString(face, "  ")
	if err != nil || got.Advance != 6 || got.HasInk || got.Bounds != (Bounds{}) {
		t.Fatalf("measurement=%+v err=%v", got, err)
	}
}

func TestMeasureStringInkCanExtendLeftOfOriginAndBeyondAdvance(t *testing.T) {
	face := newFace([]font.GlyphInfo{{Rune: 'x', Width: 4, Height: 1, AdvanceX: -2, BearingX: -3}}, "\x00")
	got, err := MeasureString(face, "x")
	want := Measurement{Advance: -2, Bounds: Bounds{MinX: -3, MinY: 0, MaxX: 1, MaxY: 1}, HasInk: true}
	if err != nil || got != want {
		t.Fatalf("measurement=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureStringErrors(t *testing.T) {
	oneByte := "\x00"
	tests := []struct {
		name  string
		face  *font.Font
		value string
		rune  string
	}{
		{"missing glyph", newFace(nil, ""), "z", "U+007A"},
		{"invalid UTF-8", newFace(nil, ""), string([]byte{0xff}), ""},
		{"glyph X overflow after positive advance", newFace([]font.GlyphInfo{{Rune: 'a', AdvanceX: math.MaxInt16}, {Rune: 'b', Width: 1, Height: 1, BearingX: 1}}, oneByte), "ab", "U+0062"},
		{"glyph X overflow after negative advance", newFace([]font.GlyphInfo{{Rune: 'a', AdvanceX: math.MinInt16}, {Rune: 'b', Width: 1, Height: 1, BearingX: -1}}, oneByte), "ab", "U+0062"},
		{"glyph maximum X overflow", newFace([]font.GlyphInfo{{Rune: 'a', Width: math.MaxInt16, Height: 1, BearingX: 1}}, strings.Repeat("\x00", 4096)), "a", "U+0061"},
		{"glyph Y overflow from bearing", newFace([]font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, BearingY: math.MinInt16}}, oneByte), "a", "U+0061"},
		{"glyph maximum Y overflow", newFace([]font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, BearingY: -math.MaxInt16}}, oneByte), "a", "U+0061"},
		{"advance overflow", newFace([]font.GlyphInfo{{Rune: 'a', AdvanceX: math.MaxInt16}}, ""), "aa", "U+0061"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MeasureString(tt.face, tt.value)
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.rune != "" && !strings.Contains(err.Error(), tt.rune) {
				t.Fatalf("error lacks rune: %v", err)
			}
		})
	}
}

func TestMeasureStringMatchesDrawString(t *testing.T) {
	face := measurementFace()
	measurement, err := MeasureString(face, "A B C")
	if err != nil {
		t.Fatal(err)
	}
	backend := &fakeBackend{}
	pen, err := DrawString(backend, face, 0, 0, "A B C", 1, 0, make([]byte, 30))
	if err != nil {
		t.Fatal(err)
	}
	if measurement.Advance != pen {
		t.Fatalf("measurement advance=%d draw pen=%d", measurement.Advance, pen)
	}
	var drawn Bounds
	for i, rect := range backend.rects {
		bounds := Bounds{MinX: rect.X, MinY: rect.Y, MaxX: rect.X + rect.Width, MaxY: rect.Y + rect.Height}
		if i == 0 {
			drawn = bounds
		} else {
			drawn = unionBounds(drawn, bounds)
		}
	}
	if drawn != measurement.Bounds {
		t.Fatalf("drawn bounds=%+v measurement bounds=%+v", drawn, measurement.Bounds)
	}
}

func measurementFace() *font.Font {
	return newFace([]font.GlyphInfo{
		{Rune: ' ', AdvanceX: 3},
		{Rune: 'A', Width: 3, Height: 5, AdvanceX: 4, BearingX: 1, BearingY: 4},
		{Rune: 'B', BitmapOffset: 5, Width: 2, Height: 2, AdvanceX: 3, BearingX: -2},
		{Rune: 'C', BitmapOffset: 7, Width: 1, Height: 3, AdvanceX: -2, BearingX: 1, BearingY: -1},
	}, strings.Repeat("\x00", 10))
}
