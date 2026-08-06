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
	styles := modgadget.StyleSet{
		Default: modgadget.Style{Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack},
		Entries: []modgadget.StyleEntry{
			{Name: "accent", Style: modgadget.Style{Font: font, Foreground: modgadget.ColorBlack, Background: modgadget.RGB565(80, 220, 255)}},
			{Name: "green", Style: modgadget.Style{Font: font, Foreground: modgadget.ColorGreen, Background: modgadget.ColorBlack}},
		},
	}
	gadget := modgadget.New(panel, modgadget.WithStyles(styles))
	if err := gadget.Clear(); err != nil {
		panic(err)
	}
	view := gadget.Viewport(modgadget.Bounds(0, 0, board.DisplayWidth, board.DisplayHeight))
	const content = "<style=accent><b>Multilingual text</b></style><br>" +
		"English: Hello<br>日本語: こんにちは<br>中文: 你好<br>한국어: 안녕하세요<br>" +
		"<style=green>named Style</style> and <b>Bold</b>"
	if err := view.SetText(content); err != nil {
		panic(err)
	}
	if err := gadget.Render(); err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Second)
	}
}
