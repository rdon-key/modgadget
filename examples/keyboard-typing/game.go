package main

import (
	"strings"

	"github.com/rdon-key/modgadget"
)

const maximumTypedRunes = 64

type typingState struct {
	runes [maximumTypedRunes]rune
	count int
}

func (state *typingState) HandleKey(event modgadget.KeyEvent) bool {
	if event.Action != modgadget.KeyDown {
		return false
	}
	if event.Code == modgadget.KeyBackspace {
		if state.count > 0 {
			state.count--
		}
		return true
	}
	if event.Rune < ' ' || event.Rune == 0 || state.count == len(state.runes) {
		return false
	}
	state.runes[state.count] = event.Rune
	state.count++
	return true
}

func (state *typingState) Text() string { return string(state.runes[:state.count]) }

func escapeMarkup(value string) string { return strings.ReplaceAll(value, "<", "<<") }
