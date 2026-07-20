package main

import (
	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/st7789"
	"machine"
	"time"
)

const (
	displayWidth  int16 = 240
	displayHeight int16 = 135
	markerSize    int16 = 8
	markerStride        = int(markerSize)*2 + 2
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

	println("before draw")
	if err := drawTestPattern(backend); err != nil {
		println("draw failed:", err.Error())
		return
	}
	println("after draw")
	println("ST7789 rectangle streaming complete")

	for {
		println("alive")
		time.Sleep(time.Second)
	}
}

func drawTestPattern(backend display.Backend) error {
	var scratch [64]byte
	leftWidth := displayWidth / 2
	topHeight := displayHeight / 2

	if err := display.FillRect(
		backend,
		display.Rect{Width: displayWidth, Height: displayHeight},
		display.ColorBlack,
		scratch[:],
	); err != nil {
		return err
	}

	regions := [...]struct {
		rect  display.Rect
		color display.Color565
	}{
		{display.Rect{X: 1, Y: 1, Width: leftWidth - 2, Height: topHeight - 2}, display.ColorRed},
		{display.Rect{X: leftWidth + 1, Y: 1, Width: displayWidth - leftWidth - 2, Height: topHeight - 2}, display.ColorGreen},
		{display.Rect{X: 1, Y: topHeight + 1, Width: leftWidth - 2, Height: displayHeight - topHeight - 2}, display.ColorBlue},
	}
	for _, region := range regions {
		if err := display.FillRect(backend, region.rect, region.color, scratch[:]); err != nil {
			return err
		}
	}

	if err := drawStripes(backend, display.Rect{
		X:      leftWidth + 1,
		Y:      topHeight + 1,
		Width:  displayWidth - leftWidth - 2,
		Height: displayHeight - topHeight - 2,
	}, scratch[:]); err != nil {
		return err
	}

	markers := [...]struct {
		rect  display.Rect
		color display.Color565
	}{
		{display.Rect{Width: markerSize, Height: markerSize}, display.ColorRed},
		{display.Rect{X: displayWidth - markerSize, Width: markerSize, Height: markerSize}, display.ColorGreen},
		{display.Rect{Y: displayHeight - markerSize, Width: markerSize, Height: markerSize}, display.ColorBlue},
		{display.Rect{X: displayWidth - markerSize, Y: displayHeight - markerSize, Width: markerSize, Height: markerSize}, display.ColorWhite},
	}
	for _, marker := range markers {
		if err := drawMarker(backend, marker.rect, marker.color); err != nil {
			return err
		}
	}
	return nil
}

func drawStripes(backend display.Backend, rect display.Rect, scratch []byte) error {
	for offset := int16(0); offset < rect.Height; offset += 8 {
		height := int16(8)
		if height > rect.Height-offset {
			height = rect.Height - offset
		}
		color := display.ColorWhite
		if offset/8&1 != 0 {
			color = display.ColorBlack
		}
		stripe := display.Rect{X: rect.X, Y: rect.Y + offset, Width: rect.Width, Height: height}
		if err := display.FillRect(backend, stripe, color, scratch); err != nil {
			return err
		}
	}
	return nil
}

func drawMarker(backend display.Backend, rect display.Rect, color display.Color565) error {
	var pixels [markerStride * int(markerSize)]byte
	high := byte(color >> 8)
	low := byte(color)
	rowBytes := int(markerSize) * 2

	for row := 0; row < int(markerSize); row++ {
		start := row * markerStride
		for column := 0; column < rowBytes; column += 2 {
			pixels[start+column] = high
			pixels[start+column+1] = low
		}
	}
	return display.BlitRGB565(backend, rect, pixels[:], markerStride)
}
