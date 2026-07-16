// Window addressing is derived from tinygo.org/x/drivers/st7789.
// Copyright The TinyGo Authors. All rights reserved.
// Licensed under the BSD 3-Clause License. See
// LICENSES/tinygo-drivers-BSD-3-Clause.txt.

package st7789

// BeginRect starts a rectangle transfer and keeps CS low until EndRect.
func (d *Device) BeginRect(x, y, width, height int16) error {
	if !d.configured {
		return ErrNotConfigured
	}
	if d.rectActive {
		return ErrRectActive
	}
	if width <= 0 || height <= 0 {
		return ErrInvalidRectSize
	}
	if x < 0 || y < 0 || x >= d.width || y >= d.height ||
		width > d.width-x || height > d.height-y {
		return ErrOutOfBounds
	}

	d.rectActive = true
	d.remaining = int32(width) * int32(height) * 2
	d.cs.Low()

	x0 := int32(x) + int32(d.columnOffset)
	y0 := int32(y) + int32(d.rowOffset)
	if err := d.sendAddress(commandColumnAddress, x0, x0+int32(width)-1); err != nil {
		d.abortRect()
		return err
	}
	if err := d.sendAddress(commandRowAddress, y0, y0+int32(height)-1); err != nil {
		d.abortRect()
		return err
	}
	if err := d.sendCommand(commandMemoryWrite, nil); err != nil {
		d.abortRect()
		return err
	}
	d.dc.High()
	return nil
}

func (d *Device) sendAddress(command byte, start, end int32) error {
	d.addressBuffer[0] = byte(start >> 8)
	d.addressBuffer[1] = byte(start)
	d.addressBuffer[2] = byte(end >> 8)
	d.addressBuffer[3] = byte(end)
	return d.sendCommand(command, d.addressBuffer[:])
}

// WritePixels sends big-endian RGB565 bytes without copying them.
func (d *Device) WritePixels(data []byte) error {
	if !d.rectActive {
		return ErrNoRectActive
	}
	if len(data)&1 != 0 {
		return ErrOddPixelData
	}
	if int32(len(data)) > d.remaining {
		return ErrTooMuchPixelData
	}
	if len(data) == 0 {
		return nil
	}
	if err := d.bus.Tx(data, nil); err != nil {
		d.abortRect()
		return err
	}
	d.remaining -= int32(len(data))
	return nil
}

// EndRect closes the transfer. It always restores CS and the idle state.
func (d *Device) EndRect() error {
	if !d.rectActive {
		return ErrNoRectActive
	}
	complete := d.remaining == 0
	d.abortRect()
	if !complete {
		return ErrIncompleteRect
	}
	return nil
}
