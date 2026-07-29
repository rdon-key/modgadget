package text

// LinesFromSpans splits spans at LF into explicit lines. Empty segments are
// retained so their font can continue to contribute line metrics.
func LinesFromSpans(spans []Span) ([]Line, error) {
	if spans == nil {
		return nil, nil
	}
	lines := make([]Line, 0, 1)
	for spanIndex := range spans {
		span := &spans[spanIndex]
		if err := validateSpan(spanIndex, span); err != nil {
			return lines, err
		}
		if len(lines) == 0 {
			lines = append(lines, Line{})
		}

		start := 0
		for index := 0; index < len(span.Value); index++ {
			if span.Value[index] != '\n' {
				continue
			}

			segment := *span
			segment.Value = span.Value[start:index]
			lineIndex := len(lines) - 1
			lines[lineIndex].Spans = append(lines[lineIndex].Spans, segment)
			lines = append(lines, Line{})
			start = index + 1
		}

		segment := *span
		segment.Value = span.Value[start:]
		lineIndex := len(lines) - 1
		lines[lineIndex].Spans = append(lines[lineIndex].Spans, segment)
	}
	return lines, nil
}
