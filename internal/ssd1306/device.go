// Package ssd1306 adapts the Zero-KB02's monochrome OLED to ModGadget's
// row-major RGB565 display interface.
package ssd1306

import (
	"errors"
	"fmt"
)

const (
	Width   int16  = 128
	Height  int16  = 64
	Address uint16 = 0x3c
	frameBytes     = int(Width) * int(Height) / 8
)

// Bus is the subset of machine.I2C needed by this display.
type Bus interface {
	Tx(address uint16, write, read []byte) error
}

var (
	ErrNotConfigured = errors.New("ssd1306: device is not configured")
	ErrActive        = errors.New("ssd1306: rectangle already active")
	ErrInactive      = errors.New("ssd1306: no active rectangle")
	ErrInvalidRect   = errors.New("ssd1306: rectangle is out of bounds")
	ErrTooMuchData   = errors.New("ssd1306: too many pixel bytes")
	ErrIncomplete    = errors.New("ssd1306: incomplete rectangle")
)

// Device holds the 1024-byte SSD1306 framebuffer. Pixel data is accepted
// as big-endian RGB565 and converted to monochrome: black is off and every
// nonblack RGB565 value is on. The OLED is physically upside down on Zero-KB02.
type Device struct {
	bus        Bus
	frame      [1 + frameBytes]byte // I2C data control byte followed by pages
	configured bool

	active   bool
	x, y     int16
	w, h     int16
	expected int32
	received int32
	pending  byte
	haveHigh bool
}

func New(bus Bus) *Device {
	d := &Device{bus: bus}
	d.frame[0] = 0x40 // SSD1306 I2C data prefix
	return d
}

func (d *Device) Size() (int16, int16) { return Width, Height }

// Configure initializes the SSD1306 in horizontal addressing mode and
// clears the display. The caller configures the I2C bus beforehand.
func (d *Device) Configure() error {
	if d == nil || d.bus == nil {
		return errors.New("ssd1306: nil I2C bus")
	}
	if d.active {
		return ErrActive
	}
	// 128x64, internal charge pump, horizontal addressing and 180-degree
	// rotation (SEGREMAP=0xA0, COMSCANINC=0xC0).
	commands := [...]byte{
		0x00, // I2C command prefix
		0xae, 0xd5, 0x80, 0xa8, 0x3f, 0xd3, 0x00, 0x40,
		0x8d, 0x14, 0x20, 0x00, 0xa0, 0xc0,
		0xda, 0x12, 0x81, 0xcf, 0xd9, 0xf1, 0xdb, 0x40,
		0xa4, 0xa6, 0x2e, 0xaf,
	}
	d.configured = false
	if err := d.bus.Tx(Address, commands[:], nil); err != nil {
		return fmt.Errorf("ssd1306: initialize: %w", err)
	}
	for i := 1; i < len(d.frame); i++ {
		d.frame[i] = 0
	}
	if err := d.flush(); err != nil {
		return err
	}
	d.configured = true
	return nil
}

func (d *Device) BeginRect(x, y, width, height int16) error {
	if d == nil || !d.configured {
		return ErrNotConfigured
	}
	if d.active {
		return ErrActive
	}
	if x < 0 || y < 0 || width <= 0 || height <= 0 ||
		int32(x)+int32(width) > int32(Width) ||
		int32(y)+int32(height) > int32(Height) {
		return ErrInvalidRect
	}
	d.x, d.y, d.w, d.h = x, y, width, height
	d.expected = int32(width) * int32(height) * 2
	d.received, d.haveHigh, d.active = 0, false, true
	return nil
}

// WritePixels accepts arbitrary byte chunks, including a split RGB565 pair.
func (d *Device) WritePixels(data []byte) error {
	if d == nil || !d.active {
		return ErrInactive
	}
	if int64(d.received)+int64(len(data)) > int64(d.expected) {
		return ErrTooMuchData
	}
	for i, b := range data {
		if !d.haveHigh {
			d.pending = b
			d.haveHigh = true
			continue
		}
		pixel := (d.received + int32(i)) / 2
		x := int(d.x) + int(pixel%int32(d.w))
		y := int(d.y) + int(pixel/int32(d.w))
		index := 1 + (y/8)*int(Width) + x
		bit := byte(1) << uint(y&7)
		if d.pending != 0 || b != 0 {
			d.frame[index] |= bit
		} else {
			d.frame[index] &^= bit
		}
		d.haveHigh = false
	}
	d.received += int32(len(data))
	return nil
}

// EndRect sends the entire page-packed framebuffer to the OLED. Full-buffer
// transfers are deliberately simple; the display is just 128x64 pixels.
func (d *Device) EndRect() error {
	if d == nil || !d.active {
		return ErrInactive
	}
	complete := d.received == d.expected && !d.haveHigh
	d.active = false
	d.haveHigh = false
	if !complete {
		return ErrIncomplete
	}
	return d.flush()
}

func (d *Device) flush() error {
	if err := d.bus.Tx(Address, d.frame[:], nil); err != nil {
		return fmt.Errorf("ssd1306: flush: %w", err)
	}
	return nil
}
