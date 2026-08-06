//go:build tinygo

package cardputeradv

import (
	"fmt"

	audio "github.com/rdon-key/modgadget/internal/audio/cardputeradv"
)

// ConfigureAudio initializes the Cardputer ADV audio player.
func ConfigureAudio() (*audio.Player, error) {
	player := audio.New()
	if err := player.Configure(); err != nil {
		return nil, fmt.Errorf("configure Cardputer ADV audio: %w", err)
	}
	return player, nil
}
