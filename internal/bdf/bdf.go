// Package bdf parses the subset of BDF 2.1 and 2.2 needed by the MGF converter.
package bdf

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Font is a parsed BDF font.
type Font struct {
	Version         string
	Name            string
	CharsetRegistry string
	CharsetEncoding string
	Ascent          int
	Descent         int
	HasAscent       bool
	HasDescent      bool
	Glyphs          []Glyph
}

// Glyph is a decoded BDF glyph. Bitmap contains raw, row-aligned bitmap bytes.
type Glyph struct {
	Name     string
	Encoding int
	AdvanceX int
	Width    int
	Height   int
	XOffset  int
	YOffset  int
	Bitmap   string
}

type parser struct {
	data           string
	position       int
	line           int
	font           Font
	globalAdvance  int
	hasGlobalDW    bool
	metricsSet     int
	hasMetricsSet  bool
	declaredChars  int
	hasChars       bool
	hasFont        bool
	hasSize        bool
	hasBoundingBox bool
	hasProperties  bool
}

// Parse parses a BDF 2.1 or 2.2 font.
func Parse(data string) (Font, error) {
	p := parser{data: data}
	line, number, ok := p.nextContentLine()
	if !ok {
		return Font{}, fmt.Errorf("bdf: missing STARTFONT")
	}
	fields := strings.Fields(line)
	if len(fields) != 2 || fields[0] != "STARTFONT" {
		return Font{}, fmt.Errorf("bdf: line %d: STARTFONT must be the first keyword", number)
	}
	if fields[1] != "2.1" && fields[1] != "2.2" {
		return Font{}, fmt.Errorf("bdf: line %d: unsupported STARTFONT version %q", number, fields[1])
	}
	p.font.Version = fields[1]

	for {
		line, number, ok = p.nextContentLine()
		if !ok {
			return Font{}, fmt.Errorf("bdf: line %d: missing ENDFONT", number)
		}
		fields = strings.Fields(line)
		keyword := fields[0]
		if p.hasChars {
			switch keyword {
			case "STARTCHAR", "ENDFONT":
			case "CHARS":
				return Font{}, p.errorf(number, "duplicate CHARS")
			default:
				return Font{}, p.errorf(number, "%s is not allowed after CHARS", keyword)
			}
		}
		switch keyword {
		case "FONT":
			if p.hasFont {
				return Font{}, p.errorf(number, "duplicate FONT")
			}
			if len(fields) < 2 {
				return Font{}, p.errorf(number, "FONT requires a name")
			}
			p.font.Name = strings.TrimSpace(line[len("FONT"):])
			p.hasFont = true
		case "SIZE":
			if p.hasSize {
				return Font{}, p.errorf(number, "duplicate SIZE")
			}
			if err := requireInts(fields, 3); err != nil {
				return Font{}, p.wrap(number, "SIZE", err)
			}
			p.hasSize = true
		case "FONTBOUNDINGBOX":
			if p.hasBoundingBox {
				return Font{}, p.errorf(number, "duplicate FONTBOUNDINGBOX")
			}
			if err := requireInts(fields, 4); err != nil {
				return Font{}, p.wrap(number, "FONTBOUNDINGBOX", err)
			}
			p.hasBoundingBox = true
		case "METRICSSET":
			if p.hasMetricsSet {
				return Font{}, p.errorf(number, "duplicate METRICSSET")
			}
			values, err := parseInts(fields[1:], 1)
			if err != nil {
				return Font{}, p.wrap(number, "METRICSSET", err)
			}
			if values[0] == 1 || (values[0] != 0 && values[0] != 2) {
				return Font{}, p.errorf(number, "METRICSSET %d does not provide supported horizontal metrics", values[0])
			}
			p.metricsSet, p.hasMetricsSet = values[0], true
		case "DWIDTH":
			if p.hasGlobalDW {
				return Font{}, p.errorf(number, "duplicate global DWIDTH")
			}
			values, err := parseInts(fields[1:], 2)
			if err != nil {
				return Font{}, p.wrap(number, "DWIDTH", err)
			}
			if values[1] != 0 {
				return Font{}, p.errorf(number, "global DWIDTH Y is %d, want 0", values[1])
			}
			p.globalAdvance, p.hasGlobalDW = values[0], true
		case "STARTPROPERTIES":
			if p.hasProperties {
				return Font{}, p.errorf(number, "duplicate STARTPROPERTIES")
			}
			if err := p.parseProperties(number, fields); err != nil {
				return Font{}, err
			}
			p.hasProperties = true
		case "CHARS":
			if p.hasChars {
				return Font{}, p.errorf(number, "duplicate CHARS")
			}
			values, err := parseInts(fields[1:], 1)
			if err != nil || values[0] < 0 {
				if err == nil {
					err = fmt.Errorf("negative count %d", values[0])
				}
				return Font{}, p.wrap(number, "CHARS", err)
			}
			p.declaredChars, p.hasChars = values[0], true
		case "STARTCHAR":
			if !p.hasChars {
				return Font{}, p.errorf(number, "STARTCHAR before CHARS")
			}
			glyph, err := p.parseGlyph(number, line)
			if err != nil {
				return Font{}, err
			}
			p.font.Glyphs = append(p.font.Glyphs, glyph)
		case "ENDFONT":
			if len(fields) != 1 {
				return Font{}, p.errorf(number, "ENDFONT has unexpected arguments")
			}
			if !p.hasChars {
				return Font{}, p.errorf(number, "missing CHARS")
			}
			if !p.hasFont {
				return Font{}, p.errorf(number, "missing FONT")
			}
			if !p.hasSize {
				return Font{}, p.errorf(number, "missing SIZE")
			}
			if !p.hasBoundingBox {
				return Font{}, p.errorf(number, "missing FONTBOUNDINGBOX")
			}
			if len(p.font.Glyphs) != p.declaredChars {
				return Font{}, p.errorf(number, "CHARS declares %d glyphs, found %d", p.declaredChars, len(p.font.Glyphs))
			}
			if extra, extraLine, exists := p.nextContentLine(); exists {
				return Font{}, p.errorf(extraLine, "content after ENDFONT: %q", extra)
			}
			return p.font, nil
		case "COMMENT", "CONTENTVERSION", "SWIDTH", "SWIDTH1", "DWIDTH1", "VVECTOR":
			// Known fields not needed by MGF conversion.
		default:
			return Font{}, p.errorf(number, "unexpected global keyword %q", keyword)
		}
	}
}

