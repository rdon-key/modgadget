package modgadget

import (
	"fmt"

	displaypkg "github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/mgf"
	"github.com/rdon-key/modgadget/internal/text"
	"github.com/rdon-key/modgadget/internal/text/markup"
)

// Font is an opaque, copyable handle to a ModGadget font.
// Its zero value is invalid.
type Font struct{ impl text.Font }

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

// Valid reports whether font refers to a usable font implementation.
func (font Font) Valid() bool { return font.impl != nil }

// HasGlyph reports whether font contains r.
func (font Font) HasGlyph(r rune) bool {
	if font.impl == nil {
		return false
	}
	_, ok := font.impl.Lookup(r)
	return ok
}

// Metrics returns the font's line metrics, or zero metrics for an invalid font.
func (font Font) Metrics() FontMetrics {
	if font.impl == nil {
		return FontMetrics{}
	}
	metrics := font.impl.Metrics()
	return FontMetrics{Ascent: metrics.Ascent, Descent: metrics.Descent, LineGap: metrics.LineGap}
}

// OpenMGF validates data and returns an opaque font without copying the MGF data.
func OpenMGF(data string) (Font, error) {
	source, err := mgf.Open(data)
	if err != nil {
		return Font{}, fmt.Errorf("modgadget: open MGF: %w", err)
	}
	return Font{impl: mgfFont{source: source}}, nil
}

// MustOpenMGF is like OpenMGF but panics if data is invalid.
func MustOpenMGF(data string) Font {
	font, err := OpenMGF(data)
	if err != nil {
		panic(err)
	}
	return font
}

type mgfFont struct{ source mgf.Font }

func (font mgfFont) Lookup(r rune) (text.Glyph, bool) {
	glyph, ok := font.source.Lookup(r)
	if !ok {
		return text.Glyph{}, false
	}
	return text.Glyph{
		Width: int16(glyph.Width), Height: int16(glyph.Height), AdvanceX: glyph.AdvanceX,
		BearingX: glyph.BearingX, BearingY: glyph.BearingY, Bitmap: glyph.Bitmap,
	}, true
}

func (font mgfFont) Metrics() text.FontMetrics {
	header := font.source.Header()
	return text.FontMetrics{Ascent: int16(header.Ascent), Descent: int16(header.Descent), LineGap: int16(header.LineGap)}
}

const maximumFontFallbacks = 3

type fontStack struct {
	fonts [1 + maximumFontFallbacks]text.Font
	count uint8
}

func (stack fontStack) Lookup(r rune) (text.Glyph, bool) {
	for i := uint8(0); i < stack.count; i++ {
		if glyph, ok := stack.fonts[i].Lookup(r); ok {
			return glyph, true
		}
	}
	return text.Glyph{}, false
}

func (stack fontStack) Metrics() text.FontMetrics {
	if stack.count == 0 {
		return text.FontMetrics{}
	}
	result := stack.fonts[0].Metrics()
	for i := uint8(1); i < stack.count; i++ {
		metrics := stack.fonts[i].Metrics()
		if metrics.Ascent > result.Ascent {
			result.Ascent = metrics.Ascent
		}
		if metrics.Descent > result.Descent {
			result.Descent = metrics.Descent
		}
		if metrics.LineGap > result.LineGap {
			result.LineGap = metrics.LineGap
		}
	}
	return result
}

// NewFontStack returns a font that searches primary, then fallbacks in order.
// At most three fallback fonts are supported.
func NewFontStack(primary Font, fallbacks ...Font) (Font, error) {
	if !primary.Valid() {
		return Font{}, fmt.Errorf("modgadget: primary font is invalid")
	}
	if len(fallbacks) > maximumFontFallbacks {
		return Font{}, fmt.Errorf("modgadget: %d fallback fonts exceeds maximum %d", len(fallbacks), maximumFontFallbacks)
	}
	stack := fontStack{count: uint8(1 + len(fallbacks))}
	stack.fonts[0] = primary.impl
	for i := range fallbacks {
		if !fallbacks[i].Valid() {
			return Font{}, fmt.Errorf("modgadget: fallback font %d is invalid", i)
		}
		stack.fonts[i+1] = fallbacks[i].impl
	}
	return Font{impl: stack}, nil
}

// Style is the complete appearance applied to a text span.
type Style struct {
	Font       Font
	Foreground Color565
	Background Color565
	Bold       bool
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

// Lookup returns the first Style whose name exactly matches name.
func (styles StyleSet) Lookup(name string) (Style, bool) {
	for i := range styles.Entries {
		if styles.Entries[i].Name == name {
			return styles.Entries[i].Style, true
		}
	}
	return Style{}, false
}

func internalStyles(styles StyleSet) text.StyleSet {
	result := text.StyleSet{Default: internalStyle(styles.Default)}
	if len(styles.Entries) != 0 {
		result.Entries = make([]text.StyleEntry, len(styles.Entries))
		for i := range styles.Entries {
			result.Entries[i] = text.StyleEntry{Name: styles.Entries[i].Name, Style: internalStyle(styles.Entries[i].Style)}
		}
	}
	return result
}

func internalStyle(style Style) text.Style {
	return text.Style{Font: style.Font.impl, Foreground: displaypkg.Color565(style.Foreground), Background: displaypkg.Color565(style.Background), Bold: style.Bold}
}

// TextMeasurement reports the width and line count of parsed and laid-out text.
type TextMeasurement struct {
	Width     int16
	LineCount int
}

// MeasureText parses and measures value using the same rules as Viewport.SetText.
func MeasureText(value string, styles StyleSet) (TextMeasurement, error) {
	if value == "" {
		return TextMeasurement{}, nil
	}
	spans, err := (markup.Parser{Styles: internalStyles(styles)}).Parse(value)
	if err != nil {
		return TextMeasurement{}, fmt.Errorf("modgadget: parse text: %w", err)
	}
	layout, err := text.NewTextLayout(spans)
	if err != nil {
		return TextMeasurement{}, fmt.Errorf("modgadget: layout text: %w", err)
	}
	measurement := layout.Measurement()
	width := measurement.MaxAdvanceX
	for i := range spans {
		if spans[i].Bold && measurement.HasInk && measurement.Bounds.MaxX > width {
			width = measurement.Bounds.MaxX
			break
		}
	}
	return TextMeasurement{Width: width, LineCount: layout.LineCount()}, nil
}
