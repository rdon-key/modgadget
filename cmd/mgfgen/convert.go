package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rdon-key/modgadget/internal/bdf"
	"github.com/rdon-key/modgadget/internal/mgf"
)

type conversionOptions struct {
	bdfPath       string
	charsPath     string
	missing       string
	assumeUnicode bool
	lineGap       uint8
	fontID        []byte
	subsetID      []byte
	region        []byte
	output        string
}

type encodedGlyph struct {
	codepoint uint32
	glyph     mgf.Glyph
}

func convertBDF(options conversionOptions, stdout, stderr io.Writer) error {
	contents, err := os.ReadFile(options.bdfPath)
	if err != nil {
		return fmt.Errorf("mgfgen: read BDF: %w", err)
	}
	font, err := bdf.Parse(string(contents))
	if err != nil {
		return fmt.Errorf("mgfgen: parse BDF: %w", err)
	}
	if !options.assumeUnicode && !unicodeCharset(font.CharsetRegistry) {
		return fmt.Errorf("mgfgen: BDF CHARSET_REGISTRY %q is not Unicode; use -assume-unicode to override", font.CharsetRegistry)
	}

	byCodepoint := make(map[rune]bdf.Glyph, len(font.Glyphs))
	for _, glyph := range font.Glyphs {
		if glyph.Encoding < 0 {
			continue
		}
		if !validScalar(glyph.Encoding) {
			return fmt.Errorf("mgfgen: glyph %q has invalid Unicode codepoint U+%X", glyph.Name, glyph.Encoding)
		}
		r := rune(glyph.Encoding)
		if previous, exists := byCodepoint[r]; exists {
			return fmt.Errorf("mgfgen: duplicate Unicode codepoint U+%04X in glyphs %q and %q", r, previous.Name, glyph.Name)
		}
		byCodepoint[r] = glyph
	}

	selected := make([]bdf.Glyph, 0, len(byCodepoint))
	missingCount := 0
	if options.charsPath == "" {
		for _, glyph := range byCodepoint {
			selected = append(selected, glyph)
		}
	} else {
		characters, err := os.ReadFile(options.charsPath)
		if err != nil {
			return fmt.Errorf("mgfgen: read -chars: %w", err)
		}
		if !utf8.Valid(characters) {
			return fmt.Errorf("mgfgen: -chars is not valid UTF-8")
		}
		text := string(characters)
		text = strings.TrimPrefix(text, "\ufeff")
		seen := make(map[rune]struct{})
		for _, r := range text {
			if r == '\r' || r == '\n' {
				continue
			}
			if _, exists := seen[r]; exists {
				continue
			}
			seen[r] = struct{}{}
			glyph, exists := byCodepoint[r]
			if !exists {
				missingCount++
				if options.missing == "error" {
					return fmt.Errorf("mgfgen: selected character U+%04X is missing from BDF", r)
				}
				continue
			}
			selected = append(selected, glyph)
		}
		if len(selected) == 0 && missingCount != 0 {
			fmt.Fprintln(stderr, "mgfgen: warning: all selected characters are missing; writing an empty MGF")
		}
	}
	sort.Slice(selected, func(left, right int) bool { return selected[left].Encoding < selected[right].Encoding })
	if len(selected) > math.MaxUint16 {
		return fmt.Errorf("mgfgen: selected glyph count %d exceeds 65535", len(selected))
	}

	ascent, descent, err := fontMetrics(font)
	if err != nil {
		return err
	}
	glyphs := make([]encodedGlyph, len(selected))
	var maxWidth, maxHeight uint8
	var recordsSize uint64
	for index, source := range selected {
		glyph, err := convertGlyph(source)
		if err != nil {
			return fmt.Errorf("mgfgen: glyph U+%04X: %w", source.Encoding, err)
		}
		glyphs[index] = encodedGlyph{codepoint: uint32(source.Encoding), glyph: glyph}
		if glyph.Width > maxWidth {
			maxWidth = glyph.Width
		}
		if glyph.Height > maxHeight {
			maxHeight = glyph.Height
		}
		recordsSize += uint64(mgf.GlyphRecordHeaderSize + len(glyph.Bitmap))
	}
	glyphDataOffset := uint64(mgf.HeaderSize) + uint64(len(glyphs))*mgf.IndexEntrySize
	fileSize := glyphDataOffset + recordsSize
	if glyphDataOffset > math.MaxUint32 || fileSize > math.MaxUint32 || fileSize > uint64(^uint(0)>>1) {
		return fmt.Errorf("mgfgen: output size %d is too large", fileSize)
	}
	header := mgf.Header{
		Version: mgf.Version1, GlyphCount: uint16(len(glyphs)), Ascent: ascent, Descent: descent,
		LineGap: options.lineGap, MaxWidth: maxWidth, MaxHeight: maxHeight, Flags: 0,
		HeaderSize: mgf.HeaderSize, IndexOffset: mgf.HeaderSize,
		GlyphDataOffset: uint32(glyphDataOffset), FileSize: uint32(fileSize),
	}
	copy(header.FontID[:], options.fontID)
	copy(header.SubsetID[:], options.subsetID)
	copy(header.Region[:], options.region)

	output := make([]byte, int(fileSize))
	if err := mgf.EncodeHeader(output[:mgf.HeaderSize], header); err != nil {
		return fmt.Errorf("mgfgen: encode header: %w", err)
	}
	entries := make([]mgf.IndexEntry, len(glyphs))
	offset := uint32(glyphDataOffset)
	for index, item := range glyphs {
		entries[index] = mgf.IndexEntry{Codepoint: item.codepoint, GlyphOffset: offset}
		offset += uint32(mgf.GlyphRecordHeaderSize + len(item.glyph.Bitmap))
	}
	if err := mgf.EncodeIndex(output[mgf.HeaderSize:glyphDataOffset], header, entries); err != nil {
		return fmt.Errorf("mgfgen: encode index: %w", err)
	}
	for index, item := range glyphs {
		start := int(entries[index].GlyphOffset)
		written, err := mgf.EncodeGlyphRecord(output[start:], item.glyph)
		if err != nil {
			return fmt.Errorf("mgfgen: encode glyph U+%04X: %w", item.codepoint, err)
		}
		if written != mgf.GlyphRecordHeaderSize+len(item.glyph.Bitmap) {
			return fmt.Errorf("mgfgen: encode glyph U+%04X wrote %d unexpected bytes", item.codepoint, written)
		}
	}
	immutable := string(output)
	decodedHeader, err := mgf.DecodeHeader(immutable)
	if err != nil {
		return fmt.Errorf("mgfgen: validate header: %w", err)
	}
	decodedIndex, err := mgf.DecodeIndex(immutable, decodedHeader)
	if err != nil {
		return fmt.Errorf("mgfgen: validate index: %w", err)
	}
	if err := mgf.ValidateGlyphData(immutable, decodedHeader, decodedIndex); err != nil {
		return fmt.Errorf("mgfgen: validate glyph data: %w", err)
	}
	if err := writeAtomic(options.output, output); err != nil {
		return fmt.Errorf("mgfgen: write %s: %w", options.output, err)
	}

	fmt.Fprintf(stdout, "wrote %s\n", options.output)
	fmt.Fprintf(stdout, "FontID: %s\n", string(options.fontID))
	fmt.Fprintf(stdout, "SubsetID: %s\n", string(options.subsetID))
	if len(options.region) == 0 {
		fmt.Fprintln(stdout, "Region: none")
	} else {
		fmt.Fprintf(stdout, "Region: %s\n", string(options.region))
	}
	fmt.Fprintf(stdout, "GlyphCount: %d\n", header.GlyphCount)
	fmt.Fprintf(stdout, "Ascent: %d\nDescent: %d\nLineGap: %d\n", header.Ascent, header.Descent, header.LineGap)
	fmt.Fprintf(stdout, "MaxWidth: %d\nMaxHeight: %d\n", header.MaxWidth, header.MaxHeight)
	fmt.Fprintf(stdout, "missing: %d\nbytes: %d\n", missingCount, header.FileSize)
	return nil
}

