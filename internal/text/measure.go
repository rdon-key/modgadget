package text

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/rdon-key/modgadget-fonts/font"
)

// Bounds is a half-open ink rectangle relative to pen X 0 and baseline Y 0.
type Bounds struct {
	MinX int16
	MinY int16
	MaxX int16
	MaxY int16
}

// Measurement contains the final pen advance and the union of glyph ink.
type Measurement struct {
	Advance int16
	Bounds  Bounds
	HasInk  bool
}

type glyphPosition struct {
	glyph font.Glyph
	x     int16
	y     int16
	nextX int16
}

// MeasureString measures one line without drawing it.
func MeasureString(face *font.Font, value string) (Measurement, error) {
	var measurement Measurement
	if face == nil {
		return measurement, fmt.Errorf("text: font is nil")
	}
	if !utf8.ValidString(value) {
		return measurement, fmt.Errorf("text: value is not valid UTF-8")
	}
	return measureValue(measurement, face, value)
}

func measureValue(measurement Measurement, face *font.Font, value string) (Measurement, error) {
	for _, r := range value {
		position, err := positionGlyph(face, r, measurement.Advance, 0)
		if err != nil {
			return measurement, err
		}
		glyph := position.glyph
		if glyph.Width != 0 && glyph.Height != 0 {
			maxX := int32(position.x) + int32(glyph.Width)
			maxY := int32(position.y) + int32(glyph.Height)
			if maxX < math.MinInt16 || maxX > math.MaxInt16 {
				return measurement, fmt.Errorf("text: glyph U+%04X maximum X coordinate is outside int16", r)
			}
			if maxY < math.MinInt16 || maxY > math.MaxInt16 {
				return measurement, fmt.Errorf("text: glyph U+%04X maximum Y coordinate is outside int16", r)
			}
			glyphBounds := Bounds{MinX: position.x, MinY: position.y, MaxX: int16(maxX), MaxY: int16(maxY)}
			if !measurement.HasInk {
				measurement.Bounds = glyphBounds
				measurement.HasInk = true
			} else {
				measurement.Bounds = unionBounds(measurement.Bounds, glyphBounds)
			}
		}
		measurement.Advance = position.nextX
	}
	return measurement, nil
}

func positionGlyph(face *font.Font, r rune, penX, baselineY int16) (glyphPosition, error) {
	glyph, ok := face.Lookup(r)
	if !ok {
		return glyphPosition{}, fmt.Errorf("text: glyph U+%04X is missing or invalid", r)
	}
	x := int32(penX) + int32(glyph.BearingX)
	if x < math.MinInt16 || x > math.MaxInt16 {
		return glyphPosition{}, fmt.Errorf("text: glyph U+%04X X coordinate is outside int16", r)
	}
	y := int32(baselineY) - int32(glyph.BearingY)
	if y < math.MinInt16 || y > math.MaxInt16 {
		return glyphPosition{}, fmt.Errorf("text: glyph U+%04X Y coordinate is outside int16", r)
	}
	nextX := int32(penX) + int32(glyph.AdvanceX)
	if nextX < math.MinInt16 || nextX > math.MaxInt16 {
		return glyphPosition{}, fmt.Errorf("text: glyph U+%04X advance is outside int16", r)
	}
	return glyphPosition{glyph: glyph, x: int16(x), y: int16(y), nextX: int16(nextX)}, nil
}

func unionBounds(left, right Bounds) Bounds {
	if right.MinX < left.MinX {
		left.MinX = right.MinX
	}
	if right.MinY < left.MinY {
		left.MinY = right.MinY
	}
	if right.MaxX > left.MaxX {
		left.MaxX = right.MaxX
	}
	if right.MaxY > left.MaxY {
		left.MaxY = right.MaxY
	}
	return left
}
