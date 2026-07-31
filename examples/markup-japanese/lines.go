package main

import (
	"fmt"

	"github.com/rdon-key/modgadget/internal/text"
)

func splitLines(destination []text.Line, spans []text.Span) ([]text.Line, error) {
	result := destination
	startLength := len(destination)
	lineStart := 0
	for index := 0; index <= len(spans); index++ {
		if index != len(spans) && spans[index].Value != "\n" {
			continue
		}
		if len(result) == cap(result) {
			return destination[:0], fmt.Errorf("markup example: line buffer too small: have %d, need at least %d", cap(destination)-startLength, len(result)-startLength+1)
		}
		result = result[:len(result)+1]
		result[len(result)-1] = text.Line{Spans: spans[lineStart:index]}
		lineStart = index + 1
	}
	return result, nil
}
