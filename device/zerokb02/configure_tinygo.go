//go:build tinygo

package zerokb02

import (
	"fmt"
	"machine"
	"time"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget/internal/ssd1306"
)

func configureDisplay() (modgadget.Display, error) {
	if err := machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 400_000,
		SDA:       machine.GPIO12,
		SCL:       machine.GPIO13,
	}); err != nil {
		return nil, fmt.Errorf("zerokb02: configure I2C0: %w", err)
	}
	time.Sleep(10 * time.Millisecond)
	display := ssd1306.New(machine.I2C0)
	if err := display.Configure(); err != nil {
		return nil, fmt.Errorf("zerokb02: configure OLED: %w", err)
	}
	return display, nil
}
