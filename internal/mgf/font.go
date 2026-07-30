package mgf

import "fmt"

// Font is an immutable view of a validated MGF font.
//
// Font retains the source string and decodes glyph records on demand. It does
// not copy the source data or Glyph Index entries.
type Font struct {
	data   string
	header Header
	index  Index
}

// Open validates data and returns an immutable font view.
func Open(data string) (Font, error) {
	header, err := DecodeHeader(data)
	if err != nil {
		return Font{}, fmt.Errorf("mgf: open header: %w", err)
	}
	index, err := DecodeIndex(data, header)
	if err != nil {
		return Font{}, fmt.Errorf("mgf: open index: %w", err)
	}
	if err := ValidateGlyphData(data, header, index); err != nil {
		return Font{}, fmt.Errorf("mgf: open glyph data: %w", err)
	}
	return Font{data: data, header: header, index: index}, nil
}

// MustOpen is like Open but panics if data is not a valid MGF font.
func MustOpen(data string) Font {
	font, err := Open(data)
	if err != nil {
		panic(err)
	}
	return font
}

// Header returns the validated MGF header by value.
func (f Font) Header() Header {
	return f.header
}

// GlyphCount returns the number of glyphs in the font.
func (f Font) GlyphCount() int {
	return int(f.header.GlyphCount)
}

// LineHeight returns Ascent + Descent + LineGap.
func (f Font) LineHeight() int {
	return int(f.header.Ascent) + int(f.header.Descent) + int(f.header.LineGap)
}

// Lookup returns the glyph for r without copying its bitmap.
func (f Font) Lookup(r rune) (Glyph, bool) {
	offset, ok := f.index.Lookup(r)
	if !ok {
		return Glyph{}, false
	}
	glyph, _, err := DecodeGlyphRecord(f.data, f.header, offset)
	if err != nil {
		return Glyph{}, false
	}
	return glyph, true
}
