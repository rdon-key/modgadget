//go:build tinygo

package main

import (
	"machine"
	"strconv"
	"time"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/efont16"
	"github.com/rdon-key/modgadget/internal/keyboard/cardputeradv"
	"github.com/rdon-key/modgadget/internal/st7789"
)

func main() {
	time.Sleep(3 * time.Second)
	if err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 40_000_000,
		Mode:      0,
		SCK:       machine.DISPLAY_SCK,
		SDO:       machine.DISPLAY_MOSI,
		SDI:       machine.NoPin,
	}); err != nil {
		panic(err)
	}
	machine.DISPLAY_BL.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.DISPLAY_BL.High()
	panel := st7789.New(machine.SPI1, machine.DISPLAY_CS, machine.DISPLAY_DC, machine.DISPLAY_RST)
	if err := panel.Configure(st7789.Config{
		Width: 135, Height: 240, Rotation: st7789.Rotation90,
		RowOffset: 40, ColumnOffset: 52, Invert: true,
	}); err != nil {
		panic(err)
	}

	if err := machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 400_000,
		SDA:       machine.GPIO8,
		SCL:       machine.GPIO9,
	}); err != nil {
		panic(err)
	}
	keyboard := cardputeradv.New(machine.I2C0)
	if err := keyboard.Configure(); err != nil {
		panic(err)
	}

	font := modgadget.NewMGFFont(efont16.Font)
	styles := modgadget.StyleSet{
		Default: modgadget.Style{Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack},
		Entries: []modgadget.StyleEntry{
			{Name: "title", Style: modgadget.Style{Font: font, Foreground: modgadget.RGB565(80, 220, 255), Background: modgadget.ColorBlack}},
			{Name: "done", Style: modgadget.Style{Font: font, Foreground: modgadget.ColorGreen, Background: modgadget.ColorBlack}},
			{Name: "next", Style: modgadget.Style{Font: font, Foreground: modgadget.RGB565(255, 220, 0), Background: modgadget.ColorBlack}},
			{Name: "miss", Style: modgadget.Style{Font: font, Foreground: modgadget.ColorRed, Background: modgadget.ColorBlack}},
		},
	}
	gadget := modgadget.New(panel, modgadget.WithStyles(styles), modgadget.WithKeyboard(keyboard))
	if err := gadget.Clear(); err != nil {
		panic(err)
	}
	titleView := gadget.Viewport(modgadget.Bounds(0, 0, 240, 18))
	targetView := gadget.Viewport(modgadget.Bounds(0, 21, 240, 36))
	typedView := gadget.Viewport(modgadget.Bounds(0, 58, 240, 36))
	statusView := gadget.Viewport(modgadget.Bounds(0, 96, 240, 18))
	helpView := gadget.Viewport(modgadget.Bounds(0, 116, 240, 18))
	if err := titleView.SetText("<style=title>ModGadget Typing</style>"); err != nil {
		panic(err)
	}
	if err := targetView.SetText("TinyGo Makes<br>Small Devices Fun!"); err != nil {
		panic(err)
	}
	if err := helpView.SetText("Enter: restart  Del: back"); err != nil {
		panic(err)
	}

	game := game{}
	refresh := func() {
		if err := typedView.SetText("<style=done>" + game.typedMarkup() + "</style>"); err != nil {
			panic(err)
		}
		status := "Miss: " + strconv.Itoa(game.misses)
		if game.Complete() {
			status = "<style=done>Complete! Enter to restart</style>"
		} else {
			status += "  <style=next>Next: " + string(target[game.position]) + "</style>"
		}
		if err := statusView.SetText(status); err != nil {
			panic(err)
		}
	}
	refresh()
	gadget.OnKey(func(event modgadget.KeyEvent) bool {
		if game.HandleKey(event) {
			refresh()
			return true
		}
		return false
	})

	for {
		now := time.Now()
		gadget.Update(now)
		if err := keyboard.Err(); err != nil {
			panic(err)
		}
		if err := gadget.Render(); err != nil {
			panic(err)
		}
		time.Sleep(16 * time.Millisecond)
	}
}
