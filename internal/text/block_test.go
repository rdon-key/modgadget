package text

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget-fonts/font"
	"github.com/rdon-key/modgadget/internal/display"
)

func TestMeasureLinesEmpty(t *testing.T) {
	for _, lines := range [][]Line{nil, {}, {{}}} {
		got, err := MeasureLines(lines)
		if err != nil || got != (BlockMeasurement{}) {
			t.Fatalf("block=%+v err=%v", got, err)
		}
	}
}

func TestMeasureLinesPlacesAndUnionsLines(t *testing.T) {
	faceA := spanFace(font.Metrics{Ascent: 3, Descent: 1, LineGap: 1}, []font.GlyphInfo{{Rune: 'a', Width: 2, Height: 2, AdvanceX: 4, BearingY: 1}}, "\x00\x00")
	faceB := spanFace(font.Metrics{Ascent: 2, Descent: 1}, []font.GlyphInfo{{Rune: 'b', Width: 1, Height: 2, AdvanceX: 2, BearingX: -1, BearingY: 2}}, "\x00\x00")
	lines := []Line{{Spans: []Span{{Face: faceA, Value: "a"}}}, {Spans: []Span{{Face: faceB, Value: "b"}}}}
	want := BlockMeasurement{Bounds: Bounds{MinX: -1, MinY: -1, MaxX: 2, MaxY: 5}, HasInk: true, MaxAdvanceX: 4, AdvanceY: 8}
	got, err := MeasureLines(lines)
	if err != nil || got != want {
		t.Fatalf("block=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureLinesFontEmptyAndWhitespaceLines(t *testing.T) {
	empty := spanFace(font.Metrics{Ascent: 4, Descent: 1}, nil, "")
	space := spanFace(font.Metrics{Ascent: 2, Descent: 1, LineGap: 1}, []font.GlyphInfo{{Rune: ' ', AdvanceX: 3}}, "")
	got, err := MeasureLines([]Line{{Spans: []Span{{Face: empty, Value: ""}}}, {}, {Spans: []Span{{Face: space, Value: "  "}}}})
	want := BlockMeasurement{MaxAdvanceX: 6, AdvanceY: 9}
	if err != nil || got != want {
		t.Fatalf("block=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureLinesMaximumNegativeAdvance(t *testing.T) {
	faceA := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'a', AdvanceX: -5}}, "")
	faceB := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'b', AdvanceX: -3}}, "")
	got, err := MeasureLines([]Line{{Spans: []Span{{Face: faceA, Value: "a"}}}, {Spans: []Span{{Face: faceB, Value: "b"}}}})
	if err != nil || got.MaxAdvanceX != -3 {
		t.Fatalf("block=%+v err=%v", got, err)
	}
}

func TestMeasureLinesAllowsNegativeAdvanceYAndOverlappingInk(t *testing.T) {
	face := spanFace(font.Metrics{Ascent: -2, Descent: -1}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 4, AdvanceX: 1, BearingY: 2}}, strings.Repeat("\x00", 4))
	got, err := MeasureLines([]Line{{Spans: []Span{{Face: face, Value: "a"}}}, {Spans: []Span{{Face: face, Value: "a"}}}})
	want := BlockMeasurement{Bounds: Bounds{MinX: 0, MinY: -5, MaxX: 1, MaxY: 2}, HasInk: true, MaxAdvanceX: 1, AdvanceY: -6}
	if err != nil || got != want {
		t.Fatalf("block=%+v want %+v err=%v", got, want, err)
	}
}

