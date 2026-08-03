//go:build tinygo

package main

import (
	"machine"
	"time"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/efont24"
	"github.com/rdon-key/modgadget/internal/st7789"
)

func main() {
	time.Sleep(3 * time.Second)
	if err := machine.SPI1.Configure(machine.SPIConfig{Frequency: 40_000_000, Mode: 0, SCK: machine.DISPLAY_SCK, SDO: machine.DISPLAY_MOSI, SDI: machine.NoPin}); err != nil {
		panic(err)
	}
	machine.DISPLAY_BL.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.DISPLAY_BL.High()
	panel := st7789.New(machine.SPI1, machine.DISPLAY_CS, machine.DISPLAY_DC, machine.DISPLAY_RST)
	if err := panel.Configure(st7789.Config{Width: 135, Height: 240, Rotation: st7789.Rotation90, RowOffset: 40, ColumnOffset: 52, Invert: true}); err != nil {
		panic(err)
	}
	font := modgadget.NewMGFFont(efont24.Font)
	styles := modgadget.StyleSet{Default: modgadget.Style{Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack}, Entries: []modgadget.StyleEntry{{Name: "news", Style: modgadget.Style{Font: font, Foreground: modgadget.RGB565(255, 220, 0), Background: modgadget.ColorBlack}}}}
	gadget := modgadget.New(panel, modgadget.WithStyles(styles))
	view := gadget.Viewport(modgadget.Bounds(0, 0, 240, 24))
	if err := view.SetText("<style=news>ModGadgetニュース：日本語表示に成功しました。</style>"); err != nil {
		panic(err)
	}
	view.SetHorizontalScroll(modgadget.ScrollSpeed(30), modgadget.ScrollGap(32), modgadget.ScrollLoop())
	for {
		gadget.Update(time.Now())
		if err := gadget.Render(); err != nil {
			panic(err)
		}
		time.Sleep(16 * time.Millisecond)
	}
}
