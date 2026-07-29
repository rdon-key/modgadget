// Package text draws bitmap-font text on display backends.
package text

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/rdon-key/modgadget-fonts/font"
	"github.com/rdon-key/modgadget/internal/display"
)

// DrawString draws value relative to baselineY and returns the final pen X.
// scratch must hold the largest glyph expanded to RGB565.
func DrawString(
	backend display.Backend,
	face *font.Font,
	penX int16,
	baselineY int16,
	value string,
	foreground display.Color565,
	background display.Color565,
	scratch []byte,
) (int16, error) {
	if backend == nil {
		return penX, fmt.Errorf("text: backend is nil")
	}
	if face == nil {
		return penX, fmt.Errorf("text: font is nil")
	}
	if !utf8.ValidString(value) {
		return penX, fmt.Errorf("text: value is not valid UTF-8")
	}

	currentX := int32(penX)
	for _, r := range value {
		glyph, ok := face.Lookup(r)
		if !ok {
			return int16(currentX), fmt.Errorf("text: glyph U+%04X is missing or invalid", r)
		}
		drawX := currentX + int32(glyph.BearingX)
		if drawX < math.MinInt16 || drawX > math.MaxInt16 {
			return int16(currentX), fmt.Errorf("text: glyph U+%04X X coordinate is outside int16", r)
		}
		drawY := int32(baselineY) - int32(glyph.BearingY)
		if drawY < math.MinInt16 || drawY > math.MaxInt16 {
			return int16(currentX), fmt.Errorf("text: glyph U+%04X Y coordinate is outside int16", r)
		}
		nextX := currentX + int32(glyph.AdvanceX)
		if nextX < math.MinInt16 || nextX > math.MaxInt16 {
			return int16(currentX), fmt.Errorf("text: glyph U+%04X advance is outside int16", r)
		}

		width, height := int(glyph.Width), int(glyph.Height)
		if width != 0 && height != 0 {
			stride, ok := checkedProduct(width, 2)
			if !ok {
				return int16(currentX), fmt.Errorf("text: glyph U+%04X RGB565 row stride overflows int", r)
			}
			required, ok := checkedProduct(stride, height)
			if !ok {
				return int16(currentX), fmt.Errorf("text: glyph U+%04X scratch size overflows int", r)
			}
			if required > len(scratch) {
				return int16(currentX), fmt.Errorf("text: scratch too small for glyph U+%04X: have %d bytes, need %d", r, len(scratch), required)
			}

			rowBytes := (width + 7) / 8
			out := scratch[:required]
			fgHigh, fgLow := byte(foreground>>8), byte(foreground)
			bgHigh, bgLow := byte(background>>8), byte(background)
			for row := 0; row < height; row++ {
				bitmapRow, bitmapOK := checkedProduct(row, rowBytes)
				outputRow, outputOK := checkedProduct(row, stride)
				if !bitmapOK || !outputOK {
					return int16(currentX), fmt.Errorf("text: glyph U+%04X row index overflows int", r)
				}
				for column := 0; column < width; column++ {
					bitmapIndex := bitmapRow + column/8
					output := outputRow + column*2
					if bitmapIndex < 0 || bitmapIndex >= len(glyph.Bitmap) || output < 0 || output+1 >= len(out) {
						return int16(currentX), fmt.Errorf("text: glyph U+%04X bitmap or scratch index is invalid", r)
					}
					if glyph.Bitmap[bitmapIndex]&(byte(0x80)>>uint(column&7)) != 0 {
						out[output], out[output+1] = fgHigh, fgLow
					} else {
						out[output], out[output+1] = bgHigh, bgLow
					}
				}
			}
			rect := display.Rect{X: int16(drawX), Y: int16(drawY), Width: glyph.Width, Height: glyph.Height}
			if err := display.BlitRGB565(backend, rect, out, stride); err != nil {
				return int16(currentX), fmt.Errorf("text: draw glyph U+%04X: %w", r, err)
			}
		}
		currentX = nextX
	}
	return int16(currentX), nil
}

func checkedProduct(left, right int) (int, bool) {
	if left < 0 || right < 0 || left != 0 && right > int(^uint(0)>>1)/left {
		return 0, false
	}
	return left * right, true
}
