// Portions of this driver are derived from tinygo.org/x/drivers/st7789.
// Copyright The TinyGo Authors. All rights reserved.
// Licensed under the BSD 3-Clause License. See
// LICENSES/tinygo-drivers-BSD-3-Clause.txt.

package st7789

import (
	"errors"
	"machine"
)

// SPI is the write operation required from a TinyGo SPI bus.
type SPI interface {
	Tx(w, r []byte) error
}

// Rotation selects a clockwise display rotation.
type Rotation uint8

const (
	Rotation0 Rotation = iota
	Rotation90
	Rotation180
	Rotation270
)

// Config describes the visible logical display area and its GRAM placement.
type Config struct {
	Width        int16
	Height       int16
	Rotation     Rotation
	RowOffset    int16
	ColumnOffset int16
	Invert       bool
}

var (
	ErrInvalidConfig    = errors.New("st7789: invalid configuration")
	ErrNotConfigured    = errors.New("st7789: device is not configured")
	ErrRectActive       = errors.New("st7789: rectangle transfer already active")
	ErrNoRectActive     = errors.New("st7789: no rectangle transfer active")
	ErrInvalidRectSize  = errors.New("st7789: rectangle width and height must be positive")
	ErrOutOfBounds      = errors.New("st7789: rectangle outside display area")
	ErrOddPixelData     = errors.New("st7789: RGB565 data length must be even")
	ErrTooMuchPixelData = errors.New("st7789: pixel data exceeds rectangle")
	ErrIncompleteRect   = errors.New("st7789: rectangle pixel data is incomplete")
)

// Device is a minimal ST7789 RGB565 rectangle streamer.
type Device struct {
	bus   SPI
	cs    machine.Pin
	dc    machine.Pin
	reset machine.Pin

	width        int16
	height       int16
	rowOffset    int16
	columnOffset int16
	configured   bool
	rectActive   bool
	remaining    int32

	commandBuffer [1]byte
	addressBuffer [4]byte
}

// New prepares the control pins. The SPI bus must be configured by the caller.
func New(bus SPI, cs, dc, reset machine.Pin) *Device {
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	dc.Configure(machine.PinConfig{Mode: machine.PinOutput})
	reset.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs.High()
	dc.High()
	reset.High()

	return &Device{bus: bus, cs: cs, dc: dc, reset: reset}
}

// Size returns the configured logical display dimensions.
func (d *Device) Size() (width, height int16) {
	return d.width, d.height
}

// sendCommand sends a command while CS is already low and leaves DC high.
func (d *Device) sendCommand(command byte, data []byte) error {
	d.commandBuffer[0] = command
	d.dc.Low()
	if err := d.bus.Tx(d.commandBuffer[:], nil); err != nil {
		d.dc.High()
		return err
	}
	d.dc.High()
	if len(data) == 0 {
		return nil
	}
	return d.bus.Tx(data, nil)
}

func (d *Device) abortRect() {
	d.cs.High()
	d.dc.High()
	d.rectActive = false
	d.remaining = 0
}
