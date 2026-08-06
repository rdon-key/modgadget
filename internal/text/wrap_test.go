package text

import (
	"math"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget/internal/display"
)

func TestWrapSpansEmptyAndWidthValidation(t *testing.T) {
	lines, err := WrapSpans(nil, 10)
	if err != nil || lines != nil {
		t.Fatalf("nil: lines=%v err=%v", lines, err)
	}
	lines, err = WrapSpans([]Span{}, 10)
	if err != nil || lines == nil || len(lines) != 0 {
		t.Fatalf("empty: lines=%v err=%v", lines, err)
	}
	for _, width := range []int16{0, -1} {
		if _, err := WrapSpans(nil, width); err == nil || !strings.Contains(err.Error(), "maximum") {
			t.Fatalf("width=%d err=%v", width, err)
		}
	}
}

func TestWrapSpansGreedyAdvance(t *testing.T) {
	tests := []struct {
		name    string
		advance int16
		width   int16
		value   string
		want    []string
	}{
		{"within width", 4, 12, "abc", []string{"abc"}},
		{"exact width", 4, 12, "abc", []string{"abc"}},
		{"one beyond", 4, 12, "abcd", []string{"abc", "d"}},
		{"oversized glyph", 5, 3, "a", []string{"a"}},
		{"all oversized", 5, 3, "ab", []string{"a", "b"}},
		{"zero advance", 0, 3, "abcd", []string{"abcd"}},
		{"negative advance", -2, 3, "abcd", []string{"abcd"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			face := asciiAdvanceFace(tt.value, tt.advance)
			lines, err := WrapSpans([]Span{{Font: face, Value: tt.value}}, tt.width)
			if err != nil || !sameSingleSpanLines(lines, tt.want) {
				t.Fatalf("lines=%v want=%v err=%v", lines, tt.want, err)
			}
		})
	}
}

func TestWrapSpansKeepsOversizedGlyphAloneBeforeNegativeAdvance(t *testing.T) {
	face := spanFace(FontMetrics{}, []testGlyphInfo{
		{Rune: 'a', AdvanceX: 5},
		{Rune: 'b', AdvanceX: -3},
		{Rune: 'c', AdvanceX: 2},
	}, "")

	lines, err := WrapSpans([]Span{{Font: face, Value: "ab"}}, 3)
	if err != nil || !sameSingleSpanLines(lines, []string{"a", "b"}) {
		t.Fatalf("same span: lines=%v err=%v", lines, err)
	}

	spans := []Span{{Font: face, Value: "a"}, {Font: face, Value: "b"}}
	lines, err = WrapSpans(spans, 3)
	if err != nil || len(lines) != 2 || len(lines[0].Spans) != 1 || len(lines[1].Spans) != 1 || lines[0].Spans[0] != spans[0] || lines[1].Spans[0] != spans[1] {
		t.Fatalf("span boundary: lines=%v err=%v", lines, err)
	}

	lines, err = WrapSpans([]Span{{Font: face, Value: "abc"}}, 3)
	if err != nil || !sameSingleSpanLines(lines, []string{"a", "bc"}) {
		t.Fatalf("negative advance after oversized: lines=%v err=%v", lines, err)
	}
}

func TestWrapSpansExplicitLFAndEmptySpans(t *testing.T) {
	face := asciiAdvanceFace("ab\r", 4)
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"empty", "", []string{""}},
		{"leading LF", "\na", []string{"", "a"}},
		{"trailing LF", "a\n", []string{"a", ""}},
		{"consecutive LF", "a\n\nb", []string{"a", "", "b"}},
		{"CR ordinary", "a\rb", []string{"a\r", "b"}},
		{"CRLF retains CR", "a\r\nb", []string{"a\r", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, err := WrapSpans([]Span{{Font: face, Value: tt.value}}, 8)
			if err != nil || !sameSingleSpanLines(lines, tt.want) {
				t.Fatalf("lines=%v want=%v err=%v", lines, tt.want, err)
			}
		})
	}
}

