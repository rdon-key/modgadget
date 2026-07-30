package mgf

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

func indexHeader(count uint16, glyphBytes uint32) Header {
	header := emptyHeader()
	header.GlyphCount = count
	header.GlyphDataOffset = uint32(HeaderSize) + uint32(count)*IndexEntrySize
	header.FileSize = header.GlyphDataOffset + glyphBytes
	return header
}

func encodeIndexFile(t *testing.T, header Header, entries []IndexEntry) []byte {
	t.Helper()
	data := make([]byte, header.FileSize)
	if err := EncodeHeader(data[:HeaderSize], header); err != nil {
		t.Fatal(err)
	}
	if err := EncodeIndex(data[header.IndexOffset:header.GlyphDataOffset], header, entries); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEncodeIndexOneAndMultipleEntries(t *testing.T) {
	oneHeader := indexHeader(1, 4)
	one := []IndexEntry{{Codepoint: 0x3042, GlyphOffset: oneHeader.GlyphDataOffset}}
	oneData := make([]byte, IndexEntrySize)
	if err := EncodeIndex(oneData, oneHeader, one); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(oneData[0:4]) != 0x3042 || binary.LittleEndian.Uint32(oneData[4:8]) != oneHeader.GlyphDataOffset {
		t.Fatalf("encoded=%x", oneData)
	}

	header := indexHeader(3, 8)
	entries := []IndexEntry{
		{Codepoint: 0, GlyphOffset: header.GlyphDataOffset},
		{Codepoint: 'A', GlyphOffset: header.GlyphDataOffset + 2},
		{Codepoint: 0x10ffff, GlyphOffset: header.FileSize - 1},
	}
	dst := make([]byte, len(entries)*IndexEntrySize+4)
	for index := range dst {
		dst[index] = 0xaa
	}
	if err := EncodeIndex(dst, header, entries); err != nil {
		t.Fatal(err)
	}
	if string(dst[len(entries)*IndexEntrySize:]) != "\xaa\xaa\xaa\xaa" {
		t.Fatalf("suffix=%x", dst[len(entries)*IndexEntrySize:])
	}
	for position, want := range entries {
		offset := position * IndexEntrySize
		got := IndexEntry{binary.LittleEndian.Uint32(dst[offset : offset+4]), binary.LittleEndian.Uint32(dst[offset+4 : offset+8])}
		if got != want {
			t.Fatalf("entry %d=%+v want=%+v", position, got, want)
		}
	}
}

func TestEncodeIndexDestinationAndEmpty(t *testing.T) {
	header := indexHeader(1, 1)
	entry := []IndexEntry{{Codepoint: 'A', GlyphOffset: header.GlyphDataOffset}}
	if err := EncodeIndex(make([]byte, IndexEntrySize-1), header, entry); err == nil {
		t.Fatal("short destination succeeded")
	}
	empty := emptyHeader()
	if err := EncodeIndex(nil, empty, nil); err != nil {
		t.Fatalf("empty index: %v", err)
	}

	dst := make([]byte, IndexEntrySize)
	for index := range dst {
		dst[index] = 0x5a
	}
	want := append([]byte(nil), dst...)
	invalid := []IndexEntry{{Codepoint: 0xd800, GlyphOffset: header.GlyphDataOffset}}
	if err := EncodeIndex(dst, header, invalid); err == nil || !reflect.DeepEqual(dst, want) {
		t.Fatalf("err=%v dst=%x", err, dst)
	}
	if err := EncodeIndex(dst, header, nil); err == nil || !reflect.DeepEqual(dst, want) {
		t.Fatalf("count err=%v dst=%x", err, dst)
	}
}

func TestEncodeIndexCodepointValidation(t *testing.T) {
	header := indexHeader(1, 1)
	valid := []uint32{0, 'A', 0x3042, 0x10ffff}
	for _, codepoint := range valid {
		entry := []IndexEntry{{Codepoint: codepoint, GlyphOffset: header.GlyphDataOffset}}
		if err := EncodeIndex(make([]byte, IndexEntrySize), header, entry); err != nil {
			t.Fatalf("U+%04X: %v", codepoint, err)
		}
	}
	for _, codepoint := range []uint32{0xd800, 0xdfff, 0x110000} {
		entry := []IndexEntry{{Codepoint: codepoint, GlyphOffset: header.GlyphDataOffset}}
		if err := EncodeIndex(make([]byte, IndexEntrySize), header, entry); err == nil || !strings.Contains(err.Error(), "codepoint") {
			t.Fatalf("U+%04X err=%v", codepoint, err)
		}
	}

	twoHeader := indexHeader(2, 1)
	for name, entries := range map[string][]IndexEntry{
		"duplicate":  {{Codepoint: 0x3042, GlyphOffset: twoHeader.GlyphDataOffset}, {Codepoint: 0x3042, GlyphOffset: twoHeader.GlyphDataOffset}},
		"descending": {{Codepoint: 0x3043, GlyphOffset: twoHeader.GlyphDataOffset}, {Codepoint: 0x3042, GlyphOffset: twoHeader.GlyphDataOffset}},
	} {
		if err := EncodeIndex(make([]byte, 2*IndexEntrySize), twoHeader, entries); err == nil || !strings.Contains(err.Error(), "not greater") {
			t.Fatalf("%s err=%v", name, err)
		}
	}
}

func TestEncodeIndexGlyphOffsetValidation(t *testing.T) {
	header := indexHeader(1, 4)
	tests := []struct {
		name   string
		offset uint32
		ok     bool
	}{
		{"GlyphDataOffset", header.GlyphDataOffset, true},
		{"FileSize minus one", header.FileSize - 1, true},
		{"before GlyphDataOffset", header.GlyphDataOffset - 1, false},
		{"FileSize", header.FileSize, false},
		{"after FileSize", header.FileSize + 1, false},
	}
	for _, tt := range tests {
		err := EncodeIndex(make([]byte, IndexEntrySize), header, []IndexEntry{{Codepoint: 'A', GlyphOffset: tt.offset}})
		if (err == nil) != tt.ok {
			t.Fatalf("%s offset=%d err=%v", tt.name, tt.offset, err)
		}
	}
}

func TestEncodeIndexGlyphOffsetMustIncrease(t *testing.T) {
	header := indexHeader(2, 4)
	tests := []struct {
		name    string
		offsets [2]uint32
	}{
		{"duplicate", [2]uint32{header.GlyphDataOffset, header.GlyphDataOffset}},
		{"descending", [2]uint32{header.GlyphDataOffset + 1, header.GlyphDataOffset}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, 2*IndexEntrySize)
			for position := range dst {
				dst[position] = 0x5a
			}
			want := append([]byte(nil), dst...)
			entries := []IndexEntry{
				{Codepoint: 'A', GlyphOffset: tt.offsets[0]},
				{Codepoint: 'B', GlyphOffset: tt.offsets[1]},
			}
			if err := EncodeIndex(dst, header, entries); err == nil || !strings.Contains(err.Error(), "GlyphOffset") || !reflect.DeepEqual(dst, want) {
				t.Fatalf("err=%v dst=%x", err, dst)
			}
		})
	}
}

