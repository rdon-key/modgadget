//go:build !tinygo

package cardputeradv

import (
	"errors"

	"github.com/rdon-key/modgadget"
	audio "github.com/rdon-key/modgadget/audio/cardputeradv"
)

var errTinyGoRequired = errors.New("Cardputer ADV device configuration requires TinyGo")

func configureDisplay() (modgadget.Display, error) { return nil, errTinyGoRequired }

func configureKeyboard() (Keyboard, error) { return nil, errTinyGoRequired }

func configureAudio() (*audio.Player, error) { return nil, errTinyGoRequired }