func TestMeasureLinesOverflow(t *testing.T) {
	positiveBaseline := spanFace(font.Metrics{Ascent: math.MaxInt16}, nil, "")
	negativeBaseline := spanFace(font.Metrics{Ascent: math.MinInt16}, nil, "")
	positiveInk := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'p', Width: 1, Height: 1, BearingY: -1}}, "\x00")
	negativeInk := spanFace(font.Metrics{}, []font.GlyphInfo{{Rune: 'n', Width: 1, Height: 1, BearingY: 1}}, "\x00")
	advanceOne := spanFace(font.Metrics{Descent: 1}, nil, "")
	tests := []struct {
		name  string
		lines []Line
		index string
	}{
		{"positive bounds shift", []Line{{Spans: []Span{{Face: positiveBaseline, Value: ""}}}, {Spans: []Span{{Face: positiveInk, Value: "p"}}}}, "line 1"},
		{"negative bounds shift", []Line{{Spans: []Span{{Face: negativeBaseline, Value: ""}}}, {Spans: []Span{{Face: negativeInk, Value: "n"}}}}, "line 1"},
		{"cumulative baseline", []Line{{Spans: []Span{{Face: positiveBaseline, Value: ""}}}, {Spans: []Span{{Face: advanceOne, Value: ""}}}}, "line 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MeasureLines(tt.lines)
			if err == nil || !strings.Contains(err.Error(), tt.index) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestMeasureLinesPartialErrorsPreserveContext(t *testing.T) {
	first := spanFace(font.Metrics{Ascent: 2}, []font.GlyphInfo{{Rune: 'a', AdvanceX: 3}}, "")
	second := spanFace(font.Metrics{Ascent: 3}, []font.GlyphInfo{{Rune: 'b', Width: 1, Height: 1, AdvanceX: 2, BearingX: 5}}, "\x00")
	tests := []struct {
		name string
		span Span
		text string
	}{
		{"nil face", Span{}, "span 0"},
		{"invalid UTF-8", Span{Face: second, Value: string([]byte{0xff})}, "span 0"},
		{"missing after glyph", Span{Face: second, Value: "bz"}, "U+007A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MeasureLines([]Line{{Spans: []Span{{Face: first, Value: "a"}}}, {Spans: []Span{tt.span}}})
			if err == nil || !strings.Contains(err.Error(), "line 1") || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("block=%+v err=%v", got, err)
			}
			if tt.name == "missing after glyph" && (got.MaxAdvanceX != 3 || !got.HasInk) {
				t.Fatalf("partial block=%+v", got)
			}
		})
	}
}

func TestDrawLinesEmpty(t *testing.T) {
	for _, lines := range [][]Line{nil, {}} {
		got, err := DrawLines(nil, lines, 4, -7, nil)
		if err != nil || got != -7 {
			t.Fatalf("baseline=%d err=%v", got, err)
		}
	}
}

func TestDrawLinesResetsPenMovesBaselineAndReusesScratch(t *testing.T) {
	face := spanFace(font.Metrics{Ascent: 2, Descent: 1}, []font.GlyphInfo{{Rune: 'a', Width: 2, Height: 1, AdvanceX: 3, BearingY: 1}}, "\x80")
	lines := []Line{
		{Spans: []Span{{Face: face, Value: "a", Foreground: 0x1234, Background: 0xabcd}}},
		{Spans: []Span{{Face: face, Value: "a", Foreground: 0x5678, Background: 0x9abc}}},
	}
	backend := &fakeBackend{}
	scratch := make([]byte, 4)
	baseline, err := DrawLines(backend, lines, 5, 7, scratch)
	if err != nil || baseline != 13 {
		t.Fatalf("baseline=%d err=%v", baseline, err)
	}
	wantRects := []display.Rect{{X: 5, Y: 6, Width: 2, Height: 1}, {X: 5, Y: 9, Width: 2, Height: 1}}
	if len(backend.rects) != 2 || backend.rects[0] != wantRects[0] || backend.rects[1] != wantRects[1] {
		t.Fatalf("rects=%v want %v", backend.rects, wantRects)
	}
	if string(scratch) != string([]byte{0x56, 0x78, 0x9a, 0xbc}) {
		t.Fatalf("scratch=%x", scratch)
	}
}

func TestDrawLinesEmptyFontLineAndEmptyLine(t *testing.T) {
	face := spanFace(font.Metrics{Ascent: 4, Descent: 1}, nil, "")
	backend := &fakeBackend{}
	baseline, err := DrawLines(backend, []Line{{Spans: []Span{{Face: face, Value: ""}}}, {}}, 0, 2, nil)
	if err != nil || baseline != 7 || backend.beginCalls != 0 {
		t.Fatalf("baseline=%d calls=%d err=%v", baseline, backend.beginCalls, err)
	}
}

func TestDrawLinesPartialErrors(t *testing.T) {
	empty := spanFace(font.Metrics{Ascent: 3}, nil, "")
	drawn := spanFace(font.Metrics{Ascent: 2}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 1}}, "\x80")
	sentinel := errors.New("backend failure")
	tests := []struct {
		name    string
		backend *fakeBackend
		value   string
		text    string
	}{
		{"backend", &fakeBackend{writeErr: sentinel}, "a", "backend failure"},
		{"missing", &fakeBackend{}, "az", "U+007A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseline, err := DrawLines(tt.backend, []Line{{Spans: []Span{{Face: empty, Value: ""}}}, {Spans: []Span{{Face: drawn, Value: tt.value}}}}, 0, 5, make([]byte, 2))
			if baseline != 8 || err == nil || !strings.Contains(err.Error(), "line 1") || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("baseline=%d err=%v", baseline, err)
			}
		})
	}
}

