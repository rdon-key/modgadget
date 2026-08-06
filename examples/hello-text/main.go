//go:build tinygo

package main

import (
	"time"

	"github.com/rdon-key/modgadget"
	board "github.com/rdon-key/modgadget/device/cardputeradv"
	"github.com/rdon-key/modgadget/font/efont16"
)

func main() {
	time.Sleep(3 * time.Second)
	panel, err := board.ConfigureDisplay()
	if err != nil {
		panic(err)
	}
	font := efont16.Font
	gadget := modgadget.New(panel, modgadget.WithStyles(modgadget.StyleSet{
		Default: modgadget.Style{Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack},
	}))
	if err := gadget.Clear(); err != nil {
		panic(err)
	}
	view := gadget.Viewport(modgadget.Bounds(0, 0, board.DisplayWidth, 16))
	if err := view.SetText("Hello, ModGadget!"); err != nil {
		panic(err)
	}
	if err := gadget.Render(); err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Second)
	}
}
