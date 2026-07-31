package text

import (
	"strings"
	"testing"

	"github.com/rdon-key/modgadget/internal/display"
	efont16mgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/efont16"
	efont24mgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/efont24"
	shinonomemgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/shinonome12"
)

type efontSink struct {
	rects     [4]display.Rect
	rectCount int
	writes    int
	bytes     int
}

func (sink *efontSink) Size() (int16, int16) { return 320, 240 }
func (sink *efontSink) BeginRect(x, y, width, height int16) error {
	sink.rects[sink.rectCount] = display.Rect{X: x, Y: y, Width: width, Height: height}
	sink.rectCount++
	return nil
}
func (sink *efontSink) WritePixels(data []byte) error {
	sink.writes++
	sink.bytes += len(data)
	return nil
}
func (sink *efontSink) EndRect() error { return nil }
func (sink *efontSink) reset()         { *sink = efontSink{} }

func embeddedSizeFonts() (MGFFont, MGFFont, MGFFont) {
	return NewMGFFont(shinonomemgf.Font), NewMGFFont(efont16mgf.Font), NewMGFFont(efont24mgf.Font)
}

func TestEfontMetricsAndMixedSizeLine(t *testing.T) {
	font12, font16, font24 := embeddedSizeFonts()
	if metrics := font16.Metrics(); metrics != (FontMetrics{Ascent: 14, Descent: 2}) || metrics.LineHeight() != 16 {
		t.Fatalf("Efont 16 metrics = %+v", metrics)
	}
	if metrics := font24.Metrics(); metrics != (FontMetrics{Ascent: 22, Descent: 2}) || metrics.LineHeight() != 24 {
		t.Fatalf("Efont 24 metrics = %+v", metrics)
	}

	spans := []Span{
		{Font: font12, Value: "\u3042"},
		{Font: font16, Value: "\u3042"},
		{Font: font24, Value: "\u3042"},
		{Font: font12, Value: "\u3042"},
	}
	measurement, err := MeasureLine(spans)
	if err != nil {
		t.Fatal(err)
	}
	if measurement.Ascent != 22 || measurement.Descent != 2 || measurement.LineGap != 0 || measurement.AdvanceY != 24 || measurement.Advance != 64 {
		t.Fatalf("measurement = %+v", measurement)
	}

	sink := &efontSink{}
	scratch := [48]byte{}
	pen, err := DrawSpans(sink, spans, 0, 30, scratch[:])
	if err != nil || pen != 64 {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
	wantRects := [4]display.Rect{
		{X: 0, Y: 20, Width: 12, Height: 12},
		{X: 12, Y: 16, Width: 16, Height: 16},
		{X: 28, Y: 8, Width: 24, Height: 24},
		{X: 52, Y: 20, Width: 12, Height: 12},
	}
	if sink.rectCount != len(wantRects) || sink.rects != wantRects {
		t.Fatalf("rects=%+v count=%d", sink.rects, sink.rectCount)
	}
}

func TestEfontPenAdvanceAndScratch(t *testing.T) {
	font12, font16, font24 := embeddedSizeFonts()
	spans := []Span{{Font: font12, Value: "\u3042"}, {Font: font16, Value: "\u3042"}, {Font: font24, Value: "\u3042"}}
	sink := &efontSink{}
	scratch := [48]byte{}
	if pen, err := DrawSpans(sink, spans, 0, 30, scratch[:]); err != nil || pen != 52 {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
	short := [47]byte{}
	sink.reset()
	if pen, err := DrawSpans(sink, []Span{{Font: font24, Value: "\u3042"}}, 7, 30, short[:]); err == nil || pen != 7 || !strings.Contains(err.Error(), "have 47") || !strings.Contains(err.Error(), "need 48") {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
	sink.reset()
	if pen, err := DrawSpans(sink, []Span{{Font: font24, Value: "\u3042"}}, 7, 30, scratch[:]); err != nil || pen != 31 {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
}

func TestEfontRendererAllocations(t *testing.T) {
	font12, font16, font24 := embeddedSizeFonts()
	spans := []Span{{Font: font12, Value: "\u3042"}, {Font: font16, Value: "\u3042"}, {Font: font24, Value: "\u3042"}, {Font: font12, Value: "\u3042"}}
	sink := &efontSink{}
	scratch := [48]byte{}
	tests := []struct {
		name string
		call func()
	}{
		{"Efont 16 lookup", func() { _, _ = font16.Lookup('\u3042') }},
		{"Efont 24 lookup", func() { _, _ = font24.Lookup('\u3042') }},
		{"mixed measure", func() {
			if _, err := MeasureLine(spans); err != nil {
				panic(err)
			}
		}},
		{"mixed draw", func() {
			sink.reset()
			if _, err := DrawSpans(sink, spans, 0, 30, scratch[:]); err != nil {
				panic(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if allocations := testing.AllocsPerRun(50, test.call); allocations != 0 {
				t.Fatalf("allocations = %v", allocations)
			}
		})
	}
}
