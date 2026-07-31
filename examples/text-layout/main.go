package main

import (
	"fmt"
	"machine"
	"time"

	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/shinonome12"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/spleen8x16"
	"github.com/rdon-key/modgadget/internal/st7789"
	textdraw "github.com/rdon-key/modgadget/internal/text"
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

	println("before layout draw")
	if err := drawLayoutDemo(backend); err != nil {
		println("layout draw failed:", err.Error())
		return
	}
	println("after layout draw")
	println("ST7789 prepared text layout display complete")

	for {
		println("alive")
		time.Sleep(time.Second)
	}
}

func drawLayoutDemo(backend display.Backend) error {
	var fillScratch [64]byte
	var glyphScratch [288]byte

	if err := display.FillRect(
		backend,
		display.Rect{Width: displayWidth, Height: displayHeight},
		display.ColorBlack,
		fillScratch[:],
	); err != nil {
		return fmt.Errorf("fill display: %w", err)
	}

	asciiLayout, err := textdraw.NewTextLayout([]textdraw.Span{
		{Font: textdraw.NewMGFFont(spleen8x16.Font), Value: "Mod", Foreground: display.ColorWhite, Background: display.ColorBlack},
		{Font: textdraw.NewMGFFont(spleen8x16.Font), Value: "Gadget\n", Foreground: display.ColorGreen, Background: display.ColorBlack},
		{Font: textdraw.NewMGFFont(spleen8x16.Font), Value: "prepared ", Foreground: display.ColorBlue, Background: display.ColorBlack},
		{Font: textdraw.NewMGFFont(spleen8x16.Font), Value: "layout", Foreground: display.ColorRed, Background: display.ColorBlack},
	})
	if err != nil {
		return fmt.Errorf("build ASCII layout: %w", err)
	}
	printLayoutMeasurement("ASCII", &asciiLayout)
	if _, err := asciiLayout.Draw(backend, 0, 16, glyphScratch[:]); err != nil {
		return fmt.Errorf("draw ASCII layout at baseline 16: %w", err)
	}
	if _, err := asciiLayout.Draw(backend, 0, 60, glyphScratch[:]); err != nil {
		return fmt.Errorf("draw ASCII layout at baseline 60: %w", err)
	}

	japaneseLayout, err := textdraw.NewTextLayout([]textdraw.Span{
		{Font: textdraw.NewMGFFont(shinonome12.Font), Value: "日本語表示", Foreground: display.ColorWhite, Background: display.ColorBlack},
	})
	if err != nil {
		return fmt.Errorf("build Japanese layout: %w", err)
	}
	printLayoutMeasurement("Japanese", &japaneseLayout)
	if _, err := japaneseLayout.Draw(backend, 12, 110, glyphScratch[:]); err != nil {
		return fmt.Errorf("draw Japanese layout at baseline 110: %w", err)
	}
	return nil
}

func printLayoutMeasurement(name string, layout *textdraw.TextLayout) {
	measurement := layout.Measurement()
	lineCount := layout.LineCount()
	println(name, "layout")
	println("line count:", lineCount)
	println("MaxAdvanceX:", measurement.MaxAdvanceX)
	println("AdvanceY:", measurement.AdvanceY)
	println("Bounds.MinX:", measurement.Bounds.MinX)
	println("Bounds.MinY:", measurement.Bounds.MinY)
	println("Bounds.MaxX:", measurement.Bounds.MaxX)
	println("Bounds.MaxY:", measurement.Bounds.MaxY)
	println("HasInk:", measurement.HasInk)
}
