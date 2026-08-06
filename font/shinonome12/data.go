// Package shinonome12 provides the embedded Shinonome 12 MGF font.
package shinonome12

import (
	_ "embed"

	"github.com/rdon-key/modgadget"
)

//go:embed shinonome12-full.mgf
var data string

// Font is the embedded full Shinonome 12 font.
var Font modgadget.Font = modgadget.MustOpenMGF(data)
