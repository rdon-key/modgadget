package text

// testGlyphInfo is the metadata needed to construct bitmap-font test fixtures.
// It deliberately models only the fields exercised by the text tests.
type testGlyphInfo struct {
	Rune         rune
	BitmapOffset uint32
	Width        int16
	Height       int16
	AdvanceX     int16
	BearingX     int16
	BearingY     int16
}

type testFixtureFont struct {
	metrics FontMetrics
	glyphs  []testGlyphInfo
	bitmap  string
}

func (font testFixtureFont) Metrics() FontMetrics { return font.metrics }

func (font testFixtureFont) Lookup(r rune) (Glyph, bool) {
	for index := range font.glyphs {
		info := font.glyphs[index]
		if info.Rune != r {
			continue
		}
		if info.Width < 0 || info.Height < 0 {
			return Glyph{}, false
		}
		rowBytes := (uint64(info.Width) + 7) / 8
		start := uint64(info.BitmapOffset)
		end := start + rowBytes*uint64(info.Height)
		if end < start || end > uint64(len(font.bitmap)) {
			return Glyph{}, false
		}
		return Glyph{
			Width: info.Width, Height: info.Height, AdvanceX: info.AdvanceX,
			BearingX: info.BearingX, BearingY: info.BearingY,
			Bitmap: font.bitmap[int(start):int(end)],
		}, true
	}
	return Glyph{}, false
}

func spanFace(metrics FontMetrics, glyphs []testGlyphInfo, bitmap string) Font {
	return &testFixtureFont{metrics: metrics, glyphs: glyphs, bitmap: bitmap}
}

type fixedFont struct {
	metrics FontMetrics
	glyphs  [4]struct {
		r rune
		g Glyph
	}
}

func (font *fixedFont) Lookup(r rune) (Glyph, bool) {
	for index := range font.glyphs {
		if font.glyphs[index].r == r {
			return font.glyphs[index].g, true
		}
	}
	return Glyph{}, false
}

func (font *fixedFont) Metrics() FontMetrics { return font.metrics }
