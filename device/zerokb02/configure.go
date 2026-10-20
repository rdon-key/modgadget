// Package zerokb02 configures ModGadget for the TinyGo Keeb Tour Zero-KB02.
// It currently supports the 128x64 monochrome OLED. Keyboard, joystick,
// encoder and LEDs are intentionally outside this display-only adapter.
package zerokb02

import "github.com/rdon-key/modgadget"

const (
	DisplayWidth  int16 = 128
	DisplayHeight int16 = 64
)

// ConfigureDisplay initializes the OLED on a Waveshare RP2040-Zero.
func ConfigureDisplay() (modgadget.Display, error) { return configureDisplay() }
