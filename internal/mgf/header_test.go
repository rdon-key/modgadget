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
	data := make([]byte, HeaderSize)
	if err := EncodeHeader(data, header); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestHeaderEncodeDecodeRoundTrip(t *testing.T) {
	header := emptyHeader()
	header.GlyphCount = 0x1234
	header.Ascent = 10
	header.Descent = 2
	header.LineGap = 1
	header.MaxWidth = 12
	header.MaxHeight = 13
	data := encodedHeader(t, header)
	if len(data) != HeaderSize || string(data[:3]) != Magic || data[3] != Version1 {
		t.Fatalf("prefix=%x len=%d", data[:4], len(data))
	}
	if string(data[4:8]) != "sh12" || string(data[8:12]) != "full" || string(data[12:14]) != "JP" {
		t.Fatalf("identity=%q/%q/%q", data[4:8], data[8:12], data[12:14])
	}
	if binary.LittleEndian.Uint16(data[14:16]) != 0x1234 || binary.LittleEndian.Uint16(data[22:24]) != HeaderSize {
		t.Fatalf("uint16 fields=%x", data[14:24])
	}
	if binary.LittleEndian.Uint32(data[24:28]) != HeaderSize || binary.LittleEndian.Uint32(data[28:32]) != HeaderSize || binary.LittleEndian.Uint32(data[32:36]) != HeaderSize {
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
		{"FileSize before glyph data", func(data []byte) { binary.LittleEndian.PutUint32(data[28:32], 37) }, "FileSize"},
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
	header.GlyphCount = 65535
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

func TestHeaderSuccessfulPathsDoNotAllocate(t *testing.T) {
	header := emptyHeader()
	data := encodedHeader(t, header)
	encoded := string(data)
	dst := make([]byte, HeaderSize)
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
