//go:build !tinygo

package main

import (
	"github.com/rdon-key/modgadget-fonts/efont24"
	"testing"
)

func TestEmbeddedFont(t *testing.T) {
	font := efont24.Font
	if !font.Valid() {
		t.Fatal("font is invalid")
	}

	for _, r := range "A日你한" {
		if !font.HasGlyph(r) {
			t.Fatalf("embedded font has no U+%04X", r)
		}
	}
}
