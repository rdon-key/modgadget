package cardputeradv

import (
	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget/audio/cardputeradv"
)

// Keyboard is a Cardputer ADV keyboard usable by ModGadget. Err reports a
// hardware polling error retained by the keyboard driver.
type Keyboard interface {
	modgadget.Keyboard
	Err() error
}

// ConfigureDisplay initializes the Cardputer ADV display and backlight.
func ConfigureDisplay() (modgadget.Display, error) { return configureDisplay() }

// ConfigureKeyboard initializes the Cardputer ADV keyboard controller.
// Configure it after audio because both devices use I2C0.
func ConfigureKeyboard() (Keyboard, error) { return configureKeyboard() }

// ConfigureAudio initializes and returns a Cardputer ADV audio player.
// Configure audio before the keyboard because both devices use I2C0.
func ConfigureAudio() (*cardputeradv.Player, error) { return configureAudio() }
