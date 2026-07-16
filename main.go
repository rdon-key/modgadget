package main

import (
	"machine"
	"modgadget-test/internal/st7789"
	"time"
)

const (
	displayWidth  int16 = 240
	displayHeight int16 = 135
)

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

	display := st7789.New(
		machine.SPI1,
		machine.DISPLAY_CS,
		machine.DISPLAY_DC,
		machine.DISPLAY_RST,
	)

	println("before display configure")
	if err := display.Configure(st7789.Config{
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

	println("before draw")
	if err := drawTestPattern(display); err != nil {
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

func drawTestPattern(display *st7789.Device) error {
	leftWidth := displayWidth / 2
	rightWidth := displayWidth - leftWidth
	topHeight := displayHeight / 2
	bottomHeight := displayHeight - topHeight

	if err := drawChunkedSolid(display, 0, 0, leftWidth, topHeight, 0xf800); err != nil {
		return err
	}
	if err := drawLineSolid(display, leftWidth, 0, rightWidth, topHeight, 0x07e0); err != nil {
		return err
	}
	if err := drawSubLineSolid(display, 0, topHeight, leftWidth, bottomHeight, 0x001f); err != nil {
		return err
	}
	return drawStripes(display, leftWidth, topHeight, rightWidth, bottomHeight)
}

func drawChunkedSolid(display *st7789.Device, x, y, width, height int16, color uint16) error {
	var chunk [64]byte
	fillRGB565(chunk[:], color)

	if err := display.BeginRect(x, y, width, height); err != nil {
		return err
	}

	remaining := int(width) * int(height) * 2
	for remaining > 0 {
		n := len(chunk)
		if n > remaining {
			n = remaining
		}
		if err := display.WritePixels(chunk[:n]); err != nil {
			return err
		}
		remaining -= n
	}

	return display.EndRect()
}

func drawLineSolid(display *st7789.Device, x, y, width, height int16, color uint16) error {
	var line [displayWidth * 2]byte
	row := line[:int(width)*2]
	fillRGB565(row, color)

	if err := display.BeginRect(x, y, width, height); err != nil {
		return err
	}

	for rowIndex := int16(0); rowIndex < height; rowIndex++ {
		if err := display.WritePixels(row); err != nil {
			return err
		}
	}

	return display.EndRect()
}

func drawSubLineSolid(display *st7789.Device, x, y, width, height int16, color uint16) error {
	var chunk [46]byte
	fillRGB565(chunk[:], color)

	if err := display.BeginRect(x, y, width, height); err != nil {
		return err
	}

	remaining := int(width) * int(height) * 2
	for remaining > 0 {
		n := len(chunk)
		if n > remaining {
			n = remaining
		}
		if err := display.WritePixels(chunk[:n]); err != nil {
			return err
		}
		remaining -= n
	}

	return display.EndRect()
}

func drawStripes(display *st7789.Device, x, y, width, height int16) error {
	var line [displayWidth * 2]byte
	row := line[:int(width)*2]

	if err := display.BeginRect(x, y, width, height); err != nil {
		return err
	}

	for rowIndex := int16(0); rowIndex < height; rowIndex++ {
		color := uint16(0xffff)
		if rowIndex&1 != 0 {
			color = 0x0000
		}
		fillRGB565(row, color)

		if err := display.WritePixels(row); err != nil {
			return err
		}
	}

	return display.EndRect()
}

func fillRGB565(buffer []byte, color uint16) {
	high := byte(color >> 8)
	low := byte(color)

	for i := 0; i < len(buffer); i += 2 {
		buffer[i] = high
		buffer[i+1] = low
	}
}
