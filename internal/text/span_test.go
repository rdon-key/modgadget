package text

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget-fonts/font"
	"github.com/rdon-key/modgadget/internal/display"
)

func TestMeasureSpansEmpty(t *testing.T) {
	for _, spans := range [][]Span{nil, {}} {
		got, err := MeasureSpans(spans)
		if err != nil || got != (Measurement{}) {
			t.Fatalf("measurement=%+v err=%v", got, err)
		}
	}
}

func TestMeasureSpansAcrossFaces(t *testing.T) {
	faceA := spanFace(font.Metrics{Ascent: 8, Descent: 2}, []font.GlyphInfo{
		{Rune: ' ', AdvanceX: 2},
		{Rune: 'A', Width: 3, Height: 4, AdvanceX: 5, BearingX: 1, BearingY: 3},
	}, strings.Repeat("\x00", 4))
	faceB := spanFace(font.Metrics{Ascent: 20, Descent: 7}, []font.GlyphInfo{
		{Rune: 'B', Width: 2, Height: 3, AdvanceX: 1, BearingX: -2},
		{Rune: 'C', BitmapOffset: 3, Width: 1, Height: 2, AdvanceX: -2, BearingX: 1, BearingY: -1},
	}, strings.Repeat("\x00", 5))
	spans := []Span{
		{Font: faceA, Value: "A"},
		{Font: faceA, Value: ""},
		{Font: faceA, Value: " "},
		{Font: faceB, Value: "BC"},
	}
	want := Measurement{Advance: 6, Bounds: Bounds{MinX: 1, MinY: -3, MaxX: 10, MaxY: 3}, HasInk: true}
	got, err := MeasureSpans(spans)
	if err != nil || got != want {
		t.Fatalf("measurement=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureSpansWhitespaceOnly(t *testing.T) {
	face := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: ' ', AdvanceX: 3}}, "")
	got, err := MeasureSpans([]Span{{Font: face, Value: "  "}, {Font: face, Value: ""}})
	if err != nil || got.Advance != 6 || got.HasInk || got.Bounds != (Bounds{}) {
		t.Fatalf("measurement=%+v err=%v", got, err)
	}
}

func TestStyleZeroValueAndCopyPreserveBold(t *testing.T) {
	if (Style{}).Bold {
		t.Fatal("zero-value Style is bold")
	}
	original := Style{Bold: true}
	copy := original
	if !copy.Bold {
		t.Fatal("Style copy lost Bold")
	}
}

func TestBoldSpanMeasurementAndDrawing(t *testing.T) {
	face := spanFace(font.Metrics{}, []font.GlyphInfo{
		{Rune: 'a', Width: 3, Height: 1, AdvanceX: 3},
		{Rune: 'b', BitmapOffset: 1, Width: 3, Height: 1, AdvanceX: 3},
	}, "\x80\x80")
	spans := []Span{
		{Font: face, Value: "a", Foreground: 0x1234, Background: 0xabcd, Bold: true},
		{Font: face, Value: "b", Foreground: 0x1234, Background: 0xabcd},
	}
	measurement, err := MeasureSpans(spans)
	if err != nil {
		t.Fatal(err)
	}
	if measurement.Advance != 6 || measurement.Bounds.MaxX != 6 {
		t.Fatalf("measurement=%+v", measurement)
	}

	backend := &fakeBackend{}
	pen, err := DrawSpans(backend, spans, 0, 0, make([]byte, 8))
	if err != nil {
		t.Fatal(err)
	}
	if pen != 6 {
		t.Fatalf("advance=%d, want 6", pen)
	}
	if len(backend.rects) != 2 || backend.rects[0].Width != 4 || backend.rects[1].Width != 3 {
		t.Fatalf("rects=%v", backend.rects)
	}
	wantBold := []byte{0x12, 0x34, 0x12, 0x34, 0xab, 0xcd, 0xab, 0xcd}
	wantNormal := []byte{0x12, 0x34, 0xab, 0xcd, 0xab, 0xcd}
	if string(backend.writes[0]) != string(wantBold) || string(backend.writes[1]) != string(wantNormal) {
		t.Fatalf("writes=%x want=%x/%x", backend.writes, wantBold, wantNormal)
	}
}

func TestBoldFinalGlyphInkFitsMeasurement(t *testing.T) {
	face := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'x', Width: 2, Height: 1, AdvanceX: 2}}, "\x80")
	measurement, err := MeasureSpans([]Span{{Font: face, Value: "x", Bold: true}})
	if err != nil {
		t.Fatal(err)
	}
	backend := &fakeBackend{}
	if _, err := DrawSpans(backend, []Span{{Font: face, Value: "x", Bold: true}}, 0, 0, make([]byte, 6)); err != nil {
		t.Fatal(err)
	}
	if got := unionRects(backend.rects); got != measurement.Bounds {
		t.Fatalf("drawn=%+v measurement=%+v", got, measurement)
	}
	if measurement.Advance != 2 || measurement.Bounds.MaxX != 3 {
		t.Fatalf("measurement=%+v", measurement)
	}
}

