//go:build tinygo

package main

import (
	"time"

	"github.com/rdon-key/modgadget"
	board "github.com/rdon-key/modgadget/examples/internal/cardputeradv"
	"github.com/rdon-key/modgadget/font/efont16"
)

func main() {
	time.Sleep(3 * time.Second)
	panel, err := board.ConfigureDisplay()
	if err != nil {
		panic(err)
	}
	player, err := board.ConfigureAudio()
	if err != nil {
		panic(err)
	}
	// Configure the keyboard last because audio and keyboard share I2C0.
	keyboard, err := board.ConfigureKeyboard()
	if err != nil {
		panic(err)
	}

	font := efont16.Font
	styles := modgadget.StyleSet{
		Default: modgadget.Style{Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack},
		Entries: []modgadget.StyleEntry{
			{Name: "title", Style: modgadget.Style{Font: font, Foreground: modgadget.RGB565(80, 220, 255), Background: modgadget.ColorBlack}},
		},
	}
	gadget := modgadget.New(panel,
		modgadget.WithStyles(styles),
		modgadget.WithKeyboard(keyboard),
		modgadget.WithVolumeController(player),
	)
	if err := gadget.Clear(); err != nil {
		panic(err)
	}
	title := gadget.Viewport(modgadget.Bounds(0, 0, board.DisplayWidth, 18))
	typed := gadget.Viewport(modgadget.Bounds(0, 28, board.DisplayWidth, 64))
	help := gadget.Viewport(modgadget.Bounds(0, 108, board.DisplayWidth, 18))
	if err := title.SetText("<style=title>Keyboard input</style>"); err != nil {
		panic(err)
	}
	if err := help.SetText("Backspace: erase "); err != nil {
		panic(err)
	}

	var state typingState
	gadget.OnKey(func(event modgadget.KeyEvent) bool {
		if !state.HandleKey(event) {
			return false
		}
		if err := typed.SetText(escapeMarkup(state.Text())); err != nil {
			panic(err)
		}
		return true
	})

	for {
		if err := player.Update(); err != nil {
			_ = player.Stop()
			panic(err)
		}
		now := time.Now()
		gadget.Update(now)
		if err := keyboard.Err(); err != nil {
			panic(err)
		}
		if err := gadget.Render(); err != nil {
			panic(err)
		}
		time.Sleep(16 * time.Millisecond)
	}
}
