//go:build tinygo

package cardputeradv

import (
	"fmt"
	"machine"

	keyboarddriver "github.com/rdon-key/modgadget/internal/keyboard/cardputeradv"
)

// ConfigureKeyboard initializes the Cardputer ADV keyboard controller.
func ConfigureKeyboard() (*keyboarddriver.Keyboard, error) {
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
