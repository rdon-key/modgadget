//go:build !tinygo

package main

import (
	_ "embed"
	"testing"

	"github.com/rdon-key/modgadget"
)

//go:embed efont24-full.mgz
var testFontData string

func TestEmbeddedFont(t *testing.T) {
	font, err := modgadget.OpenMGZ(testFontData)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range "A日你한" {
		if !font.HasGlyph(r) {
			t.Fatalf("embedded font has no U+%04X", r)
		}
	}
}