func (p *parser) parseProperties(startLine int, fields []string) error {
	values, err := parseInts(fields[1:], 1)
	if err != nil || values[0] < 0 {
		return p.wrap(startLine, "STARTPROPERTIES", err)
	}
	count := values[0]
	for index := 0; index < count; index++ {
		line, number, ok := p.nextLine()
		if !ok {
			return p.errorf(startLine, "unterminated STARTPROPERTIES")
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			return p.errorf(number, "empty property")
		}
		switch fields[0] {
		case "FONT_ASCENT":
			if p.font.HasAscent {
				return p.errorf(number, "duplicate FONT_ASCENT")
			}
			v, err := parseInts(fields[1:], 1)
			if err != nil {
				return p.wrap(number, "FONT_ASCENT", err)
			}
			p.font.Ascent, p.font.HasAscent = v[0], true
		case "FONT_DESCENT":
			if p.font.HasDescent {
				return p.errorf(number, "duplicate FONT_DESCENT")
			}
			v, err := parseInts(fields[1:], 1)
			if err != nil {
				return p.wrap(number, "FONT_DESCENT", err)
			}
			p.font.Descent, p.font.HasDescent = v[0], true
		case "CHARSET_REGISTRY":
			if p.font.CharsetRegistry != "" {
				return p.errorf(number, "duplicate CHARSET_REGISTRY")
			}
			p.font.CharsetRegistry, err = propertyString(fields)
			if err != nil {
				return p.wrap(number, "CHARSET_REGISTRY", err)
			}
		case "CHARSET_ENCODING":
			if p.font.CharsetEncoding != "" {
				return p.errorf(number, "duplicate CHARSET_ENCODING")
			}
			p.font.CharsetEncoding, err = propertyString(fields)
			if err != nil {
				return p.wrap(number, "CHARSET_ENCODING", err)
			}
		}
	}
	line, number, ok := p.nextContentLine()
	if !ok || strings.TrimSpace(line) != "ENDPROPERTIES" {
		return p.errorf(number, "expected ENDPROPERTIES")
	}
	return nil
}

