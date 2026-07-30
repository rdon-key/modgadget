package mgf

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestRawBitmapSize(t *testing.T) {
	tests := []struct {
		width, height uint8
		want          uint16
	}{
		{0, 0, 0},
		{0, 12, 0},
		{8, 1, 1},
		{9, 1, 2},
		{12, 12, 24},
		{8, 16, 16},
		{255, 255, 8160},
	}
	for _, tt := range tests {
		if got := RawBitmapSize(tt.width, tt.height); got != tt.want {
			t.Fatalf("%dx%d=%d want=%d", tt.width, tt.height, got, tt.want)
		}
	}
}

func TestGlyphRecordEncodeDecodeRoundTrip(t *testing.T) {
	tests := []Glyph{
		{Width: 12, Height: 12, AdvanceX: 12, BearingX: 1, BearingY: 10, Bitmap: bitmapString(12, 12)},
		{Width: 8, Height: 16, AdvanceX: 8, BearingX: -1, BearingY: -2, Bitmap: bitmapString(8, 16)},
		{Width: 9, Height: 1, AdvanceX: -3, BearingX: math.MinInt16, BearingY: math.MaxInt16, Bitmap: bitmapString(9, 1)},
		{Width: 0, Height: 12, Bitmap: ""},
		{Width: 12, Height: 0, Bitmap: ""},
		{Width: 0, Height: 0, Bitmap: ""},
	}
	for _, want := range tests {
		t.Run(glyphTestName(want), func(t *testing.T) {
			recordSize := GlyphRecordHeaderSize + len(want.Bitmap)
			header := indexHeader(1, uint32(recordSize))
			header.MaxWidth = want.Width
			header.MaxHeight = want.Height
			data := make([]byte, header.FileSize)
			originalBitmap := want.Bitmap
			written, err := EncodeGlyphRecord(data[header.GlyphDataOffset:], want)
			if err != nil || written != recordSize {
				t.Fatalf("written=%d err=%v", written, err)
			}
			if want.Bitmap != originalBitmap {
				t.Fatal("Bitmap changed")
			}
			if binary.LittleEndian.Uint16(data[header.GlyphDataOffset+2:header.GlyphDataOffset+4]) != uint16(want.AdvanceX) ||
				binary.LittleEndian.Uint16(data[header.GlyphDataOffset+4:header.GlyphDataOffset+6]) != uint16(want.BearingX) ||
				binary.LittleEndian.Uint16(data[header.GlyphDataOffset+6:header.GlyphDataOffset+8]) != uint16(want.BearingY) {
				t.Fatalf("signed fields=%x", data[header.GlyphDataOffset+2:header.GlyphDataOffset+8])
			}
			file := string(data)
			got, next, err := DecodeGlyphRecord(file, header, header.GlyphDataOffset)
			if err != nil || got != want || next != header.FileSize {
				t.Fatalf("got=%+v next=%d err=%v want=%+v", got, next, err, want)
			}
			if got.Bitmap != file[int(header.GlyphDataOffset)+GlyphRecordHeaderSize:] {
				t.Fatal("Bitmap is not the file substring")
			}
		})
	}
}

func TestEncodeGlyphRecordErrorsAndDestination(t *testing.T) {
	valid := Glyph{Width: 9, Height: 1, Bitmap: "\x80\x00"}
	required := GlyphRecordHeaderSize + len(valid.Bitmap)
	if _, err := EncodeGlyphRecord(make([]byte, required-1), valid); err == nil {
		t.Fatal("short destination succeeded")
	}
	for _, bitmap := range []string{"\x80", "\x80\x00\x00"} {
		dst := make([]byte, required)
		for index := range dst {
			dst[index] = 0x5a
		}
		want := append([]byte(nil), dst...)
		glyph := valid
		glyph.Bitmap = bitmap
		if _, err := EncodeGlyphRecord(dst, glyph); err == nil || !reflect.DeepEqual(dst, want) {
			t.Fatalf("bitmap len=%d err=%v dst=%x", len(bitmap), err, dst)
		}
	}
	dst := make([]byte, required+3)
	for index := range dst {
		dst[index] = 0xaa
	}
	if written, err := EncodeGlyphRecord(dst, valid); err != nil || written != required {
		t.Fatalf("written=%d err=%v", written, err)
	}
	if string(dst[required:]) != "\xaa\xaa\xaa" {
		t.Fatalf("suffix=%x", dst[required:])
	}
}

