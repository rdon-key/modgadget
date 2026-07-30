// Package mgf implements the stable portions of the ModGadget Font format.
package mgf

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	Magic      = "MGF"
	Version1   = uint8(1)
	HeaderSize = 36
)

// Header is the fixed-size MGF1 file header.
type Header struct {
	Version         uint8
	FontID          [4]byte
	SubsetID        [4]byte
	Region          [2]byte
	GlyphCount      uint16
	Ascent          uint8
	Descent         uint8
	LineGap         uint8
	MaxWidth        uint8
	MaxHeight       uint8
	Flags           uint8
	HeaderSize      uint16
	IndexOffset     uint32
	GlyphDataOffset uint32
	FileSize        uint32
}

// EncodeHeader writes header to the first 36 bytes of dst in MGF1 format.
func EncodeHeader(dst []byte, header Header) error {
	if len(dst) < HeaderSize {
		return fmt.Errorf("mgf: header destination is %d bytes, want at least %d", len(dst), HeaderSize)
	}
	if err := header.Validate(header.FileSize); err != nil {
		return err
	}

	var encoded [HeaderSize]byte
	copy(encoded[0:3], Magic)
	encoded[3] = header.Version
	copy(encoded[4:8], header.FontID[:])
	copy(encoded[8:12], header.SubsetID[:])
	copy(encoded[12:14], header.Region[:])
	binary.LittleEndian.PutUint16(encoded[14:16], header.GlyphCount)
	encoded[16] = header.Ascent
	encoded[17] = header.Descent
	encoded[18] = header.LineGap
	encoded[19] = header.MaxWidth
	encoded[20] = header.MaxHeight
	encoded[21] = header.Flags
	binary.LittleEndian.PutUint16(encoded[22:24], header.HeaderSize)
	binary.LittleEndian.PutUint32(encoded[24:28], header.IndexOffset)
	binary.LittleEndian.PutUint32(encoded[28:32], header.GlyphDataOffset)
	binary.LittleEndian.PutUint32(encoded[32:36], header.FileSize)
	copy(dst[:HeaderSize], encoded[:])
	return nil
}

// DecodeHeader validates and decodes the MGF1 header at the start of data.
func DecodeHeader(data string) (Header, error) {
	var header Header
	if len(data) < HeaderSize {
		return header, fmt.Errorf("mgf: data is %d bytes, want at least %d", len(data), HeaderSize)
	}
	if data[0] != Magic[0] || data[1] != Magic[1] || data[2] != Magic[2] {
		return header, fmt.Errorf("mgf: invalid magic")
	}
	if data[3] != Version1 {
		return header, fmt.Errorf("mgf: unsupported version %d", data[3])
	}
	if uint64(len(data)) > math.MaxUint32 {
		return header, fmt.Errorf("mgf: data length exceeds uint32")
	}

	header.Version = uint8(data[3])
	copy(header.FontID[:], data[4:8])
	copy(header.SubsetID[:], data[8:12])
	copy(header.Region[:], data[12:14])
	header.GlyphCount = uint16(data[14]) | uint16(data[15])<<8
	header.Ascent = uint8(data[16])
	header.Descent = uint8(data[17])
	header.LineGap = uint8(data[18])
	header.MaxWidth = uint8(data[19])
	header.MaxHeight = uint8(data[20])
	header.Flags = uint8(data[21])
	header.HeaderSize = uint16(data[22]) | uint16(data[23])<<8
	header.IndexOffset = decodeUint32(data, 24)
	header.GlyphDataOffset = decodeUint32(data, 28)
	header.FileSize = decodeUint32(data, 32)
	if err := header.Validate(uint32(len(data))); err != nil {
		return Header{}, err
	}
	return header, nil
}

// Validate checks the MGF1 header against the complete file length.
func (header Header) Validate(dataLength uint32) error {
	if header.Version != Version1 {
		return fmt.Errorf("mgf: unsupported version %d", header.Version)
	}
	if !printableASCII(header.FontID[:]) {
		return fmt.Errorf("mgf: FontID contains non-printable ASCII")
	}
	if !printableASCII(header.SubsetID[:]) {
		return fmt.Errorf("mgf: SubsetID contains non-printable ASCII")
	}
	if header.Region != ([2]byte{}) && !printableASCII(header.Region[:]) {
		return fmt.Errorf("mgf: Region must be zero or contain printable ASCII")
	}
	if header.Flags != 0 {
		return fmt.Errorf("mgf: Flags is %d, want 0", header.Flags)
	}
	if header.HeaderSize != HeaderSize {
		return fmt.Errorf("mgf: HeaderSize is %d, want %d", header.HeaderSize, HeaderSize)
	}
	if header.IndexOffset != HeaderSize {
		return fmt.Errorf("mgf: IndexOffset is %d, want %d", header.IndexOffset, HeaderSize)
	}
	expectedGlyphDataOffset := uint64(header.IndexOffset) + uint64(header.GlyphCount)*IndexEntrySize
	if expectedGlyphDataOffset > math.MaxUint32 {
		return fmt.Errorf("mgf: GlyphDataOffset calculation overflows uint32")
	}
	if header.GlyphDataOffset != uint32(expectedGlyphDataOffset) {
		return fmt.Errorf("mgf: GlyphDataOffset is %d, want %d", header.GlyphDataOffset, expectedGlyphDataOffset)
	}
	if header.FileSize < header.GlyphDataOffset {
		return fmt.Errorf("mgf: FileSize %d is before GlyphDataOffset %d", header.FileSize, header.GlyphDataOffset)
	}
	if header.FileSize != dataLength {
		return fmt.Errorf("mgf: FileSize is %d, data length is %d", header.FileSize, dataLength)
	}
	return nil
}

func printableASCII(value []byte) bool {
	for _, b := range value {
		if b < 0x20 || b > 0x7e {
			return false
		}
	}
	return true
}

func decodeUint32(data string, offset int) uint32 {
	return uint32(data[offset]) |
		uint32(data[offset+1])<<8 |
		uint32(data[offset+2])<<16 |
		uint32(data[offset+3])<<24
}
