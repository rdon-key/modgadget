package markup

import (
	"errors"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/text"
)

type testFont struct {
	id      int16
	metrics text.FontMetrics
}

func (font *testFont) Lookup(r rune) (text.Glyph, bool) {
	return text.Glyph{AdvanceX: 1, BearingX: font.id}, true
}
func (font *testFont) Metrics() text.FontMetrics { return font.metrics }

func testParser() (Parser, *testFont, *testFont, *testFont) {
	font12 := &testFont{id: 12, metrics: text.FontMetrics{Ascent: 10, Descent: 2}}
	font16 := &testFont{id: 16, metrics: text.FontMetrics{Ascent: 14, Descent: 2}}
	font24 := &testFont{id: 24, metrics: text.FontMetrics{Ascent: 22, Descent: 2}}
	return Parser{
		Fonts:      Fonts{Size12: font12, Size16: font16, Size24: font24},
		Foreground: display.ColorWhite, Background: display.ColorBlack,
	}, font12, font16, font24
}

func fontIdentifier(t *testing.T, font text.Font) int16 {
	t.Helper()
	glyph, ok := font.Lookup('x')
	if !ok {
		t.Fatal("test font lookup failed")
	}
	return glyph.BearingX
}

func TestParsePlainText(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("Hello 日本語")
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 1 || spans[0].Value != "Hello 日本語" || fontIdentifier(t, spans[0].Font) != 12 || spans[0].Foreground != display.ColorWhite || spans[0].Background != display.ColorBlack {
		t.Fatalf("spans = %+v", spans)
	}
}

func TestParseSizes(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("a<size=16>b</size>c<size=24>d</size>e")
	if err != nil {
		t.Fatal(err)
	}
	wantValues := []string{"a", "b", "c", "d", "e"}
	wantFonts := []int16{12, 16, 12, 24, 12}
	if len(spans) != len(wantValues) {
		t.Fatalf("len = %d", len(spans))
	}
	for index := range spans {
		if spans[index].Value != wantValues[index] || fontIdentifier(t, spans[index].Font) != wantFonts[index] {
			t.Fatalf("span %d = %+v", index, spans[index])
		}
	}
}

func TestParseNestedStyles(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("<size=24>A<fg=#ff0000>B<bg=#ffffff>C</bg>D</fg>E</size>")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		value      string
		font       int16
		foreground display.Color565
		background display.Color565
	}{
		{"A", 24, display.ColorWhite, display.ColorBlack},
		{"B", 24, display.RGB565(255, 0, 0), display.ColorBlack},
		{"C", 24, display.RGB565(255, 0, 0), display.ColorWhite},
		{"D", 24, display.RGB565(255, 0, 0), display.ColorBlack},
		{"E", 24, display.ColorWhite, display.ColorBlack},
	}
	if len(spans) != len(want) {
		t.Fatalf("len = %d", len(spans))
	}
	for index := range spans {
		if spans[index].Value != want[index].value || fontIdentifier(t, spans[index].Font) != want[index].font || spans[index].Foreground != want[index].foreground || spans[index].Background != want[index].background {
			t.Fatalf("span %d = %+v", index, spans[index])
		}
	}
}

func TestParseColors(t *testing.T) {
	parser, _, _, _ := testParser()
	tests := []struct {
		hex  string
		want display.Color565
	}{
		{"000000", display.RGB565(0x00, 0x00, 0x00)},
		{"ffffff", display.RGB565(0xff, 0xff, 0xff)},
		{"ff0000", display.RGB565(0xff, 0x00, 0x00)},
		{"00ff00", display.RGB565(0x00, 0xff, 0x00)},
		{"0000ff", display.RGB565(0x00, 0x00, 0xff)},
		{"123456", display.RGB565(0x12, 0x34, 0x56)},
		{"abcdef", display.RGB565(0xab, 0xcd, 0xef)},
		{"ABCDEF", display.RGB565(0xab, 0xcd, 0xef)},
	}
	for _, test := range tests {
		t.Run(test.hex, func(t *testing.T) {
			spans, err := parser.Parse("<fg=#" + test.hex + ">x</fg>")
			if err != nil {
				t.Fatal(err)
			}
			if len(spans) != 1 || spans[0].Foreground != test.want {
				t.Fatalf("spans = %+v", spans)
			}
		})
	}
}

func TestParseBreakEscapeNewlineAndEmptyTags(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("a<br>b<br/>c")
	if err != nil {
		t.Fatal(err)
	}
	if got := spanValues(spans); got != "a|\n|b|\n|c" {
		t.Fatalf("values = %q", got)
	}
	spans, err = parser.Parse("1 << 2")
	if err != nil {
		t.Fatal(err)
	}
	if got := concatenateValues(spans); got != "1 < 2" {
		t.Fatalf("escaped = %q", got)
	}
	spans, err = parser.Parse("a\nb")
	if err != nil || len(spans) != 1 || spans[0].Value != "a\nb" {
		t.Fatalf("raw newline spans=%+v err=%v", spans, err)
	}
	spans, err = parser.Parse("a<size=16></size>b")
	if err != nil || len(spans) != 2 || spans[0].Value != "a" || spans[1].Value != "b" {
		t.Fatalf("empty tag spans=%+v err=%v", spans, err)
	}
	spans, err = parser.Parse("a<size=16>b<br>c</size>d")
	if err != nil || len(spans) != 5 || spans[2].Value != "\n" || fontIdentifier(t, spans[2].Font) != 16 {
		t.Fatalf("styled break spans=%+v err=%v", spans, err)
	}
}

