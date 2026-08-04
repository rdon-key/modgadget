package main

import (
	"testing"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/efont16"
)

func TestTargetGlyphCoverage(t *testing.T) {
	for _, r := range target {
		if _, ok := efont16.Font.Lookup(r); !ok {
			t.Errorf("target rune %q is missing", r)
		}
	}
}

func TestGameCorrectMissBackspaceAndReset(t *testing.T) {
	game := game{}
	if !game.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyT, Rune: 'T', Action: modgadget.KeyDown, Modifiers: modgadget.ModShift}) || game.position != 1 {
		t.Fatalf("correct input position=%d", game.position)
	}
	if !game.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyX, Rune: 'x', Action: modgadget.KeyDown}) || game.position != 1 || game.misses != 1 {
		t.Fatalf("miss position=%d misses=%d", game.position, game.misses)
	}
	game.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyBackspace, Action: modgadget.KeyDown})
	game.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyBackspace, Action: modgadget.KeyDown})
	if game.position != 0 {
		t.Fatalf("backspace position=%d", game.position)
	}
	game.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyEnter, Action: modgadget.KeyDown})
	if game.position != 0 || game.misses != 0 {
		t.Fatalf("reset=%+v", game)
	}
}

func TestGameCompleteShiftBangAndIgnoredEvents(t *testing.T) {
	game := game{position: len(target) - 1}
	if target[game.position] != '!' {
		t.Fatal("target no longer ends in !")
	}
	if !game.HandleKey(modgadget.KeyEvent{Code: modgadget.Key1, Rune: '!', Action: modgadget.KeyDown, Modifiers: modgadget.ModShift}) || !game.Complete() {
		t.Fatal("Aa+1 did not complete game")
	}
	position, misses := game.position, game.misses
	if game.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyA, Rune: 'a', Action: modgadget.KeyDown}) || game.position != position || game.misses != misses {
		t.Fatal("printable input changed completed game")
	}
	if game.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyA, Action: modgadget.KeyUp}) || game.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyEscape, Action: modgadget.KeyDown}) {
		t.Fatal("KeyUp or zero Rune special key was handled")
	}
}

func TestTypedMarkupWrapsWithinTwoLines(t *testing.T) {
	game := game{position: len(target)}
	want := "TinyGo Makes <br>Small Devices Fun!"
	if got := game.typedMarkup(); got != want {
		t.Fatalf("typed markup = %q, want %q", got, want)
	}
}

func TestTargetLinesFitDisplayWidth(t *testing.T) {
	for _, line := range []string{target[:targetLineBreak], target[targetLineBreak:]} {
		width := int16(0)
		for _, r := range line {
			glyph, ok := efont16.Font.Lookup(r)
			if !ok {
				t.Fatalf("missing rune %q", r)
			}
			width += glyph.AdvanceX
		}
		if width > 240 {
			t.Fatalf("line %q width = %d", line, width)
		}
	}
}
