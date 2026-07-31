package main

import (
	"testing"

	"github.com/rdon-key/modgadget/internal/display"
)

func TestDisplayForSpecialKeys(t *testing.T) {
	tests := []struct {
		row        int
		col        int
		label      string
		lineHeight int16
		ascent     int16
		foreground display.Color565
		background display.Color565
	}{
		{2, 11, "上", directionLineHeight, 22, display.ColorBlack, display.RGB565(64, 192, 255)},
		{3, 11, "下", directionLineHeight, 22, display.ColorBlack, display.RGB565(96, 224, 128)},
		{3, 10, "左", directionLineHeight, 22, display.ColorBlack, display.RGB565(255, 208, 64)},
		{3, 12, "右", directionLineHeight, 22, display.ColorWhite, display.RGB565(224, 64, 64)},
		{2, 13, "OK", regularLineHeight, 14, display.ColorBlack, display.RGB565(96, 224, 128)},
		{0, 13, "ESC", regularLineHeight, 14, display.ColorWhite, display.RGB565(224, 64, 64)},
	}
	for _, test := range tests {
		got := displayForKey(test.row, test.col)
		if got.label != test.label || got.lineHeight != test.lineHeight {
			t.Errorf("displayForKey(%d,%d) label/lineHeight = %q/%d, want %q/%d", test.row, test.col, got.label, got.lineHeight, test.label, test.lineHeight)
		}
		if got.foreground != test.foreground || got.background != test.background {
			t.Errorf("displayForKey(%d,%d) colors = %#04x/%#04x, want %#04x/%#04x", test.row, test.col, got.foreground, got.background, test.foreground, test.background)
		}
		if ascent := got.font.Metrics().Ascent; ascent != test.ascent {
			t.Errorf("displayForKey(%d,%d) font ascent = %d, want %d", test.row, test.col, ascent, test.ascent)
		}
	}
}

func TestDisplayForRegularKey(t *testing.T) {
	got := displayForKey(1, 4)
	if got.label != "row=1 col=4" {
		t.Fatalf("label = %q", got.label)
	}
	if got.font.Metrics().Ascent != 14 || got.lineHeight != regularLineHeight {
		t.Fatalf("regular font ascent/line height = %d/%d", got.font.Metrics().Ascent, got.lineHeight)
	}
	if got.foreground != display.ColorWhite || got.background != display.ColorBlack {
		t.Fatalf("regular colors = %#04x/%#04x", got.foreground, got.background)
	}
}
