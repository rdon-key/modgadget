package main

import (
	"fmt"

	"github.com/rdon-key/modgadget/internal/display"
	efont16mgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/efont16"
	efont24mgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/efont24"
	"github.com/rdon-key/modgadget/internal/text"
)

const (
	regularLineHeight   int16 = 19
	directionLineHeight int16 = 27
)

var (
	keyFont16 text.Font = text.NewMGFFont(efont16mgf.Font)
	keyFont24 text.Font = text.NewMGFFont(efont24mgf.Font)
)

type keyDisplay struct {
	label      string
	font       text.Font
	foreground display.Color565
	background display.Color565
	lineHeight int16
}

func displayForKey(row, col int) keyDisplay {
	displayValue := keyDisplay{
		font:       keyFont16,
		foreground: display.ColorWhite,
		background: display.ColorBlack,
		lineHeight: regularLineHeight,
	}
	switch {
	case row == 2 && col == 11:
		displayValue.label = "上"
		displayValue.font = keyFont24
		displayValue.foreground = display.ColorBlack
		displayValue.background = display.RGB565(64, 192, 255)
		displayValue.lineHeight = directionLineHeight
	case row == 3 && col == 11:
		displayValue.label = "下"
		displayValue.font = keyFont24
		displayValue.foreground = display.ColorBlack
		displayValue.background = display.RGB565(96, 224, 128)
		displayValue.lineHeight = directionLineHeight
	case row == 3 && col == 10:
		displayValue.label = "左"
		displayValue.font = keyFont24
		displayValue.foreground = display.ColorBlack
		displayValue.background = display.RGB565(255, 208, 64)
		displayValue.lineHeight = directionLineHeight
	case row == 3 && col == 12:
		displayValue.label = "右"
		displayValue.font = keyFont24
		displayValue.foreground = display.ColorWhite
		displayValue.background = display.RGB565(224, 64, 64)
		displayValue.lineHeight = directionLineHeight
	case row == 2 && col == 13:
		displayValue.label = "OK"
		displayValue.foreground = display.ColorBlack
		displayValue.background = display.RGB565(96, 224, 128)
	case row == 0 && col == 13:
		displayValue.label = "ESC"
		displayValue.foreground = display.ColorWhite
		displayValue.background = display.RGB565(224, 64, 64)
	default:
		displayValue.label = fmt.Sprintf("row=%d col=%d", row, col)
	}
	return displayValue
}
