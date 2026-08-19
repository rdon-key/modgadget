package text

import (
	"github.com/rdon-key/modgadget/internal/display"
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

// GlyphMetadata is glyph placement information without bitmap data.
type GlyphMetadata struct {
	Width    int16
	Height   int16
	AdvanceX int16
	BearingX int16
	BearingY int16
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

// MetadataFont optionally provides glyph placement without loading a bitmap.
type MetadataFont interface {
	LookupMetadata(r rune) (GlyphMetadata, bool)
}

// ResolvedGlyph identifies the leaf font selected for a glyph. It is a
// transient drawing value and is not retained by fonts or layouts.
type ResolvedGlyph struct {
	metadata GlyphMetadata
	leaf     Font
	glyph    Glyph
	hasGlyph bool
}

// Metadata returns the resolved placement information.
func (resolved ResolvedGlyph) Metadata() GlyphMetadata { return resolved.metadata }

func (resolved ResolvedGlyph) load(r rune) (Glyph, bool) {
	if resolved.hasGlyph {
		return resolved.glyph, true
	}
	if resolved.leaf == nil {
		return Glyph{}, false
	}
	return resolved.leaf.Lookup(r)
}

type glyphResolver interface {
	ResolveGlyph(r rune) (ResolvedGlyph, bool)
}

// ResolveGlyph selects one leaf font and preserves a Glyph already loaded
// while falling back from metadata lookup.
func ResolveGlyph(font Font, r rune) (ResolvedGlyph, bool) {
	if font == nil {
		return ResolvedGlyph{}, false
	}
	if resolver, ok := font.(glyphResolver); ok {
		return resolver.ResolveGlyph(r)
	}
	if metadataFont, ok := font.(MetadataFont); ok {
		metadata, ok := metadataFont.LookupMetadata(r)
		if !ok {
			return ResolvedGlyph{}, false
		}
		return ResolvedGlyph{metadata: metadata, leaf: font}, true
	}
	glyph, ok := font.Lookup(r)
	if !ok {
		return ResolvedGlyph{}, false
	}
	return ResolvedGlyph{metadata: metadataFromGlyph(glyph), leaf: font, glyph: glyph, hasGlyph: true}, true
}

// LookupMetadata uses a font's metadata-only path when available and otherwise
// falls back to Lookup.
func LookupMetadata(font Font, r rune) (GlyphMetadata, bool) {
	resolved, ok := ResolveGlyph(font, r)
	if !ok {
		return GlyphMetadata{}, false
	}
	return resolved.metadata, true
}

func metadataFromGlyph(glyph Glyph) GlyphMetadata {
	return GlyphMetadata{
		Width: glyph.Width, Height: glyph.Height, AdvanceX: glyph.AdvanceX,
		BearingX: glyph.BearingX, BearingY: glyph.BearingY,
	}
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

// LookupMetadata searches Primary followed by Fallbacks in array order.
func (stack FontStack) LookupMetadata(r rune) (GlyphMetadata, bool) {
	resolved, ok := stack.ResolveGlyph(r)
	return resolved.metadata, ok
}

// ResolveGlyph searches Primary followed by Fallbacks and preserves the leaf
// font chosen by the first successful metadata resolution.
func (stack FontStack) ResolveGlyph(r rune) (ResolvedGlyph, bool) {
	if stack.Primary != nil {
		if glyph, ok := ResolveGlyph(stack.Primary, r); ok {
			return glyph, true
		}
	}
	for index := range stack.Fallbacks {
		if stack.Fallbacks[index] != nil {
			if glyph, ok := ResolveGlyph(stack.Fallbacks[index], r); ok {
				return glyph, true
			}
		}
	}
	return ResolvedGlyph{}, false
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
	// Bold enables synthetic bold rendering by extending glyph ink one pixel
	// to the right. It does not change the font's glyph advance.
	Bold bool
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
