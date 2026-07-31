package main

import (
	"strings"
	"testing"

	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/text"
)

type lineTestFont struct{ id int16 }

func (font *lineTestFont) Lookup(r rune) (text.Glyph, bool) {
	return text.Glyph{BearingX: font.id}, true
}
func (font *lineTestFont) Metrics() text.FontMetrics { return text.FontMetrics{} }

func TestSplitLines(t *testing.T) {
	fontA := &lineTestFont{id: 1}
	fontB := &lineTestFont{id: 2}
	spans := []text.Span{
		{Font: fontA, Value: "a", Foreground: 1, Background: 2},
		{Font: fontA, Value: "\n", Foreground: 3, Background: 4},
		{Font: fontB, Value: "b", Foreground: 5, Background: 6},
	}
	var storage [2]text.Line
	lines, err := splitLines(storage[:0], spans)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || len(lines[0].Spans) != 1 || len(lines[1].Spans) != 1 || lines[0].Spans[0].Value != "a" || lines[1].Spans[0].Value != "b" {
		t.Fatalf("lines = %+v", lines)
	}
	if lines[0].Spans[0].Font != fontA || lines[0].Spans[0].Foreground != display.Color565(1) || lines[0].Spans[0].Background != display.Color565(2) || lines[1].Spans[0].Font != fontB || lines[1].Spans[0].Foreground != display.Color565(5) || lines[1].Spans[0].Background != display.Color565(6) {
		t.Fatalf("styles were not preserved: %+v", lines)
	}
	if &lines[0].Spans[0] != &spans[0] || &lines[1].Spans[0] != &spans[2] {
		t.Fatal("line spans do not reference the input slice")
	}

	var short [1]text.Line
	if result, err := splitLines(short[:0], spans); err == nil || len(result) != 0 || !strings.Contains(err.Error(), "have 1") || !strings.Contains(err.Error(), "need at least 2") {
		t.Fatalf("short result=%+v err=%v", result, err)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		result, err := splitLines(storage[:0], spans)
		if err != nil || len(result) != 2 {
			panic("split")
		}
	}); allocations != 0 {
		t.Fatalf("allocations = %v", allocations)
	}
}

func TestSplitLinesEmptySeparator(t *testing.T) {
	spans := []text.Span{{Value: "\n"}}
	var storage [2]text.Line
	lines, err := splitLines(storage[:0], spans)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || len(lines[0].Spans) != 0 || len(lines[1].Spans) != 0 {
		t.Fatalf("lines = %+v", lines)
	}
}

func TestMarkupContent(t *testing.T) {
	var spans [32]text.Span
	parsed, err := messageParser.ParseInto(spans[:0], message)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 8 {
		t.Fatalf("span count = %d", len(parsed))
	}
	for index := range parsed {
		for _, r := range parsed[index].Value {
			if r == '\n' {
				continue
			}
			if _, ok := parsed[index].Font.Lookup(r); !ok {
				t.Fatalf("span %d missing U+%04X", index, r)
			}
		}
	}
	var lines [4]text.Line
	result, err := splitLines(lines[:0], parsed)
	if err != nil || len(result) != 2 {
		t.Fatalf("lines=%d err=%v", len(result), err)
	}
	measurement, err := text.MeasureLines(result)
	if err != nil || !measurement.HasInk {
		t.Fatalf("measurement=%+v err=%v", measurement, err)
	}
	t.Logf("spans=%d lines=%d measurement=%+v", len(parsed), len(result), measurement)
}