func unicodeCharset(registry string) bool {
	return strings.EqualFold(registry, "ISO10646") || strings.EqualFold(registry, "Unicode")
}

func validScalar(value int) bool {
	return value >= 0 && value <= utf8.MaxRune && (value < 0xd800 || value > 0xdfff)
}

func fontMetrics(font bdf.Font) (uint8, uint8, error) {
	if font.HasAscent != font.HasDescent {
		return 0, 0, fmt.Errorf("mgfgen: FONT_ASCENT and FONT_DESCENT must both be present or absent")
	}
	ascent, descent := int64(font.Ascent), int64(font.Descent)
	if !font.HasAscent {
		ascent, descent = 0, 0
		for _, glyph := range font.Glyphs {
			if glyph.Encoding < 0 {
				continue
			}
			y := int64(glyph.YOffset)
			height := int64(glyph.Height)
			if height < 0 || (height > 0 && y > math.MaxInt64-height) {
				return 0, 0, fmt.Errorf("mgfgen: glyph %q U+%04X Ascent calculation overflows", glyph.Name, glyph.Encoding)
			}
			if top := y + height; top > ascent {
				if top > math.MaxUint8 {
					return 0, 0, fmt.Errorf("mgfgen: glyph %q U+%04X Ascent %d does not fit uint8", glyph.Name, glyph.Encoding, top)
				}
				ascent = top
			}
			if y < 0 {
				if y == math.MinInt64 {
					return 0, 0, fmt.Errorf("mgfgen: glyph %q U+%04X Descent calculation overflows", glyph.Name, glyph.Encoding)
				}
				if below := -y; below > descent {
					if below > math.MaxUint8 {
						return 0, 0, fmt.Errorf("mgfgen: glyph %q U+%04X Descent %d does not fit uint8", glyph.Name, glyph.Encoding, below)
					}
					descent = below
				}
			}
		}
	}
	if ascent < 0 || ascent > math.MaxUint8 {
		return 0, 0, fmt.Errorf("mgfgen: Ascent %d does not fit uint8", ascent)
	}
	if descent < 0 || descent > math.MaxUint8 {
		return 0, 0, fmt.Errorf("mgfgen: Descent %d does not fit uint8", descent)
	}
	return uint8(ascent), uint8(descent), nil
}