func TestWrapSpansPreservesStylesAndSpanBoundaries(t *testing.T) {
	faceA := asciiAdvanceFace("ab", 4)
	faceB := asciiAdvanceFace("cd", 4)
	input := []Span{
		{Font: faceA, Value: "ab", Foreground: 0x1234, Background: 0xabcd},
		{Font: faceB, Value: "cd", Foreground: 0x5678, Background: 0x9abc},
	}
	lines, err := WrapSpans(input, 12)
	if err != nil || len(lines) != 2 || len(lines[0].Spans) != 2 || len(lines[1].Spans) != 1 {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
	if lines[0].Spans[0] != input[0] || lines[0].Spans[1].Font != faceB || lines[0].Spans[1].Value != "c" || lines[0].Spans[1].Foreground != 0x5678 || lines[0].Spans[1].Background != 0x9abc || lines[1].Spans[0].Value != "d" {
		t.Fatalf("lines=%+v", lines)
	}
	if input[0].Value != "ab" || input[1].Value != "cd" {
		t.Fatalf("input changed: %+v", input)
	}
}

func TestWrapSpansKeepsExplicitEmptySpanWithoutAutomaticEmptySpan(t *testing.T) {
	faceA := asciiAdvanceFace("a", 4)
	faceB := asciiAdvanceFace("b", 4)
	lines, err := WrapSpans([]Span{{Font: faceA, Value: "a\n"}, {Font: faceB, Value: "b"}}, 3)
	if err != nil || len(lines) != 2 {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
	if len(lines[0].Spans) != 1 || lines[0].Spans[0].Value != "a" {
		t.Fatalf("automatic line=%v", lines[0].Spans)
	}
	if len(lines[1].Spans) != 2 || lines[1].Spans[0].Font != faceA || lines[1].Spans[0].Value != "" || lines[1].Spans[1].Font != faceB || lines[1].Spans[1].Value != "b" {
		t.Fatalf("explicit line=%v", lines[1].Spans)
	}
}

func TestWrapSpansDoesNotSplitUTF8Rune(t *testing.T) {
	face := spanFace(FontMetrics{}, []testGlyphInfo{{Rune: '日', AdvanceX: 4}, {Rune: '本', AdvanceX: 4}, {Rune: '語', AdvanceX: 4}}, "")
	lines, err := WrapSpans([]Span{{Font: face, Value: "日本語"}}, 8)
	if err != nil || !sameSingleSpanLines(lines, []string{"日本", "語"}) {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
}

func TestWrapSpansErrorsAndPartialResult(t *testing.T) {
	face := asciiAdvanceFace("abcd", 2)
	t.Run("unfinished explicit line", func(t *testing.T) {
		lines, err := WrapSpans([]Span{{Font: face, Value: "a"}, {Value: ""}}, 4)
		if err == nil || len(lines) != 0 || !strings.Contains(err.Error(), "explicit line 0") || !strings.Contains(err.Error(), "span 1") {
			t.Fatalf("lines=%v err=%v", lines, err)
		}
	})
	t.Run("completed line before LF", func(t *testing.T) {
		lines, err := WrapSpans([]Span{{Font: face, Value: "a\n"}, {Value: ""}}, 4)
		if err == nil || !sameSingleSpanLines(lines, []string{"a"}) || !strings.Contains(err.Error(), "explicit line 1") {
			t.Fatalf("lines=%v err=%v", lines, err)
		}
	})
	t.Run("completed automatic wrap in partial explicit line", func(t *testing.T) {
		lines, err := WrapSpans([]Span{{Font: face, Value: "a\nabcd"}, {Value: ""}}, 4)
		if err == nil || !sameSingleSpanLines(lines, []string{"a", "ab"}) || !strings.Contains(err.Error(), "explicit line 1") {
			t.Fatalf("lines=%v err=%v", lines, err)
		}
	})
	t.Run("input span index after explicit lines", func(t *testing.T) {
		lines, err := WrapSpans([]Span{{Font: face, Value: "a\n"}, {Font: face, Value: "b\n"}, {Font: face, Value: "Z"}}, 4)
		if err == nil || !sameLineText(lines, []string{"a", "b"}) || !strings.Contains(err.Error(), "explicit line 2") || !strings.Contains(err.Error(), "span 2") || !strings.Contains(err.Error(), "U+005A") {
			t.Fatalf("lines=%v err=%v", lines, err)
		}
	})
	t.Run("input span index after automatic wrap", func(t *testing.T) {
		lines, err := WrapSpans([]Span{{Font: face, Value: "abc"}, {Font: face, Value: "Z"}}, 4)
		if err == nil || !sameSingleSpanLines(lines, []string{"ab"}) || !strings.Contains(err.Error(), "explicit line 0") || !strings.Contains(err.Error(), "span 1") || !strings.Contains(err.Error(), "U+005A") {
			t.Fatalf("lines=%v err=%v", lines, err)
		}
	})
	t.Run("invalid UTF-8", func(t *testing.T) {
		lines, err := WrapSpans([]Span{{Font: face, Value: "a\n"}, {Font: face, Value: string([]byte{0xff})}}, 4)
		if err == nil || !sameSingleSpanLines(lines, []string{"a"}) || !strings.Contains(err.Error(), "explicit line 1") || !strings.Contains(err.Error(), "span 1") {
			t.Fatalf("lines=%v err=%v", lines, err)
		}
	})
	t.Run("advance overflow", func(t *testing.T) {
		overflowFace := spanFace(FontMetrics{}, []testGlyphInfo{{Rune: 'a', AdvanceX: math.MaxInt16}}, "")
		lines, err := WrapSpans([]Span{{Font: overflowFace, Value: "aa"}}, math.MaxInt16)
		if err == nil || len(lines) != 0 || !strings.Contains(err.Error(), "span 0") || !strings.Contains(err.Error(), "U+0061") {
			t.Fatalf("lines=%v err=%v", lines, err)
		}
	})
}

func TestNewWrappedTextLayoutEmptyInputIsZeroValue(t *testing.T) {
	for _, spans := range [][]Span{nil, {}} {
		layout, err := NewWrappedTextLayout(spans, 8)
		if err != nil || layout.lines != nil || layout.measurement != (BlockMeasurement{}) || layout.LineCount() != 0 {
			t.Fatalf("spans=%v layout=%+v err=%v", spans, layout, err)
		}
	}
}

func TestWrappedTextLayoutIntegration(t *testing.T) {
	face := spanFace(FontMetrics{Ascent: 2, Descent: 1}, []testGlyphInfo{
		{Rune: 'a', Width: 1, Height: 1, AdvanceX: 4, BearingY: 1},
		{Rune: 'b', BitmapOffset: 1, Width: 1, Height: 1, AdvanceX: 4, BearingY: 1},
	}, "\x80\x80")
	spans := []Span{{Font: face, Value: "ab", Foreground: display.ColorWhite}, {Font: face, Value: "ab", Foreground: display.ColorGreen}}
	lines, err := WrapSpans(spans, 8)
	if err != nil {
		t.Fatal(err)
	}
	measurement, err := MeasureLines(lines)
	if err != nil {
		t.Fatal(err)
	}
	for index := range lines {
		line, err := MeasureLine(lines[index].Spans)
		if err != nil || line.Advance > 8 {
			t.Fatalf("line %d=%+v err=%v", index, line, err)
		}
	}
	layout, err := NewWrappedTextLayout(spans, 8)
	if err != nil || layout.LineCount() != len(lines) || layout.Measurement() != measurement {
		t.Fatalf("layout=%+v lines=%d measurement=%+v err=%v", layout, len(lines), measurement, err)
	}
	backend := &fakeBackend{}
	baseline, err := layout.Draw(backend, 0, 0, make([]byte, 2))
	if err != nil || baseline != measurement.AdvanceY || len(backend.rects) != 4 {
		t.Fatalf("baseline=%d rects=%v err=%v", baseline, backend.rects, err)
	}
	if string(backend.writes[0]) != "\xff\xff" || string(backend.writes[2]) != "\x07\xe0" {
		t.Fatalf("writes=%x", backend.writes)
	}

	bad, err := NewWrappedTextLayout([]Span{{Value: ""}}, 8)
	if err == nil || bad.lines != nil || bad.measurement != (BlockMeasurement{}) {
		t.Fatalf("bad layout=%+v err=%v", bad, err)
	}
}

func asciiAdvanceFace(chars string, advance int16) Font {
	seen := map[rune]bool{}
	glyphs := make([]testGlyphInfo, 0, len(chars))
	for _, r := range chars {
		if r == '\n' || seen[r] {
			continue
		}
		seen[r] = true
		glyphs = append(glyphs, testGlyphInfo{Rune: r, AdvanceX: advance})
	}
	for left := 1; left < len(glyphs); left++ {
		for right := left; right > 0 && glyphs[right].Rune < glyphs[right-1].Rune; right-- {
			glyphs[right], glyphs[right-1] = glyphs[right-1], glyphs[right]
		}
	}
	return spanFace(FontMetrics{}, glyphs, "")
}

func sameSingleSpanLines(lines []Line, values []string) bool {
	if len(lines) != len(values) {
		return false
	}
	for index, value := range values {
		if len(lines[index].Spans) != 1 || lines[index].Spans[0].Value != value {
			return false
		}
	}
	return true
}

func sameLineText(lines []Line, values []string) bool {
	if len(lines) != len(values) {
		return false
	}
	for lineIndex := range lines {
		value := ""
		for spanIndex := range lines[lineIndex].Spans {
			value += lines[lineIndex].Spans[spanIndex].Value
		}
		if value != values[lineIndex] {
			return false
		}
	}
	return true
}
