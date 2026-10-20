//go:build tinygo

package main

import (
	"time"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget-fonts/efont24"
	board "github.com/rdon-key/modgadget/device/zerokb02"
)

func main() {
	panel, err := board.ConfigureDisplay()
	if err != nil {
		panic(err)
	}

	font := efont24.Font // MGZ-compressed 24-pixel CJK bitmap font
	const greeting = "恭喜進軍台灣！"
	for _, r := range greeting {
		if !font.HasGlyph(r) {
			panic("efont24 is missing a greeting glyph")
		}
	}

	gadget := modgadget.New(panel, modgadget.WithStyles(modgadget.StyleSet{
		Default: modgadget.Style{
			Font:       font,
			Foreground: modgadget.ColorWhite,
			Background: modgadget.ColorBlack,
		},
	}))
	if err := gadget.Clear(); err != nil {
		panic(err)
	}

	// A 128-pixel OLED cannot hold seven 24-pixel-wide characters in one row.
	first := gadget.Viewport(modgadget.Bounds(16, 0, 112, 32))
	if err := first.SetText("恭喜進軍"); err != nil {
		panic(err)
	}
	second := gadget.Viewport(modgadget.Bounds(28, 32, 100, 32))
	if err := second.SetText("台灣！"); err != nil {
		panic(err)
	}
	if err := gadget.Render(); err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Second)
	}
}
