// Package efont16 provides the embedded Efont Biwidth 16 MGF font.
package efont16

import (
	_ "embed"

	"github.com/rdon-key/modgadget/internal/mgf"
)

//go:embed efont16-full.mgf
var data string

// Font is the embedded full Efont Biwidth 16 font.
var Font mgf.Font = mgf.MustOpen(data)
