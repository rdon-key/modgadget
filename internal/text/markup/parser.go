// Package markup converts a small, allocation-conscious markup language into
// text spans.
package markup

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/rdon-key/modgadget/internal/text"
)

const maximumNestingDepth = 16

type tagKind uint8

const (
	tagStyle tagKind = iota + 1
	tagBold
)

// Parser holds the styles used while parsing markup.
type Parser struct {
	Styles text.StyleSet
}

// SyntaxError reports a markup error at a byte offset in the input string.
type SyntaxError struct {
	Offset  int
	Message string
}

// Error implements error.
func (err *SyntaxError) Error() string {
	return fmt.Sprintf("markup: byte %d: %s", err.Offset, err.Message)
}

// Parse parses value and allocates a result slice of the exact required size.
func (parser Parser) Parse(value string) ([]text.Span, error) {
	_, count, err := parser.parse(nil, value, true)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return []text.Span{}, nil
	}
	result := make([]text.Span, 0, count)
	result, _, err = parser.parse(result, value, false)
	return result, err
}

// ParseInto parses value into destination without growing its backing array.
// On error it returns an empty slice.
func (parser Parser) ParseInto(destination []text.Span, value string) ([]text.Span, error) {
	result, _, err := parser.parse(destination, value, false)
	if err != nil {
		return destination[:0], err
	}
	return result, nil
}

func (parser Parser) parse(destination []text.Span, value string, countOnly bool) ([]text.Span, int, error) {
	if !utf8.ValidString(value) {
		return destination, 0, syntaxError(0, "input is not valid UTF-8")
	}
	if parser.Styles.Default.Font == nil {
		return destination, 0, syntaxError(0, "default style font is nil")
	}

	current := parser.Styles.Default
	var stack [maximumNestingDepth]text.Style
	var tags [maximumNestingDepth]tagKind
	depth := 0
	emitted := 0
	startLength := len(destination)

	emit := func(value string) error {
		if value == "" {
			return nil
		}
		emitted++
		if countOnly {
			return nil
		}
		if len(destination) == cap(destination) {
			return fmt.Errorf("markup: span buffer too small: have %d, need at least %d", cap(destination)-startLength, emitted)
		}
		destination = append(destination, text.Span{
			Font: current.Font, Value: value,
			Foreground: current.Foreground, Background: current.Background, Bold: current.Bold,
		})
		return nil
	}

	for offset := 0; offset < len(value); {
		if value[offset] != '<' {
			end := strings.IndexByte(value[offset:], '<')
			if end < 0 {
				end = len(value)
			} else {
				end += offset
			}
			if err := emit(value[offset:end]); err != nil {
				return destination, emitted, err
			}
			offset = end
			continue
		}
		if offset+1 < len(value) && value[offset+1] == '<' {
			if err := emit(value[offset : offset+1]); err != nil {
				return destination, emitted, err
			}
			offset += 2
			continue
		}

		endRelative := strings.IndexByte(value[offset+1:], '>')
		if endRelative < 0 {
			return destination, emitted, syntaxError(offset, "unterminated tag")
		}
		end := offset + 1 + endRelative
		tag := value[offset : end+1]

		switch tag {
		case "</style>":
			if depth == 0 || tags[depth-1] != tagStyle {
				return destination, emitted, syntaxError(offset, "unexpected closing style tag")
			}
			depth--
			current = stack[depth]
		case "<b>":
			if depth == len(stack) {
				return destination, emitted, syntaxError(offset, "nesting is too deep")
			}
			stack[depth] = current
			tags[depth] = tagBold
			depth++
			current.Bold = true
		case "</b>":
			if depth == 0 || tags[depth-1] != tagBold {
				return destination, emitted, syntaxError(offset, "unexpected closing bold tag")
			}
			depth--
			current = stack[depth]
		case "<br>", "<br/>":
			if err := emit("\n"); err != nil {
				return destination, emitted, err
			}
		default:
			if !strings.HasPrefix(tag, "<style=") {
				return destination, emitted, syntaxError(offset, "unknown tag")
			}
			name := tag[len("<style=") : len(tag)-1]
			if !validStyleName(name) {
				return destination, emitted, syntaxError(offset, "invalid style name")
			}
			selected, ok := parser.Styles.Lookup(name)
			if !ok {
				return destination, emitted, syntaxError(offset, "unknown style")
			}
			if selected.Font == nil {
				return destination, emitted, syntaxError(offset, "selected style font is nil")
			}
			if depth == len(stack) {
				return destination, emitted, syntaxError(offset, "nesting is too deep")
			}
			stack[depth] = current
			tags[depth] = tagStyle
			depth++
			if hasActiveBoldTag(tags[:depth]) {
				selected.Bold = true
			}
			current = selected
		}
		offset = end + 1
	}

	if depth != 0 {
		return destination, emitted, syntaxError(len(value), "unclosed tag")
	}
	return destination, emitted, nil
}

func hasActiveBoldTag(tags []tagKind) bool {
	for _, tag := range tags {
		if tag == tagBold {
			return true
		}
	}
	return false
}

func validStyleName(name string) bool {
	if len(name) == 0 || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for index := 1; index < len(name); index++ {
		value := name[index]
		if (value < 'a' || value > 'z') && (value < '0' || value > '9') && value != '-' {
			return false
		}
	}
	return true
}

func syntaxError(offset int, message string) *SyntaxError {
	return &SyntaxError{Offset: offset, Message: message}
}
