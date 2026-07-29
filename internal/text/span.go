package text

import (
	"fmt"
	"unicode/utf8"

	"github.com/rdon-key/modgadget-fonts/font"
	"github.com/rdon-key/modgadget/internal/display"
)

// Span is a consecutive string sharing one font and pair of drawing colors.
type Span struct {
	Face       *font.Font
	Value      string
	Foreground display.Color565
	Background display.Color565
}

// MeasureSpans measures a single baseline-aligned line of spans.
// Span indexes in validation errors are zero-based.
func MeasureSpans(spans []Span) (Measurement, error) {
	var measurement Measurement
	for index := range spans {
		span := &spans[index]
		if span.Face == nil {
			return measurement, fmt.Errorf("text: span %d: font is nil", index)
		}
		if !utf8.ValidString(span.Value) {
			return measurement, fmt.Errorf("text: span %d: value is not valid UTF-8", index)
		}
		var err error
		measurement, err = measureValue(measurement, span.Face, span.Value)
		if err != nil {
			return measurement, err
		}
	}
	return measurement, nil
}

// DrawSpans draws a single baseline-aligned line of spans and returns the
// final pen X coordinate.
func DrawSpans(backend display.Backend, spans []Span, penX, baselineY int16, scratch []byte) (int16, error) {
	if len(spans) == 0 {
		return penX, nil
	}
	if backend == nil {
		return penX, fmt.Errorf("text: backend is nil")
	}
	currentX := penX
	for index := range spans {
		span := &spans[index]
		if span.Face == nil {
			return currentX, fmt.Errorf("text: span %d: font is nil", index)
		}
		if !utf8.ValidString(span.Value) {
			return currentX, fmt.Errorf("text: span %d: value is not valid UTF-8", index)
		}
		var err error
		currentX, err = drawValue(backend, span.Face, currentX, baselineY, span.Value, span.Foreground, span.Background, scratch)
		if err != nil {
			return currentX, err
		}
	}
	return currentX, nil
}
