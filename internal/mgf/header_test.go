package mgf

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

func emptyHeader() Header {
	return Header{
		Version:         Version1,
		FontID:          [4]byte{'s', 'h', '1', '2'},
		SubsetID:        [4]byte{'f', 'u', 'l', 'l'},
		Region:          [2]byte{'J', 'P'},
		HeaderSize:      HeaderSize,
		IndexOffset:     HeaderSize,
		GlyphDataOffset: HeaderSize,
		FileSize:        HeaderSize,
	}
}

func encodedHeader(t *testing.T, header Header) []byte {
	t.Helper()
	data := make([]byte, header.FileSize)
	if err := EncodeHeader(data, header); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestHeaderEncodeDecodeRoundTrip(t *testing.T) {
	header := emptyHeader()
	setHeaderGlyphCount(&header, 2)
	header.Ascent = 10
	header.Descent = 2
	header.LineGap = 1
	header.MaxWidth = 12
	header.MaxHeight = 13
	data := encodedHeader(t, header)
	if len(data) != int(header.FileSize) || string(data[:3]) != Magic || data[3] != Version1 {
		t.Fatalf("prefix=%x len=%d", data[:4], len(data))
	}
	if string(data[4:8]) != "sh12" || string(data[8:12]) != "full" || string(data[12:14]) != "JP" {
		t.Fatalf("identity=%q/%q/%q", data[4:8], data[8:12], data[12:14])
	}
	if binary.LittleEndian.Uint16(data[14:16]) != 2 || binary.LittleEndian.Uint16(data[22:24]) != HeaderSize {
		t.Fatalf("uint16 fields=%x", data[14:24])
	}
	if binary.LittleEndian.Uint32(data[24:28]) != HeaderSize || binary.LittleEndian.Uint32(data[28:32]) != HeaderSize+2*IndexEntrySize || binary.LittleEndian.Uint32(data[32:36]) != HeaderSize+2*IndexEntrySize {
		t.Fatalf("offsets=%x", data[24:36])
	}
	decoded, err := DecodeHeader(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, header) {
		t.Fatalf("decoded=%+v want=%+v", decoded, header)
	}
}

func TestEncodeHeaderDestinationAndNoPartialWrite(t *testing.T) {
	header := emptyHeader()
	if err := EncodeHeader(make([]byte, HeaderSize-1), header); err == nil {
		t.Fatal("35-byte destination succeeded")
	}
	dst := make([]byte, HeaderSize+4)
	for index := range dst {
		dst[index] = 0xaa
	}
	if err := EncodeHeader(dst, header); err != nil {
		t.Fatal(err)
	}
	if string(dst[HeaderSize:]) != "\xaa\xaa\xaa\xaa" {
		t.Fatalf("suffix changed: %x", dst[HeaderSize:])
	}

	invalid := header
	invalid.Flags = 1
	unchanged := make([]byte, HeaderSize)
	for index := range unchanged {
		unchanged[index] = 0x5a
	}
	want := append([]byte(nil), unchanged...)
	if err := EncodeHeader(unchanged, invalid); err == nil || !reflect.DeepEqual(unchanged, want) {
		t.Fatalf("err=%v dst=%x", err, unchanged)
	}
}

func TestDecodeHeaderValidation(t *testing.T) {
	valid := encodedHeader(t, emptyHeader())
	tests := []struct {
		name   string
		mutate func([]byte)
		text   string
	}{
		{"magic", func(data []byte) { data[0] = 'X' }, "magic"},
		{"version", func(data []byte) { data[3] = 2 }, "version"},
		{"FontID", func(data []byte) { data[4] = 0 }, "FontID"},
		{"SubsetID", func(data []byte) { data[8] = 0x7f }, "SubsetID"},
		{"region one NUL", func(data []byte) { data[12] = 0 }, "Region"},
		{"region non-printable", func(data []byte) { data[13] = 0x1f }, "Region"},
		{"flags", func(data []byte) { data[21] = 1 }, "Flags"},
		{"HeaderSize", func(data []byte) { binary.LittleEndian.PutUint16(data[22:24], 35) }, "HeaderSize"},
		{"IndexOffset", func(data []byte) { binary.LittleEndian.PutUint32(data[24:28], 35) }, "IndexOffset"},
		{"GlyphDataOffset", func(data []byte) { binary.LittleEndian.PutUint32(data[28:32], 35) }, "GlyphDataOffset"},
		{"GlyphDataOffset too large", func(data []byte) { binary.LittleEndian.PutUint32(data[28:32], 37) }, "GlyphDataOffset"},
		{"FileSize mismatch", func(data []byte) { binary.LittleEndian.PutUint32(data[32:36], 37) }, "data length"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := append([]byte(nil), valid...)
			tt.mutate(data)
			if _, err := DecodeHeader(string(data)); err == nil || !strings.Contains(err.Error(), tt.text) {
				t.Fatalf("err=%v", err)
			}
		})
	}
	for _, size := range []int{0, 34, 35} {
		if _, err := DecodeHeader(string(make([]byte, size))); err == nil {
			t.Fatalf("size %d succeeded", size)
		}
	}
}

