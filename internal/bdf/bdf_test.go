package bdf

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

const testBDF = `STARTFONT 2.2
FONT test
SIZE 12 75 75
FONTBOUNDINGBOX 9 2 0 -1
METRICSSET 0
DWIDTH 8 0
STARTPROPERTIES 4
FONT_ASCENT 9
FONT_DESCENT 3
CHARSET_REGISTRY "ISO10646"
CHARSET_ENCODING "1"
ENDPROPERTIES
CHARS 2
STARTCHAR A
ENCODING 65
SWIDTH 500 0
DWIDTH 9 0
BBX 9 2 -1 -1
BITMAP
aa80
5500
ENDCHAR
STARTCHAR unencoded
ENCODING -1 1234
BBX 0 0 0 0
BITMAP
ENDCHAR
ENDFONT
`

func TestParse(t *testing.T) {
	font, err := Parse(testBDF)
	if err != nil {
		t.Fatal(err)
	}
	if font.Version != "2.2" || font.Name != "test" || font.CharsetRegistry != "ISO10646" || font.CharsetEncoding != "1" {
		t.Fatalf("font metadata = %+v", font)
	}
	if !font.HasAscent || font.Ascent != 9 || !font.HasDescent || font.Descent != 3 {
		t.Fatalf("metrics = %+v", font)
	}
	if len(font.Glyphs) != 2 {
		t.Fatalf("glyph count = %d", len(font.Glyphs))
	}
	a := font.Glyphs[0]
	if a.Encoding != 65 || a.AdvanceX != 9 || a.Width != 9 || a.Height != 2 || a.XOffset != -1 || a.YOffset != -1 {
		t.Fatalf("A = %+v", a)
	}
	if a.Bitmap != string([]byte{0xaa, 0x80, 0x55, 0x00}) {
		t.Fatalf("bitmap = % x", []byte(a.Bitmap))
	}
	if font.Glyphs[1].Encoding != -1 || font.Glyphs[1].Bitmap != "" {
		t.Fatalf("unencoded = %+v", font.Glyphs[1])
	}
}

func TestParseBDF21CRLFAndGlobalDWIDTH(t *testing.T) {
	fixture := strings.Replace(strings.Replace(testBDF, "STARTFONT 2.2", "STARTFONT 2.1", 1), "DWIDTH 9 0\n", "", 1)
	font, err := Parse(strings.ReplaceAll(fixture, "\n", "\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if font.Version != "2.1" || font.Glyphs[0].AdvanceX != 8 {
		t.Fatalf("font = %+v", font)
	}
}

func TestParseBitmapWidthsAndCase(t *testing.T) {
	fixture := strings.Replace(testBDF, "CHARS 2", "CHARS 1", 1)
	fixture = fixture[:strings.Index(fixture, "STARTCHAR unencoded")] + "ENDFONT\n"
	font, err := Parse(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(font.Glyphs[0].Bitmap); got != 4 {
		t.Fatalf("bitmap bytes = %d", got)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		contains string
	}{
		{"version", "STARTFONT 2.2", "STARTFONT 2.0", "version"},
		{"chars mismatch", "CHARS 2", "CHARS 3", "CHARS"},
		{"metrics set one", "METRICSSET 0", "METRICSSET 1", "METRICSSET"},
		{"dwidth y", "DWIDTH 9 0", "DWIDTH 9 1", "DWIDTH Y"},
		{"encoding", "ENCODING 65", "ENCODING -2", "ENCODING"},
		{"duplicate encoding", "ENCODING 65", "ENCODING 65\nENCODING 66", "duplicate ENCODING"},
		{"missing dwidth", "DWIDTH 8 0\n", "", "missing DWIDTH"},
		{"short bitmap", "aa80\n5500", "aa80", "BITMAP"},
		{"bad hex", "aa80", "zz80", "BITMAP"},
		{"long row", "aa80", "aa8000", "hex digits"},
		{"missing endfont", "ENDFONT\n", "", "ENDFONT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := strings.Replace(testBDF, test.old, test.new, 1)
			_, err := Parse(fixture)
			if err == nil || !strings.Contains(err.Error(), test.contains) || !strings.Contains(err.Error(), "line") {
				t.Fatalf("error = %v, want line and %q", err, test.contains)
			}
		})
	}
}

func TestParseLongIgnoredLine(t *testing.T) {
	fixture := strings.Replace(testBDF, "FONT test", "FONT "+strings.Repeat("x", 100000), 1)
	if _, err := Parse(fixture); err != nil {
		t.Fatal(err)
	}
}

func TestParseHugeBitmapDimensionsReturnErrorWithoutPanic(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"maximum width", math.MaxInt, 1},
		{"maximum height", 8, math.MaxInt},
		{"product exceeds int", math.MaxInt, 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := strings.Replace(testBDF, "BBX 9 2 -1 -1", "BBX "+strconv.Itoa(test.width)+" "+strconv.Itoa(test.height)+" -1 -1", 1)
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("Parse panicked: %v", recovered)
				}
			}()
			_, err := Parse(fixture)
			if err == nil || (!strings.Contains(err.Error(), "A") && !strings.Contains(err.Error(), "bitmap size")) {
				t.Fatalf("error = %v, want glyph or bitmap size", err)
			}
		})
	}
}

func TestParseRejectsGlobalFieldsAfterCHARS(t *testing.T) {
	fields := []string{
		"FONT later",
		"SIZE 12 75 75",
		"FONTBOUNDINGBOX 8 8 0 0",
		"STARTPROPERTIES 0\nENDPROPERTIES",
		"DWIDTH 8 0",
		"METRICSSET 0",
	}
	for _, field := range fields {
		name := strings.Fields(field)[0]
		t.Run(name, func(t *testing.T) {
			fixture := strings.Replace(testBDF, "CHARS 2\n", "CHARS 2\n"+field+"\n", 1)
			_, err := Parse(fixture)
			if err == nil || !strings.Contains(err.Error(), "line") || !strings.Contains(err.Error(), name) {
				t.Fatalf("error = %v, want line and %s", err, name)
			}
		})
	}
}

func TestParseRejectsDuplicateAndMissingGlobalFields(t *testing.T) {
	duplicates := []string{
		"FONT test",
		"SIZE 12 75 75",
		"FONTBOUNDINGBOX 9 2 0 -1",
		"STARTPROPERTIES 0\nENDPROPERTIES",
		"METRICSSET 0",
		"DWIDTH 8 0",
	}
	for _, field := range duplicates {
		name := strings.Fields(field)[0]
		t.Run("duplicate "+name, func(t *testing.T) {
			fixture := strings.Replace(testBDF, "CHARS 2", field+"\nCHARS 2", 1)
			_, err := Parse(fixture)
			if err == nil || !strings.Contains(err.Error(), "line") || !strings.Contains(err.Error(), name) {
				t.Fatalf("error = %v, want line and %s", err, name)
			}
		})
	}
	missing := []string{
		"FONT test\n",
		"SIZE 12 75 75\n",
		"FONTBOUNDINGBOX 9 2 0 -1\n",
	}
	for _, field := range missing {
		name := strings.Fields(field)[0]
		t.Run("missing "+name, func(t *testing.T) {
			fixture := strings.Replace(testBDF, field, "", 1)
			_, err := Parse(fixture)
			if err == nil || !strings.Contains(err.Error(), "line") || !strings.Contains(err.Error(), name) {
				t.Fatalf("error = %v, want line and %s", err, name)
			}
		})
	}
}
