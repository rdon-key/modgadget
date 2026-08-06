package main

import (
	"strings"
	"testing"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/efont16"
	"github.com/rdon-key/modgadget/internal/text/markup"
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

func TestEscapeMarkupForDisplay(t *testing.T) {
	tests := []struct {
		name, raw, display string
	}{
		{name: "ordinary", raw: "abc", display: "abc"},
		{name: "literal less-than", raw: "<", display: "<<"},
		{name: "tag-like input", raw: "<b>hello</b>", display: "<<b>hello<</b>"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var state typingState
			for _, r := range test.raw {
				if !state.HandleKey(modgadget.KeyEvent{Rune: r, Action: modgadget.KeyDown}) {
					t.Fatalf("Rune %q was not accepted", r)
				}
			}
			if got := state.Text(); got != test.raw {
				t.Fatalf("raw=%q want=%q", got, test.raw)
			}
			if got := escapeMarkup(state.Text()); got != test.display {
				t.Fatalf("display markup=%q want=%q", got, test.display)
			}
		})
	}
}

func TestEscapedTagLikeInputParsesAsPlainText(t *testing.T) {
	const raw = "<b>hello</b>"
	parser := markup.Parser{Styles: modgadget.StyleSet{Default: modgadget.Style{
		Font: modgadget.NewMGFFont(efont16.Font),
	}}}
	spans, err := parser.Parse(escapeMarkup(raw))
	if err != nil {
		t.Fatal(err)
	}
	var displayed strings.Builder
	for _, span := range spans {
		if span.Bold {
			t.Fatalf("input was interpreted as Bold: %+v", spans)
		}
		displayed.WriteString(span.Value)
	}
	if got := displayed.String(); got != raw {
		t.Fatalf("displayed=%q want=%q", got, raw)
	}
}

func TestLessThanBackspaceClearsRawAndDisplayText(t *testing.T) {
	var state typingState
	state.HandleKey(modgadget.KeyEvent{Rune: '<', Action: modgadget.KeyDown})
	if state.Text() != "<" || escapeMarkup(state.Text()) != "<<" {
		t.Fatalf("before backspace raw=%q display=%q", state.Text(), escapeMarkup(state.Text()))
	}
	state.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyBackspace, Action: modgadget.KeyDown})
	if state.Text() != "" || escapeMarkup(state.Text()) != "" {
		t.Fatalf("after backspace raw=%q display=%q", state.Text(), escapeMarkup(state.Text()))
	}
}

func TestTypingStateIgnoresNonPrintableSystemKey(t *testing.T) {
	var state typingState
	if state.HandleKey(modgadget.KeyEvent{Code: modgadget.KeyF12, Action: modgadget.KeyDown, Modifiers: modgadget.ModFn}) {
		t.Fatal("system key was treated as text")
	}
}
