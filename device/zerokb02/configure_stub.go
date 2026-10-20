//go:build !tinygo

package zerokb02

import (
	"errors"

	"github.com/rdon-key/modgadget"
)

var errTinyGoRequired = errors.New("zerokb02: display configuration requires TinyGo")

func configureDisplay() (modgadget.Display, error) { return nil, errTinyGoRequired }
