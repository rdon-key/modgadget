package main

import (
	"fmt"
	"machine"
	"time"

	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/fontdata/spleen8x16"
	"github.com/rdon-key/modgadget/internal/st7789"
	textdraw "github.com/rdon-key/modgadget/internal/text"
)

const (
	displayWidth  int16 = 240
	displayHeight int16 = 135
)

type viewportDemo struct {
	name        string
	bounds      display.Rect
	penX        int16
	baselineY   int16
	borderColor display.Color565
}

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

	println("before viewport text draw")
	if err := drawViewportDemo(backend); err != nil {
		println("viewport text draw failed:", err.Error())
		return
	}
	println("after viewport text draw")
	println("ST7789 viewport text clipping complete")

	for {
		println("alive")
		time.Sleep(time.Second)
	}
}

func drawViewportDemo(backend display.Backend) error {
	var fillScratch [64]byte
	var glyphScratch [288]byte

	if err := display.FillRect(
		backend,
		display.Rect{Width: displayWidth, Height: displayHeight},
		display.RGB565(16, 16, 24),
		fillScratch[:],
	); err != nil {
		return fmt.Errorf("fill display: %w", err)
	}

	layout, err := textdraw.NewTextLayout([]textdraw.Span{
		{Face: &spleen8x16.Font, Value: "View", Foreground: display.ColorWhite, Background: display.ColorBlack},
		{Face: &spleen8x16.Font, Value: "port\n", Foreground: display.ColorGreen, Background: display.ColorBlack},
		{Face: &spleen8x16.Font, Value: "clip", Foreground: display.RGB565(0, 255, 255), Background: display.ColorBlack},
		{Face: &spleen8x16.Font, Value: "ping", Foreground: display.RGB565(255, 255, 0), Background: display.ColorBlack},
	})
	if err != nil {
		return fmt.Errorf("build viewport text layout: %w", err)
	}
	printLayoutMeasurement(&layout)

	demos := [...]viewportDemo{
		{name: "left clip", bounds: display.Rect{X: 6, Y: 6, Width: 108, Height: 52}, penX: -16, baselineY: 18, borderColor: display.ColorWhite},
		{name: "right clip", bounds: display.Rect{X: 126, Y: 6, Width: 108, Height: 52}, penX: 72, baselineY: 18, borderColor: display.ColorGreen},
		{name: "top clip", bounds: display.Rect{X: 6, Y: 72, Width: 108, Height: 52}, penX: 8, baselineY: 5, borderColor: display.RGB565(0, 255, 255)},
		{name: "bottom clip", bounds: display.Rect{X: 126, Y: 72, Width: 108, Height: 52}, penX: 8, baselineY: 40, borderColor: display.RGB565(255, 255, 0)},
	}
	for index := range demos {
		if err := drawViewportPanel(backend, &layout, demos[index], fillScratch[:], glyphScratch[:]); err != nil {
			return err
		}
	}
	return nil
}

func drawViewportPanel(backend display.Backend, layout *textdraw.TextLayout, demo viewportDemo, fillScratch, glyphScratch []byte) error {
	println("panel:", demo.name)
	println("bounds:", demo.bounds.X, demo.bounds.Y, demo.bounds.Width, demo.bounds.Height)
	println("penX:", demo.penX)
	println("first baseline:", demo.baselineY)

	border := display.Rect{X: demo.bounds.X - 1, Y: demo.bounds.Y - 1, Width: demo.bounds.Width + 2, Height: demo.bounds.Height + 2}
	if err := display.FillRect(backend, border, demo.borderColor, fillScratch); err != nil {
		println("draw error: true")
		return fmt.Errorf("%s border: %w", demo.name, err)
	}
	if err := display.FillRect(backend, demo.bounds, display.ColorBlack, fillScratch); err != nil {
		println("draw error: true")
		return fmt.Errorf("%s background: %w", demo.name, err)
	}

	viewport, err := display.NewViewport(demo.bounds)
	if err != nil {
		println("draw error: true")
		return fmt.Errorf("%s viewport: %w", demo.name, err)
	}
	clippedBackend, err := display.NewViewportBackend(backend, viewport)
	if err != nil {
		println("draw error: true")
		return fmt.Errorf("%s viewport backend: %w", demo.name, err)
	}
	nextBaseline, err := layout.Draw(clippedBackend, demo.penX, demo.baselineY, glyphScratch)
	if err != nil {
		println("next baseline:", nextBaseline)
		println("draw error: true")
		return fmt.Errorf("%s layout draw: %w", demo.name, err)
	}
	println("next baseline:", nextBaseline)
	println("draw error: false")
	return nil
}

func printLayoutMeasurement(layout *textdraw.TextLayout) {
	measurement := layout.Measurement()
	println("layout line count:", layout.LineCount())
	println("layout MaxAdvanceX:", measurement.MaxAdvanceX)
	println("layout AdvanceY:", measurement.AdvanceY)
	println("layout bounds:", measurement.Bounds.MinX, measurement.Bounds.MinY, measurement.Bounds.MaxX, measurement.Bounds.MaxY)
	println("layout HasInk:", measurement.HasInk)
}
