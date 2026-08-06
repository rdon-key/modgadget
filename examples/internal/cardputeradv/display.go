//go:build tinygo

package cardputeradv

import (
	"fmt"
	"machine"

	"github.com/rdon-key/modgadget/internal/st7789"
)

// ConfigureDisplay initializes the Cardputer ADV display and backlight.
func ConfigureDisplay() (*st7789.Device, error) {
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
