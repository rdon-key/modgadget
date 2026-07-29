package text

import (
	"fmt"
	"math"
)

// LineMeasurement combines horizontal ink measurement with the typographic
// metrics of one baseline-aligned line.
type LineMeasurement struct {
	Measurement

	Ascent   int16
	Descent  int16
	LineGap  int16
	AdvanceY int16
}

// MeasureLine measures one baseline-aligned line of spans. After a span's font
// and UTF-8 are validated, its line metrics are included before its glyphs are
// measured. A glyph error therefore returns those metrics and the horizontal
// result through the last successfully processed glyph.
func MeasureLine(spans []Span) (LineMeasurement, error) {
	var line LineMeasurement
	hasMetrics := false
	for index := range spans {
		span := &spans[index]
		if err := validateSpan(index, span); err != nil {
			return line, err
		}

		metrics := span.Face.Metrics()
		candidate := line
		if !hasMetrics {
			candidate.Ascent = metrics.Ascent
			candidate.Descent = metrics.Descent
			candidate.LineGap = metrics.LineGap
		} else {
			if metrics.Ascent > candidate.Ascent {
				candidate.Ascent = metrics.Ascent
			}
			if metrics.Descent > candidate.Descent {
				candidate.Descent = metrics.Descent
			}
			if metrics.LineGap > candidate.LineGap {
				candidate.LineGap = metrics.LineGap
			}
		}
		advanceY := int32(candidate.Ascent) + int32(candidate.Descent) + int32(candidate.LineGap)
		if advanceY < math.MinInt16 || advanceY > math.MaxInt16 {
			return line, fmt.Errorf("text: span %d: line advance is outside int16", index)
		}
		candidate.AdvanceY = int16(advanceY)
		line = candidate
		hasMetrics = true

		var err error
		line.Measurement, err = measureValue(line.Measurement, span.Face, span.Value)
		if err != nil {
			return line, err
		}
	}
	return line, nil
}
