package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget/internal/bdf"
	"github.com/rdon-key/modgadget/internal/mgf"
)

const conversionBDF = `STARTFONT 2.2
FONT conversion-test
SIZE 12 75 75
FONTBOUNDINGBOX 9 2 0 -1
STARTPROPERTIES 4
FONT_ASCENT 9
FONT_DESCENT 3
CHARSET_REGISTRY "ISO10646"
CHARSET_ENCODING "1"
ENDPROPERTIES
CHARS 3
STARTCHAR hiragana-a
ENCODING 12354
DWIDTH 9 0
BBX 9 2 -1 -1
BITMAP
aa80
5500
ENDCHAR
STARTCHAR A
ENCODING 65
DWIDTH 8 0
BBX 8 1 1 2
BITMAP
81
ENDCHAR
STARTCHAR unencoded
ENCODING -1
DWIDTH 8 0
BBX 0 0 0 0
BITMAP
ENDCHAR
ENDFONT
`

func TestRunBDFConversionAndRawBytes(t *testing.T) {
	directory := t.TempDir()
	bdfPath := filepath.Join(directory, "font.bdf")
	outputPath := filepath.Join(directory, "font.mgf")
	if err := os.WriteFile(bdfPath, []byte(conversionBDF), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-bdf", bdfPath, "-font-id", "tst1", "-subset-id", "full", "-region", "JP", "-line-gap", "2", "-o", outputPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run = %d, stderr %q", code, stderr.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	immutable := string(data)
	header, err := mgf.DecodeHeader(immutable)
	if err != nil {
		t.Fatal(err)
	}
	index, err := mgf.DecodeIndex(immutable, header)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgf.ValidateGlyphData(immutable, header, index); err != nil {
		t.Fatal(err)
	}
	if header.GlyphCount != 2 || header.Ascent != 9 || header.Descent != 3 || header.LineGap != 2 || header.MaxWidth != 9 || header.MaxHeight != 2 || int(header.FileSize) != len(data) {
		t.Fatalf("header = %+v, bytes = %d", header, len(data))
	}
	// Check raw offsets independently of the MGF decoder.
	if string(data[:3]) != "MGF" || binary.LittleEndian.Uint16(data[14:16]) != 2 || binary.LittleEndian.Uint32(data[28:32]) != 52 {
		t.Fatalf("raw header = % x", data[:36])
	}
	if binary.LittleEndian.Uint32(data[36:40]) != 65 || binary.LittleEndian.Uint32(data[44:48]) != 12354 {
		t.Fatalf("raw codepoints = % x", data[36:52])
	}
	firstOffset := binary.LittleEndian.Uint32(data[40:44])
	if firstOffset != 52 || binary.LittleEndian.Uint32(data[48:52]) != 63 {
		t.Fatalf("raw glyph offsets = % x", data[40:52])
	}
	record := data[firstOffset:]
	if record[0] != 8 || record[1] != 1 || int16(binary.LittleEndian.Uint16(record[2:4])) != 8 || int16(binary.LittleEndian.Uint16(record[4:6])) != 1 || int16(binary.LittleEndian.Uint16(record[6:8])) != 3 || binary.LittleEndian.Uint16(record[8:10]) != 1 || record[10] != 0x81 {
		t.Fatalf("raw first glyph = % x", record[:11])
	}
	if !strings.Contains(stdout.String(), "GlyphCount: 2") || !strings.Contains(stdout.String(), "bytes:") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunBDFSelectionMissingAndAssumeUnicode(t *testing.T) {
	directory := t.TempDir()
	bdfPath := filepath.Join(directory, "font.bdf")
	charsPath := filepath.Join(directory, "chars.txt")
	outputPath := filepath.Join(directory, "font.mgf")
	fixture := strings.Replace(conversionBDF, "ISO10646", "JISX0208", 1)
	if err := os.WriteFile(bdfPath, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(charsPath, []byte("\ufeffあ\r\nあZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-bdf", bdfPath, "-chars", charsPath, "-missing", "skip", "-assume-unicode", "-font-id", "tst1", "-subset-id", "ui01", "-o", outputPath}
	var stdout, stderr bytes.Buffer
	if code := run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("run = %d, stderr %q", code, stderr.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	header, err := mgf.DecodeHeader(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if header.GlyphCount != 1 || !strings.Contains(stdout.String(), "missing: 1") {
		t.Fatalf("header=%+v stdout=%q", header, stdout.String())
	}
}

func TestRunBDFErrorsDoNotCreateOutput(t *testing.T) {
	directory := t.TempDir()
	bdfPath := filepath.Join(directory, "font.bdf")
	outputPath := filepath.Join(directory, "font.mgf")
	if err := os.WriteFile(bdfPath, []byte(strings.Replace(conversionBDF, "ISO10646", "JISX0208", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-bdf", bdfPath, "-font-id", "tst1", "-subset-id", "full", "-o", outputPath}, &stdout, &stderr); code == 0 {
		t.Fatal("charset conversion succeeded")
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("output stat error = %v", err)
	}
	if code := run([]string{"-chars", "unused", "-font-id", "tst1", "-subset-id", "full", "-o", outputPath}, &stdout, &stderr); code == 0 {
		t.Fatal("header-only conversion flag succeeded")
	}
}

func TestMetricConversionAndDerivation(t *testing.T) {
	glyph, err := convertGlyph(testBDFGlyph(255, 255, -32768, -32768, -255))
	if err != nil {
		t.Fatal(err)
	}
	if glyph.BearingY != 0 || glyph.AdvanceX != -32768 || glyph.BearingX != -32768 {
		t.Fatalf("glyph = %+v", glyph)
	}
	bad := testBDFGlyph(256, 1, 0, 0, 0)
	if _, err := convertGlyph(bad); err == nil {
		t.Fatal("Width 256 accepted")
	}
}

func TestFontMetricsPropertiesAndDerivation(t *testing.T) {
	propertyFont := bdf.Font{Ascent: 10, Descent: 2, HasAscent: true, HasDescent: true}
	ascent, descent, err := fontMetrics(propertyFont)
	if err != nil || ascent != 10 || descent != 2 {
		t.Fatalf("property metrics = %d,%d,%v", ascent, descent, err)
	}
	derived := bdf.Font{Glyphs: []bdf.Glyph{
		{Encoding: 65, Height: 7, YOffset: 2},
		{Encoding: 66, Height: 5, YOffset: -3},
		{Encoding: -1, Height: 255, YOffset: 100},
	}}
	ascent, descent, err = fontMetrics(derived)
	if err != nil || ascent != 9 || descent != 3 {
		t.Fatalf("derived metrics = %d,%d,%v", ascent, descent, err)
	}
	if _, _, err := fontMetrics(bdf.Font{HasAscent: true}); err == nil {
		t.Fatal("single metric property accepted")
	}
	if _, _, err := fontMetrics(bdf.Font{Glyphs: []bdf.Glyph{{Encoding: 65, Height: 256}}}); err == nil {
		t.Fatal("derived ascent 256 accepted")
	}
	for _, glyph := range []bdf.Glyph{
		{Name: "minimum-y", Encoding: 65, YOffset: math.MinInt},
		{Name: "top-overflow", Encoding: 66, YOffset: math.MaxInt, Height: 1},
	} {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("fontMetrics panicked: %v", recovered)
			}
		}()
		if _, _, err := fontMetrics(bdf.Font{Glyphs: []bdf.Glyph{glyph}}); err == nil || (!strings.Contains(err.Error(), glyph.Name) && !strings.Contains(err.Error(), "U+")) {
			t.Fatalf("glyph %+v error = %v", glyph, err)
		}
	}
}

func TestRunBDFInvalidUTF8AndAllMissing(t *testing.T) {
	directory := t.TempDir()
	bdfPath := filepath.Join(directory, "font.bdf")
	charsPath := filepath.Join(directory, "chars.txt")
	outputPath := filepath.Join(directory, "font.mgf")
	if err := os.WriteFile(bdfPath, []byte(conversionBDF), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(charsPath, []byte{0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-bdf", bdfPath, "-chars", charsPath, "-font-id", "tst1", "-subset-id", "full", "-o", outputPath}
	var stdout, stderr bytes.Buffer
	if code := run(args, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "UTF-8") {
		t.Fatalf("invalid UTF-8 code=%d stderr=%q", code, stderr.String())
	}
	if err := os.WriteFile(charsPath, []byte("Z"), 0o600); err != nil {
		t.Fatal(err)
	}
	args = append(args, "-missing", "skip")
	stdout.Reset()
	stderr.Reset()
	if code := run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("all missing code=%d stderr=%q", code, stderr.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	header, err := mgf.DecodeHeader(string(data))
	if err != nil || header.GlyphCount != 0 || len(data) != mgf.HeaderSize || !strings.Contains(stderr.String(), "warning") {
		t.Fatalf("header=%+v bytes=%d err=%v stderr=%q", header, len(data), err, stderr.String())
	}
}

func testBDFGlyph(width, height, advance, xOffset, yOffset int) (glyph bdf.Glyph) {
	glyph.Width, glyph.Height, glyph.AdvanceX, glyph.XOffset, glyph.YOffset = width, height, advance, xOffset, yOffset
	glyph.Bitmap = string(make([]byte, ((width+7)/8)*height))
	return glyph
}

func TestRunBDFRejectsUnselectedGlyphMetricOverflow(t *testing.T) {
	directory := t.TempDir()
	bdfPath := filepath.Join(directory, "font.bdf")
	charsPath := filepath.Join(directory, "chars.txt")
	outputPath := filepath.Join(directory, "font.mgf")
	fixture := `STARTFONT 2.2
FONT metric-overflow
SIZE 8 75 75
FONTBOUNDINGBOX 8 1 0 0
STARTPROPERTIES 2
CHARSET_REGISTRY "ISO10646"
CHARSET_ENCODING "1"
ENDPROPERTIES
CHARS 2
STARTCHAR A
ENCODING 65
DWIDTH 8 0
BBX 8 1 0 0
BITMAP
81
ENDCHAR
STARTCHAR bad
ENCODING 66
DWIDTH 8 0
BBX 0 0 0 ` + strconv.Itoa(math.MinInt) + `
BITMAP
ENDCHAR
ENDFONT
`
	if err := os.WriteFile(bdfPath, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(charsPath, []byte("A"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("run panicked: %v", recovered)
		}
	}()
	var stdout, stderr bytes.Buffer
	code := run([]string{"-bdf", bdfPath, "-chars", charsPath, "-font-id", "tst1", "-subset-id", "full", "-o", outputPath}, &stdout, &stderr)
	if code == 0 || (!strings.Contains(stderr.String(), "bad") && !strings.Contains(stderr.String(), "Descent")) {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("output exists or stat failed: %v", err)
	}
}

func TestRunBDFHugeBitmapDoesNotCreateOutput(t *testing.T) {
	directory := t.TempDir()
	bdfPath := filepath.Join(directory, "font.bdf")
	outputPath := filepath.Join(directory, "font.mgf")
	fixture := strings.Replace(conversionBDF, "BBX 9 2 -1 -1", "BBX "+strconv.Itoa(math.MaxInt)+" 2 -1 -1", 1)
	if err := os.WriteFile(bdfPath, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("run panicked: %v", recovered)
		}
	}()
	var stdout, stderr bytes.Buffer
	code := run([]string{"-bdf", bdfPath, "-font-id", "tst1", "-subset-id", "full", "-o", outputPath}, &stdout, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), "bitmap size") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("output exists or stat failed: %v", err)
	}
}
