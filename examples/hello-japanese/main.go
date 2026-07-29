package main

import (
	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/fontdata/shinonome12"
	"github.com/rdon-key/modgadget/internal/st7789"
	textdraw "github.com/rdon-key/modgadget/internal/text"
	"machine"
	"time"
)

const (
	displayWidth  int16 = 240
	displayHeight int16 = 135
)

var _ display.Backend = (*st7789.Device)(nil)

func main() {
	time.Sleep(3 * time.Second)
	println("boot")

	println("before SPI configure")
	if err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 10_000_000,
		Mode:      0,
		SCK:       machine.DISPLAY_SCK,
		SDO:       machine.DISPLAY_MOSI,
		SDI:       machine.NoPin,
	}); err != nil {
		println("SPI configure failed:", err.Error())
		return
	}
	println("after SPI configure")

	machine.DISPLAY_BL.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.DISPLAY_BL.High()
	println("backlight on")

	panel := st7789.New(
		machine.SPI1,
		machine.DISPLAY_CS,
		machine.DISPLAY_DC,
		machine.DISPLAY_RST,
	)

	println("before display configure")
	if err := panel.Configure(st7789.Config{
		Width:        135,
		Height:       240,
		Rotation:     st7789.Rotation90,
		RowOffset:    40,
		ColumnOffset: 52,
		Invert:       true,
	}); err != nil {
		println("display configure failed:", err.Error())
		return
	}
	println("after display configure")
	var backend display.Backend = panel

	println("before text draw")
	if err := drawTextDemo(backend); err != nil {
		println("text draw failed:", err.Error())
		return
	}
	println("after text draw")
	println("ST7789 text display complete")

	for {
		println("alive")
		time.Sleep(time.Second)
	}
}

func drawTextDemo(backend display.Backend) error {
	var fillScratch [64]byte
	var glyphScratch [288]byte

	if err := display.FillRect(
		backend,
		display.Rect{Width: displayWidth, Height: displayHeight},
		display.ColorBlack,
		fillScratch[:],
	); err != nil {
		return err
	}

	_, err := textdraw.DrawString(
		backend,
		&shinonome12.Font,
		12,
		shinonome12.Font.Metrics().Ascent,
		"日本語表示",
		display.ColorWhite,
		display.ColorBlack,
		glyphScratch[:],
	)
	return err
}
