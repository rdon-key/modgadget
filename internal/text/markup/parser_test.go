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
	base := &testFont{id: 12, metrics: text.FontMetrics{Ascent: 10, Descent: 2}}
	medium := &testFont{id: 16, metrics: text.FontMetrics{Ascent: 14, Descent: 2}}
	large := &testFont{id: 24, metrics: text.FontMetrics{Ascent: 22, Descent: 2}}
	return Parser{Styles: text.StyleSet{
		Default: text.Style{Font: base, Foreground: display.ColorWhite, Background: display.ColorBlack},
		Entries: []text.StyleEntry{
			{Name: "medium", Style: text.Style{Font: medium, Foreground: display.ColorGreen, Background: display.ColorBlack}},
			{Name: "large-red", Style: text.Style{Font: large, Foreground: display.ColorRed, Background: display.ColorBlack}},
			{Name: "inverse", Style: text.Style{Font: base, Foreground: display.ColorBlack, Background: display.ColorWhite}},
			{Name: "bold", Style: text.Style{Font: large, Foreground: display.ColorRed, Background: display.ColorBlack, Bold: true}},
			{Name: "normal", Style: text.Style{Font: medium, Foreground: display.ColorGreen, Background: display.ColorWhite}},
			{Name: "nil-font", Style: text.Style{Foreground: display.ColorWhite}},
		},
	}}, base, medium, large
}

func fontIdentifier(t *testing.T, font text.Font) int16 {
	t.Helper()
	glyph, ok := font.Lookup('x')
	if !ok {
		t.Fatal("test font lookup failed")
	}
	return glyph.BearingX
}

func TestParseDefaultStyle(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("Hello 日本語")
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 1 || spans[0].Value != "Hello 日本語" || fontIdentifier(t, spans[0].Font) != 12 || spans[0].Foreground != display.ColorWhite || spans[0].Background != display.ColorBlack {
		t.Fatalf("spans = %+v", spans)
	}
	parser.Styles.Default.Font = nil
	if spans, err := parser.Parse("x"); err == nil || spans != nil {
		t.Fatalf("nil default font spans=%+v err=%v", spans, err)
	}
}

func TestParseStyleSwitchAndRestore(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("a<style=medium>b</style>c")
	if err != nil {
		t.Fatal(err)
	}
	wantFonts := []int16{12, 16, 12}
	wantValues := []string{"a", "b", "c"}
	for index := range spans {
		if spans[index].Value != wantValues[index] || fontIdentifier(t, spans[index].Font) != wantFonts[index] {
			t.Fatalf("span %d = %+v", index, spans[index])
		}
	}
	if spans[1].Foreground != display.ColorGreen || spans[1].Background != display.ColorBlack {
		t.Fatalf("medium style colors = %#04x/%#04x", spans[1].Foreground, spans[1].Background)
	}
}

func TestParseBoldAndRestore(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("plain <b>bold</b> plain")
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 3 || spans[0].Bold || !spans[1].Bold || spans[2].Bold {
		t.Fatalf("spans=%+v", spans)
	}
	if concatenateValues(spans) != "plain bold plain" {
		t.Fatalf("values=%q", concatenateValues(spans))
	}
}

func TestParseBoldNestedInStyleRestoresCompleteStyle(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("<style=medium>normal <b>bold</b> normal</style> plain")
	if err != nil {
		t.Fatal(err)
	}
	wantFonts := []int16{16, 16, 16, 12}
	wantBold := []bool{false, true, false, false}
	for index := range spans {
		if fontIdentifier(t, spans[index].Font) != wantFonts[index] || spans[index].Bold != wantBold[index] {
			t.Fatalf("span %d=%+v", index, spans[index])
		}
	}
	for _, span := range spans[:3] {
		if span.Foreground != display.ColorGreen || span.Background != display.ColorBlack {
			t.Fatalf("nested style was not preserved: %+v", span)
		}
	}
}

func TestParseStyleNestedInBoldRemainsBold(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("<b>bold <style=medium>medium bold</style> bold</b> plain")
	if err != nil {
		t.Fatal(err)
	}
	wantBold := []bool{true, true, true, false}
	wantFonts := []int16{12, 16, 12, 12}
	for index := range spans {
		if spans[index].Bold != wantBold[index] || fontIdentifier(t, spans[index].Font) != wantFonts[index] {
			t.Fatalf("span %d=%+v", index, spans[index])
		}
	}
}

