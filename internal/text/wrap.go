package text

import "fmt"

// WrapSpans greedily wraps spans by glyph advance after applying explicit LF
// line breaks. On an error, completed wrapped lines are returned. If splitting
// failed, the last explicit line is still under construction: wraps completed
// within it are returned, but its final unfinished line is not.
func WrapSpans(spans []Span, maxAdvanceX int16) ([]Line, error) {
	if maxAdvanceX <= 0 {
		return nil, fmt.Errorf("text: wrap maximum advance X must be positive")
	}
	explicitLines, splitErr := splitSpans(spans)
	if explicitLines == nil {
		return nil, splitErr
	}
	wrapped := make([]Line, 0, len(explicitLines))
	for lineIndex := range explicitLines {
		includeFinal := splitErr == nil || lineIndex < len(explicitLines)-1
		completed, err := wrapExplicitLine(explicitLines[lineIndex], lineIndex, maxAdvanceX, includeFinal)
		wrapped = append(wrapped, completed...)
		if err != nil {
			return wrapped, err
		}
	}
	if splitErr != nil {
		lineIndex := len(explicitLines) - 1
		if lineIndex < 0 {
			lineIndex = 0
		}
		return wrapped, fmt.Errorf("text: explicit line %d: %w", lineIndex, splitErr)
	}
	return wrapped, nil
}

func wrapExplicitLine(explicit indexedLine, lineIndex int, maxAdvanceX int16, includeFinal bool) ([]Line, error) {
	var wrapped []Line
	current := Line{}
	penX := int16(0)
	hasGlyph := false
	oversized := false
	for spanIndex := range explicit.spans {
		indexed := &explicit.spans[spanIndex]
		span := &indexed.span
		if span.Value == "" {
			current.Spans = append(current.Spans, *span)
			continue
		}

		segmentStart := 0
		for runeStart, r := range span.Value {
			if oversized {
				if segmentStart < runeStart {
					current.Spans = append(current.Spans, spanSubstring(span, segmentStart, runeStart))
				}
				wrapped = append(wrapped, current)
				current = Line{}
				penX = 0
				hasGlyph = false
				oversized = false
				segmentStart = runeStart
			}
			position, err := positionGlyph(spanFont(span), r, penX, 0)
			if err != nil {
				return wrapped, fmt.Errorf("text: explicit line %d span %d: %w", lineIndex, indexed.inputIndex, err)
			}
			if hasGlyph && position.nextX > maxAdvanceX {
				if segmentStart < runeStart {
					current.Spans = append(current.Spans, spanSubstring(span, segmentStart, runeStart))
				}
				wrapped = append(wrapped, current)
				current = Line{}
				penX = 0
				hasGlyph = false
				oversized = false
				segmentStart = runeStart
				position, err = positionGlyph(spanFont(span), r, penX, 0)
				if err != nil {
					return wrapped, fmt.Errorf("text: explicit line %d span %d: %w", lineIndex, indexed.inputIndex, err)
				}
			}
			penX = position.nextX
			oversized = !hasGlyph && position.nextX > maxAdvanceX
			hasGlyph = true
		}
		current.Spans = append(current.Spans, spanSubstring(span, segmentStart, len(span.Value)))
	}
	if includeFinal {
		wrapped = append(wrapped, current)
	}
	return wrapped, nil
}

// NewWrappedTextLayout prepares a measured layout using greedy glyph wrapping.
func NewWrappedTextLayout(spans []Span, maxAdvanceX int16) (TextLayout, error) {
	lines, err := WrapSpans(spans, maxAdvanceX)
	if err != nil {
		return TextLayout{}, err
	}
	if len(spans) == 0 {
		return TextLayout{}, nil
	}
	measurement, err := MeasureLines(lines)
	if err != nil {
		return TextLayout{}, err
	}
	return TextLayout{lines: lines, measurement: measurement}, nil
}

func spanSubstring(span *Span, start, end int) Span {
	segment := *span
	segment.Value = span.Value[start:end]
	return segment
}
