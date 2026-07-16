// Initialization and rotation handling are derived from
// tinygo.org/x/drivers/st7789.
// Copyright The TinyGo Authors. All rights reserved.
// Licensed under the BSD 3-Clause License. See
// LICENSES/tinygo-drivers-BSD-3-Clause.txt.

package st7789

import "time"

// Configure resets and initializes the display for RGB565 streaming.
func (d *Device) Configure(config Config) error {
	if d.rectActive {
		return ErrRectActive
	}
	if d.bus == nil || config.Width <= 0 || config.Height <= 0 ||
		config.Rotation > Rotation270 || config.RowOffset < 0 || config.ColumnOffset < 0 {
		return ErrInvalidConfig
	}

	d.configured = false
	d.cs.High()
	d.dc.High()

	d.reset.High()
	time.Sleep(10 * time.Millisecond)
	d.reset.Low()
	time.Sleep(20 * time.Millisecond)
	d.reset.High()
	time.Sleep(120 * time.Millisecond)

	if err := d.configureCommand(commandSoftwareReset, nil); err != nil {
		return err
	}
	time.Sleep(150 * time.Millisecond)

	d.cs.Low()
	defer d.cs.High()

	if err := d.sendCommand(commandSleepOut, nil); err != nil {
		return err
	}
	if err := d.sendCommand(commandPixelFormat, []byte{pixelFormatRGB565}); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	madctl, rowOffset, columnOffset := rotationSettings(config)
	if err := d.sendCommand(commandMemoryAccess, []byte{madctl}); err != nil {
		return err
	}
	if err := d.sendCommand(commandFrameRate2, []byte{0x0f}); err != nil {
		return err
	}
	if err := d.sendCommand(commandPorchControl, []byte{0x08, 0x08, 0x00, 0x22, 0x22}); err != nil {
		return err
	}

	invertCommand := byte(commandInvertOff)
	if config.Invert {
		invertCommand = commandInvertOn
	}
	if err := d.sendCommand(invertCommand, nil); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)
	if err := d.sendCommand(commandNormalOn, nil); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)
	if err := d.sendCommand(commandDisplayOn, nil); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	d.cs.High()

	d.width = config.Width
	d.height = config.Height
	if config.Rotation == Rotation90 ||
		config.Rotation == Rotation270 {
		d.width, d.height = config.Height, config.Width
	}
	d.rowOffset = rowOffset
	d.columnOffset = columnOffset
	d.configured = true
	return nil
}

func (d *Device) configureCommand(command byte, data []byte) error {
	d.cs.Low()
	err := d.sendCommand(command, data)
	d.cs.High()
	if err != nil {
		d.dc.High()
		d.rectActive = false
		d.remaining = 0
	}
	return err
}

func rotationSettings(config Config) (madctl byte, rowOffset, columnOffset int16) {
	switch config.Rotation {
	case Rotation0:
		return 0, 0, 0
	case Rotation90:
		return madctlMX | madctlMV, config.ColumnOffset, config.RowOffset
	case Rotation180:
		return madctlMX | madctlMY, config.RowOffset, config.ColumnOffset
	case Rotation270:
		return madctlMY | madctlMV, config.ColumnOffset, config.RowOffset
	default:
		return 0, 0, 0
	}
}
