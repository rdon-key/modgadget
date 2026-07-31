// Package markup converts a small, allocation-conscious markup language into
// text spans.
package markup

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/text"
)

const maximumNestingDepth = 16

// Fonts contains the fonts selected by size tags.
type Fonts struct {
	Size12 text.Font
	Size16 text.Font
	Size24 text.Font
}

// Parser holds the fonts and default colors used while parsing markup.
type Parser struct {
	Fonts      Fonts
	Foreground display.Color565
	Background display.Color565
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

type tagKind uint8

const (
	tagSize tagKind = iota + 1
	tagForeground
	tagBackground
)

type style struct {
	font       text.Font
	foreground display.Color565
	background display.Color565
}

type styleFrame struct {
	style style
	kind  tagKind
}

func (parser Parser) parse(destination []text.Span, value string, countOnly bool) ([]text.Span, int, error) {
	if !utf8.ValidString(value) {
		return destination, 0, syntaxError(0, "input is not valid UTF-8")
	}
	if parser.Fonts.Size12 == nil {
		return destination, 0, syntaxError(0, "Size12 font is nil")
	}

	current := style{font: parser.Fonts.Size12, foreground: parser.Foreground, background: parser.Background}
	var stack [maximumNestingDepth]styleFrame
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
			Font: current.font, Value: value,
			Foreground: current.foreground, Background: current.background,
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
		case "<size=12>", "<size=16>", "<size=24>":
			if depth == len(stack) {
				return destination, emitted, syntaxError(offset, "nesting is too deep")
			}
			font := parser.Fonts.Size12
			if tag == "<size=16>" {
				font = parser.Fonts.Size16
			} else if tag == "<size=24>" {
				font = parser.Fonts.Size24
			}
			if font == nil {
				return destination, emitted, syntaxError(offset, "selected font is nil")
			}
			stack[depth] = styleFrame{style: current, kind: tagSize}
			depth++
			current.font = font
		case "</size>":
			if err := closeStyle(&current, &stack, &depth, tagSize, offset, "size"); err != nil {
				return destination, emitted, err
			}
		case "</fg>":
			if err := closeStyle(&current, &stack, &depth, tagForeground, offset, "fg"); err != nil {
				return destination, emitted, err
			}
		case "</bg>":
			if err := closeStyle(&current, &stack, &depth, tagBackground, offset, "bg"); err != nil {
				return destination, emitted, err
			}
		case "<br>", "<br/>":
			if err := emit("\n"); err != nil {
				return destination, emitted, err
			}
		default:
			if strings.HasPrefix(tag, "<size=") {
				return destination, emitted, syntaxError(offset, "invalid size tag")
			}
			if strings.HasPrefix(tag, "<fg=") || strings.HasPrefix(tag, "<bg=") {
				kind := tagForeground
				name := "foreground"
				if strings.HasPrefix(tag, "<bg=") {
					kind = tagBackground
					name = "background"
				}
				color, ok := parseColorTag(tag)
				if !ok {
					return destination, emitted, syntaxError(offset, "invalid "+name+" color")
				}
				if depth == len(stack) {
					return destination, emitted, syntaxError(offset, "nesting is too deep")
				}
				stack[depth] = styleFrame{style: current, kind: kind}
				depth++
				if kind == tagForeground {
					current.foreground = color
				} else {
					current.background = color
				}
			} else {
				return destination, emitted, syntaxError(offset, "unknown tag")
			}
		}
		offset = end + 1
	}

	if depth != 0 {
		return destination, emitted, syntaxError(len(value), "unclosed tag")
	}
	return destination, emitted, nil
}

func closeStyle(current *style, stack *[maximumNestingDepth]styleFrame, depth *int, kind tagKind, offset int, name string) error {
	if *depth == 0 {
		return syntaxError(offset, "unexpected closing "+name+" tag")
	}
	frame := stack[*depth-1]
	if frame.kind != kind {
		return syntaxError(offset, "mismatched closing "+name+" tag")
	}
	*depth = *depth - 1
	*current = frame.style
	return nil
}

func parseColorTag(tag string) (display.Color565, bool) {
	if len(tag) != 12 || tag[3] != '=' || tag[4] != '#' || tag[11] != '>' {
		return 0, false
	}
	digits := tag[5:11]
	var values [6]uint8
	for index := range values {
		var ok bool
		values[index], ok = hexDigit(digits[index])
		if !ok {
			return 0, false
		}
	}
	return display.RGB565(values[0]<<4|values[1], values[2]<<4|values[3], values[4]<<4|values[5]), true
}

func hexDigit(value byte) (uint8, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

func syntaxError(offset int, message string) *SyntaxError {
	return &SyntaxError{Offset: offset, Message: message}
}