func TestDecodeGlyphRecordErrors(t *testing.T) {
	glyph := Glyph{Width: 9, Height: 2, Bitmap: bitmapString(9, 2)}
	header, data, _ := buildGlyphFile(t, []Glyph{glyph})
	offset := header.GlyphDataOffset
	tests := []struct {
		name   string
		change func(*Header, []byte, *uint32)
		text   string
	}{
		{"before GlyphDataOffset", func(header *Header, _ []byte, offset *uint32) { *offset = header.GlyphDataOffset - 1 }, "GlyphDataOffset"},
		{"at FileSize", func(header *Header, _ []byte, offset *uint32) { *offset = header.FileSize }, "FileSize"},
		{"fixed header truncated", func(header *Header, data []byte, _ *uint32) {
			header.FileSize = uint32(len(data))
			header.GlyphDataOffset = HeaderSize + IndexEntrySize
		}, "truncated"},
		{"bitmap truncated", func(header *Header, data []byte, _ *uint32) {
			binary.LittleEndian.PutUint16(data[offset+8:offset+10], uint16(len(glyph.Bitmap)+1))
		}, "truncated"},
		{"DataLength short", func(_ *Header, data []byte, _ *uint32) {
			binary.LittleEndian.PutUint16(data[offset+8:offset+10], uint16(len(glyph.Bitmap)-1))
		}, "DataLength"},
		{"DataLength long", func(header *Header, data []byte, _ *uint32) {
			data = append(data, 0)
			header.FileSize++
			binary.LittleEndian.PutUint16(data[offset+8:offset+10], uint16(len(glyph.Bitmap)+1))
		}, "DataLength"},
		{"Width exceeds", func(header *Header, _ []byte, _ *uint32) { header.MaxWidth = glyph.Width - 1 }, "MaxWidth"},
		{"Height exceeds", func(header *Header, _ []byte, _ *uint32) { header.MaxHeight = glyph.Height - 1 }, "MaxHeight"},
		{"unsupported Flags", func(header *Header, _ []byte, _ *uint32) { header.Flags = 1 }, "Flags"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changedHeader := header
			changedData := append([]byte(nil), data...)
			changedOffset := offset
			tt.change(&changedHeader, changedData, &changedOffset)
			if tt.name == "fixed header truncated" {
				changedData = changedData[:changedOffset+GlyphRecordHeaderSize-1]
				changedHeader.FileSize = uint32(len(changedData))
			}
			if tt.name == "DataLength long" {
				changedData = append(changedData, 0)
				changedHeader.FileSize = uint32(len(changedData))
				binary.LittleEndian.PutUint16(changedData[offset+8:offset+10], uint16(len(glyph.Bitmap)+1))
			}
			_, _, err := DecodeGlyphRecord(string(changedData), changedHeader, changedOffset)
			if err == nil || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("err=%v", err)
			}
		})
	}
	mismatchHeader := header
	mismatchHeader.FileSize++
	if _, _, err := DecodeGlyphRecord(string(data), mismatchHeader, offset); err == nil || !strings.Contains(err.Error(), "data length") {
		t.Fatalf("FileSize mismatch err=%v", err)
	}
}

