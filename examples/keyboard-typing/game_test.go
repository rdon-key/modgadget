package main

import (
	"testing"

	"github.com/rdon-key/modgadget"
)

func TestTypingStatePrintableRuneAndBackspace(t *testing.T) {
	var state typingState
	if !state.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyA, Rune: 'a', Action: modgadget.KeyDown}) || state.Text() != "a" {
		t.Fatalf("typed=%q", state.Text())
	}
	if state.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyB, Rune: 'b', Action: modgadget.KeyUp}) || state.Text() != "a" {
		t.Fatalf("KeyUp changed text=%q", state.Text())
	}
	if !state.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyBackspace, Action: modgadget.KeyDown}) || state.Text() != "" {
		t.Fatalf("backspace text=%q", state.Text())
	}
}

func TestTypingStateIgnoresNonPrintableSystemKey(t *testing.T) {
	var state typingState
	if state.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyF12, Action: modgadget.KeyDown, Modifiers: modgadget.ModFn}) {
		t.Fatal("system key was treated as text")
	}
}
