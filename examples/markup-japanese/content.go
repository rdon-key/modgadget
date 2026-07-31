package main

import (
	"github.com/rdon-key/modgadget/internal/display"
	efont16mgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/efont16"
	efont24mgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/efont24"
	shinonomemgf "github.com/rdon-key/modgadget/internal/fontdata/mgf/shinonome12"
	"github.com/rdon-key/modgadget/internal/text"
	"github.com/rdon-key/modgadget/internal/text/markup"
)

// Shinonome 12 does not contain the ASCII used by this demo, so those runs
// explicitly select Efont 16 while the untagged Japanese runs remain 12px.
const message = "通常<size=16> ModGadget 日本語16px </size><size=24><fg=#ff4040>大きな24px</fg></size><br><bg=#ffffff><fg=#000000>背景色</fg></bg><size=16> 1 << 2</size>"

var messageParser = markup.Parser{
	Fonts: markup.Fonts{
		Size12: text.MGFFont{Font: shinonomemgf.Font},
		Size16: text.MGFFont{Font: efont16mgf.Font},
		Size24: text.MGFFont{Font: efont24mgf.Font},
	},
	Foreground: display.ColorWhite,
	Background: display.ColorBlack,
}
