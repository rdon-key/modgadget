package text

import (
	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/mgf"
)

// Glyph is the bitmap and placement information used by the text renderer.
type Glyph struct {
	Width    int16
	Height   int16
	AdvanceX int16
	BearingX int16
	BearingY int16
	Bitmap   string
}

// FontMetrics describes a font's baseline-relative line box.
type FontMetrics struct {
	Ascent  int16
	Descent int16
	LineGap int16
}

// LineHeight returns Ascent + Descent + LineGap.
func (metrics FontMetrics) LineHeight() int16 {
	return metrics.Ascent + metrics.Descent + metrics.LineGap
}

// Font is the minimal allocation-free font view used by MGF text rendering.
type Font interface {
	Lookup(r rune) (Glyph, bool)
	Metrics() FontMetrics
}

// MGFFont adapts a validated MGF font to the text renderer.
type MGFFont struct {
	source mgf.Font
}

// NewMGFFont adapts an already validated MGF font.
func NewMGFFont(source mgf.Font) MGFFont { return MGFFont{source: source} }

// Lookup returns a glyph whose bitmap still refers to the embedded MGF data.
func (font MGFFont) Lookup(r rune) (Glyph, bool) {
	glyph, ok := font.source.Lookup(r)
	if !ok {
		return Glyph{}, false
	}
	return Glyph{
		Width: int16(glyph.Width), Height: int16(glyph.Height), AdvanceX: glyph.AdvanceX,
		BearingX: glyph.BearingX, BearingY: glyph.BearingY, Bitmap: glyph.Bitmap,
	}, true
}

// Metrics returns the MGF header's typographic metrics.
func (font MGFFont) Metrics() FontMetrics {
	header := font.source.Header()
	return FontMetrics{Ascent: int16(header.Ascent), Descent: int16(header.Descent), LineGap: int16(header.LineGap)}
}

// FontStack searches fonts of the same display size in priority order.
// When more than one font contains a code point, the first font wins.
type FontStack struct {
	Primary   Font
	Fallbacks [3]Font
}

// Lookup searches Primary followed by Fallbacks in array order.
func (stack FontStack) Lookup(r rune) (Glyph, bool) {
	if stack.Primary != nil {
		if glyph, ok := stack.Primary.Lookup(r); ok {
			return glyph, true
		}
	}
	for index := range stack.Fallbacks {
		if stack.Fallbacks[index] != nil {
			if glyph, ok := stack.Fallbacks[index].Lookup(r); ok {
				return glyph, true
			}
		}
	}
	return Glyph{}, false
}

// Metrics returns the component-wise maximum line metrics.
func (stack FontStack) Metrics() FontMetrics {
	var result FontMetrics
	hasFont := false
	if stack.Primary != nil {
		result = stack.Primary.Metrics()
		hasFont = true
	}
	for index := range stack.Fallbacks {
		if stack.Fallbacks[index] != nil {
			metrics := stack.Fallbacks[index].Metrics()
			if !hasFont {
				result = metrics
				hasFont = true
			} else {
				result = maximumFontMetrics(result, metrics)
			}
		}
	}
	return result
}

func maximumFontMetrics(left, right FontMetrics) FontMetrics {
	if right.Ascent > left.Ascent {
		left.Ascent = right.Ascent
	}
	if right.Descent > left.Descent {
		left.Descent = right.Descent
	}
	if right.LineGap > left.LineGap {
		left.LineGap = right.LineGap
	}
	return left
}

// Style is the complete appearance applied to a span.
type Style struct {
	Font       Font
	Foreground display.Color565
	Background display.Color565
}

// StyleEntry associates a case-sensitive name with a complete Style.
type StyleEntry struct {
	Name  string
	Style Style
}

// StyleSet contains the default appearance and named styles used by markup.
type StyleSet struct {
	Default Style
	Entries []StyleEntry
}

// Lookup linearly searches Entries and returns the first exact name match.
func (styles StyleSet) Lookup(name string) (Style, bool) {
	for index := range styles.Entries {
		if styles.Entries[index].Name == name {
			return styles.Entries[index].Style, true
		}
	}
	return Style{}, false
}
