// Package text draws bitmap-font text on display backends.
package text

import (
	"fmt"
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
	return drawLegacyValue(backend, face, penX, baselineY, value, foreground, background, scratch)
}

func drawLegacyValue(backend display.Backend, face *font.Font, penX, baselineY int16, value string, foreground, background display.Color565, scratch []byte) (int16, error) {
	currentX := penX
	view := legacyFont{face: face}
	for _, r := range value {
		position, err := positionGlyph(view, r, currentX, baselineY)
		if err != nil {
			return currentX, err
		}
		glyph := position.glyph

		width, height := int(glyph.Width), int(glyph.Height)
		if width != 0 && height != 0 {
			stride, ok := checkedProduct(width, 2)
			if !ok {
				return currentX, fmt.Errorf("text: glyph U+%04X RGB565 row stride overflows int", r)
			}
			required, ok := checkedProduct(stride, height)
			if !ok {
				return currentX, fmt.Errorf("text: glyph U+%04X scratch size overflows int", r)
			}
			if required > len(scratch) {
				return currentX, fmt.Errorf("text: scratch too small for glyph U+%04X: have %d bytes, need %d", r, len(scratch), required)
			}

			rowBytes := (width + 7) / 8
			out := scratch[:required]
			fgHigh, fgLow := byte(foreground>>8), byte(foreground)
			bgHigh, bgLow := byte(background>>8), byte(background)
			for row := 0; row < height; row++ {
				bitmapRow, bitmapOK := checkedProduct(row, rowBytes)
				outputRow, outputOK := checkedProduct(row, stride)
				if !bitmapOK || !outputOK {
					return currentX, fmt.Errorf("text: glyph U+%04X row index overflows int", r)
				}
				for column := 0; column < width; column++ {
					bitmapIndex := bitmapRow + column/8
					output := outputRow + column*2
					if bitmapIndex < 0 || bitmapIndex >= len(glyph.Bitmap) || output < 0 || output+1 >= len(out) {
						return currentX, fmt.Errorf("text: glyph U+%04X bitmap or scratch index is invalid", r)
					}
					if glyph.Bitmap[bitmapIndex]&(byte(0x80)>>uint(column&7)) != 0 {
						out[output], out[output+1] = fgHigh, fgLow
					} else {
						out[output], out[output+1] = bgHigh, bgLow
					}
				}
			}
			rect := display.Rect{X: position.x, Y: position.y, Width: glyph.Width, Height: glyph.Height}
			if err := display.BlitRGB565(backend, rect, out, stride); err != nil {
				return currentX, fmt.Errorf("text: draw glyph U+%04X: %w", r, err)
			}
		}
		currentX = position.nextX
	}
	return currentX, nil
}

// drawFontValue streams RGB565 pixels directly from immutable 1-bit bitmap
// strings. It deliberately does not expand a glyph into a scratch buffer.
func drawFontValue(backend display.Backend, face Font, penX, baselineY int16, value string, foreground, background display.Color565, scratch []byte) (int16, error) {
	currentX := penX
	if len(scratch) < 4 && value != "" {
		return currentX, fmt.Errorf("text: scratch too small for direct glyph pixels: have %d bytes, need 4", len(scratch))
	}
	fg := scratch[:2]
	bg := scratch[2:4]
	fg[0], fg[1] = byte(foreground>>8), byte(foreground)
	bg[0], bg[1] = byte(background>>8), byte(background)
	for _, r := range value {
		position, err := positionGlyph(face, r, currentX, baselineY)
		if err != nil {
			return currentX, err
		}
		glyph := position.glyph
		width, height := int(glyph.Width), int(glyph.Height)
		if width != 0 && height != 0 {
			rect := display.Rect{X: position.x, Y: position.y, Width: glyph.Width, Height: glyph.Height}
			if err := backend.BeginRect(rect.X, rect.Y, rect.Width, rect.Height); err != nil {
				return currentX, fmt.Errorf("text: draw glyph U+%04X: %w", r, err)
			}
			rowBytes := (width + 7) / 8
			for y := 0; y < height; y++ {
				row := y * rowBytes
				for x := 0; x < width; x++ {
					pixel := bg
					if glyph.Bitmap[row+x/8]&(byte(0x80)>>uint(x&7)) != 0 {
						pixel = fg
					}
					if err := backend.WritePixels(pixel); err != nil {
						return currentX, fmt.Errorf("text: draw glyph U+%04X: %w", r, err)
					}
				}
			}
			if err := backend.EndRect(); err != nil {
				return currentX, fmt.Errorf("text: draw glyph U+%04X: %w", r, err)
			}
		}
		currentX = position.nextX
	}
	return currentX, nil
}

func checkedProduct(left, right int) (int, bool) {
	if left < 0 || right < 0 || left != 0 && right > int(^uint(0)>>1)/left {
		return 0, false
	}
	return left * right, true
}