func convertGlyph(source bdf.Glyph) (mgf.Glyph, error) {
	if source.Width < 0 || source.Width > math.MaxUint8 {
		return mgf.Glyph{}, fmt.Errorf("Width %d does not fit uint8", source.Width)
	}
	if source.Height < 0 || source.Height > math.MaxUint8 {
		return mgf.Glyph{}, fmt.Errorf("Height %d does not fit uint8", source.Height)
	}
	bearingY := int64(source.YOffset) + int64(source.Height)
	if source.AdvanceX < math.MinInt16 || source.AdvanceX > math.MaxInt16 {
		return mgf.Glyph{}, fmt.Errorf("AdvanceX %d does not fit int16", source.AdvanceX)
	}
	if source.XOffset < math.MinInt16 || source.XOffset > math.MaxInt16 {
		return mgf.Glyph{}, fmt.Errorf("BearingX %d does not fit int16", source.XOffset)
	}
	if bearingY < math.MinInt16 || bearingY > math.MaxInt16 {
		return mgf.Glyph{}, fmt.Errorf("BearingY %d does not fit int16", bearingY)
	}
	glyph := mgf.Glyph{Width: uint8(source.Width), Height: uint8(source.Height), AdvanceX: int16(source.AdvanceX), BearingX: int16(source.XOffset), BearingY: int16(bearingY), Bitmap: source.Bitmap}
	if len(glyph.Bitmap) != int(mgf.RawBitmapSize(glyph.Width, glyph.Height)) {
		return mgf.Glyph{}, fmt.Errorf("Bitmap has %d bytes, want %d", len(glyph.Bitmap), mgf.RawBitmapSize(glyph.Width, glyph.Height))
	}
	return glyph, nil
}
