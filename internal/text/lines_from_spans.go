package text

type indexedSpan struct {
	span       Span
	inputIndex int
}

type indexedLine struct {
	spans []indexedSpan
}

// LinesFromSpans splits spans at LF into explicit lines. Empty segments are
// retained so their font can continue to contribute line metrics.
func LinesFromSpans(spans []Span) ([]Line, error) {
	indexedLines, err := splitSpans(spans)
	if indexedLines == nil {
		return nil, err
	}
	lines := make([]Line, len(indexedLines))
	for lineIndex := range indexedLines {
		lines[lineIndex].Spans = make([]Span, len(indexedLines[lineIndex].spans))
		for spanIndex := range indexedLines[lineIndex].spans {
			lines[lineIndex].Spans[spanIndex] = indexedLines[lineIndex].spans[spanIndex].span
		}
	}
	return lines, err
}

func splitSpans(spans []Span) ([]indexedLine, error) {
	if spans == nil {
		return nil, nil
	}
	lines := make([]indexedLine, 0, 1)
	for spanIndex := range spans {
		span := &spans[spanIndex]
		if err := validateSpan(spanIndex, span); err != nil {
			return lines, err
		}
		if len(lines) == 0 {
			lines = append(lines, indexedLine{})
		}

		start := 0
		for index := 0; index < len(span.Value); index++ {
			if span.Value[index] != '\n' {
				continue
			}

			segment := *span
			segment.Value = span.Value[start:index]
			lineIndex := len(lines) - 1
			lines[lineIndex].spans = append(lines[lineIndex].spans, indexedSpan{span: segment, inputIndex: spanIndex})
			lines = append(lines, indexedLine{})
			start = index + 1
		}

		segment := *span
		segment.Value = span.Value[start:]
		lineIndex := len(lines) - 1
		lines[lineIndex].spans = append(lines[lineIndex].spans, indexedSpan{span: segment, inputIndex: spanIndex})
	}
	return lines, nil
}