func TestHeaderRegions(t *testing.T) {
	for _, region := range [][2]byte{{'J', 'P'}, {'C', 'N'}, {0, 0}} {
		header := emptyHeader()
		header.Region = region
		data := encodedHeader(t, header)
		decoded, err := DecodeHeader(string(data))
		if err != nil || decoded.Region != region {
			t.Fatalf("region=%v decoded=%v err=%v", region, decoded.Region, err)
		}
	}
	for _, region := range [][2]byte{{0, 'P'}, {'J', 0}} {
		header := emptyHeader()
		header.Region = region
		if err := header.Validate(HeaderSize); err == nil {
			t.Fatalf("region %v succeeded", region)
		}
	}
}

func TestHeaderMaximumValuesAndZeroMetrics(t *testing.T) {
	header := emptyHeader()
	setHeaderGlyphCount(&header, 65535)
	header.Ascent = 255
	header.Descent = 255
	header.LineGap = 255
	header.MaxWidth = 255
	header.MaxHeight = 255
	data := encodedHeader(t, header)
	decoded, err := DecodeHeader(string(data))
	if err != nil || decoded != header {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}

	zero := emptyHeader()
	if err := zero.Validate(HeaderSize); err != nil {
		t.Fatalf("zero metrics: %v", err)
	}
}

func TestHeaderFixedIndexOffsets(t *testing.T) {
	tests := []struct {
		count uint16
		want  uint32
	}{
		{0, HeaderSize},
		{1, HeaderSize + IndexEntrySize},
		{3, HeaderSize + 3*IndexEntrySize},
		{65535, HeaderSize + 65535*IndexEntrySize},
	}
	for _, tt := range tests {
		header := emptyHeader()
		setHeaderGlyphCount(&header, tt.count)
		if header.GlyphDataOffset != tt.want {
			t.Fatalf("count=%d offset=%d want=%d", tt.count, header.GlyphDataOffset, tt.want)
		}
		if err := header.Validate(header.FileSize); err != nil {
			t.Fatalf("count=%d: %v", tt.count, err)
		}
	}

	header := emptyHeader()
	setHeaderGlyphCount(&header, 2)
	for _, offset := range []uint32{header.GlyphDataOffset - 1, header.GlyphDataOffset + 1} {
		invalid := header
		invalid.GlyphDataOffset = offset
		if err := invalid.Validate(invalid.FileSize); err == nil || !strings.Contains(err.Error(), "GlyphDataOffset") {
			t.Fatalf("offset=%d err=%v", offset, err)
		}
	}
	invalidIndex := header
	invalidIndex.IndexOffset = HeaderSize + 1
	if err := invalidIndex.Validate(invalidIndex.FileSize); err == nil || !strings.Contains(err.Error(), "IndexOffset") {
		t.Fatalf("IndexOffset err=%v", err)
	}
	invalidFile := header
	invalidFile.FileSize = invalidFile.GlyphDataOffset - 1
	if err := invalidFile.Validate(invalidFile.FileSize); err == nil || !strings.Contains(err.Error(), "FileSize") {
		t.Fatalf("FileSize err=%v", err)
	}

	expected := uint64(HeaderSize) + uint64(uint16(65535))*uint64(IndexEntrySize)
	if expected > uint64(^uint32(0)) || expected != uint64(HeaderSize+65535*IndexEntrySize) {
		t.Fatalf("overflow-safe offset=%d", expected)
	}
}

func TestHeaderSuccessfulPathsDoNotAllocate(t *testing.T) {
	header := emptyHeader()
	data := encodedHeader(t, header)
	encoded := string(data)
	dst := make([]byte, HeaderSize)
	if allocations := testing.AllocsPerRun(100, func() {
		if err := header.Validate(header.FileSize); err != nil {
			panic(err)
		}
	}); allocations != 0 {
		t.Fatalf("Header.Validate allocations=%v", allocations)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if err := EncodeHeader(dst, header); err != nil {
			panic(err)
		}
	}); allocations != 0 {
		t.Fatalf("EncodeHeader allocations=%v", allocations)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if _, err := DecodeHeader(encoded); err != nil {
			panic(err)
		}
	}); allocations != 0 {
		t.Fatalf("DecodeHeader allocations=%v", allocations)
	}
}

func setHeaderGlyphCount(header *Header, count uint16) {
	header.GlyphCount = count
	header.GlyphDataOffset = uint32(HeaderSize) + uint32(count)*IndexEntrySize
	header.FileSize = header.GlyphDataOffset
}
