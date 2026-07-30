package mgf

import (
	"encoding/binary"
	"fmt"
)

// IndexEntrySize is the fixed byte size of one MGF1 Glyph Index entry.
const IndexEntrySize = 8

// IndexEntry maps a Unicode scalar value to an absolute Glyph Record offset.
type IndexEntry struct {
	Codepoint   uint32
	GlyphOffset uint32
}

// Index is an immutable view over an encoded MGF1 Glyph Index.
type Index struct {
	data  string
	count int
}

// EncodeIndex validates and encodes entries into the required prefix of dst.
func EncodeIndex(dst []byte, header Header, entries []IndexEntry) error {
	if err := header.Validate(header.FileSize); err != nil {
		return err
	}
	if len(entries) != int(header.GlyphCount) {
		return fmt.Errorf("mgf: index has %d entries, GlyphCount is %d", len(entries), header.GlyphCount)
	}
	required := len(entries) * IndexEntrySize
	if len(dst) < required {
		return fmt.Errorf("mgf: index destination is %d bytes, want at least %d", len(dst), required)
	}
	if err := validateEntries(entries, header); err != nil {
		return err
	}
	for position, entry := range entries {
		offset := position * IndexEntrySize
		binary.LittleEndian.PutUint32(dst[offset:offset+4], entry.Codepoint)
		binary.LittleEndian.PutUint32(dst[offset+4:offset+8], entry.GlyphOffset)
	}
	return nil
}

// DecodeIndex validates an MGF1 Glyph Index and returns a view over data.
func DecodeIndex(data string, header Header) (Index, error) {
	if uint64(len(data)) > uint64(^uint32(0)) {
		return Index{}, fmt.Errorf("mgf: data length exceeds uint32")
	}
	if err := header.Validate(uint32(len(data))); err != nil {
		return Index{}, err
	}
	start := int(header.IndexOffset)
	end := int(header.GlyphDataOffset)
	if start < 0 || end < start || end > len(data) {
		return Index{}, fmt.Errorf("mgf: index region [%d,%d) is outside data", start, end)
	}
	region := data[start:end]
	if len(region) != int(header.GlyphCount)*IndexEntrySize {
		return Index{}, fmt.Errorf("mgf: index region is %d bytes, want %d", len(region), int(header.GlyphCount)*IndexEntrySize)
	}
	index := Index{data: region, count: int(header.GlyphCount)}
	if err := validateIndex(index, header); err != nil {
		return Index{}, err
	}
	return index, nil
}

// Len returns the number of entries in the Index.
func (index Index) Len() int {
	return index.count
}

// Entry decodes the entry at position.
func (index Index) Entry(position int) (IndexEntry, bool) {
	if position < 0 || position >= index.count {
		return IndexEntry{}, false
	}
	offset := position * IndexEntrySize
	return IndexEntry{
		Codepoint:   decodeUint32(index.data, offset),
		GlyphOffset: decodeUint32(index.data, offset+4),
	}, true
}

// Lookup finds r using binary search over the sorted Codepoint entries.
func (index Index) Lookup(r rune) (uint32, bool) {
	if r < 0 || !validCodepoint(uint32(r)) {
		return 0, false
	}
	target := uint32(r)
	left, right := 0, index.count
	for left < right {
		middle := left + (right-left)/2
		entry, _ := index.Entry(middle)
		if entry.Codepoint < target {
			left = middle + 1
		} else {
			right = middle
		}
	}
	entry, ok := index.Entry(left)
	if !ok || entry.Codepoint != target {
		return 0, false
	}
	return entry.GlyphOffset, true
}

func validateEntries(entries []IndexEntry, header Header) error {
	var previous uint32
	for position, entry := range entries {
		if !validCodepoint(entry.Codepoint) {
			return fmt.Errorf("mgf: index entry %d has invalid codepoint U+%04X", position, entry.Codepoint)
		}
		if position != 0 && entry.Codepoint <= previous {
			return fmt.Errorf("mgf: index entry %d codepoint U+%04X is not greater than previous U+%04X", position, entry.Codepoint, previous)
		}
		if entry.GlyphOffset < header.GlyphDataOffset {
			return fmt.Errorf("mgf: index entry %d GlyphOffset %d is before GlyphDataOffset %d", position, entry.GlyphOffset, header.GlyphDataOffset)
		}
		if entry.GlyphOffset >= header.FileSize {
			return fmt.Errorf("mgf: index entry %d GlyphOffset %d is not before FileSize %d", position, entry.GlyphOffset, header.FileSize)
		}
		previous = entry.Codepoint
	}
	return nil
}

func validateIndex(index Index, header Header) error {
	var previous uint32
	for position := 0; position < index.count; position++ {
		entry, _ := index.Entry(position)
		if !validCodepoint(entry.Codepoint) {
			return fmt.Errorf("mgf: index entry %d has invalid codepoint U+%04X", position, entry.Codepoint)
		}
		if position != 0 && entry.Codepoint <= previous {
			return fmt.Errorf("mgf: index entry %d codepoint U+%04X is not greater than previous U+%04X", position, entry.Codepoint, previous)
		}
		if entry.GlyphOffset < header.GlyphDataOffset {
			return fmt.Errorf("mgf: index entry %d GlyphOffset %d is before GlyphDataOffset %d", position, entry.GlyphOffset, header.GlyphDataOffset)
		}
		if entry.GlyphOffset >= header.FileSize {
			return fmt.Errorf("mgf: index entry %d GlyphOffset %d is not before FileSize %d", position, entry.GlyphOffset, header.FileSize)
		}
		previous = entry.Codepoint
	}
	return nil
}

func validCodepoint(codepoint uint32) bool {
	return codepoint <= 0x10ffff && (codepoint < 0xd800 || codepoint > 0xdfff)
}