func TestDecodeIndexAndMethods(t *testing.T) {
	header := indexHeader(3, 8)
	entries := []IndexEntry{
		{Codepoint: 'A', GlyphOffset: header.GlyphDataOffset},
		{Codepoint: 0x3042, GlyphOffset: header.GlyphDataOffset + 2},
		{Codepoint: 0x10ffff, GlyphOffset: header.FileSize - 1},
	}
	data := encodeIndexFile(t, header, entries)
	index, err := DecodeIndex(string(data), header)
	if err != nil {
		t.Fatal(err)
	}
	if index.Len() != len(entries) {
		t.Fatalf("Len=%d", index.Len())
	}
	for position, want := range entries {
		got, ok := index.Entry(position)
		if !ok || got != want {
			t.Fatalf("Entry(%d)=%+v/%v want=%+v", position, got, ok, want)
		}
		offset, ok := index.Lookup(rune(want.Codepoint))
		if !ok || offset != want.GlyphOffset {
			t.Fatalf("Lookup(U+%04X)=%d/%v", want.Codepoint, offset, ok)
		}
	}
	for _, position := range []int{-1, len(entries)} {
		if _, ok := index.Entry(position); ok {
			t.Fatalf("Entry(%d) succeeded", position)
		}
	}
	for _, r := range []rune{'B', -1, rune(0xd800)} {
		if _, ok := index.Lookup(r); ok {
			t.Fatalf("Lookup(%d) succeeded", r)
		}
	}
}

func TestDecodeIndexOneEntry(t *testing.T) {
	header := indexHeader(1, 1)
	want := IndexEntry{Codepoint: 0x3042, GlyphOffset: header.GlyphDataOffset}
	data := encodeIndexFile(t, header, []IndexEntry{want})
	index, err := DecodeIndex(string(data), header)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := index.Entry(0)
	if index.Len() != 1 || !ok || got != want {
		t.Fatalf("Len=%d Entry=%+v/%v", index.Len(), got, ok)
	}
}

