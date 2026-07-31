// Package text draws bitmap-font text on display backends.
package text

import (
	"fmt"
	"unicode/utf8"

	"github.com/rdon-key/modgadget/internal/display"
)

// DrawString draws value relative to baselineY and returns the final pen X.
// scratch must hold one RGB565 row of the widest glyph.
func DrawString(
	backend display.Backend,
	font Font,
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
	if font == nil {
		return penX, fmt.Errorf("text: font is nil")
	}
	if !utf8.ValidString(value) {
		return penX, fmt.Errorf("text: value is not valid UTF-8")
	}
	return drawFontValue(backend, font, penX, baselineY, value, foreground, background, scratch)
}

// drawFontValue streams RGB565 rows from immutable 1-bit bitmap strings. It
// expands only one row at a time into caller-provided scratch space.
func drawFontValue(backend display.Backend, face Font, penX, baselineY int16, value string, foreground, background display.Color565, scratch []byte) (int16, error) {
	currentX := penX
	for _, r := range value {
		position, err := positionGlyph(face, r, currentX, baselineY)
		if err != nil {
			return currentX, err
		}
		glyph := position.glyph
		width, height := int(glyph.Width), int(glyph.Height)
		if width != 0 && height != 0 {
			rowPixelBytes := width * 2
			if len(scratch) < rowPixelBytes {
				return currentX, fmt.Errorf("text: scratch too small for glyph U+%04X: have %d bytes, need %d", r, len(scratch), rowPixelBytes)
			}
			rect := display.Rect{X: position.x, Y: position.y, Width: glyph.Width, Height: glyph.Height}
			if err := backend.BeginRect(rect.X, rect.Y, rect.Width, rect.Height); err != nil {
				return currentX, fmt.Errorf("text: draw glyph U+%04X: %w", r, err)
			}
			rowBytes := (width + 7) / 8
			for y := 0; y < height; y++ {
				row := y * rowBytes
				for x := 0; x < width; x++ {
					color := background
					if glyph.Bitmap[row+x/8]&(byte(0x80)>>uint(x&7)) != 0 {
						color = foreground
					}
					offset := x * 2
					scratch[offset], scratch[offset+1] = byte(color>>8), byte(color)
				}
				if err := backend.WritePixels(scratch[:rowPixelBytes]); err != nil {
					return currentX, fmt.Errorf("text: draw glyph U+%04X: %w", r, err)
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