func TestNamedBoldDoesNotOverrideNestedNormalStyle(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("<style=bold>A<style=normal>B</style>C</style>D")
	if err != nil {
		t.Fatal(err)
	}
	assertSpanStyles(t, spans, []expectedSpanStyle{
		{value: "A", font: 24, foreground: display.ColorRed, background: display.ColorBlack, bold: true},
		{value: "B", font: 16, foreground: display.ColorGreen, background: display.ColorWhite},
		{value: "C", font: 24, foreground: display.ColorRed, background: display.ColorBlack, bold: true},
		{value: "D", font: 12, foreground: display.ColorWhite, background: display.ColorBlack},
	})
}

func TestExplicitBoldOverridesNestedNormalStyle(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("<b>A<style=normal>B</style>C</b>D")
	if err != nil {
		t.Fatal(err)
	}
	assertSpanStyles(t, spans, []expectedSpanStyle{
		{value: "A", font: 12, foreground: display.ColorWhite, background: display.ColorBlack, bold: true},
		{value: "B", font: 16, foreground: display.ColorGreen, background: display.ColorWhite, bold: true},
		{value: "C", font: 12, foreground: display.ColorWhite, background: display.ColorBlack, bold: true},
		{value: "D", font: 12, foreground: display.ColorWhite, background: display.ColorBlack},
	})
}

func TestNamedBoldAndExplicitBoldRestoreIndependently(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("<style=bold>A<b>B<style=normal>C</style>D</b>E</style>F")
	if err != nil {
		t.Fatal(err)
	}
	assertSpanStyles(t, spans, []expectedSpanStyle{
		{value: "A", font: 24, foreground: display.ColorRed, background: display.ColorBlack, bold: true},
		{value: "B", font: 24, foreground: display.ColorRed, background: display.ColorBlack, bold: true},
		{value: "C", font: 16, foreground: display.ColorGreen, background: display.ColorWhite, bold: true},
		{value: "D", font: 24, foreground: display.ColorRed, background: display.ColorBlack, bold: true},
		{value: "E", font: 24, foreground: display.ColorRed, background: display.ColorBlack, bold: true},
		{value: "F", font: 12, foreground: display.ColorWhite, background: display.ColorBlack},
	})
}

func TestNestedBoldTagsRestoreByDepth(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("<b>A<b>B</b>C</b>D")
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 4 || !spans[0].Bold || !spans[1].Bold || !spans[2].Bold || spans[3].Bold {
		t.Fatalf("spans=%+v", spans)
	}
}

func TestParseBoldErrors(t *testing.T) {
	parser, _, _, _ := testParser()
	for _, value := range []string{"</b>", "<b>x", "<b><style=medium>x</b></style>"} {
		if spans, err := parser.Parse(value); err == nil || spans != nil {
			t.Fatalf("value=%q spans=%+v err=%v", value, spans, err)
		}
	}
}

type expectedSpanStyle struct {
	value                  string
	font                   int16
	foreground, background display.Color565
	bold                   bool
}

func assertSpanStyles(t *testing.T, spans []text.Span, want []expectedSpanStyle) {
	t.Helper()
	if len(spans) != len(want) {
		t.Fatalf("len=%d want=%d spans=%+v", len(spans), len(want), spans)
	}
	for index := range want {
		got := spans[index]
		if got.Value != want[index].value || fontIdentifier(t, got.Font) != want[index].font ||
			got.Foreground != want[index].foreground || got.Background != want[index].background || got.Bold != want[index].bold {
			t.Fatalf("span %d=%+v want=%+v", index, got, want[index])
		}
	}
}

