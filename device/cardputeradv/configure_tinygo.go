//go:build tinygo

package cardputeradv

import (
	"fmt"
	"machine"

	"github.com/rdon-key/modgadget"
	audio "github.com/rdon-key/modgadget/audio/cardputeradv"
	keyboarddriver "github.com/rdon-key/modgadget/internal/keyboard/cardputeradv"
	"github.com/rdon-key/modgadget/internal/st7789"
)

func configureDisplay() (modgadget.Display, error) {
	if err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 40_000_000,
		Mode:      0,
		SCK:       machine.DISPLAY_SCK,
		SDO:       machine.DISPLAY_MOSI,
		SDI:       machine.NoPin,
	}); err != nil {
		return nil, fmt.Errorf("configure Cardputer ADV SPI: %w", err)
	}
	machine.DISPLAY_BL.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.DISPLAY_BL.High()
	panel := st7789.New(machine.SPI1, machine.DISPLAY_CS, machine.DISPLAY_DC, machine.DISPLAY_RST)
	if err := panel.Configure(st7789.Config{
		Width: 135, Height: 240, Rotation: st7789.Rotation90,
		RowOffset: 40, ColumnOffset: 52, Invert: true,
	}); err != nil {
		return nil, fmt.Errorf("configure Cardputer ADV display: %w", err)
	}
	return panel, nil
}

func configureKeyboard() (Keyboard, error) {
	if err := machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 400_000,
		SDA:       machine.GPIO8,
		SCL:       machine.GPIO9,
	}); err != nil {
		return nil, fmt.Errorf("configure Cardputer ADV keyboard I2C: %w", err)
	}
	keyboard := keyboarddriver.New(machine.I2C0)
	if err := keyboard.Configure(); err != nil {
		return nil, fmt.Errorf("configure Cardputer ADV keyboard: %w", err)
	}
	return keyboard, nil
}

func configureAudio() (*audio.Player, error) {
	player, err := audio.Configure()
	if err != nil {
		return nil, fmt.Errorf("configure Cardputer ADV audio: %w", err)
	}
	return player, nil
}