func TestBoldGlyphIsClippedByViewportBackend(t *testing.T) {
	face := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'x', Width: 2, Height: 1, AdvanceX: 2}}, "\x80")
	physical := &fakeBackend{}
	viewport, err := display.NewViewport(display.Rect{X: 10, Y: 20, Width: 2, Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	backend, err := display.NewViewportBackend(physical, viewport)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DrawSpans(backend, []Span{{Font: face, Value: "x", Bold: true}}, 1, 0, make([]byte, 6)); err != nil {
		t.Fatal(err)
	}
	if len(physical.rects) != 1 || physical.rects[0] != (display.Rect{X: 11, Y: 20, Width: 1, Height: 1}) {
		t.Fatalf("physical rects=%v", physical.rects)
	}
	if len(physical.writes) != 1 || len(physical.writes[0]) != 2 {
		t.Fatalf("physical writes=%x", physical.writes)
	}
}

func TestMeasureSpansPartialErrors(t *testing.T) {
	first := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'a', AdvanceX: 3}}, "")
	second := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'b', AdvanceX: 2}}, "")
	tests := []struct {
		name  string
		spans []Span
		want  int16
		text  string
	}{
		{"nil face", []Span{{Font: first, Value: "a"}, {Value: ""}}, 3, "span 1"},
		{"missing glyph", []Span{{Font: first, Value: "a"}, {Font: second, Value: "bz"}}, 5, "U+007A"},
		{"invalid UTF-8", []Span{{Font: first, Value: "a"}, {Font: second, Value: string([]byte{0xff})}}, 3, "span 1"},
		{"span boundary advance overflow", []Span{{Font: spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'x', AdvanceX: math.MaxInt16}}, ""), Value: "x"}, {Font: spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'y', AdvanceX: 1}}, ""), Value: "y"}}, math.MaxInt16, "U+0079"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MeasureSpans(tt.spans)
			if err == nil || got.Advance != tt.want || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("measurement=%+v err=%v", got, err)
			}
		})
	}
}

func TestDrawSpansEmpty(t *testing.T) {
	for _, spans := range [][]Span{nil, {}} {
		backend := &fakeBackend{}
		pen, err := DrawSpans(backend, spans, -4, 9, nil)
		if err != nil || pen != -4 || backend.beginCalls != 0 {
			t.Fatalf("pen=%d err=%v calls=%d", pen, err, backend.beginCalls)
		}
	}
	pen, err := DrawSpans(nil, nil, 7, 9, nil)
	if err != nil || pen != 7 {
		t.Fatalf("nil backend with no spans: pen=%d err=%v", pen, err)
	}
}

func TestDrawSpansSwitchesFacesAndColorsAndReusesScratch(t *testing.T) {
	faceA := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'a', Width: 2, Height: 1, AdvanceX: 3, BearingY: 1}}, "\x80")
	faceB := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'b', Width: 2, Height: 1, AdvanceX: 4, BearingX: 1, BearingY: -1}}, "\x80")
	spans := []Span{
		{Font: faceA, Value: "a", Foreground: 0x1234, Background: 0xabcd},
		{Font: faceB, Value: "b", Foreground: 0x5678, Background: 0x9abc},
	}
	backend := &fakeBackend{}
	scratch := make([]byte, 4)
	pen, err := DrawSpans(backend, spans, 0, 0, scratch)
	if err != nil || pen != 7 {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
	wantRects := []display.Rect{{X: 0, Y: -1, Width: 2, Height: 1}, {X: 4, Y: 1, Width: 2, Height: 1}}
	if len(backend.rects) != 2 || backend.rects[0] != wantRects[0] || backend.rects[1] != wantRects[1] {
		t.Fatalf("rects=%v want %v", backend.rects, wantRects)
	}
	wantWrites := [][]byte{{0x12, 0x34, 0xab, 0xcd}, {0x56, 0x78, 0x9a, 0xbc}}
	if len(backend.writes) != 2 {
		t.Fatalf("writes=%x", backend.writes)
	}
	for index := range wantWrites {
		if string(backend.writes[index]) != string(wantWrites[index]) {
			t.Fatalf("write %d=%x want %x", index, backend.writes[index], wantWrites[index])
		}
	}
	if string(scratch) != string(wantWrites[1]) {
		t.Fatalf("scratch=%x want %x", scratch, wantWrites[1])
	}
}

