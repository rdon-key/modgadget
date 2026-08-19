//go:build tinygo

package main

import (
	_ "embed"
	"time"

	"github.com/rdon-key/modgadget"
	board "github.com/rdon-key/modgadget/device/cardputeradv"
)

//go:embed efont24-full.mgz
var fontData string

func main() {
	time.Sleep(3 * time.Second)
	panel, err := board.ConfigureDisplay()
	if err != nil {
		panic(err)
	}
	font := modgadget.MustOpenMGZ(fontData)
	gadget := modgadget.New(panel, modgadget.WithStyles(modgadget.StyleSet{Default: modgadget.Style{
		Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack,
	}}))
	if err := gadget.Clear(); err != nil {
		panic(err)
	}
	view := gadget.Viewport(modgadget.Bounds(0, 0, board.DisplayWidth, board.DisplayHeight))
	if err := view.SetText("Hello, world!<br>日本語: こんにちは<br>中文: 你好<br>한국어: 안녕하세요"); err != nil {
		panic(err)
	}
	if err := gadget.Render(); err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Second)
	}
}