func TestDrawLinesDrawsEarlierSpansBeforeLaterSpanError(t *testing.T) {
	good := spanFace(font.Metrics{Ascent: 2, Descent: 1}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 1}}, "\x80")
	maxMetrics := spanFace(font.Metrics{Ascent: math.MaxInt16}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 1}}, "\x80")
	overflowMetrics := spanFace(font.Metrics{Descent: 1}, []font.GlyphInfo{{Rune: 'b', Width: 1, Height: 1, AdvanceX: 1}}, "\x80")
	first := func(face *font.Font) Span {
		return Span{Face: face, Value: "a", Foreground: 0x1234}
	}
	tests := []struct {
		name       string
		spans      []Span
		wantCalls  int
		errorTexts []string
	}{
		{"nil face", []Span{first(good), {}}, 1, []string{"line 0", "span 1"}},
		{"invalid UTF-8", []Span{first(good), {Face: good, Value: string([]byte{0xff})}}, 1, []string{"line 0", "span 1"}},
		{"metrics overflow", []Span{first(maxMetrics), {Face: overflowMetrics, Value: "b"}}, 1, []string{"line 0", "span 1"}},
		{"missing glyph", []Span{first(good), {Face: good, Value: "az"}}, 2, []string{"line 0", "U+007A"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeBackend{}
			baseline, err := DrawLines(backend, []Line{{Spans: tt.spans}}, 3, 7, make([]byte, 2))
			if baseline != 7 || err == nil {
				t.Fatalf("baseline=%d err=%v", baseline, err)
			}
			for _, text := range tt.errorTexts {
				if !strings.Contains(err.Error(), text) {
					t.Fatalf("error %q lacks %q", err, text)
				}
			}
			if backend.beginCalls != tt.wantCalls || len(backend.writes) != tt.wantCalls {
				t.Fatalf("begin calls=%d writes=%d want %d", backend.beginCalls, len(backend.writes), tt.wantCalls)
			}
			if len(backend.rects) == 0 || backend.rects[0] != (display.Rect{X: 3, Y: 7, Width: 1, Height: 1}) {
				t.Fatalf("rects=%v", backend.rects)
			}
			if string(backend.writes[0]) != "\x12\x34" {
				t.Fatalf("first pixels=%x", backend.writes[0])
			}
		})
	}
}

func TestDrawLinesBaselineOverflow(t *testing.T) {
	face := spanFace(font.Metrics{Descent: 1}, nil, "")
	baseline, err := DrawLines(&fakeBackend{}, []Line{{Spans: []Span{{Face: face, Value: ""}}}}, 0, math.MaxInt16, nil)
	if baseline != math.MaxInt16 || err == nil || !strings.Contains(err.Error(), "line 0") {
		t.Fatalf("baseline=%d err=%v", baseline, err)
	}
}

func TestBlockMeasurementAndDrawingAgree(t *testing.T) {
	face := spanFace(font.Metrics{Ascent: 2, Descent: 1}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 2, BearingY: 1}}, "\x80")
	lines := []Line{{Spans: []Span{{Face: face, Value: "a"}}}, {Spans: []Span{{Face: face, Value: "a"}}}}
	measurement, err := MeasureLines(lines)
	if err != nil {
		t.Fatal(err)
	}
	backend := &fakeBackend{}
	baseline, err := DrawLines(backend, lines, 0, 0, make([]byte, 2))
	if err != nil || baseline != measurement.AdvanceY || unionRects(backend.rects) != measurement.Bounds {
		t.Fatalf("baseline=%d block=%+v rects=%v err=%v", baseline, measurement, backend.rects, err)
	}
	line, err := MeasureLine(lines[0].Spans)
	one, oneErr := MeasureLines(lines[:1])
	if err != nil || oneErr != nil || line.Advance != one.MaxAdvanceX || line.Bounds != one.Bounds {
		t.Fatalf("line=%+v block=%+v errors=%v/%v", line, one, err, oneErr)
	}
}
