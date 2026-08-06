//go:build tinygo

package main

import (
	"time"

	"github.com/rdon-key/modgadget"
	board "github.com/rdon-key/modgadget/examples/internal/cardputeradv"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/efont24"
)

func main() {
	time.Sleep(3 * time.Second)
	panel, err := board.ConfigureDisplay()
	if err != nil {
		panic(err)
	}
	font := modgadget.NewMGFFont(efont24.Font)
	gadget := modgadget.New(panel, modgadget.WithStyles(modgadget.StyleSet{
		Default: modgadget.Style{Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack},
	}))
	if err := gadget.Clear(); err != nil {
		panic(err)
	}

	still := gadget.Viewport(modgadget.Bounds(0, 0, board.DisplayWidth, 24))
	oneShot := gadget.Viewport(modgadget.Bounds(0, 32, board.DisplayWidth, 24))
	loop := gadget.Viewport(modgadget.Bounds(0, 64, board.DisplayWidth, 24))
	if err := still.SetText("Static text"); err != nil {
		panic(err)
	}
	if err := oneShot.SetText("One-shot text enters from the right."); err != nil {
		panic(err)
	}
	oneShot.SetHorizontalScroll(modgadget.ScrollSpeed(24), modgadget.ScrollFromRight())
	if err := loop.SetText("Loop: 日本語 ◆ English ◆ 中文 ◆ 한국어"); err != nil {
		panic(err)
	}
	loop.SetHorizontalScroll(
		modgadget.ScrollSpeed(24),
		modgadget.ScrollGap(32),
		modgadget.ScrollLoop(),
		modgadget.ScrollFromRight(),
	)

	for {
		now := time.Now()
		gadget.Update(now)
		if err := gadget.Render(); err != nil {
			panic(err)
		}
		time.Sleep(16 * time.Millisecond)
	}
}
