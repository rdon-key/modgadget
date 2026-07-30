package text

import (
	"errors"
	"fmt"
	"math"
)

var errLineAdvanceOverflow = errors.New("line advance is outside int16")

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
	line, _, err := measureLine(spans)
	return line, err
}

func measureLine(spans []Span) (LineMeasurement, bool, error) {
	var line LineMeasurement
	var accumulator lineMetricsAccumulator
	processed := false
	for index := range spans {
		span := &spans[index]
		if err := validateSpan(index, span); err != nil {
			return line, processed, err
		}
		if err := accumulator.add(spanFont(span).Metrics()); err != nil {
			return line, processed, fmt.Errorf("text: span %d: %w", index, err)
		}
		line.Ascent, line.Descent = accumulator.ascent, accumulator.descent
		line.LineGap, line.AdvanceY = accumulator.lineGap, accumulator.advanceY

		var spanProcessed bool
		var err error
		line.Measurement, spanProcessed, err = measureValueProgress(line.Measurement, spanFont(span), span.Value)
		processed = processed || spanProcessed
		if err != nil {
			return line, processed, err
		}
	}
	return line, processed, nil
}

type lineMetricsAccumulator struct {
	ascent   int16
	descent  int16
	lineGap  int16
	advanceY int16
	hasValue bool
}

func (accumulator *lineMetricsAccumulator) add(metrics FontMetrics) error {
	candidate := *accumulator
	if !candidate.hasValue {
		candidate.ascent = metrics.Ascent
		candidate.descent = metrics.Descent
		candidate.lineGap = metrics.LineGap
		candidate.hasValue = true
	} else {
		if metrics.Ascent > candidate.ascent {
			candidate.ascent = metrics.Ascent
		}
		if metrics.Descent > candidate.descent {
			candidate.descent = metrics.Descent
		}
		if metrics.LineGap > candidate.lineGap {
			candidate.lineGap = metrics.LineGap
		}
	}
	advanceY := int32(candidate.ascent) + int32(candidate.descent) + int32(candidate.lineGap)
	if advanceY < math.MinInt16 || advanceY > math.MaxInt16 {
		return errLineAdvanceOverflow
	}
	candidate.advanceY = int16(advanceY)
	*accumulator = candidate
	return nil
}
