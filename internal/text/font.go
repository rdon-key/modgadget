package text

import (
	fontpkg "github.com/rdon-key/modgadget-fonts/font"
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
	Font mgf.Font
}

// Lookup returns a glyph whose bitmap still refers to the embedded MGF data.
func (font MGFFont) Lookup(r rune) (Glyph, bool) {
	glyph, ok := font.Font.Lookup(r)
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
	header := font.Font.Header()
	return FontMetrics{Ascent: int16(header.Ascent), Descent: int16(header.Descent), LineGap: int16(header.LineGap)}
}

// FallbackFont searches Primary before Fallback.
type FallbackFont struct {
	Primary  Font
	Fallback Font
}

// Lookup returns the primary glyph when both fonts contain r.
func (font FallbackFont) Lookup(r rune) (Glyph, bool) {
	if font.Primary != nil {
		if glyph, ok := font.Primary.Lookup(r); ok {
			return glyph, true
		}
	}
	if font.Fallback != nil {
		return font.Fallback.Lookup(r)
	}
	return Glyph{}, false
}

// Metrics returns the component-wise maximum line metrics.
func (font FallbackFont) Metrics() FontMetrics {
	if font.Primary == nil {
		if font.Fallback == nil {
			return FontMetrics{}
		}
		return font.Fallback.Metrics()
	}
	metrics := font.Primary.Metrics()
	if font.Fallback != nil {
		fallback := font.Fallback.Metrics()
		if fallback.Ascent > metrics.Ascent {
			metrics.Ascent = fallback.Ascent
		}
		if fallback.Descent > metrics.Descent {
			metrics.Descent = fallback.Descent
		}
		if fallback.LineGap > metrics.LineGap {
			metrics.LineGap = fallback.LineGap
		}
	}
	return metrics
}

type legacyFont struct{ face *fontpkg.Font }

func (font legacyFont) Lookup(r rune) (Glyph, bool) {
	glyph, ok := font.face.Lookup(r)
	if !ok {
		return Glyph{}, false
	}
	return Glyph{Width: glyph.Width, Height: glyph.Height, AdvanceX: glyph.AdvanceX, BearingX: glyph.BearingX, BearingY: glyph.BearingY, Bitmap: glyph.Bitmap}, true
}
func (font legacyFont) Metrics() FontMetrics {
	metrics := font.face.Metrics()
	return FontMetrics{Ascent: metrics.Ascent, Descent: metrics.Descent, LineGap: metrics.LineGap}
}