func TestDecodeIndexValidation(t *testing.T) {
	header := indexHeader(2, 4)
	entries := []IndexEntry{
		{Codepoint: 'A', GlyphOffset: header.GlyphDataOffset},
		{Codepoint: 'B', GlyphOffset: header.GlyphDataOffset + 1},
	}
	valid := encodeIndexFile(t, header, entries)
	tests := []struct {
		name   string
		mutate func([]byte)
		text   string
	}{
		{"invalid codepoint", func(data []byte) { binary.LittleEndian.PutUint32(data[HeaderSize:HeaderSize+4], 0xd800) }, "codepoint"},
		{"duplicate", func(data []byte) {
			binary.LittleEndian.PutUint32(data[HeaderSize+IndexEntrySize:HeaderSize+IndexEntrySize+4], 'A')
		}, "not greater"},
		{"descending", func(data []byte) { binary.LittleEndian.PutUint32(data[HeaderSize:HeaderSize+4], 'C') }, "not greater"},
		{"offset before", func(data []byte) {
			binary.LittleEndian.PutUint32(data[HeaderSize+4:HeaderSize+8], header.GlyphDataOffset-1)
		}, "GlyphOffset"},
		{"offset at FileSize", func(data []byte) { binary.LittleEndian.PutUint32(data[HeaderSize+4:HeaderSize+8], header.FileSize) }, "GlyphOffset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := append([]byte(nil), valid...)
			tt.mutate(data)
			if _, err := DecodeIndex(string(data), header); err == nil || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("err=%v", err)
			}
		})
	}
	if _, err := DecodeIndex(string(valid[:header.GlyphDataOffset-1]), header); err == nil {
		t.Fatal("short index region succeeded")
	}
}

func TestDecodeIndexGlyphOffsetMustIncrease(t *testing.T) {
	header := indexHeader(2, 4)
	entries := []IndexEntry{
		{Codepoint: 'A', GlyphOffset: header.GlyphDataOffset},
		{Codepoint: 'B', GlyphOffset: header.GlyphDataOffset + 1},
	}
	valid := encodeIndexFile(t, header, entries)
	for _, tt := range []struct {
		name    string
		offsets [2]uint32
	}{
		{"duplicate", [2]uint32{header.GlyphDataOffset, header.GlyphDataOffset}},
		{"descending", [2]uint32{header.GlyphDataOffset + 1, header.GlyphDataOffset}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := append([]byte(nil), valid...)
			binary.LittleEndian.PutUint32(data[HeaderSize+4:HeaderSize+IndexEntrySize], tt.offsets[0])
			binary.LittleEndian.PutUint32(data[HeaderSize+IndexEntrySize+4:HeaderSize+2*IndexEntrySize], tt.offsets[1])
			if _, err := DecodeIndex(string(data), header); err == nil || !strings.Contains(err.Error(), "GlyphOffset") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestZeroValueIndex(t *testing.T) {
	var index Index
	if index.Len() != 0 {
		t.Fatalf("Len=%d", index.Len())
	}
	if _, ok := index.Entry(0); ok {
		t.Fatal("Entry succeeded")
	}
	if _, ok := index.Lookup('A'); ok {
		t.Fatal("Lookup succeeded")
	}
}

func TestIndexSuccessfulPathsDoNotAllocate(t *testing.T) {
	header := indexHeader(3, 4)
	entries := []IndexEntry{
		{Codepoint: 'A', GlyphOffset: header.GlyphDataOffset},
		{Codepoint: 'B', GlyphOffset: header.GlyphDataOffset + 1},
		{Codepoint: 'C', GlyphOffset: header.GlyphDataOffset + 2},
	}
	dst := make([]byte, len(entries)*IndexEntrySize)
	if allocations := testing.AllocsPerRun(100, func() {
		if err := EncodeIndex(dst, header, entries); err != nil {
			panic(err)
		}
	}); allocations != 0 {
		t.Fatalf("EncodeIndex allocations=%v", allocations)
	}
	data := string(encodeIndexFile(t, header, entries))
	if allocations := testing.AllocsPerRun(100, func() {
		if _, err := DecodeIndex(data, header); err != nil {
			panic(err)
		}
	}); allocations != 0 {
		t.Fatalf("DecodeIndex allocations=%v", allocations)
	}
	index, err := DecodeIndex(data, header)
	if err != nil {
		t.Fatal(err)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if _, ok := index.Entry(1); !ok {
			panic("entry missing")
		}
	}); allocations != 0 {
		t.Fatalf("Entry allocations=%v", allocations)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if _, ok := index.Lookup('B'); !ok {
			panic("lookup missing")
		}
	}); allocations != 0 {
		t.Fatalf("Lookup allocations=%v", allocations)
	}
}
