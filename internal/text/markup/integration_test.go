package markup

import (
	"testing"

	"github.com/rdon-key/modgadget/internal/display"
	efont16mgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/efont16"
	efont24mgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/efont24"
	shinonomemgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/shinonome12"
	"github.com/rdon-key/modgadget/internal/text"
)

type integrationSink struct {
	rects     [4]display.Rect
	rectCount int
	writes    int
	bytes     int
}

func (sink *integrationSink) Size() (int16, int16) { return 320, 240 }
func (sink *integrationSink) BeginRect(x, y, width, height int16) error {
	sink.rects[sink.rectCount] = display.Rect{X: x, Y: y, Width: width, Height: height}
	sink.rectCount++
	return nil
}
func (sink *integrationSink) WritePixels(data []byte) error {
	sink.writes++
	sink.bytes += len(data)
	return nil
}
func (sink *integrationSink) EndRect() error { return nil }
func (sink *integrationSink) reset()         { *sink = integrationSink{} }

func integrationParser() Parser {
	return Parser{
		Fonts: Fonts{
			Size12: text.MGFFont{Font: shinonomemgf.Font},
			Size16: text.MGFFont{Font: efont16mgf.Font},
			Size24: text.MGFFont{Font: efont24mgf.Font},
		},
		Foreground: display.ColorWhite,
		Background: display.ColorBlack,
	}
}

func TestRealFontIntegration(t *testing.T) {
	parser := integrationParser()
	value := "小<size=16>中</size><size=24><fg=#ff0000>大</fg></size>小"
	var storage [8]text.Span
	spans, err := parser.ParseInto(storage[:0], value)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 4 {
		t.Fatalf("span count = %d", len(spans))
	}
	wantValues := [4]string{"小", "中", "大", "小"}
	wantAdvances := [4]int16{12, 16, 24, 12}
	for index := range spans {
		if spans[index].Value != wantValues[index] {
			t.Fatalf("span %d value = %q", index, spans[index].Value)
		}
		var r rune
		for _, r = range spans[index].Value {
			break
		}
		glyph, ok := spans[index].Font.Lookup(r)
		if !ok || glyph.AdvanceX != wantAdvances[index] {
			t.Fatalf("span %d glyph=%+v ok=%v", index, glyph, ok)
		}
	}
	if spans[2].Foreground != display.RGB565(255, 0, 0) {
		t.Fatalf("large foreground = %#04x", spans[2].Foreground)
	}

	measurement, err := text.MeasureLine(spans)
	if err != nil {
		t.Fatal(err)
	}
	if measurement.Ascent != 22 || measurement.Descent != 2 || measurement.LineGap != 0 || measurement.AdvanceY != 24 || measurement.Advance != 64 {
		t.Fatalf("measurement = %+v", measurement)
	}

	sink := &integrationSink{}
	scratch := [48]byte{}
	pen, err := text.DrawSpans(sink, spans, 0, 30, scratch[:])
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

func TestRealFontIntegrationAllocations(t *testing.T) {
	parser := integrationParser()
	value := "小<size=16>中</size><size=24><fg=#ff0000>大</fg></size>小"
	var storage [8]text.Span
	spans, err := parser.ParseInto(storage[:0], value)
	if err != nil {
		t.Fatal(err)
	}
	sink := &integrationSink{}
	scratch := [48]byte{}
	tests := []struct {
		name string
		call func()
	}{
		{"ParseInto", func() {
			result, err := parser.ParseInto(storage[:0], value)
			if err != nil || len(result) != 4 {
				panic("parse")
			}
		}},
		{"MeasureLine", func() {
			if _, err := text.MeasureLine(spans); err != nil {
				panic(err)
			}
		}},
		{"DrawSpans", func() {
			sink.reset()
			if _, err := text.DrawSpans(sink, spans, 0, 30, scratch[:]); err != nil {
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