func TestParseUTF8(t *testing.T) {
	parser, _, _, _ := testParser()
	value := "日本語🙂"
	spans, err := parser.Parse(value)
	if err != nil || len(spans) != 1 || spans[0].Value != value {
		t.Fatalf("spans=%+v err=%v", spans, err)
	}
	if spans, err := parser.Parse(string([]byte{0xff})); err == nil || spans != nil {
		t.Fatalf("invalid UTF-8 spans=%+v err=%v", spans, err)
	}
}

func TestParseErrors(t *testing.T) {
	parser, _, _, _ := testParser()
	tests := []struct {
		name   string
		value  string
		offset int
	}{
		{"unknown", "a<unknown>b", 1},
		{"unsupported size", "<size=13>x</size>", 0},
		{"quoted size", "<size=\"16\">x</size>", 0},
		{"malformed color", "<fg=#12xx56>x</fg>", 0},
		{"short color", "<fg=#fff>x</fg>", 0},
		{"named color", "<fg=red>x</fg>", 0},
		{"uppercase", "<SIZE=16>x</SIZE>", 0},
		{"tag whitespace", "<size = 16>x</size>", 0},
		{"break whitespace", "<br />", 0},
		{"closing break", "</br>", 0},
		{"unterminated", "a<size=16", 1},
		{"unexpected close", "</size>", 0},
		{"mismatch", "<size=16><fg=#ff0000>x</size></fg>", 22},
		{"wrong close", "<fg=#ff0000>x</bg>", 13},
		{"unclosed size", "<size=16>x", 10},
		{"unclosed fg", "<fg=#ff0000>x", 13},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spans, err := parser.Parse(test.value)
			if err == nil || spans != nil {
				t.Fatalf("spans=%+v err=%v", spans, err)
			}
			var syntax *SyntaxError
			if !errors.As(err, &syntax) || syntax.Offset != test.offset {
				t.Fatalf("error=%v offset=%v", err, syntax)
			}
		})
	}
	deep := strings.Repeat("<size=12>", 17) + "x" + strings.Repeat("</size>", 17)
	if _, err := parser.Parse(deep); err == nil {
		t.Fatal("depth 17 succeeded")
	} else {
		var syntax *SyntaxError
		if !errors.As(err, &syntax) || syntax.Offset != 16*len("<size=12>") {
			t.Fatalf("depth error = %v", err)
		}
	}
}

func TestParseFontValidation(t *testing.T) {
	parser, _, _, _ := testParser()
	parser.Fonts.Size12 = nil
	if spans, err := parser.Parse("x"); err == nil || spans != nil {
		t.Fatalf("nil Size12 spans=%+v err=%v", spans, err)
	}
	parser, _, _, _ = testParser()
	parser.Fonts.Size16 = nil
	if _, err := parser.Parse("x"); err != nil {
		t.Fatalf("unused nil Size16: %v", err)
	}
	if spans, err := parser.Parse("<size=16>x</size>"); err == nil || spans != nil {
		t.Fatalf("nil Size16 spans=%+v err=%v", spans, err)
	}
	parser, _, _, _ = testParser()
	parser.Fonts.Size24 = nil
	if _, err := parser.Parse("x"); err != nil {
		t.Fatalf("unused nil Size24: %v", err)
	}
	if spans, err := parser.Parse("<size=24>x</size>"); err == nil || spans != nil {
		t.Fatalf("nil Size24 spans=%+v err=%v", spans, err)
	}
}

func TestParseInto(t *testing.T) {
	parser, _, _, _ := testParser()
	value := "a<size=16>b<fg=#ff0000>c</fg>d</size>e<br>f"
	want, err := parser.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	var storage [32]text.Span
	got, err := parser.ParseInto(storage[:0], value)
	if err != nil {
		t.Fatal(err)
	}
	assertSameSpans(t, got, want)

	var exact [7]text.Span
	if result, err := parser.ParseInto(exact[:0], value); err != nil || len(result) != len(exact) {
		t.Fatalf("exact result=%d err=%v", len(result), err)
	}
	var short [6]text.Span
	if result, err := parser.ParseInto(short[:0], value); err == nil || len(result) != 0 || !strings.Contains(err.Error(), "have 6") || !strings.Contains(err.Error(), "need at least 7") {
		t.Fatalf("short result=%d err=%v", len(result), err)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		result, err := parser.ParseInto(storage[:0], value)
		if err != nil || len(result) != 7 {
			panic("parse")
		}
	}); allocations != 0 {
		t.Fatalf("allocations = %v", allocations)
	}
}

func assertSameSpans(t *testing.T, got, want []text.Span) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d", len(got), len(want))
	}
	for index := range got {
		if got[index].Value != want[index].Value || got[index].Foreground != want[index].Foreground || got[index].Background != want[index].Background || fontIdentifier(t, got[index].Font) != fontIdentifier(t, want[index].Font) {
			t.Fatalf("span %d got=%+v want=%+v", index, got[index], want[index])
		}
	}
}

func spanValues(spans []text.Span) string {
	values := make([]string, len(spans))
	for index := range spans {
		values[index] = spans[index].Value
	}
	return strings.Join(values, "|")
}

func concatenateValues(spans []text.Span) string {
	var builder strings.Builder
	for index := range spans {
		builder.WriteString(spans[index].Value)
	}
	return builder.String()
}