func TestValidateGlyphData(t *testing.T) {
	empty := emptyHeader()
	emptyData := make([]byte, HeaderSize)
	if err := EncodeHeader(emptyData, empty); err != nil {
		t.Fatal(err)
	}
	emptyIndex, err := DecodeIndex(string(emptyData), empty)
	if err != nil || ValidateGlyphData(string(emptyData), empty, emptyIndex) != nil {
		t.Fatalf("empty index=%+v err=%v", emptyIndex, err)
	}

	glyphs := []Glyph{
		{Width: 8, Height: 1, Bitmap: bitmapString(8, 1)},
		{Width: 12, Height: 2, Bitmap: bitmapString(12, 2)},
	}
	header, data, index := buildGlyphFile(t, glyphs)
	if err := ValidateGlyphData(string(data), header, index); err != nil {
		t.Fatalf("multiple glyphs: %v", err)
	}
	oneHeader, oneData, oneIndex := buildGlyphFile(t, glyphs[:1])
	if err := ValidateGlyphData(string(oneData), oneHeader, oneIndex); err != nil {
		t.Fatalf("one glyph: %v", err)
	}

	tests := []struct {
		name   string
		change func(*Header, *[]byte, *Index)
		text   string
	}{
		{"first offset shifted", func(_ *Header, _ *[]byte, index *Index) {
			*index = changedIndexOffset(t, *index, 0, header.GlyphDataOffset+1)
		}, "glyph 0 offset"},
		{"gap", func(_ *Header, _ *[]byte, index *Index) {
			entry, _ := index.Entry(1)
			*index = changedIndexOffset(t, *index, 1, entry.GlyphOffset+1)
		}, "glyph 1 offset"},
		{"overlap", func(_ *Header, _ *[]byte, index *Index) {
			entry, _ := index.Entry(1)
			*index = changedIndexOffset(t, *index, 1, entry.GlyphOffset-1)
		}, "glyph 1 offset"},
		{"duplicate offset", func(_ *Header, _ *[]byte, index *Index) {
			first, _ := index.Entry(0)
			*index = changedIndexOffset(t, *index, 1, first.GlyphOffset)
		}, "GlyphOffset"},
		{"trailing byte", func(header *Header, data *[]byte, _ *Index) { *data = append(*data, 0); header.FileSize++ }, "trailing"},
		{"last record exceeds", func(_ *Header, data *[]byte, _ *Index) {
			last, _ := index.Entry(1)
			binary.LittleEndian.PutUint16((*data)[last.GlyphOffset+8:last.GlyphOffset+10], 0xffff)
		}, "truncated"},
		{"MaxWidth mismatch", func(header *Header, _ *[]byte, _ *Index) { header.MaxWidth++ }, "MaxWidth"},
		{"MaxHeight mismatch", func(header *Header, _ *[]byte, _ *Index) { header.MaxHeight++ }, "MaxHeight"},
		{"zero index", func(_ *Header, _ *[]byte, index *Index) { *index = Index{} }, "GlyphCount"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changedHeader := header
			changedData := append([]byte(nil), data...)
			changedIndex := index
			tt.change(&changedHeader, &changedData, &changedIndex)
			if tt.name == "first offset shifted" || tt.name == "gap" || tt.name == "overlap" || tt.name == "duplicate offset" {
				copy(changedData[changedHeader.IndexOffset:changedHeader.GlyphDataOffset], changedIndex.data)
			}
			if err := ValidateGlyphData(string(changedData), changedHeader, changedIndex); err == nil || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("err=%v", err)
			}
		})
	}

	badEmpty := empty
	badEmpty.MaxWidth = 1
	if err := ValidateGlyphData(string(emptyData), badEmpty, emptyIndex); err == nil || !strings.Contains(err.Error(), "MaxWidth") {
		t.Fatalf("empty maxima err=%v", err)
	}
}

func TestValidateGlyphDataRequiresMatchingFileIndex(t *testing.T) {
	glyphs := []Glyph{
		{Width: 8, Height: 1, Bitmap: bitmapString(8, 1)},
		{Width: 12, Height: 2, Bitmap: bitmapString(12, 2)},
	}
	header, data, index := buildGlyphFile(t, glyphs)
	changedData := append([]byte(nil), data...)
	changedData[header.IndexOffset] ^= 1
	if err := ValidateGlyphData(string(changedData), header, index); err == nil || !strings.Contains(err.Error(), "index") {
		t.Fatalf("mismatched file index err=%v", err)
	}
}

