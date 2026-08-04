package main

import "github.com/rdon-key/modgadget"

const target = "TinyGo Makes Small Devices Fun!"

const targetLineBreak = len("TinyGo Makes ")

type game struct {
	position int
	misses   int
}

func (game *game) Reset() { game.position, game.misses = 0, 0 }

func (game *game) HandleKey(event modgadget.KeyEvent) bool {
	if event.Action != modgadget.KeyDown {
		return false
	}
	switch event.Code {
	case modgadget.KeyEnter:
		game.Reset()
		return true
	case modgadget.KeyBackspace:
		if game.position > 0 {
			game.position--
		}
		return true
	}
	if event.Rune == 0 || game.Complete() {
		return false
	}
	if event.Rune == rune(target[game.position]) {
		game.position++
	} else {
		game.misses++
	}
	return true
}

func (game *game) Complete() bool { return game.position == len(target) }

func (game *game) typedMarkup() string {
	typed := target[:game.position]
	if game.position > targetLineBreak {
		return typed[:targetLineBreak] + "<br>" + typed[targetLineBreak:]
	}
	return typed
}
