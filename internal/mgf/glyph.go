package mgf

import (
	"encoding/binary"
	"fmt"
	"math"
)

// GlyphRecordHeaderSize is the fixed byte size before raw bitmap data.
const GlyphRecordHeaderSize = 10

// Glyph contains decoded metrics and an immutable raw 1-bit bitmap view.
type Glyph struct {
	Width    uint8
	Height   uint8
	AdvanceX int16
	BearingX int16
	BearingY int16
	Bitmap   string
}

// RawBitmapSize returns the byte length of an uncompressed row-aligned bitmap.
func RawBitmapSize(width, height uint8) uint16 {
	rowBytes := (uint16(width) + 7) / 8
	return rowBytes * uint16(height)
}

// EncodeGlyphRecord validates and writes one uncompressed MGF1 Glyph Record.
func EncodeGlyphRecord(dst []byte, glyph Glyph) (int, error) {
	if len(glyph.Bitmap) > math.MaxUint16 {
		return 0, fmt.Errorf("mgf: glyph Bitmap is %d bytes, exceeds uint16", len(glyph.Bitmap))
	}
	wantLength := int(RawBitmapSize(glyph.Width, glyph.Height))
	if len(glyph.Bitmap) != wantLength {
		return 0, fmt.Errorf("mgf: glyph DataLength is %d, want %d", len(glyph.Bitmap), wantLength)
	}
	required := GlyphRecordHeaderSize + len(glyph.Bitmap)
	if len(dst) < required {
		return 0, fmt.Errorf("mgf: glyph destination is %d bytes, want at least %d", len(dst), required)
	}

	dst[0] = glyph.Width
	dst[1] = glyph.Height
	binary.LittleEndian.PutUint16(dst[2:4], uint16(glyph.AdvanceX))
	binary.LittleEndian.PutUint16(dst[4:6], uint16(glyph.BearingX))
	binary.LittleEndian.PutUint16(dst[6:8], uint16(glyph.BearingY))
	binary.LittleEndian.PutUint16(dst[8:10], uint16(len(glyph.Bitmap)))
	copy(dst[GlyphRecordHeaderSize:required], glyph.Bitmap)
	return required, nil
}

// DecodeGlyphRecord validates and returns one Glyph Record view from data.
func DecodeGlyphRecord(data string, header Header, glyphOffset uint32) (Glyph, uint32, error) {
	var glyph Glyph
	if uint64(len(data)) > math.MaxUint32 {
		return glyph, 0, fmt.Errorf("mgf: data length exceeds uint32")
	}
	if err := header.Validate(uint32(len(data))); err != nil {
		return glyph, 0, err
	}
	if glyphOffset < header.GlyphDataOffset {
		return glyph, 0, fmt.Errorf("mgf: glyph offset %d is before GlyphDataOffset %d", glyphOffset, header.GlyphDataOffset)
	}
	if glyphOffset >= header.FileSize {
		return glyph, 0, fmt.Errorf("mgf: glyph offset %d is not before FileSize %d", glyphOffset, header.FileSize)
	}
	headerEnd := uint64(glyphOffset) + GlyphRecordHeaderSize
	if headerEnd > uint64(header.FileSize) {
		return glyph, 0, fmt.Errorf("mgf: glyph at offset %d is truncated", glyphOffset)
	}
	offset := int(glyphOffset)
	glyph.Width = uint8(data[offset])
	glyph.Height = uint8(data[offset+1])
	glyph.AdvanceX = int16(uint16(data[offset+2]) | uint16(data[offset+3])<<8)
	glyph.BearingX = int16(uint16(data[offset+4]) | uint16(data[offset+5])<<8)
	glyph.BearingY = int16(uint16(data[offset+6]) | uint16(data[offset+7])<<8)
	dataLength := uint16(data[offset+8]) | uint16(data[offset+9])<<8
	recordEnd := headerEnd + uint64(dataLength)
	if recordEnd > uint64(header.FileSize) {
		return Glyph{}, 0, fmt.Errorf("mgf: glyph at offset %d bitmap data is truncated", glyphOffset)
	}
	if glyph.Width > header.MaxWidth {
		return Glyph{}, 0, fmt.Errorf("mgf: glyph at offset %d Width %d exceeds MaxWidth %d", glyphOffset, glyph.Width, header.MaxWidth)
	}
	if glyph.Height > header.MaxHeight {
		return Glyph{}, 0, fmt.Errorf("mgf: glyph at offset %d Height %d exceeds MaxHeight %d", glyphOffset, glyph.Height, header.MaxHeight)
	}
	wantLength := RawBitmapSize(glyph.Width, glyph.Height)
	if dataLength != wantLength {
		return Glyph{}, 0, fmt.Errorf("mgf: glyph at offset %d DataLength is %d, want %d", glyphOffset, dataLength, wantLength)
	}
	glyph.Bitmap = data[int(headerEnd):int(recordEnd)]
	return glyph, uint32(recordEnd), nil
}

