package text

import (
	"testing"

	"github.com/rdon-key/modgadget/internal/display"
)

func markerFont(r rune, marker int16, metrics FontMetrics) *fixedFont {
	return &fixedFont{metrics: metrics, glyphs: [4]struct {
		r rune
		g Glyph
	}{{r: r, g: Glyph{AdvanceX: 1, BearingX: marker}}}}
}

func TestFontStackLookupOrderAndNil(t *testing.T) {
	primary := markerFont('p', 1, FontMetrics{})
	first := markerFont('a', 2, FontMetrics{})
	second := markerFont('b', 3, FontMetrics{})
	third := markerFont('c', 4, FontMetrics{})
	stack := FontStack{Primary: primary, Fallbacks: [3]Font{first, second, third}}

	for _, test := range []struct {
		r      rune
		marker int16
	}{{'p', 1}, {'a', 2}, {'b', 3}, {'c', 4}} {
		glyph, ok := stack.Lookup(test.r)
		if !ok || glyph.BearingX != test.marker {
			t.Fatalf("Lookup(%q) = %+v, %v", test.r, glyph, ok)
		}
	}
	if _, ok := stack.Lookup('x'); ok {
		t.Fatal("all-font miss succeeded")
	}

	sharedPrimary := markerFont('s', 10, FontMetrics{})
	sharedFirst := markerFont('s', 20, FontMetrics{})
	stack = FontStack{Primary: sharedPrimary, Fallbacks: [3]Font{sharedFirst}}
	if glyph, ok := stack.Lookup('s'); !ok || glyph.BearingX != 10 {
		t.Fatalf("primary precedence = %+v, %v", glyph, ok)
	}
	stack = FontStack{Fallbacks: [3]Font{nil, sharedFirst}}
	if glyph, ok := stack.Lookup('s'); !ok || glyph.BearingX != 20 {
		t.Fatalf("nil primary/fallback handling = %+v, %v", glyph, ok)
	}
	if glyph, ok := (FontStack{}).Lookup('s'); ok || glyph != (Glyph{}) {
		t.Fatalf("zero stack = %+v, %v", glyph, ok)
	}

	if allocations := testing.AllocsPerRun(100, func() {
		_, _ = stack.Lookup('s')
		_, _ = stack.Lookup('x')
	}); allocations != 0 {
		t.Fatalf("Lookup allocations = %v", allocations)
	}
}

func TestFontStackMetrics(t *testing.T) {
	primary := markerFont('p', 1, FontMetrics{Ascent: 4, Descent: 1, LineGap: 1})
	first := markerFont('a', 2, FontMetrics{Ascent: 8})
	second := markerFont('b', 3, FontMetrics{Descent: 3})
	third := markerFont('c', 4, FontMetrics{LineGap: 2})
	stack := FontStack{Primary: primary, Fallbacks: [3]Font{first, second, third}}
	if got := stack.Metrics(); got != (FontMetrics{Ascent: 8, Descent: 3, LineGap: 2}) {
		t.Fatalf("metrics = %+v", got)
	}
	if got := (FontStack{Primary: primary}).Metrics(); got != primary.Metrics() {
		t.Fatalf("primary metrics = %+v", got)
	}
	if got := (FontStack{}).Metrics(); got != (FontMetrics{}) {
		t.Fatalf("zero metrics = %+v", got)
	}
}

func TestStyleSetLookup(t *testing.T) {
	first := Style{Font: markerFont('x', 1, FontMetrics{}), Foreground: display.ColorWhite}
	middle := Style{Font: markerFont('x', 2, FontMetrics{}), Background: display.ColorBlack}
	last := Style{Font: FontStack{Primary: markerFont('x', 3, FontMetrics{})}}
	duplicate := Style{Font: markerFont('x', 4, FontMetrics{})}
	styles := StyleSet{Default: duplicate, Entries: []StyleEntry{
		{Name: "first", Style: first},
		{Name: "middle", Style: middle},
		{Name: "last", Style: last},
		{Name: "first", Style: duplicate},
	}}
	for _, test := range []struct {
		name   string
		marker int16
	}{{"first", 1}, {"middle", 2}, {"last", 3}} {
		style, ok := styles.Lookup(test.name)
		if !ok {
			t.Fatalf("Lookup(%q) missed", test.name)
		}
		glyph, ok := style.Font.Lookup('x')
		if !ok || glyph.BearingX != test.marker {
			t.Fatalf("Lookup(%q) glyph = %+v, %v", test.name, glyph, ok)
		}
	}
	if _, ok := styles.Lookup("First"); ok {
		t.Fatal("Lookup is not case-sensitive")
	}
	if _, ok := styles.Lookup("unknown"); ok {
		t.Fatal("unknown style succeeded")
	}
	if _, ok := styles.Lookup("default"); ok {
		t.Fatal("Default was exposed as a named style")
	}
	if allocations := testing.AllocsPerRun(100, func() {
		_, _ = styles.Lookup("middle")
		_, _ = styles.Lookup("unknown")
	}); allocations != 0 {
		t.Fatalf("Lookup allocations = %v", allocations)
	}
}
