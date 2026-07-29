package text

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget-fonts/font"
)

func TestTextLayoutZeroAndNilReceiver(t *testing.T) {
	var zero TextLayout
	if zero.Measurement() != (BlockMeasurement{}) || zero.LineCount() != 0 {
		t.Fatalf("zero layout: measurement=%+v lines=%d", zero.Measurement(), zero.LineCount())
	}
	baseline, err := zero.Draw(nil, 4, -3, nil)
	if err != nil || baseline != -3 {
		t.Fatalf("zero draw: baseline=%d err=%v", baseline, err)
	}

	var layout *TextLayout
	if layout.Measurement() != (BlockMeasurement{}) || layout.LineCount() != 0 {
		t.Fatalf("nil layout: measurement=%+v lines=%d", layout.Measurement(), layout.LineCount())
	}
	baseline, err = layout.Draw(nil, 4, 7, nil)
	if err != nil || baseline != 7 {
		t.Fatalf("nil draw: baseline=%d err=%v", baseline, err)
	}
}

func TestNewTextLayoutEmptyInputs(t *testing.T) {
	for _, spans := range [][]Span{nil, {}} {
		layout, err := NewTextLayout(spans)
		if err != nil || layout.lines != nil || layout.measurement != (BlockMeasurement{}) {
			t.Fatalf("layout=%+v err=%v", layout, err)
		}
	}
}

func TestTextLayoutMatchesExistingPipeline(t *testing.T) {
	face := spanFace(font.Metrics{Ascent: 2, Descent: 1}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 2, BearingY: 1}}, "\x80")
	spans := []Span{{Face: face, Value: "a\na\n", Foreground: 0x1234, Background: 0xabcd}}
	lines, err := LinesFromSpans(spans)
	if err != nil {
		t.Fatal(err)
	}
	want, err := MeasureLines(lines)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := NewTextLayout(spans)
	if err != nil || layout.LineCount() != 3 || layout.Measurement() != want {
		t.Fatalf("layout measurement=%+v lines=%d want=%+v err=%v", layout.Measurement(), layout.LineCount(), want, err)
	}
}

func TestTextLayoutDrawsRepeatedlyWithSavedStyles(t *testing.T) {
	face := spanFace(font.Metrics{Ascent: 2, Descent: 1}, []font.GlyphInfo{{Rune: 'a', Width: 2, Height: 1, AdvanceX: 3, BearingY: 1}}, "\x80")
	otherFace := spanFace(font.Metrics{Ascent: 9}, nil, "")
	spans := []Span{
		{Face: face, Value: "a", Foreground: 0x1234, Background: 0xabcd},
		{Face: face, Value: "a\na", Foreground: 0x5678, Background: 0x9abc},
	}
	layout, err := NewTextLayout(spans)
	if err != nil {
		t.Fatal(err)
	}
	wantMeasurement := layout.Measurement()
	spans[0].Face = otherFace
	spans[0].Value = "changed"
	spans[0].Foreground = 0
	spans[0].Background = 0
	spans[1] = Span{}
	if layout.Measurement() != wantMeasurement || layout.LineCount() != 2 {
		t.Fatalf("layout changed: measurement=%+v lines=%d", layout.Measurement(), layout.LineCount())
	}

	backends := []*fakeBackend{{}, {}}
	for _, backend := range backends {
		scratch := make([]byte, 4)
		baseline, err := layout.Draw(backend, 0, 0, scratch)
		if err != nil || baseline != wantMeasurement.AdvanceY {
			t.Fatalf("baseline=%d want=%d err=%v", baseline, wantMeasurement.AdvanceY, err)
		}
		if unionRects(backend.rects) != wantMeasurement.Bounds {
			t.Fatalf("rect bounds=%+v want=%+v", unionRects(backend.rects), wantMeasurement.Bounds)
		}
		wantWrites := [][]byte{
			{0x12, 0x34, 0xab, 0xcd},
			{0x56, 0x78, 0x9a, 0xbc},
			{0x56, 0x78, 0x9a, 0xbc},
		}
		if !reflect.DeepEqual(backend.writes, wantWrites) {
			t.Fatalf("writes=%x want=%x", backend.writes, wantWrites)
		}
		if string(scratch) != string(wantWrites[len(wantWrites)-1]) {
			t.Fatalf("scratch=%x", scratch)
		}
	}
	if !reflect.DeepEqual(backends[0].rects, backends[1].rects) || !reflect.DeepEqual(backends[0].writes, backends[1].writes) {
		t.Fatalf("draws differ: first=%v/%x second=%v/%x", backends[0].rects, backends[0].writes, backends[1].rects, backends[1].writes)
	}
}

func TestNewTextLayoutErrorsReturnZeroLayout(t *testing.T) {
	valid := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'a', AdvanceX: 1}}, "")
	lineOverflowA := spanFace(font.Metrics{Ascent: math.MaxInt16}, nil, "")
	lineOverflowB := spanFace(font.Metrics{Descent: 1}, nil, "")
	boundsOverflow := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'a', Width: math.MaxInt16, Height: 1, BearingX: 1}}, strings.Repeat("\x00", 4096))
	tests := []struct {
		name  string
		spans []Span
		text  string
	}{
		{"nil face", []Span{{Value: ""}}, "span 0"},
		{"invalid UTF-8", []Span{{Face: valid, Value: string([]byte{0xff})}}, "span 0"},
		{"missing glyph", []Span{{Face: valid, Value: "z"}}, "U+007A"},
		{"line metrics overflow", []Span{{Face: lineOverflowA, Value: ""}, {Face: lineOverflowB, Value: ""}}, "line advance"},
		{"glyph bounds overflow", []Span{{Face: boundsOverflow, Value: "a"}}, "U+0061"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout, err := NewTextLayout(tt.spans)
			if err == nil || !strings.Contains(err.Error(), tt.text) || layout.lines != nil || layout.measurement != (BlockMeasurement{}) {
				t.Fatalf("layout=%+v err=%v", layout, err)
			}
		})
	}
}

func TestTextLayoutDrawPreservesDrawLinesPartialError(t *testing.T) {
	empty := spanFace(font.Metrics{Ascent: 3}, nil, "")
	drawn := spanFace(font.Metrics{Ascent: 2}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 1}}, "\x80")
	layout, err := NewTextLayout([]Span{{Face: empty, Value: "\n"}, {Face: drawn, Value: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("backend failure")
	backend := &fakeBackend{writeErr: sentinel}
	baseline, err := layout.Draw(backend, 0, 5, make([]byte, 2))
	if baseline != 8 || !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("baseline=%d err=%v", baseline, err)
	}
}
