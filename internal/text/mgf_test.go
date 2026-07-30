package text

import (
	"testing"

	"github.com/rdon-key/modgadget/internal/display"
	shinonomemgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/shinonome12"
	spleenmgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/spleen8x16"
)

func embeddedFallbackFont() (MGFFont, MGFFont, FallbackFont) {
	primary := MGFFont{Font: shinonomemgf.Font}
	fallback := MGFFont{Font: spleenmgf.Font}
	return primary, fallback, FallbackFont{Primary: primary, Fallback: fallback}
}

func TestMGFFontAdapter(t *testing.T) {
	primary, _, _ := embeddedFallbackFont()
	metrics := primary.Metrics()
	if metrics != (FontMetrics{Ascent: 10, Descent: 2}) {
		t.Fatalf("metrics = %+v", metrics)
	}
	glyph, ok := primary.Lookup('あ')
	if !ok || glyph.Width != 12 || glyph.Height != 12 || glyph.AdvanceX != 12 || glyph.BearingX != 0 || glyph.BearingY != 10 || len(glyph.Bitmap) != 24 {
		t.Fatalf("glyph=%+v ok=%v bitmap=%d", glyph, ok, len(glyph.Bitmap))
	}
	want, _ := shinonomemgf.Font.Lookup('あ')
	if glyph.Bitmap != want.Bitmap {
		t.Fatal("adapter bitmap differs from MGF bitmap")
	}
	if _, ok := primary.Lookup('M'); ok {
		t.Fatal("unexpected M")
	}
}

func TestFallbackFontLookupAndMetrics(t *testing.T) {
	primary, fallback, fonts := embeddedFallbackFont()
	if glyph, ok := fonts.Lookup('あ'); !ok || glyph.BearingY != 10 {
		t.Fatalf("primary glyph=%+v ok=%v", glyph, ok)
	}
	if glyph, ok := fonts.Lookup('M'); !ok || glyph.BearingY != 12 {
		t.Fatalf("fallback glyph=%+v ok=%v", glyph, ok)
	}
	shared, ok := fonts.Lookup('\\')
	primaryShared, primaryOK := primary.Lookup('\\')
	fallbackShared, fallbackOK := fallback.Lookup('\\')
	if !ok || !primaryOK || !fallbackOK || shared.Bitmap != primaryShared.Bitmap || shared.Bitmap == fallbackShared.Bitmap {
		t.Fatalf("primary precedence was not preserved")
	}
	if _, ok := fonts.Lookup('😀'); ok {
		t.Fatal("unexpected emoji")
	}
	metrics := fonts.Metrics()
	if metrics != (FontMetrics{Ascent: 12, Descent: 4}) || metrics.LineHeight() != 16 {
		t.Fatalf("metrics = %+v height=%d", metrics, metrics.LineHeight())
	}
}

type fixedFont struct {
	metrics FontMetrics
	glyphs  [4]struct {
		r rune
		g Glyph
	}
}

func (font *fixedFont) Lookup(r rune) (Glyph, bool) {
	for index := range font.glyphs {
		if font.glyphs[index].r == r {
			return font.glyphs[index].g, true
		}
	}
	return Glyph{}, false
}
func (font *fixedFont) Metrics() FontMetrics { return font.metrics }

type pixelSink struct {
	rects                    [8]display.Rect
	rectCount, writes, bytes int
	pixels                   [2048]byte
}

func (sink *pixelSink) Size() (int16, int16) { return 320, 240 }
func (sink *pixelSink) BeginRect(x, y, width, height int16) error {
	sink.rects[sink.rectCount] = display.Rect{X: x, Y: y, Width: width, Height: height}
	sink.rectCount++
	return nil
}
func (sink *pixelSink) WritePixels(data []byte) error {
	copy(sink.pixels[sink.bytes:], data)
	sink.bytes += len(data)
	sink.writes++
	return nil
}
func (sink *pixelSink) EndRect() error { return nil }
func (sink *pixelSink) reset()         { *sink = pixelSink{} }

