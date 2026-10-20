package zerokb02

import (
	"testing"

	"github.com/rdon-key/modgadget"
)

var _ func() (modgadget.Display, error) = ConfigureDisplay

func TestDisplayDimensions(t *testing.T) {
	if DisplayWidth != 128 || DisplayHeight != 64 {
		t.Fatalf("dimensions: %dx%d", DisplayWidth, DisplayHeight)
	}
}