// ValidateGlyphData validates canonical, contiguous MGF1 Glyph Record layout.
func ValidateGlyphData(data string, header Header, index Index) error {
	if uint64(len(data)) > math.MaxUint32 {
		return fmt.Errorf("mgf: data length exceeds uint32")
	}
	if err := header.Validate(uint32(len(data))); err != nil {
		return err
	}
	indexStart64 := uint64(header.IndexOffset)
	indexEnd64 := uint64(header.GlyphDataOffset)
	if indexEnd64 < indexStart64 || indexEnd64 > uint64(len(data)) {
		return fmt.Errorf("mgf: index region [%d,%d) is outside data", indexStart64, indexEnd64)
	}
	indexStart := int(indexStart64)
	indexEnd := int(indexEnd64)
	if index.count != int(header.GlyphCount) {
		return fmt.Errorf("mgf: index has %d entries, GlyphCount is %d", index.count, header.GlyphCount)
	}
	if len(index.data) != indexEnd-indexStart {
		return fmt.Errorf("mgf: index data is %d bytes, want %d", len(index.data), indexEnd-indexStart)
	}
	if index.data != data[indexStart:indexEnd] {
		return fmt.Errorf("mgf: index does not match file data")
	}
	if header.GlyphCount == 0 {
		if header.GlyphDataOffset != header.FileSize {
			return fmt.Errorf("mgf: empty glyph data offset %d does not equal FileSize %d", header.GlyphDataOffset, header.FileSize)
		}
		if header.MaxWidth != 0 || header.MaxHeight != 0 {
			return fmt.Errorf("mgf: empty glyph data requires MaxWidth and MaxHeight 0")
		}
		return nil
	}
	if err := validateIndex(index, header); err != nil {
		return err
	}

	expectedOffset := header.GlyphDataOffset
	var actualMaxWidth uint8
	var actualMaxHeight uint8
	for position := 0; position < index.count; position++ {
		entry, ok := index.Entry(position)
		if !ok {
			return fmt.Errorf("mgf: glyph %d index entry is unavailable", position)
		}
		if entry.GlyphOffset != expectedOffset {
			return fmt.Errorf("mgf: glyph %d offset is %d, want previous record end %d", position, entry.GlyphOffset, expectedOffset)
		}
		glyph, nextOffset, err := DecodeGlyphRecord(data, header, entry.GlyphOffset)
		if err != nil {
			return fmt.Errorf("mgf: glyph %d: %w", position, err)
		}
		if glyph.Width > actualMaxWidth {
			actualMaxWidth = glyph.Width
		}
		if glyph.Height > actualMaxHeight {
			actualMaxHeight = glyph.Height
		}
		expectedOffset = nextOffset
	}
	if expectedOffset != header.FileSize {
		return fmt.Errorf("mgf: trailing glyph data starts at %d, FileSize is %d", expectedOffset, header.FileSize)
	}
	if header.MaxWidth != actualMaxWidth {
		return fmt.Errorf("mgf: MaxWidth is %d, actual maximum is %d", header.MaxWidth, actualMaxWidth)
	}
	if header.MaxHeight != actualMaxHeight {
		return fmt.Errorf("mgf: MaxHeight is %d, actual maximum is %d", header.MaxHeight, actualMaxHeight)
	}
	return nil
}