func (p *parser) parseGlyph(startLine int, start string) (Glyph, error) {
	name := strings.TrimSpace(start[len("STARTCHAR"):])
	if name == "" {
		return Glyph{}, p.errorf(startLine, "STARTCHAR requires a name")
	}
	g := Glyph{Name: name}
	hasEncoding, hasBBX, hasDW := false, false, false
	for {
		line, number, ok := p.nextContentLine()
		if !ok {
			return Glyph{}, p.errorf(startLine, "glyph %q: missing ENDCHAR", name)
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "ENCODING":
			if hasEncoding {
				return Glyph{}, p.glyphError(number, name, "duplicate ENCODING")
			}
			if len(fields) != 2 && len(fields) != 3 {
				return Glyph{}, p.glyphError(number, name, "ENCODING requires one or two integers")
			}
			values, err := parseInts(fields[1:], len(fields)-1)
			if err != nil {
				return Glyph{}, p.glyphError(number, name, "invalid ENCODING: %v", err)
			}
			if values[0] < -1 {
				return Glyph{}, p.glyphError(number, name, "ENCODING is %d, want -1 or non-negative", values[0])
			}
			g.Encoding, hasEncoding = values[0], true
		case "DWIDTH":
			if hasDW {
				return Glyph{}, p.glyphError(number, name, "duplicate DWIDTH")
			}
			values, err := parseInts(fields[1:], 2)
			if err != nil {
				return Glyph{}, p.glyphError(number, name, "invalid DWIDTH: %v", err)
			}
			if values[1] != 0 {
				return Glyph{}, p.glyphError(number, name, "DWIDTH Y is %d, want 0", values[1])
			}
			g.AdvanceX, hasDW = values[0], true
		case "BBX":
			if hasBBX {
				return Glyph{}, p.glyphError(number, name, "duplicate BBX")
			}
			values, err := parseInts(fields[1:], 4)
			if err != nil {
				return Glyph{}, p.glyphError(number, name, "invalid BBX: %v", err)
			}
			if values[0] < 0 || values[1] < 0 {
				return Glyph{}, p.glyphError(number, name, "BBX has negative dimensions %d x %d", values[0], values[1])
			}
			g.Width, g.Height, g.XOffset, g.YOffset = values[0], values[1], values[2], values[3]
			hasBBX = true
		case "BITMAP":
			if !hasEncoding {
				return Glyph{}, p.glyphError(number, name, "BITMAP before ENCODING")
			}
			if !hasBBX {
				return Glyph{}, p.glyphError(number, name, "BITMAP before BBX")
			}
			if !hasDW {
				if !p.hasGlobalDW {
					return Glyph{}, p.glyphError(number, name, "missing DWIDTH")
				}
				g.AdvanceX = p.globalAdvance
			}
			bitmap, err := p.parseBitmap(name, g.Width, g.Height)
			if err != nil {
				return Glyph{}, err
			}
			g.Bitmap = bitmap
			return g, nil
		case "ENDCHAR":
			return Glyph{}, p.glyphError(number, name, "missing BITMAP")
		case "SWIDTH", "SWIDTH1", "DWIDTH1", "VVECTOR", "COMMENT":
			// Known glyph fields not needed by MGF conversion.
		default:
			return Glyph{}, p.glyphError(number, name, "unexpected keyword %q", fields[0])
		}
	}
}

