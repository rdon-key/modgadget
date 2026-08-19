// Package mgz reads the experimental block-compressed MGZ1 font format.
package mgz

import (
	"compress/flate"
	"fmt"
	"io"
	"math"
	"strings"
)

const (
	HeaderSize     = 64
	GlyphEntrySize = 16
	BlockEntrySize = 16
	Version1       = 1
	CodecDeflate   = 1
	blockDeflated  = 1
)

type Header struct {
	FontID, SubsetID         [4]byte
	Region                   [2]byte
	GlyphCount, BlockCount   uint32
	GlyphsPerBlock           uint16
	Ascent, Descent, LineGap uint8
	MaxWidth, MaxHeight      uint8
	OriginalMGFSize          uint32
}

type Glyph struct {
	Width, Height                uint8
	AdvanceX, BearingX, BearingY int16
	Bitmap                       string
}

type block struct {
	offset, storedLen, rawLen uint32
	flags                     uint32
}

// Font retains the source data and one expanded block. A Font is not safe for
// concurrent Lookup calls; callers should share it through the public handle.
type Font struct {
	data                   string
	header                 Header
	glyphTable, blockTable uint32
	cacheBlock             uint32
	cache                  string
	cacheValid             bool
	inflations             uint64
}

func u16(s string, p int) uint16 { return uint16(s[p]) | uint16(s[p+1])<<8 }
func u32(s string, p int) uint32 {
	return uint32(s[p]) | uint32(s[p+1])<<8 | uint32(s[p+2])<<16 | uint32(s[p+3])<<24
}

// Open strictly validates an MGZ1 file and returns a font with an empty cache.
func Open(data string) (*Font, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("mgz: header is truncated")
	}
	if data[:4] != "MGZ1" {
		return nil, fmt.Errorf("mgz: invalid magic")
	}
	if u16(data, 4) != Version1 || u16(data, 6) != HeaderSize {
		return nil, fmt.Errorf("mgz: unsupported version or header size")
	}
	if uint64(len(data)) > math.MaxUint32 || u32(data, 20) != uint32(len(data)) {
		return nil, fmt.Errorf("mgz: file size does not match data")
	}
	if u16(data, 18) != 0 || u16(data, 34) != CodecDeflate {
		return nil, fmt.Errorf("mgz: unsupported header flags or codec")
	}
	for _, p := range []int{41, 42, 43, 60, 61, 62, 63} {
		if data[p] != 0 {
			return nil, fmt.Errorf("mgz: reserved header bytes are nonzero")
		}
	}

	gc, bc, gpb := u32(data, 24), u32(data, 28), u16(data, 32)
	if gpb == 0 {
		return nil, fmt.Errorf("mgz: GlyphsPerBlock is zero")
	}
	wantBlocks := uint32(0)
	if gc != 0 {
		wantBlocks = (gc-1)/uint32(gpb) + 1
	}
	if bc != wantBlocks {
		return nil, fmt.Errorf("mgz: BlockCount is %d, want %d", bc, wantBlocks)
	}
	gt, bt, bd := u32(data, 44), u32(data, 48), u32(data, 52)
	if uint64(gt) != HeaderSize || uint64(bt) != uint64(gt)+uint64(gc)*GlyphEntrySize || uint64(bd) != uint64(bt)+uint64(bc)*BlockEntrySize || uint64(bd) > uint64(len(data)) {
		return nil, fmt.Errorf("mgz: noncanonical or out-of-range table offsets")
	}
	f := &Font{data: data, glyphTable: gt, blockTable: bt, cacheBlock: math.MaxUint32}
	copy(f.header.FontID[:], data[8:12])
	copy(f.header.SubsetID[:], data[12:16])
	copy(f.header.Region[:], data[16:18])
	f.header.GlyphCount, f.header.BlockCount, f.header.GlyphsPerBlock = gc, bc, gpb
	f.header.Ascent, f.header.Descent, f.header.LineGap = data[36], data[37], data[38]
	f.header.MaxWidth, f.header.MaxHeight, f.header.OriginalMGFSize = data[39], data[40], u32(data, 56)
	nextData := bd
	for i := uint32(0); i < bc; i++ {
		p := int(bt + i*BlockEntrySize)
		b := block{u32(data, p), u32(data, p+4), u32(data, p+8), u32(data, p+12)}
		if b.flags != 0 && b.flags != blockDeflated {
			return nil, fmt.Errorf("mgz: block %d has unsupported flags", i)
		}
		if b.offset != nextData || uint64(b.offset)+uint64(b.storedLen) > uint64(len(data)) {
			return nil, fmt.Errorf("mgz: block %d has invalid data offset", i)
		}
		if b.flags == 0 && b.storedLen != b.rawLen {
			return nil, fmt.Errorf("mgz: raw block %d length mismatch", i)
		}
		nextData = b.offset + b.storedLen
	}
	if nextData != uint32(len(data)) {
		return nil, fmt.Errorf("mgz: trailing or missing block data")
	}
	if err := f.validateGlyphs(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *Font) blockAt(index uint32) block {
	p := int(f.blockTable + index*BlockEntrySize)
	return block{u32(f.data, p), u32(f.data, p+4), u32(f.data, p+8), u32(f.data, p+12)}
}

// ValidateAll expands and validates every DEFLATE stream. It is intended for
// tests and offline verification; Open and normal startup do not call it.
func (f *Font) ValidateAll() error {
	if f == nil {
		return fmt.Errorf("mgz: nil font")
	}
	for i := uint32(0); i < f.header.BlockCount; i++ {
		if err := f.validateBlock(i); err != nil {
			return err
		}
	}
	return nil
}

