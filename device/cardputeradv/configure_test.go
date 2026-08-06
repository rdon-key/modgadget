package cardputeradv

import (
	"testing"

	"github.com/rdon-key/modgadget"
	audio "github.com/rdon-key/modgadget/audio/cardputeradv"
)

var (
	_ func() (modgadget.Display, error) = ConfigureDisplay
	_ func() (Keyboard, error)          = ConfigureKeyboard
	_ func() (*audio.Player, error)     = ConfigureAudio
)

func TestDisplayDimensions(t *testing.T) {
	if DisplayWidth != 240 || DisplayHeight != 135 {
		t.Fatalf("display dimensions=%dx%d", DisplayWidth, DisplayHeight)
	}
}
