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
	rects     [5]display.Rect
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
	return Parser{Styles: text.StyleSet{
		Default: text.Style{
			Font:       text.NewMGFFont(shinonomemgf.Font),
			Foreground: display.ColorWhite,
			Background: display.ColorBlack,
		},
		Entries: []text.StyleEntry{
			{Name: "medium", Style: text.Style{
				Font:       text.NewMGFFont(efont16mgf.Font),
				Foreground: display.RGB565(0, 255, 255),
				Background: display.ColorBlack,
			}},
			{Name: "large-red", Style: text.Style{
				Font:       text.NewMGFFont(efont24mgf.Font),
				Foreground: display.ColorRed,
				Background: display.ColorBlack,
			}},
			{Name: "inverse", Style: text.Style{
				Font:       text.NewMGFFont(shinonomemgf.Font),
				Foreground: display.ColorBlack,
				Background: display.ColorWhite,
			}},
		},
	}}
}

const integrationValue = "小<style=medium>中</style><style=large-red>大</style><style=inverse>小</style>小"

func TestRealFontIntegration(t *testing.T) {
	parser := integrationParser()
	var storage [8]text.Span
	spans, err := parser.ParseInto(storage[:0], integrationValue)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 5 {
		t.Fatalf("span count = %d", len(spans))
	}
	wantValues := [5]string{"小", "中", "大", "小", "小"}
	wantAdvances := [5]int16{12, 16, 24, 12, 12}
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
	if spans[1].Foreground != display.RGB565(0, 255, 255) || spans[2].Foreground != display.ColorRed {
		t.Fatalf("named foregrounds = %#04x/%#04x", spans[1].Foreground, spans[2].Foreground)
	}
	if spans[3].Foreground != display.ColorBlack || spans[3].Background != display.ColorWhite {
		t.Fatalf("inverse colors = %#04x/%#04x", spans[3].Foreground, spans[3].Background)
	}
	if spans[4].Foreground != display.ColorWhite || spans[4].Background != display.ColorBlack {
		t.Fatalf("default restoration colors = %#04x/%#04x", spans[4].Foreground, spans[4].Background)
	}

	measurement, err := text.MeasureLine(spans)
	if err != nil {
		t.Fatal(err)
	}
	if measurement.Ascent != 22 || measurement.Descent != 2 || measurement.LineGap != 0 || measurement.AdvanceY != 24 || measurement.Advance != 76 {
		t.Fatalf("measurement = %+v", measurement)
	}

	sink := &integrationSink{}
	scratch := [48]byte{}
	pen, err := text.DrawSpans(sink, spans, 0, 30, scratch[:])
	if err != nil || pen != 76 {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
	wantRects := [5]display.Rect{
		{X: 0, Y: 20, Width: 12, Height: 12},
		{X: 12, Y: 16, Width: 16, Height: 16},
		{X: 28, Y: 8, Width: 24, Height: 24},
		{X: 52, Y: 20, Width: 12, Height: 12},
		{X: 64, Y: 20, Width: 12, Height: 12},
	}
	if sink.rectCount != len(wantRects) || sink.rects != wantRects {
		t.Fatalf("rects=%+v count=%d", sink.rects, sink.rectCount)
	}
}

func TestRealFontIntegrationAllocations(t *testing.T) {
	parser := integrationParser()
	var storage [8]text.Span
	spans, err := parser.ParseInto(storage[:0], integrationValue)
	if err != nil {
		t.Fatal(err)
	}
	spans[0].Bold = true
	sink := &integrationSink{}
	scratch := [48]byte{}
	tests := []struct {
		name string
		call func()
	}{
		{"ParseInto", func() {
			result, err := parser.ParseInto(storage[:0], integrationValue)
			if err != nil || len(result) != 5 {
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