func (f *Font) validateBlock(index uint32) error {
	b := f.blockAt(index)
	if b.flags == 0 {
		return nil
	}
	stored := f.data[int(b.offset):int(b.offset+b.storedLen)]
	r := flate.NewReader(strings.NewReader(stored))
	n, err := io.Copy(io.Discard, io.LimitReader(r, int64(b.rawLen)+1))
	closeErr := r.Close()
	if err != nil {
		return fmt.Errorf("mgz: inflate block %d: %w", index, err)
	}
	if closeErr != nil {
		return fmt.Errorf("mgz: close block %d: %w", index, closeErr)
	}
	if n != int64(b.rawLen) {
		return fmt.Errorf("mgz: block %d expands to %d bytes, want %d", index, n, b.rawLen)
	}
	return nil
}

func MustOpen(data string) *Font {
	f, err := Open(data)
	if err != nil {
		panic(err)
	}
	return f
}
func (f *Font) Header() Header {
	if f == nil {
		return Header{}
	}
	return f.header
}
func (f *Font) GlyphCount() int {
	if f == nil {
		return 0
	}
	return int(f.header.GlyphCount)
}
func (f *Font) LineHeight() int {
	if f == nil {
		return 0
	}
	return int(f.header.Ascent) + int(f.header.Descent) + int(f.header.LineGap)
}

func (f *Font) validateGlyphs() error {
	var prev uint32
	var blockIndex, blockEnd uint32
	var maxW, maxH uint8
	for i := uint32(0); i < f.header.GlyphCount; i++ {
		p := int(f.glyphTable + i*GlyphEntrySize)
		cp := u32(f.data, p)
		if cp > 0x10ffff || cp >= 0xd800 && cp <= 0xdfff || i > 0 && cp <= prev {
			return fmt.Errorf("mgz: glyph %d code point is invalid or unsorted", i)
		}
		w, h := uint8(f.data[p+4]), uint8(f.data[p+5])
		n, off := uint32(u16(f.data, p+12)), uint32(u16(f.data, p+14))
		bi := i / uint32(f.header.GlyphsPerBlock)
		if bi != blockIndex {
			if blockEnd != f.blockAt(blockIndex).rawLen {
				return fmt.Errorf("mgz: block %d raw extent mismatch", blockIndex)
			}
			blockIndex, blockEnd = bi, 0
		}
		if w > f.header.MaxWidth || h > f.header.MaxHeight || n != uint32((uint16(w)+7)/8)*uint32(h) {
			return fmt.Errorf("mgz: glyph %d has invalid dimensions or bitmap length", i)
		}
		if off != blockEnd || uint64(off)+uint64(n) > uint64(f.blockAt(bi).rawLen) {
			return fmt.Errorf("mgz: glyph %d has invalid block offset", i)
		}
		blockEnd = off + n
		if w > maxW {
			maxW = w
		}
		if h > maxH {
			maxH = h
		}
		prev = cp
	}
	if f.header.BlockCount != 0 && blockEnd != f.blockAt(blockIndex).rawLen {
		return fmt.Errorf("mgz: block %d raw extent mismatch", blockIndex)
	}
	if maxW != f.header.MaxWidth || maxH != f.header.MaxHeight {
		return fmt.Errorf("mgz: maximum glyph dimensions mismatch")
	}
	return nil
}

func (f *Font) expand(index uint32, cache bool) (string, error) {
	if cache && f.cacheValid && f.cacheBlock == index {
		return f.cache, nil
	}
	b := f.blockAt(index)
	stored := f.data[int(b.offset):int(b.offset+b.storedLen)]
	raw := stored
	if b.flags == blockDeflated {
		r := flate.NewReader(strings.NewReader(stored))
		var output strings.Builder
		output.Grow(int(b.rawLen))
		_, err := io.Copy(&output, io.LimitReader(r, int64(b.rawLen)+1))
		closeErr := r.Close()
		if err != nil {
			return "", fmt.Errorf("mgz: inflate block %d: %w", index, err)
		}
		if closeErr != nil {
			return "", fmt.Errorf("mgz: close block %d: %w", index, closeErr)
		}
		raw = output.String()
	}
	if len(raw) != int(b.rawLen) {
		return "", fmt.Errorf("mgz: block %d expands to %d bytes, want %d", index, len(raw), b.rawLen)
	}
	if cache {
		f.cacheBlock, f.cache, f.cacheValid = index, raw, true
		if b.flags == blockDeflated {
			f.inflations++
		}
	}
	return raw, nil
}

func (f *Font) Lookup(r rune) (Glyph, bool) {
	if f == nil || r < 0 || r > 0x10ffff || r >= 0xd800 && r <= 0xdfff {
		return Glyph{}, false
	}
	lo, hi := uint32(0), f.header.GlyphCount
	for lo < hi {
		mid := lo + (hi-lo)/2
		cp := u32(f.data, int(f.glyphTable+mid*GlyphEntrySize))
		if cp < uint32(r) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == f.header.GlyphCount {
		return Glyph{}, false
	}
	p := int(f.glyphTable + lo*GlyphEntrySize)
	if u32(f.data, p) != uint32(r) {
		return Glyph{}, false
	}
	raw, err := f.expand(lo/uint32(f.header.GlyphsPerBlock), true)
	if err != nil {
		return Glyph{}, false
	}
	off, n := int(u16(f.data, p+14)), int(u16(f.data, p+12))
	return Glyph{uint8(f.data[p+4]), uint8(f.data[p+5]), int16(u16(f.data, p+6)), int16(u16(f.data, p+8)), int16(u16(f.data, p+10)), raw[off : off+n]}, true
}
