package text

import (
	"strings"
	"testing"

	"github.com/rdon-key/modgadget-fonts/font"
)

func TestLinesFromSpansEmptyInputs(t *testing.T) {
	lines, err := LinesFromSpans(nil)
	if err != nil || lines != nil {
		t.Fatalf("nil input: lines=%v err=%v", lines, err)
	}
	lines, err = LinesFromSpans([]Span{})
	if err != nil || lines == nil || len(lines) != 0 {
		t.Fatalf("empty input: lines=%v err=%v", lines, err)
	}
}

func TestLinesFromSpansSplitsNewlinesAndRetainsEmptySegments(t *testing.T) {
	face := spanFace(font.Metrics{}, nil, "")
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"empty string", "", []string{""}},
		{"no newline", "A", []string{"A"}},
		{"LF", "A\nB", []string{"A", "B"}},
		{"CRLF retains CR", "A\r\nB", []string{"A\r", "B"}},
		{"CRLF only", "\r\n", []string{"\r", ""}},
		{"standalone CR", "A\rB", []string{"A\rB"}},
		{"leading", "\nA", []string{"", "A"}},
		{"trailing", "A\n", []string{"A", ""}},
		{"consecutive", "A\n\nB", []string{"A", "", "B"}},
		{"multiple mixed", "\nA\r\n\n", []string{"", "A\r", "", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []Span{{Font: face, Value: tt.value, Foreground: 0x1234, Background: 0xabcd}}
			lines, err := LinesFromSpans(input)
			if err != nil || len(lines) != len(tt.want) {
				t.Fatalf("lines=%v err=%v", lines, err)
			}
			for index, value := range tt.want {
				if len(lines[index].Spans) != 1 {
					t.Fatalf("line %d spans=%v", index, lines[index].Spans)
				}
				got := lines[index].Spans[0]
				if got.Font != face || got.Value != value || got.Foreground != 0x1234 || got.Background != 0xabcd {
					t.Fatalf("line %d span=%+v", index, got)
				}
			}
			if input[0].Value != tt.value {
				t.Fatalf("input changed to %q", input[0].Value)
			}
		})
	}
}

func TestLinesFromSpansCRAndLFAcrossSpanBoundary(t *testing.T) {
	faceA := spanFace(font.Metrics{}, nil, "")
	faceB := spanFace(font.Metrics{}, nil, "")
	lines, err := LinesFromSpans([]Span{{Font: faceA, Value: "\r"}, {Font: faceB, Value: "\n"}})
	if err != nil || len(lines) != 2 {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
	if len(lines[0].Spans) != 2 || lines[0].Spans[0].Font != faceA || lines[0].Spans[0].Value != "\r" || lines[0].Spans[1].Font != faceB || lines[0].Spans[1].Value != "" {
		t.Fatalf("first line=%v", lines[0].Spans)
	}
	if len(lines[1].Spans) != 1 || lines[1].Spans[0].Font != faceB || lines[1].Spans[0].Value != "" {
		t.Fatalf("second line=%v", lines[1].Spans)
	}
}

func TestLinesFromSpansPreservesSpanOrderAcrossLines(t *testing.T) {
	faceA := spanFace(font.Metrics{}, nil, "")
	faceB := spanFace(font.Metrics{}, nil, "")
	spans := []Span{
		{Font: faceA, Value: "a\n", Foreground: 1, Background: 2},
		{Font: faceB, Value: "b\nc", Foreground: 3, Background: 4},
	}
	lines, err := LinesFromSpans(spans)
	if err != nil || len(lines) != 3 {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
	want := [][]struct {
		face  Font
		value string
		fg    uint16
		bg    uint16
	}{
		{{faceA, "a", 1, 2}},
		{{faceA, "", 1, 2}, {faceB, "b", 3, 4}},
		{{faceB, "c", 3, 4}},
	}
	for lineIndex := range want {
		if len(lines[lineIndex].Spans) != len(want[lineIndex]) {
			t.Fatalf("line %d spans=%v", lineIndex, lines[lineIndex].Spans)
		}
		for spanIndex, expected := range want[lineIndex] {
			got := lines[lineIndex].Spans[spanIndex]
			if got.Font != expected.face || got.Value != expected.value || uint16(got.Foreground) != expected.fg || uint16(got.Background) != expected.bg {
				t.Fatalf("line %d span %d=%+v", lineIndex, spanIndex, got)
			}
		}
	}
	if spans[0].Value != "a\n" || spans[1].Value != "b\nc" {
		t.Fatalf("input changed: %+v", spans)
	}
}

func TestLinesFromSpansMultipleSpansWithoutNewline(t *testing.T) {
	face := spanFace(font.Metrics{}, nil, "")
	lines, err := LinesFromSpans([]Span{{Font: face, Value: "Hello "}, {Font: face, Value: "world"}})
	if err != nil || len(lines) != 1 || len(lines[0].Spans) != 2 || lines[0].Spans[0].Value != "Hello " || lines[0].Spans[1].Value != "world" {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
}

func TestLinesFromSpansValidationReturnsPartialLines(t *testing.T) {
	face := spanFace(font.Metrics{}, nil, "")
	tests := []struct {
		name string
		bad  Span
		text string
	}{
		{"nil face", Span{Value: "ignored"}, "span 1"},
		{"empty with nil face", Span{Value: ""}, "span 1"},
		{"invalid UTF-8", Span{Font: face, Value: string([]byte{0xff})}, "span 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, err := LinesFromSpans([]Span{{Font: face, Value: "a\n"}, tt.bad})
			if err == nil || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("lines=%v err=%v", lines, err)
			}
			if len(lines) != 2 || len(lines[0].Spans) != 1 || lines[0].Spans[0].Value != "a" || len(lines[1].Spans) != 1 || lines[1].Spans[0].Value != "" {
				t.Fatalf("partial lines=%v", lines)
			}
		})
	}
}

func TestLinesFromSpansMeasureLinesIntegration(t *testing.T) {
	face := spanFace(font.Metrics{Ascent: 2, Descent: 1}, []font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 2, BearingY: 1}}, "\x80")
	lines, err := LinesFromSpans([]Span{{Font: face, Value: "a\na"}})
	if err != nil || len(lines) != 2 {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
	measurement, err := MeasureLines(lines)
	want := BlockMeasurement{Bounds: Bounds{MinX: 0, MinY: -1, MaxX: 1, MaxY: 3}, HasInk: true, MaxAdvanceX: 2, AdvanceY: 6}
	if err != nil || measurement != want {
		t.Fatalf("measurement=%+v want %+v err=%v", measurement, want, err)
	}
}
