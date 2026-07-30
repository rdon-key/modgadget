package mgf

import (
	"strings"
	"testing"
)

func TestOpenAndFontMethods(t *testing.T) {
	data := fontTestData(t)
	font, err := Open(data)
	if err != nil {
		t.Fatal(err)
	}
	if font.Header().GlyphCount != 3 || font.GlyphCount() != 3 {
		t.Fatalf("header=%+v count=%d", font.Header(), font.GlyphCount())
	}
	if font.LineHeight() != 13 {
		t.Fatalf("LineHeight = %d", font.LineHeight())
	}
	minimum, ok := font.Lookup(0)
	if !ok {
		t.Fatal("minimum Codepoint not found")
	}
	if minimum.Width != 0 || minimum.Height != 0 || minimum.AdvanceX != 4 || minimum.Bitmap != "" {
		t.Fatalf("minimum = %+v", minimum)
	}
	a, ok := font.Lookup('A')
	if !ok {
		t.Fatal("A not found")
	}
	if a.Width != 8 || a.Height != 1 || a.AdvanceX != 9 || a.BearingX != -1 || a.BearingY != 2 || a.Bitmap != "\x81" {
		t.Fatalf("A = %+v", a)
	}
	if _, ok := font.Lookup('B'); ok {
		t.Fatal("unexpected B")
	}
	if _, ok := font.Lookup(-1); ok {
		t.Fatal("unexpected negative rune")
	}
	if _, ok := font.Lookup(rune(0xd800)); ok {
		t.Fatal("unexpected surrogate")
	}
	maximum, ok := font.Lookup(rune(0x10ffff))
	if !ok || maximum.Bitmap != "\xff" {
		t.Fatalf("maximum = %+v, %v", maximum, ok)
	}
	if _, ok := font.Lookup(rune(0x110000)); ok {
		t.Fatal("unexpected rune above Unicode maximum")
	}
}

func TestZeroValueFont(t *testing.T) {
	var font Font
	if font.Header() != (Header{}) || font.GlyphCount() != 0 || font.LineHeight() != 0 {
		t.Fatalf("zero font = header %+v count %d height %d", font.Header(), font.GlyphCount(), font.LineHeight())
	}
	if _, ok := font.Lookup('A'); ok {
		t.Fatal("zero Font lookup succeeded")
	}
}

func TestOpenErrorsHaveContext(t *testing.T) {
	if _, err := Open("bad"); err == nil || !strings.Contains(err.Error(), "open header") {
		t.Fatalf("header error = %v", err)
	}
	data := []byte(fontTestData(t))
	data[36] = 0xff
	data[37] = 0xff
	data[38] = 0x11
	if _, err := Open(string(data)); err == nil || !strings.Contains(err.Error(), "open index") {
		t.Fatalf("index error = %v", err)
	}
	data = []byte(fontTestData(t))
	data[len(data)-1] ^= 1
	// Padding-bit changes remain valid, so make the first record length invalid.
	data[HeaderSize+3*IndexEntrySize+8] = 2
	if _, err := Open(string(data)); err == nil || !strings.Contains(err.Error(), "open glyph data") {
		t.Fatalf("glyph data error = %v", err)
	}
	data = append([]byte(fontTestData(t)), 0)
	if _, err := Open(string(data)); err == nil || !strings.Contains(err.Error(), "open header") {
		t.Fatalf("trailing data error = %v", err)
	}
	data = []byte(fontTestData(t))
	for index := HeaderSize + IndexEntrySize; index < HeaderSize+IndexEntrySize+4; index++ {
		data[index] = 0
	}
	if _, err := Open(string(data)); err == nil || !strings.Contains(err.Error(), "open index") {
		t.Fatalf("file index corruption error = %v", err)
	}
}

func TestMustOpen(t *testing.T) {
	font := MustOpen(fontTestData(t))
	if font.GlyphCount() != 3 {
		t.Fatalf("count = %d", font.GlyphCount())
	}
	defer func() {
		if recover() == nil {
			t.Fatal("MustOpen did not panic")
		}
	}()
	_ = MustOpen("invalid")
}

func TestOpenAllocations(t *testing.T) {
	data := fontTestData(t)
	allocations := testing.AllocsPerRun(100, func() {
		font, err := Open(data)
		if err != nil || font.GlyphCount() != 3 {
			panic("Open failed")
		}
	})
	if allocations != 0 {
		t.Fatalf("Open allocations = %v, want 0", allocations)
	}
}

func TestFontLookupAllocations(t *testing.T) {
	font := MustOpen(fontTestData(t))
	tests := []struct {
		name string
		call func()
	}{
		{"lookup hit", func() {
			if glyph, ok := font.Lookup('A'); !ok || glyph.Bitmap != "\x81" {
				panic("hit")
			}
		}},
		{"lookup miss", func() {
			if _, ok := font.Lookup('B'); ok {
				panic("miss")
			}
		}},
		{"header", func() {
			if font.Header().GlyphCount != 3 {
				panic("header")
			}
		}},
		{"glyph count", func() {
			if font.GlyphCount() != 3 {
				panic("count")
			}
		}},
		{"line height", func() {
			if font.LineHeight() != 13 {
				panic("height")
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if allocations := testing.AllocsPerRun(100, test.call); allocations != 0 {
				t.Fatalf("allocations = %v, want 0", allocations)
			}
		})
	}
}

func fontTestData(t *testing.T) string {
	t.Helper()
	glyphs := []Glyph{
		{Width: 0, Height: 0, AdvanceX: 4},
		{Width: 8, Height: 1, AdvanceX: 9, BearingX: -1, BearingY: 2, Bitmap: "\x81"},
		{Width: 1, Height: 1, AdvanceX: 1, Bitmap: "\xff"},
	}
	header := Header{
		Version: Version1, FontID: [4]byte{'t', 'e', 's', 't'}, SubsetID: [4]byte{'f', 'u', 'l', 'l'},
		GlyphCount: 3, Ascent: 10, Descent: 2, LineGap: 1, MaxWidth: 8, MaxHeight: 1,
		HeaderSize: HeaderSize, IndexOffset: HeaderSize,
		GlyphDataOffset: HeaderSize + 3*IndexEntrySize,
	}
	header.FileSize = header.GlyphDataOffset + uint32(3*GlyphRecordHeaderSize+2)
	data := make([]byte, header.FileSize)
	if err := EncodeHeader(data, header); err != nil {
		t.Fatal(err)
	}
	entries := []IndexEntry{
		{Codepoint: 0, GlyphOffset: header.GlyphDataOffset},
		{Codepoint: 'A', GlyphOffset: header.GlyphDataOffset + GlyphRecordHeaderSize},
		{Codepoint: 0x10ffff, GlyphOffset: header.GlyphDataOffset + 2*GlyphRecordHeaderSize + 1},
	}
	if err := EncodeIndex(data[header.IndexOffset:header.GlyphDataOffset], header, entries); err != nil {
		t.Fatal(err)
	}
	for index, glyph := range glyphs {
		if _, err := EncodeGlyphRecord(data[entries[index].GlyphOffset:], glyph); err != nil {
			t.Fatal(err)
		}
	}
	return string(data)
}