func TestParseNestedStyles(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("<style=medium>A<style=large-red>B<style=inverse>C</style>D</style>E</style>F")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		value      string
		font       int16
		foreground display.Color565
		background display.Color565
	}{
		{"A", 16, display.ColorGreen, display.ColorBlack},
		{"B", 24, display.ColorRed, display.ColorBlack},
		{"C", 12, display.ColorBlack, display.ColorWhite},
		{"D", 24, display.ColorRed, display.ColorBlack},
		{"E", 16, display.ColorGreen, display.ColorBlack},
		{"F", 12, display.ColorWhite, display.ColorBlack},
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

func TestStyleNames(t *testing.T) {
	parser, _, medium, _ := testParser()
	parser.Styles.Entries = []text.StyleEntry{}
	for _, name := range []string{"main", "date", "sub1", "ja-main"} {
		parser.Styles.Entries = append(parser.Styles.Entries, text.StyleEntry{Name: name, Style: text.Style{Font: medium}})
		if _, err := parser.Parse("<style=" + name + ">x</style>"); err != nil {
			t.Errorf("valid name %q: %v", name, err)
		}
	}
	for _, name := range []string{"", "Main", "main_style", "-main", "main.class", "日本語", "main style"} {
		if _, err := parser.Parse("<style=" + name + ">x</style>"); err == nil {
			t.Errorf("invalid name %q succeeded", name)
		}
	}
}

func TestParseBreakEscapeNewlineAndEmptyStyle(t *testing.T) {
	parser, _, _, _ := testParser()
	spans, err := parser.Parse("a<br>b<br/>c")
	if err != nil || spanValues(spans) != "a|\n|b|\n|c" {
		t.Fatalf("break spans=%+v err=%v", spans, err)
	}
	spans, err = parser.Parse("1 << 2")
	if err != nil || concatenateValues(spans) != "1 < 2" {
		t.Fatalf("escape spans=%+v err=%v", spans, err)
	}
	spans, err = parser.Parse("a\nb")
	if err != nil || len(spans) != 1 || spans[0].Value != "a\nb" {
		t.Fatalf("newline spans=%+v err=%v", spans, err)
	}
	spans, err = parser.Parse("a<style=medium></style>b")
	if err != nil || len(spans) != 2 || spans[0].Value != "a" || spans[1].Value != "b" {
		t.Fatalf("empty style spans=%+v err=%v", spans, err)
	}
	spans, err = parser.Parse("<style=medium>a<br>b</style>")
	if err != nil || len(spans) != 3 || spans[1].Value != "\n" || fontIdentifier(t, spans[1].Font) != 16 {
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

func TestParseErrorsAndOffsets(t *testing.T) {
	parser, _, _, _ := testParser()
	tests := []struct {
		name   string
		value  string
		offset int
	}{
		{"unknown tag", "a<unknown>b", 1},
		{"unknown style", "<style=missing>x</style>", 0},
		{"nil selected font", "<style=nil-font>x</style>", 0},
		{"unterminated", "a<style=medium", 1},
		{"unexpected close", "</style>", 0},
		{"unclosed", "<style=medium>x", len("<style=medium>x")},
		{"old size 12", "<size=12>x</size>", 0},
		{"old size 16", "<size=16>x</size>", 0},
		{"old size 24", "<size=24>x</size>", 0},
		{"old foreground", "<fg=#ff0000>x</fg>", 0},
		{"old background", "<bg=#ffffff>x</bg>", 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spans, err := parser.Parse(test.value)
			if err == nil || spans != nil {
				t.Fatalf("spans=%+v err=%v", spans, err)
			}
			var syntax *SyntaxError
			if !errors.As(err, &syntax) || syntax.Offset != test.offset {
				t.Fatalf("error=%v syntax=%+v", err, syntax)
			}
		})
	}
}

func TestNestingDepth(t *testing.T) {
	parser, _, _, _ := testParser()
	depth16 := strings.Repeat("<style=medium>", 16) + "x" + strings.Repeat("</style>", 16)
	if _, err := parser.Parse(depth16); err != nil {
		t.Fatalf("depth 16: %v", err)
	}
	depth17 := strings.Repeat("<style=medium>", 17) + "x" + strings.Repeat("</style>", 17)
	if _, err := parser.Parse(depth17); err == nil {
		t.Fatal("depth 17 succeeded")
	} else {
		var syntax *SyntaxError
		if !errors.As(err, &syntax) || syntax.Offset != 16*len("<style=medium>") {
			t.Fatalf("depth error = %v", err)
		}
	}
}

func TestParseInto(t *testing.T) {
	parser, _, _, _ := testParser()
	value := "a<style=medium>b<b>c</b>d</style>e<br>f"
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
		if got[index].Value != want[index].Value || got[index].Foreground != want[index].Foreground || got[index].Background != want[index].Background || got[index].Bold != want[index].Bold || fontIdentifier(t, got[index].Font) != fontIdentifier(t, want[index].Font) {
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
