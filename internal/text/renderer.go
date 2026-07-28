// Package text draws bitmap-font text on display backends.
package text

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/rdon-key/modgadget-fonts/font"
	"github.com/rdon-key/modgadget/internal/display"
)

// DrawString expands each glyph in value to RGB565 and draws it at the
// current pen position. scratch must hold the largest expanded glyph.
func DrawString(
	backend display.Backend,
	face *font.Font,
	x int16,
	lineTop int16,
	value string,
	foreground display.Color565,
	background display.Color565,
	scratch []byte,
) (advance int32, err error) {
	if backend == nil {
		return 0, fmt.Errorf("text: backend is nil")
	}
	if face == nil {
		return 0, fmt.Errorf("text: font is nil")
	}
	if !utf8.ValidString(value) {
		return 0, fmt.Errorf("text: value is not valid UTF-8")
	}

	startX := int64(x)
	penX := startX
	for _, r := range value {
		glyph, ok := face.Lookup(r)
		if !ok {
			return prefixAdvance(startX, penX), fmt.Errorf("text: glyph U+%04X is missing or invalid", r)
		}
		if glyph.Width < 0 || glyph.Height < 0 {
			return prefixAdvance(startX, penX), fmt.Errorf("text: glyph U+%04X has negative dimensions", r)
		}

		width := int64(glyph.Width)
		height := int64(glyph.Height)
		rowBytes := (width + 7) / 8
		bitmapBytes := rowBytes * height
		if bitmapBytes != int64(len(glyph.Bitmap)) {
			return prefixAdvance(startX, penX), fmt.Errorf("text: glyph U+%04X bitmap length is %d, want %d", r, len(glyph.Bitmap), bitmapBytes)
		}

		if width == 0 || height == 0 {
			penX += int64(glyph.Advance)
			continue
		}

		required := width * height * 2
		if required < 0 || required > int64(len(scratch)) {
			return prefixAdvance(startX, penX), fmt.Errorf("text: scratch too small for glyph U+%04X: have %d bytes, need %d", r, len(scratch), required)
		}

		drawX := penX + int64(glyph.XOffset)
		drawY := int64(lineTop) + int64(glyph.YOffset)
		if drawX < math.MinInt16 || drawX > math.MaxInt16 {
			return prefixAdvance(startX, penX), fmt.Errorf("text: glyph U+%04X X coordinate is outside int16", r)
		}
		if drawY < math.MinInt16 || drawY > math.MaxInt16 {
			return prefixAdvance(startX, penX), fmt.Errorf("text: glyph U+%04X Y coordinate is outside int16", r)
		}

		fgHigh, fgLow := byte(foreground>>8), byte(foreground)
		bgHigh, bgLow := byte(background>>8), byte(background)
		out := scratch[:int(required)]
		for row := int64(0); row < height; row++ {
			bitmapRow := row * rowBytes
			outputRow := row * width * 2
			for column := int64(0); column < width; column++ {
				set := glyph.Bitmap[int(bitmapRow+column/8)]&(byte(0x80)>>uint(column&7)) != 0
				output := int(outputRow + column*2)
				if set {
					out[output], out[output+1] = fgHigh, fgLow
				} else {
					out[output], out[output+1] = bgHigh, bgLow
				}
			}
		}

		rect := display.Rect{X: int16(drawX), Y: int16(drawY), Width: glyph.Width, Height: glyph.Height}
		if err := display.BlitRGB565(backend, rect, out, int(width*2)); err != nil {
			return prefixAdvance(startX, penX), fmt.Errorf("text: draw glyph U+%04X: %w", r, err)
		}
		penX += int64(glyph.Advance)
	}

	total := penX - startX
	if total < math.MinInt32 || total > math.MaxInt32 {
		return 0, fmt.Errorf("text: string advance is outside int32")
	}
	return int32(total), nil
}

func prefixAdvance(startX, penX int64) int32 {
	advance := penX - startX
	if advance < math.MinInt32 || advance > math.MaxInt32 {
		return 0
	}
	return int32(advance)
}