func TestMGFBaselineBitmapAndAdvance(t *testing.T) {
	font := &fixedFont{glyphs: [4]struct {
		r rune
		g Glyph
	}{
		{'S', Glyph{Width: 8, Height: 1, AdvanceX: 8, BearingY: 12, Bitmap: "\x81"}},
		{'J', Glyph{Width: 9, Height: 1, AdvanceX: 12, BearingX: -2, BearingY: 10, Bitmap: "\x80\xff"}},
		{'Z', Glyph{AdvanceX: 5}},
	}}
	sink := &pixelSink{}
	scratch := [4]byte{}
	pen, err := drawFontValue(sink, font, 10, 12, "SJZ", 0xffff, 0, scratch[:])
	if err != nil || pen != 35 {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
	wantRects := []display.Rect{{X: 10, Y: 0, Width: 8, Height: 1}, {X: 16, Y: 2, Width: 9, Height: 1}}
	if sink.rectCount != len(wantRects) {
		t.Fatalf("rect count=%d", sink.rectCount)
	}
	for index := range wantRects {
		if sink.rects[index] != wantRects[index] {
			t.Fatalf("rect %d=%+v", index, sink.rects[index])
		}
	}
	if sink.writes != 17 {
		t.Fatalf("pixel writes=%d", sink.writes)
	}
	// Width 9 draws exactly nine pixels; padding bits in the second byte are ignored.
	if sink.bytes != (8+9)*2 {
		t.Fatalf("pixel bytes=%d", sink.bytes)
	}
	if sink.pixels[0] != 0xff || sink.pixels[1] != 0xff || sink.pixels[2] != 0 || sink.pixels[14] != 0xff {
		t.Fatalf("MSB-first pixels=%x", sink.pixels[:16])
	}
	if _, err := drawFontValue(sink, font, 0, 0, "?", 1, 0, scratch[:]); err == nil {
		t.Fatal("missing glyph did not preserve existing error behavior")
	}
}

func TestMGFMixedTextCommonBaseline(t *testing.T) {
	_, _, fonts := embeddedFallbackFont()
	sink := &pixelSink{}
	scratch := [4]byte{}
	pen, err := drawFontValue(sink, &fonts, 0, 12, "MあA日", 0xffff, 0, scratch[:])
	if err != nil || pen != 40 {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
	want := []display.Rect{{X: 0, Y: 0, Width: 8, Height: 16}, {X: 8, Y: 2, Width: 12, Height: 12}, {X: 20, Y: 0, Width: 8, Height: 16}, {X: 28, Y: 2, Width: 12, Height: 12}}
	if sink.rectCount != len(want) {
		t.Fatalf("rects=%d", sink.rectCount)
	}
	for index := range want {
		if sink.rects[index] != want[index] {
			t.Fatalf("rect %d=%+v want=%+v", index, sink.rects[index], want[index])
		}
	}
}

func TestMGFClipping(t *testing.T) {
	pixels := make([]byte, 4*2)
	surface, err := display.NewSurface(2, 2, pixels)
	if err != nil {
		t.Fatal(err)
	}
	viewport, err := display.NewViewport(display.Rect{Width: 2, Height: 2})
	if err != nil {
		t.Fatal(err)
	}
	backend, err := display.NewViewportBackend(surface, viewport)
	if err != nil {
		t.Fatal(err)
	}
	font := &fixedFont{glyphs: [4]struct {
		r rune
		g Glyph
	}{{'x', Glyph{Width: 3, Height: 2, AdvanceX: 3, BearingX: -1, BearingY: 1, Bitmap: "\xe0\xa0"}}}}
	scratch := [4]byte{}
	if _, err := drawFontValue(backend, font, 0, 1, "x", 0xffff, 0, scratch[:]); err != nil {
		t.Fatal(err)
	}
	if string(pixels) != "\xff\xff\xff\xff\x00\x00\xff\xff" {
		t.Fatalf("pixels=%x", pixels)
	}
}

func TestMGFRenderingAllocations(t *testing.T) {
	primary, _, fonts := embeddedFallbackFont()
	sink := &pixelSink{}
	scratch := [4]byte{}
	tests := []struct {
		name string
		call func()
	}{
		{"adapter hit", func() {
			if _, ok := primary.Lookup('あ'); !ok {
				panic("hit")
			}
		}},
		{"adapter miss", func() {
			if _, ok := primary.Lookup('M'); ok {
				panic("miss")
			}
		}},
		{"primary hit", func() {
			if _, ok := fonts.Lookup('あ'); !ok {
				panic("primary")
			}
		}},
		{"fallback hit", func() {
			if _, ok := fonts.Lookup('M'); !ok {
				panic("fallback")
			}
		}},
		{"total miss", func() {
			if _, ok := fonts.Lookup('😀'); ok {
				panic("miss")
			}
		}},
		{"metrics", func() {
			if fonts.Metrics().LineHeight() != 16 {
				panic("metrics")
			}
		}},
		{"draw glyph", func() {
			sink.reset()
			if _, err := drawFontValue(sink, &fonts, 0, 12, "あ", 1, 0, scratch[:]); err != nil {
				panic(err)
			}
		}},
		{"draw mixed", func() {
			sink.reset()
			if _, err := drawFontValue(sink, &fonts, 0, 12, "MあA日", 1, 0, scratch[:]); err != nil {
				panic(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(50, test.call); got != 0 {
				t.Fatalf("allocations=%v", got)
			}
		})
	}
}
