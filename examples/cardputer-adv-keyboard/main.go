//go:build tinygo

package main

import (
	"fmt"
	"machine"
	"time"

	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/st7789"
	"github.com/rdon-key/modgadget/internal/text"
)

const (
	displayWidth  int16 = 240
	displayHeight int16 = 135

	keyboardAddress = uint16(0x34)
	keyboardSDA     = machine.GPIO8
	keyboardSCL     = machine.GPIO9
	keyboardINT     = machine.GPIO11

	registerCFG       = byte(0x01)
	registerINTStat   = byte(0x02)
	registerKeyLockEC = byte(0x03)
	registerKeyEventA = byte(0x04)
	registerKPGPIO1   = byte(0x1d)
	registerKPGPIO2   = byte(0x1e)
	registerKPGPIO3   = byte(0x1f)

	configKeyEventInterrupt = byte(1 << 0)
	configOverflowInterrupt = byte(1 << 3)
	interruptOverflow       = byte(1 << 3)
	interruptClearMask      = byte(0x1f)
	maximumEvents           = 10
	displayedKeysBottom     = int16(112)
)

var (
	pressed     [4][14]bool
	fillScratch [1024]byte
	rowScratch  [24 * 2]byte
)

var _ display.Backend = (*st7789.Device)(nil)

type keyboard struct {
	bus *machine.I2C
}

func main() {
	time.Sleep(3 * time.Second)
	println("boot")

	if err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 40_000_000,
		Mode:      0,
		SCK:       machine.DISPLAY_SCK,
		SDO:       machine.DISPLAY_MOSI,
		SDI:       machine.NoPin,
	}); err != nil {
		println("SPI configure failed:", err.Error())
		return
	}
	machine.DISPLAY_BL.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.DISPLAY_BL.High()
	panel := st7789.New(machine.SPI1, machine.DISPLAY_CS, machine.DISPLAY_DC, machine.DISPLAY_RST)
	if err := panel.Configure(st7789.Config{
		Width: 135, Height: 240, Rotation: st7789.Rotation90,
		RowOffset: 40, ColumnOffset: 52, Invert: true,
	}); err != nil {
		println("display configure failed:", err.Error())
		return
	}

	if err := machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 400_000,
		SDA:       keyboardSDA,
		SCL:       keyboardSCL,
	}); err != nil {
		println("I2C configure failed:", err.Error())
		return
	}
	keyboardINT.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	keys := keyboard{bus: machine.I2C0}
	if err := keys.configure(); err != nil {
		println("keyboard configure failed:", err.Error())
		return
	}

	var backend display.Backend = panel
	if err := redraw(backend); err != nil {
		println("initial draw failed:", err.Error())
		return
	}
	for {
		if keyboardINT.Get() {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		changed, err := keys.readEvents()
		if err != nil {
			println("keyboard read failed:", err.Error())
			return
		}
		if changed {
			if err := redraw(backend); err != nil {
				println("display update failed:", err.Error())
				return
			}
		}
	}
}

func (keyboard keyboard) configure() error {
	if err := keyboard.writeRegister(registerCFG, 0); err != nil {
		return err
	}
	if err := keyboard.writeRegister(registerKPGPIO1, 0x7f); err != nil {
		return err
	}
	if err := keyboard.writeRegister(registerKPGPIO2, 0xff); err != nil {
		return err
	}
	if err := keyboard.writeRegister(registerKPGPIO3, 0x00); err != nil {
		return err
	}
	count, err := keyboard.eventCount()
	if err != nil {
		return err
	}
	if count > maximumEvents {
		count = maximumEvents
	}
	for index := 0; index < count; index++ {
		if _, err := keyboard.readRegister(registerKeyEventA); err != nil {
			return err
		}
	}
	if err := keyboard.writeRegister(registerINTStat, interruptClearMask); err != nil {
		return err
	}
	return keyboard.writeRegister(registerCFG, configKeyEventInterrupt|configOverflowInterrupt)
}