func (p *parser) parseBitmap(name string, width, height int) (string, error) {
	if width < 0 || height < 0 {
		return "", p.glyphError(p.line, name, "BBX has negative dimensions %d x %d", width, height)
	}
	width64, height64 := uint64(width), uint64(height)
	if width64 > math.MaxUint64-7 {
		return "", p.glyphError(p.line, name, "bitmap size for BBX %d x %d is too large", width, height)
	}
	rowBytes64 := (width64 + 7) / 8
	bitmapRows64 := height64
	if width == 0 || height == 0 {
		bitmapRows64 = 0
	}
	if bitmapRows64 != 0 && rowBytes64 > math.MaxUint64/bitmapRows64 {
		return "", p.glyphError(p.line, name, "bitmap size for BBX %d x %d is too large", width, height)
	}
	bitmapSize64 := rowBytes64 * bitmapRows64
	maxInt := uint64(^uint(0) >> 1)
	if rowBytes64 > maxInt || bitmapSize64 > maxInt || rowBytes64 > math.MaxUint64/2 || rowBytes64*2 > maxInt {
		return "", p.glyphError(p.line, name, "bitmap size for BBX %d x %d is too large", width, height)
	}
	if bitmapSize64 > math.MaxUint64/2 || bitmapSize64*2 > uint64(len(p.data)-p.position) {
		return "", p.glyphError(p.line, name, "bitmap size for BBX %d x %d exceeds remaining BDF data", width, height)
	}
	rowBytes := int(rowBytes64)
	bitmapRows := int(bitmapRows64)
	bitmap := make([]byte, int(bitmapSize64))
	for row := 0; row < bitmapRows; row++ {
		line, number, ok := p.nextLine()
		if !ok {
			return "", p.glyphError(number, name, "BITMAP has %d rows, want %d", row, bitmapRows)
		}
		hexText := strings.TrimSpace(line)
		if len(hexText) != rowBytes*2 {
			return "", p.glyphError(number, name, "BITMAP row has %d hex digits, want %d", len(hexText), rowBytes*2)
		}
		if _, err := hex.Decode(bitmap[row*rowBytes:(row+1)*rowBytes], []byte(hexText)); err != nil {
			return "", p.glyphError(number, name, "invalid BITMAP row: %v", err)
		}
	}
	line, number, ok := p.nextLine()
	if !ok || strings.TrimSpace(line) != "ENDCHAR" {
		return "", p.glyphError(number, name, "expected ENDCHAR after %d BITMAP rows", bitmapRows)
	}
	return string(bitmap), nil
}

func (p *parser) nextLine() (string, int, bool) {
	if p.position >= len(p.data) {
		return "", p.line + 1, false
	}
	start := p.position
	if end := strings.IndexByte(p.data[start:], '\n'); end >= 0 {
		p.position = start + end + 1
		end += start
		if end > start && p.data[end-1] == '\r' {
			end--
		}
		p.line++
		return p.data[start:end], p.line, true
	}
	p.position = len(p.data)
	p.line++
	line := p.data[start:]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line, p.line, true
}

func (p *parser) nextContentLine() (string, int, bool) {
	for {
		line, number, ok := p.nextLine()
		if !ok {
			return "", number, false
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "COMMENT") {
			continue
		}
		return trimmed, number, true
	}
}

func parseInts(fields []string, count int) ([]int, error) {
	if len(fields) != count {
		return nil, fmt.Errorf("has %d values, want %d", len(fields), count)
	}
	values := make([]int, count)
	for index := range fields {
		value, err := strconv.ParseInt(fields[index], 10, 0)
		if err != nil {
			return nil, fmt.Errorf("value %q: %w", fields[index], err)
		}
		values[index] = int(value)
	}
	return values, nil
}

func requireInts(fields []string, count int) error {
	_, err := parseInts(fields[1:], count)
	return err
}

func propertyString(fields []string) (string, error) {
	if len(fields) != 2 {
		return "", fmt.Errorf("requires one value")
	}
	value := fields[1]
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	return value, nil
}

func (p *parser) errorf(line int, format string, args ...any) error {
	return fmt.Errorf("bdf: line %d: %s", line, fmt.Sprintf(format, args...))
}

func (p *parser) glyphError(line int, name, format string, args ...any) error {
	return p.errorf(line, "glyph %q: %s", name, fmt.Sprintf(format, args...))
}

func (p *parser) wrap(line int, field string, err error) error {
	if err == nil {
		return p.errorf(line, "invalid %s", field)
	}
	return p.errorf(line, "invalid %s: %v", field, err)
}
