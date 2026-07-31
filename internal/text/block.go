package text

import (
	"errors"
	"fmt"
	"math"

	"github.com/rdon-key/modgadget/internal/display"
)

// Line is one explicit baseline-aligned sequence of spans.
type Line struct {
	Spans []Span
}

// BlockMeasurement describes the ink and advances of explicit lines relative
// to the first baseline at Y 0.
type BlockMeasurement struct {
	Bounds Bounds
	HasInk bool

	MaxAdvanceX int16
	AdvanceY    int16
}

// MeasureLines measures explicit lines, placing the first baseline at Y 0.
func MeasureLines(lines []Line) (BlockMeasurement, error) {
	var block BlockMeasurement
	baselineY := int16(0)
	hasAdvance := false
	for index := range lines {
		line, processed, err := measureLine(lines[index].Spans)
		if err != nil {
			if !errors.Is(err, errLineAdvanceOverflow) && processed {
				if integrateErr := integrateLine(&block, line.Measurement, baselineY, &hasAdvance); integrateErr != nil {
					return block, fmt.Errorf("text: line %d: %w", index, integrateErr)
				}
			}
			block.AdvanceY = baselineY
			return block, fmt.Errorf("text: line %d: %w", index, err)
		}
		if err := integrateLine(&block, line.Measurement, baselineY, &hasAdvance); err != nil {
			return block, fmt.Errorf("text: line %d: %w", index, err)
		}
		nextBaseline := int32(baselineY) + int32(line.AdvanceY)
		if nextBaseline < math.MinInt16 || nextBaseline > math.MaxInt16 {
			block.AdvanceY = baselineY
			return block, fmt.Errorf("text: line %d: baseline advance is outside int16", index)
		}
		baselineY = int16(nextBaseline)
		block.AdvanceY = baselineY
	}
	return block, nil
}

// DrawLines draws explicit lines and returns the next baseline Y.
func DrawLines(backend display.Backend, lines []Line, penX, firstBaselineY int16, scratch []byte) (int16, error) {
	baselineY := firstBaselineY
	for index := range lines {
		var accumulator lineMetricsAccumulator
		currentPenX := penX
		for spanIndex := range lines[index].Spans {
			span := &lines[index].Spans[spanIndex]
			if err := validateSpan(spanIndex, span); err != nil {
				return baselineY, fmt.Errorf("text: line %d: %w", index, err)
			}
			if err := accumulator.add(span.Font.Metrics()); err != nil {
				return baselineY, fmt.Errorf("text: line %d: text: span %d: %w", index, spanIndex, err)
			}
			var err error
			currentPenX, err = drawFontValue(backend, span.Font, currentPenX, baselineY, span.Value, span.Foreground, span.Background, scratch)
			if err != nil {
				return baselineY, fmt.Errorf("text: line %d: %w", index, err)
			}
		}
		nextBaseline := int32(baselineY) + int32(accumulator.advanceY)
		if nextBaseline < math.MinInt16 || nextBaseline > math.MaxInt16 {
			return baselineY, fmt.Errorf("text: line %d: baseline advance is outside int16", index)
		}
		baselineY = int16(nextBaseline)
	}
	return baselineY, nil
}

func integrateLine(block *BlockMeasurement, measurement Measurement, baselineY int16, hasAdvance *bool) error {
	if !*hasAdvance {
		block.MaxAdvanceX = measurement.Advance
		*hasAdvance = true
	} else if measurement.Advance > block.MaxAdvanceX {
		block.MaxAdvanceX = measurement.Advance
	}
	if !measurement.HasInk {
		return nil
	}
	minY := int32(baselineY) + int32(measurement.Bounds.MinY)
	maxY := int32(baselineY) + int32(measurement.Bounds.MaxY)
	if minY < math.MinInt16 || minY > math.MaxInt16 || maxY < math.MinInt16 || maxY > math.MaxInt16 {
		return fmt.Errorf("ink bounds are outside int16")
	}
	bounds := Bounds{MinX: measurement.Bounds.MinX, MinY: int16(minY), MaxX: measurement.Bounds.MaxX, MaxY: int16(maxY)}
	if !block.HasInk {
		block.Bounds = bounds
		block.HasInk = true
	} else {
		block.Bounds = unionBounds(block.Bounds, bounds)
	}
	return nil
}
