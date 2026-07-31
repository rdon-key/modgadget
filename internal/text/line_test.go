package text

import (
	"math"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget-fonts/font"
)

func TestMeasureLineEmpty(t *testing.T) {
	for _, spans := range [][]Span{nil, {}} {
		got, err := MeasureLine(spans)
		if err != nil || got != (LineMeasurement{}) {
			t.Fatalf("line=%+v err=%v", got, err)
		}
	}
}

func TestMeasureLineEmptyStringIncludesMetrics(t *testing.T) {
	face := spanFace(font.Metrics{Ascent: 8, Descent: 2, LineGap: 1}, nil, "")
	want := LineMeasurement{Ascent: 8, Descent: 2, LineGap: 1, AdvanceY: 11}
	got, err := MeasureLine([]Span{{Font: face, Value: ""}})
	if err != nil || got != want {
		t.Fatalf("line=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureLineSelectsMaximumMetricsAndMeasuresInk(t *testing.T) {
	faceA := spanFace(font.Metrics{Ascent: 8, Descent: 1, LineGap: 0}, []font.GlyphInfo{
		{Rune: 'a', Width: 1, Height: 14, AdvanceX: 4, BearingY: 10},
	}, strings.Repeat("\x00", 14))
	faceB := spanFace(font.Metrics{Ascent: 6, Descent: 3, LineGap: 1}, []font.GlyphInfo{
		{Rune: 'b', Width: 1, Height: 1, AdvanceX: -2, BearingX: -1},
	}, "\x00")
	lineGapFace := spanFace(font.Metrics{Ascent: 7, Descent: 2, LineGap: 2}, nil, "")
	spans := []Span{{Font: faceA, Value: "a"}, {Font: lineGapFace, Value: ""}, {Font: faceB, Value: "b"}}
	want := LineMeasurement{
		Measurement: Measurement{Advance: 2, Bounds: Bounds{MinX: 0, MinY: -10, MaxX: 4, MaxY: 4}, HasInk: true},
		Ascent:      8, Descent: 3, LineGap: 2, AdvanceY: 13,
	}
	got, err := MeasureLine(spans)
	if err != nil || got != want {
		t.Fatalf("line=%+v want %+v err=%v", got, want, err)
	}
	measurement, err := MeasureSpans(spans)
	if err != nil || got.Measurement != measurement {
		t.Fatalf("line measurement=%+v spans=%+v err=%v", got.Measurement, measurement, err)
	}
}

func TestMeasureLineSelectsMaximumNegativeMetrics(t *testing.T) {
	faceA := spanFace(font.Metrics{Ascent: -5, Descent: -2, LineGap: -4}, nil, "")
	faceB := spanFace(font.Metrics{Ascent: -3, Descent: -6, LineGap: -1}, nil, "")
	want := LineMeasurement{Ascent: -3, Descent: -2, LineGap: -1, AdvanceY: -6}
	got, err := MeasureLine([]Span{{Font: faceA, Value: ""}, {Font: faceB, Value: ""}})
	if err != nil || got != want || got.Measurement != (Measurement{}) || got.HasInk || got.Advance != 0 {
		t.Fatalf("line=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureLineWhitespaceHasAdvanceWithoutInk(t *testing.T) {
	face := spanFace(font.Metrics{Ascent: 5, Descent: 2, LineGap: 1}, []font.GlyphInfo{{Rune: ' ', AdvanceX: 3}}, "")
	got, err := MeasureLine([]Span{{Font: face, Value: "  "}})
	want := LineMeasurement{Measurement: Measurement{Advance: 6}, Ascent: 5, Descent: 2, LineGap: 1, AdvanceY: 8}
	if err != nil || got != want {
		t.Fatalf("line=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureLineValidationPartialResults(t *testing.T) {
	first := spanFace(font.Metrics{Ascent: 2, Descent: 1}, []font.GlyphInfo{{Rune: 'a', AdvanceX: 3}}, "")
	second := spanFace(font.Metrics{Ascent: 4, LineGap: 1}, []font.GlyphInfo{{Rune: 'b', AdvanceX: 2}}, "")
	prior := LineMeasurement{Measurement: Measurement{Advance: 3}, Ascent: 2, Descent: 1, AdvanceY: 3}
	withSecondMetrics := LineMeasurement{Measurement: Measurement{Advance: 5}, Ascent: 4, Descent: 1, LineGap: 1, AdvanceY: 6}
	tests := []struct {
		name  string
		spans []Span
		want  LineMeasurement
		text  string
	}{
		{"nil face", []Span{{Font: first, Value: "a"}, {Value: ""}}, prior, "span 1"},
		{"invalid UTF-8", []Span{{Font: first, Value: "a"}, {Font: second, Value: string([]byte{0xff})}}, prior, "span 1"},
		{"missing glyph", []Span{{Font: first, Value: "a"}, {Font: second, Value: "bz"}}, withSecondMetrics, "U+007A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MeasureLine(tt.spans)
			if err == nil || got != tt.want || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("line=%+v want %+v err=%v", got, tt.want, err)
			}
		})
	}
}

func TestMeasureLineGlyphAndHorizontalOverflow(t *testing.T) {
	maxAdvance := spanFace(font.Metrics{Ascent: 1}, []font.GlyphInfo{{Rune: 'a', AdvanceX: math.MaxInt16}}, "")
	coordinate := spanFace(font.Metrics{Descent: 1}, []font.GlyphInfo{{Rune: 'b', Width: 1, Height: 1, BearingX: 1}}, "\x00")
	advance := spanFace(font.Metrics{Descent: 1}, []font.GlyphInfo{{Rune: 'c', AdvanceX: 1}}, "")
	tests := []struct {
		name string
		face Font
		char string
		rune string
	}{
		{"glyph coordinate overflow", coordinate, "b", "U+0062"},
		{"horizontal advance overflow", advance, "c", "U+0063"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MeasureLine([]Span{{Font: maxAdvance, Value: "a"}, {Font: tt.face, Value: tt.char}})
			if err == nil || got.Advance != math.MaxInt16 || !strings.Contains(err.Error(), tt.rune) {
				t.Fatalf("line=%+v err=%v", got, err)
			}
		})
	}
}

func TestMeasureLineAdvanceYOverflowReturnsPriorLine(t *testing.T) {
	first := spanFace(font.Metrics{Ascent: math.MaxInt16}, nil, "")
	second := spanFace(font.Metrics{Descent: 1}, nil, "")
	want := LineMeasurement{Ascent: math.MaxInt16, AdvanceY: math.MaxInt16}
	got, err := MeasureLine([]Span{{Font: first, Value: ""}, {Font: second, Value: ""}})
	if err == nil || got != want || !strings.Contains(err.Error(), "span 1") {
		t.Fatalf("line=%+v want %+v err=%v", got, want, err)
	}
}
