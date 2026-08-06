// Package efont24 provides the embedded Efont Biwidth 24 MGF font.
package efont24

import (
	_ "embed"

	"github.com/rdon-key/modgadget"
)

//go:embed efont24-full.mgf
var data string

// Font is the embedded full Efont Biwidth 24 font.
var Font modgadget.Font = modgadget.MustOpenMGF(data)