func TestValidateGlyphDataAcceptsCopiedIndexContent(t *testing.T) {
	glyph := Glyph{Width: 8, Height: 1, Bitmap: bitmapString(8, 1)}
	header, data, index := buildGlyphFile(t, []Glyph{glyph})
	copiedData := string(append([]byte(nil), index.data...))
	copiedIndex := Index{data: copiedData, count: index.count}
	if err := ValidateGlyphData(string(data), header, copiedIndex); err != nil {
		t.Fatalf("copied index content: %v", err)
	}
}

func TestGlyphSuccessfulPathsDoNotAllocate(t *testing.T) {
	glyph := Glyph{Width: 12, Height: 12, Bitmap: bitmapString(12, 12)}
	dst := make([]byte, GlyphRecordHeaderSize+len(glyph.Bitmap))
	if allocations := testing.AllocsPerRun(100, func() {
		if _, err := EncodeGlyphRecord(dst, glyph); err != nil {
			panic(err)
		}
	}); allocations != 0 {
		t.Fatalf("EncodeGlyphRecord allocations=%v", allocations)
	}
	header, data, index := buildGlyphFile(t, []Glyph{glyph})
	encoded := string(data)
	if allocations := testing.AllocsPerRun(100, func() {
		if _, _, err := DecodeGlyphRecord(encoded, header, header.GlyphDataOffset); err != nil {
			panic(err)
		}
	}); allocations != 0 {
		t.Fatalf("DecodeGlyphRecord allocations=%v", allocations)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if err := ValidateGlyphData(encoded, header, index); err != nil {
			panic(err)
		}
	}); allocations != 0 {
		t.Fatalf("ValidateGlyphData allocations=%v", allocations)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		_ = RawBitmapSize(255, 255)
	}); allocations != 0 {
		t.Fatalf("RawBitmapSize allocations=%v", allocations)
	}
}

func buildGlyphFile(t *testing.T, glyphs []Glyph) (Header, []byte, Index) {
	t.Helper()
	header := indexHeader(uint16(len(glyphs)), 0)
	var maxWidth, maxHeight uint8
	recordBytes := 0
	for _, glyph := range glyphs {
		recordBytes += GlyphRecordHeaderSize + len(glyph.Bitmap)
		if glyph.Width > maxWidth {
			maxWidth = glyph.Width
		}
		if glyph.Height > maxHeight {
			maxHeight = glyph.Height
		}
	}
	header.MaxWidth = maxWidth
	header.MaxHeight = maxHeight
	header.FileSize = header.GlyphDataOffset + uint32(recordBytes)
	entries := make([]IndexEntry, len(glyphs))
	offset := header.GlyphDataOffset
	for position, glyph := range glyphs {
		entries[position] = IndexEntry{Codepoint: uint32('A' + position), GlyphOffset: offset}
		offset += uint32(GlyphRecordHeaderSize + len(glyph.Bitmap))
	}
	data := make([]byte, header.FileSize)
	if err := EncodeHeader(data[:HeaderSize], header); err != nil {
		t.Fatal(err)
	}
	if err := EncodeIndex(data[header.IndexOffset:header.GlyphDataOffset], header, entries); err != nil {
		t.Fatal(err)
	}
	for position, glyph := range glyphs {
		if _, err := EncodeGlyphRecord(data[entries[position].GlyphOffset:], glyph); err != nil {
			t.Fatal(err)
		}
	}
	index, err := DecodeIndex(string(data), header)
	if err != nil {
		t.Fatal(err)
	}
	return header, data, index
}

func changedIndexOffset(t *testing.T, index Index, position int, offset uint32) Index {
	t.Helper()
	data := []byte(index.data)
	binary.LittleEndian.PutUint32(data[position*IndexEntrySize+4:position*IndexEntrySize+8], offset)
	return Index{data: string(data), count: index.count}
}

func bitmapString(width, height uint8) string {
	return string(make([]byte, RawBitmapSize(width, height)))
}

func glyphTestName(glyph Glyph) string {
	return fmt.Sprintf("%dx%d", glyph.Width, glyph.Height)
}
