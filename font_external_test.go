package modgadget_test

import (
	_ "embed"
	"testing"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget/font/efont16"
	"github.com/rdon-key/modgadget/font/efont24"
	"github.com/rdon-key/modgadget/font/shinonome12"
	"github.com/rdon-key/modgadget/font/spleen8x16"
)

//go:embed font/efont16/efont16-full.mgf
var validMGF string

func TestZeroFontIsSafe(t *testing.T) {
	var font modgadget.Font
	if font.Valid() || font.HasGlyph('A') || font.Metrics() != (modgadget.FontMetrics{}) {
		t.Fatalf("zero font valid=%v glyph=%v metrics=%+v", font.Valid(), font.HasGlyph('A'), font.Metrics())
	}
	if _, err := modgadget.MeasureText("A", modgadget.StyleSet{Default: modgadget.Style{Font: font}}); err == nil {
		t.Fatal("invalid font was accepted")
	}
}

func TestOpenMGFAndMustOpenMGF(t *testing.T) {
	font, err := modgadget.OpenMGF(validMGF)
	if err != nil {
		t.Fatal(err)
	}
	if !font.Valid() || !font.HasGlyph('A') || font.HasGlyph('\U0010ffff') {
		t.Fatalf("valid=%v A=%v missing=%v", font.Valid(), font.HasGlyph('A'), font.HasGlyph('\U0010ffff'))
	}
	if got := font.Metrics(); got != (modgadget.FontMetrics{Ascent: 14, Descent: 2}) {
		t.Fatalf("metrics=%+v", got)
	}
	if !modgadget.MustOpenMGF(validMGF).Valid() {
		t.Fatal("MustOpenMGF returned invalid font")
	}
	badMagic := "BAD" + validMGF[3:]
	invalidIndex := validMGF[:24] + "\x00\x00\x00\x00" + validMGF[28:]
	for name, data := range map[string]string{
		"truncated": validMGF[:20], "magic": badMagic, "index": invalidIndex,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := modgadget.OpenMGF(data); err == nil {
				t.Fatal("invalid MGF was accepted")
			}
		})
	}
	defer func() {
		if recover() == nil {
			t.Fatal("MustOpenMGF did not panic")
		}
	}()
	modgadget.MustOpenMGF("invalid")
}

func TestStandardFonts(t *testing.T) {
	tests := []struct {
		name string
		font modgadget.Font
		rune rune
	}{
		{"efont16", efont16.Font, 'あ'},
		{"efont24", efont24.Font, 'あ'},
		{"shinonome12", shinonome12.Font, 'あ'},
		{"spleen8x16", spleen8x16.Font, 'A'},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !test.font.Valid() || !test.font.HasGlyph(test.rune) {
				t.Fatalf("valid=%v glyph=%v", test.font.Valid(), test.font.HasGlyph(test.rune))
			}
		})
	}
}

func TestFontStack(t *testing.T) {
	stack, err := modgadget.NewFontStack(efont24.Font, efont16.Font, spleen8x16.Font)
	if err != nil {
		t.Fatal(err)
	}
	if !stack.HasGlyph('A') || !stack.HasGlyph('あ') || stack.HasGlyph('\U0010ffff') {
		t.Fatalf("stack lookup A=%v Japanese=%v missing=%v", stack.HasGlyph('A'), stack.HasGlyph('あ'), stack.HasGlyph('\U0010ffff'))
	}
	measurement, err := modgadget.MeasureText("A", modgadget.StyleSet{Default: modgadget.Style{Font: stack}})
	if err != nil || measurement.Width != 12 {
		t.Fatalf("primary measurement=%+v err=%v", measurement, err)
	}
	if stack.Metrics() != (modgadget.FontMetrics{Ascent: 22, Descent: 4}) {
		t.Fatalf("metrics=%+v", stack.Metrics())
	}
	firstFallback, err := modgadget.NewFontStack(shinonome12.Font, efont24.Font, spleen8x16.Font)
	if err != nil {
		t.Fatal(err)
	}
	secondFallback, err := modgadget.NewFontStack(shinonome12.Font, spleen8x16.Font, efont24.Font)
	if err != nil {
		t.Fatal(err)
	}
	firstMeasurement, err := modgadget.MeasureText("M", modgadget.StyleSet{Default: modgadget.Style{Font: firstFallback}})
	if err != nil {
		t.Fatal(err)
	}
	secondMeasurement, err := modgadget.MeasureText("M", modgadget.StyleSet{Default: modgadget.Style{Font: secondFallback}})
	if err != nil || firstMeasurement.Width != 12 || secondMeasurement.Width != 8 {
		t.Fatalf("fallback order first=%+v second=%+v err=%v", firstMeasurement, secondMeasurement, err)
	}
	var invalid modgadget.Font
	if _, err := modgadget.NewFontStack(invalid); err == nil {
		t.Fatal("invalid primary was accepted")
	}
	if _, err := modgadget.NewFontStack(efont16.Font, invalid); err == nil {
		t.Fatal("invalid fallback was accepted")
	}
	if _, err := modgadget.NewFontStack(efont16.Font, efont16.Font, efont16.Font, efont16.Font, efont16.Font); err == nil {
		t.Fatal("too many fallbacks were accepted")
	}
	if allocations := testing.AllocsPerRun(100, func() { _ = stack.HasGlyph('A') }); allocations != 0 {
		t.Fatalf("lookup allocations=%v", allocations)
	}
}

func TestMeasureTextUsesViewportMarkupRules(t *testing.T) {
	styles := modgadget.StyleSet{
		Default: modgadget.Style{Font: efont16.Font},
		Entries: []modgadget.StyleEntry{{Name: "large", Style: modgadget.Style{Font: efont24.Font}}},
	}
	tests := []struct {
		name  string
		value string
		lines int
	}{
		{"plain", "ABC", 1},
		{"markup", "A<style=large>B</style>C", 1},
		{"bold", "A<b>B</b>C", 1},
		{"break", "A<br>B", 2},
		{"self closing break", "A<br/>B", 2},
		{"escaped less", "A<<B", 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := modgadget.MeasureText(test.value, styles)
			if err != nil || got.LineCount != test.lines || got.Width <= 0 {
				t.Fatalf("measurement=%+v err=%v", got, err)
			}
		})
	}
	for _, value := range []string{"<unknown>x</unknown>", "<b>unclosed", "\U0010ffff"} {
		if _, err := modgadget.MeasureText(value, styles); err == nil {
			t.Fatalf("value %q was accepted", value)
		}
	}
	if got, err := modgadget.MeasureText("", styles); err != nil || got != (modgadget.TextMeasurement{}) {
		t.Fatalf("empty measurement=%+v err=%v", got, err)
	}
	if got, err := modgadget.MeasureText("A<br>B", styles); err != nil || got.LineCount != 2 {
		t.Fatalf("multiple lines=%+v err=%v", got, err)
	}
}