func (keyboard keyboard) readEvents() (bool, error) {
	status, err := keyboard.readRegister(registerINTStat)
	if err != nil {
		return false, err
	}
	if status&interruptOverflow != 0 {
		println("keyboard fifo overflow")
	}
	count, err := keyboard.eventCount()
	if err != nil {
		return false, err
	}
	if count > maximumEvents {
		count = maximumEvents
	}
	changed := false
	for index := 0; index < count; index++ {
		raw, err := keyboard.readRegister(registerKeyEventA)
		if err != nil {
			return changed, err
		}
		keyIndex := int(raw&0x7f) - 1
		matrixRow, matrixCol := -1, -1
		if keyIndex >= 0 {
			matrixRow = keyIndex / 10
			matrixCol = keyIndex % 10
		}
		row, col, ok := cardputerCoordinate(matrixRow, matrixCol)
		if !ok {
			fmt.Printf("invalid key event raw=%d\n", raw)
			continue
		}
		isPressed := raw&0x80 != 0
		if isPressed {
			fmt.Printf("press row=%d col=%d\n", row, col)
		} else {
			fmt.Printf("release row=%d col=%d\n", row, col)
		}
		if pressed[row][col] != isPressed {
			pressed[row][col] = isPressed
			changed = true
		}
	}
	if status != 0 {
		if err := keyboard.writeRegister(registerINTStat, status&interruptClearMask); err != nil {
			return changed, err
		}
	}
	return changed, nil
}

func (keyboard keyboard) eventCount() (int, error) {
	value, err := keyboard.readRegister(registerKeyLockEC)
	if err != nil {
		return 0, err
	}
	return int(value & 0x0f), nil
}

func (keyboard keyboard) readRegister(register byte) (byte, error) {
	var value [1]byte
	if err := keyboard.bus.Tx(keyboardAddress, []byte{register}, value[:]); err != nil {
		return 0, fmt.Errorf("read register %#02x: %w", register, err)
	}
	return value[0], nil
}

func (keyboard keyboard) writeRegister(register, value byte) error {
	data := [2]byte{register, value}
	if err := keyboard.bus.Tx(keyboardAddress, data[:], nil); err != nil {
		return fmt.Errorf("write register %#02x: %w", register, err)
	}
	return nil
}

func redraw(backend display.Backend) error {
	if err := display.FillRect(backend, display.Rect{Width: displayWidth, Height: displayHeight}, display.ColorBlack, fillScratch[:]); err != nil {
		return err
	}
	if err := drawText(backend, 8, 20, "キーボード", keyFont16, display.ColorWhite, display.ColorBlack); err != nil {
		return err
	}
	count := 0
	baseline := int16(44)
	displayFull := false
	for row := range pressed {
		for col := range pressed[row] {
			if !pressed[row][col] {
				continue
			}
			count++
			if !displayFull {
				key := displayForKey(row, col)
				if baseline+key.font.Metrics().Descent > displayedKeysBottom {
					displayFull = true
				} else if err := drawText(backend, 8, baseline, key.label, key.font, key.foreground, key.background); err != nil {
					return err
				} else {
					baseline += key.lineHeight
				}
			}
		}
	}
	if count == 0 {
		if err := drawText(backend, 8, 44, "none", keyFont16, display.ColorWhite, display.ColorBlack); err != nil {
			return err
		}
	}
	return drawText(backend, 8, 129, fmt.Sprintf("pressed=%d", count), keyFont16, display.RGB565(128, 192, 255), display.ColorBlack)
}

func drawText(backend display.Backend, x, baseline int16, value string, font text.Font, foreground, background display.Color565) error {
	spans := [1]text.Span{{
		Font: font, Value: value, Foreground: foreground, Background: background,
	}}
	_, err := text.DrawSpans(backend, spans[:], x, baseline, rowScratch[:])
	return err
}
