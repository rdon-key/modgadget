package text

import "github.com/rdon-key/modgadget/internal/display"

// TextLayout is a prepared set of explicit lines and its block measurement.
type TextLayout struct {
	lines       []Line
	measurement BlockMeasurement
}

// NewTextLayout splits and measures spans for repeated drawing.
func NewTextLayout(spans []Span) (TextLayout, error) {
	if len(spans) == 0 {
		return TextLayout{}, nil
	}
	lines, err := LinesFromSpans(spans)
	if err != nil {
		return TextLayout{}, err
	}
	measurement, err := MeasureLines(lines)
	if err != nil {
		return TextLayout{}, err
	}
	return TextLayout{lines: lines, measurement: measurement}, nil
}

// Measurement returns the measurement saved when layout was constructed.
func (layout *TextLayout) Measurement() BlockMeasurement {
	if layout == nil {
		return BlockMeasurement{}
	}
	return layout.measurement
}

// LineCount returns the number of prepared lines.
func (layout *TextLayout) LineCount() int {
	if layout == nil {
		return 0
	}
	return len(layout.lines)
}

// Draw draws the prepared lines without splitting or measuring them again.
func (layout *TextLayout) Draw(backend display.Backend, penX, firstBaselineY int16, scratch []byte) (int16, error) {
	if layout == nil || len(layout.lines) == 0 {
		return firstBaselineY, nil
	}
	return DrawLines(backend, layout.lines, penX, firstBaselineY, scratch)
}