func TestDrawSpansWhitespaceAndPartialErrors(t *testing.T) {
	space := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: ' ', AdvanceX: 3}}, "")
	drawn := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 2}}, "\x80")
	backend := &fakeBackend{}
	pen, err := DrawSpans(backend, []Span{{Font: space, Value: " "}, {Font: drawn, Value: "a"}}, 5, 0, make([]byte, 2))
	if err != nil || pen != 10 || len(backend.rects) != 1 || backend.rects[0].X != 8 {
		t.Fatalf("pen=%d rects=%v err=%v", pen, backend.rects, err)
	}

	sentinel := errors.New("backend failure")
	backend = &fakeBackend{writeErr: sentinel}
	pen, err = DrawSpans(backend, []Span{{Font: space, Value: " "}, {Font: drawn, Value: "a"}}, 5, 0, make([]byte, 2))
	if pen != 8 || !errors.Is(err, sentinel) {
		t.Fatalf("pen=%d err=%v", pen, err)
	}

	pen, err = DrawSpans(&fakeBackend{}, []Span{{Font: space, Value: " "}, {Font: drawn, Value: "az"}}, 5, 0, make([]byte, 2))
	if pen != 10 || err == nil || !strings.Contains(err.Error(), "U+007A") {
		t.Fatalf("pen=%d err=%v", pen, err)
	}

	pen, err = DrawSpans(&fakeBackend{}, []Span{{Font: space, Value: " "}, {Value: ""}}, 5, 0, nil)
	if pen != 8 || err == nil || !strings.Contains(err.Error(), "span 1") {
		t.Fatalf("nil face: pen=%d err=%v", pen, err)
	}
}

func TestDrawSpansOverflow(t *testing.T) {
	tests := []struct {
		name  string
		spans []Span
		pen   int16
	}{
		{"coordinate", []Span{{Font: spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, BearingX: 1}}, "\x80"), Value: "a"}}, math.MaxInt16},
		{"advance", []Span{{Font: spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'a', AdvanceX: 1}}, ""), Value: "a"}}, math.MaxInt16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DrawSpans(&fakeBackend{}, tt.spans, tt.pen, 0, make([]byte, 2))
			if err == nil || got != tt.pen || !strings.Contains(err.Error(), "U+0061") {
				t.Fatalf("pen=%d err=%v", got, err)
			}
		})
	}
}

func TestSpanMeasurementAndDrawingAgree(t *testing.T) {
	face := measurementFace()
	spans := []Span{{Font: face, Value: "A "}, {Font: face, Value: "B"}, {Font: face, Value: " C"}}
	measurement, err := MeasureSpans(spans)
	if err != nil {
		t.Fatal(err)
	}
	backend := &fakeBackend{}
	pen, err := DrawSpans(backend, spans, 0, 0, make([]byte, 30))
	if err != nil || pen != measurement.Advance {
		t.Fatalf("pen=%d measurement=%+v err=%v", pen, measurement, err)
	}
	if unionRects(backend.rects) != measurement.Bounds {
		t.Fatalf("rect bounds=%+v measurement=%+v", unionRects(backend.rects), measurement.Bounds)
	}
}

func TestSingleSpanAPIsAgree(t *testing.T) {
	face := measurementFace()
	span := Span{Font: face, Value: "A", Foreground: 0x1234, Background: 0xabcd}
	stringMeasurement, err := MeasureString(face, span.Value)
	if err != nil {
		t.Fatal(err)
	}
	spanMeasurement, err := MeasureSpans([]Span{span})
	if err != nil || spanMeasurement != stringMeasurement {
		t.Fatalf("string=%+v span=%+v err=%v", stringMeasurement, spanMeasurement, err)
	}
	stringBackend, spanBackend := &fakeBackend{}, &fakeBackend{}
	stringPen, stringErr := DrawString(stringBackend, face, 0, 0, span.Value, span.Foreground, span.Background, make([]byte, 30))
	spanPen, spanErr := DrawSpans(spanBackend, []Span{span}, 0, 0, make([]byte, 30))
	if stringErr != nil || spanErr != nil || stringPen != spanPen || stringPen != stringMeasurement.Advance {
		t.Fatalf("string pen=%d span pen=%d measurement=%+v errors=%v/%v", stringPen, spanPen, stringMeasurement, stringErr, spanErr)
	}
	if unionRects(stringBackend.rects) != unionRects(spanBackend.rects) {
		t.Fatalf("string rects=%v span rects=%v", stringBackend.rects, spanBackend.rects)
	}
}

type fixtureFont struct{ source *font.Font }

func (adapter fixtureFont) Lookup(r rune) (Glyph, bool) {
	glyph, ok := adapter.source.Lookup(r)
	if !ok {
		return Glyph{}, false
	}
	return Glyph{Width: glyph.Width, Height: glyph.Height, AdvanceX: glyph.AdvanceX, BearingX: glyph.BearingX, BearingY: glyph.BearingY, Bitmap: glyph.Bitmap}, true
}

func (adapter fixtureFont) Metrics() FontMetrics {
	metrics := adapter.source.Metrics()
	return FontMetrics{Ascent: metrics.Ascent, Descent: metrics.Descent, LineGap: metrics.LineGap}
}

func spanFace(metrics font.Metrics, glyphs []font.GlyphInfo, bitmap string) Font {
	face := font.New(metrics, glyphs, bitmap)
	return fixtureFont{source: &face}
}

func unionRects(rects []display.Rect) Bounds {
	var bounds Bounds
	for index, rect := range rects {
		rectBounds := Bounds{MinX: rect.X, MinY: rect.Y, MaxX: rect.X + rect.Width, MaxY: rect.Y + rect.Height}
		if index == 0 {
			bounds = rectBounds
		} else {
			bounds = unionBounds(bounds, rectBounds)
		}
	}
	return bounds
}
