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
const message = "通常<style=body16> ModGadget 日本語16px </style><style=large-red>大きな24px</style><br><style=inverse>背景色</style><style=body16> 1 << 2</style>"

var messageStyles = [...]text.StyleEntry{
	{Name: "body16", Style: text.Style{
		Font:       text.NewMGFFont(efont16mgf.Font),
		Foreground: display.ColorWhite,
		Background: display.ColorBlack,
	}},
	{Name: "large-red", Style: text.Style{
		Font:       text.NewMGFFont(efont24mgf.Font),
		Foreground: display.RGB565(255, 64, 64),
		Background: display.ColorBlack,
	}},
	{Name: "inverse", Style: text.Style{
		Font:       text.NewMGFFont(shinonomemgf.Font),
		Foreground: display.ColorBlack,
		Background: display.ColorWhite,
	}},
}

var messageParser = markup.Parser{
	Styles: text.StyleSet{
		Default: text.Style{
			Font:       text.NewMGFFont(shinonomemgf.Font),
			Foreground: display.ColorWhite,
			Background: display.ColorBlack,
		},
		Entries: messageStyles[:],
	},
}
