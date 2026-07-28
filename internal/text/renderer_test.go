package text

import (
	"errors"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget-fonts/font"
	"github.com/rdon-key/modgadget/internal/display"
)

type fakeBackend struct {
	rects      []display.Rect
	writes     [][]byte
	beginErr   error
	writeErr   error
	endErr     error
	beginCalls int
	writeCalls int
	endCalls   int
}

func (b *fakeBackend) Size() (int16, int16) { return 320, 240 }
func (b *fakeBackend) BeginRect(x, y, width, height int16) error {
	b.beginCalls++
	b.rects = append(b.rects, display.Rect{X: x, Y: y, Width: width, Height: height})
	return b.beginErr
}
func (b *fakeBackend) WritePixels(data []byte) error {
	b.writeCalls++
	b.writes = append(b.writes, append([]byte(nil), data...))
	return b.writeErr
}
func (b *fakeBackend) EndRect() error { b.endCalls++; return b.endErr }

func newFace(glyphs []font.GlyphInfo, bitmap string) *font.Font {
	f := font.New(font.Metrics{}, glyphs, bitmap)
	return &f
}

func TestDrawStringEmpty(t *testing.T) {
	b := &fakeBackend{}
	advance, err := DrawString(b, newFace(nil, ""), 0, 0, "", 0, 0, nil)
	if err != nil || advance != 0 || b.beginCalls != 0 {
		t.Fatalf("advance=%d err=%v calls=%d", advance, err, b.beginCalls)
	}
}

func TestDrawStringExpandsAndPositionsGlyphs(t *testing.T) {
	face := newFace([]font.GlyphInfo{
		{Rune: 'A', Width: 3, Height: 2, XOffset: 2, YOffset: -1, Advance: 4},
		{Rune: '\u3042', BitmapOffset: 2, Width: 1, Height: 1, XOffset: -1, YOffset: 3, Advance: 2},
	}, "\xa0\x40\x80")
	b := &fakeBackend{}
	scratch := make([]byte, 12)
	advance, err := DrawString(b, face, 10, 20, "A\u3042", 0x1234, 0xabcd, scratch)
	if err != nil || advance != 6 {
		t.Fatalf("advance=%d err=%v", advance, err)
	}
	wantRects := []display.Rect{{X: 12, Y: 19, Width: 3, Height: 2}, {X: 13, Y: 23, Width: 1, Height: 1}}
	if len(b.rects) != len(wantRects) || b.rects[0] != wantRects[0] || b.rects[1] != wantRects[1] {
		t.Fatalf("rects=%v, want %v", b.rects, wantRects)
	}
	wantWrites := [][]byte{
		{0x12, 0x34, 0xab, 0xcd, 0x12, 0x34},
		{0xab, 0xcd, 0x12, 0x34, 0xab, 0xcd},
		{0x12, 0x34},
	}
	if len(b.writes) != len(wantWrites) {
		t.Fatalf("writes=%v", b.writes)
	}
	for i := range wantWrites {
		if string(b.writes[i]) != string(wantWrites[i]) {
			t.Errorf("write %d=%x, want %x", i, b.writes[i], wantWrites[i])
		}
	}
}

func TestDrawStringIgnoresUnusedBits(t *testing.T) {
	b := &fakeBackend{}
	face := newFace([]font.GlyphInfo{{Rune: 'x', Width: 9, Height: 1, Advance: 9}}, "\x80\x7f")
	_, err := DrawString(b, face, 0, 0, "x", 0xffff, 0, make([]byte, 18))
	if err != nil {
		t.Fatal(err)
	}
	if got := b.writes[0]; string(got) != string([]byte{0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("pixels=%x", got)
	}
}

func TestDrawStringEmptyGlyphAdvancesWithoutDrawing(t *testing.T) {
	b := &fakeBackend{}
	face := newFace([]font.GlyphInfo{{Rune: ' ', Width: 0, Height: 3, Advance: 5}}, "")
	advance, err := DrawString(b, face, 0, 0, " ", 0, 0, nil)
	if err != nil || advance != 5 || b.beginCalls != 0 {
		t.Fatalf("advance=%d err=%v calls=%d", advance, err, b.beginCalls)
	}
}

func TestDrawStringValidation(t *testing.T) {
	valid := newFace([]font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, Advance: 1}}, "\x80")
	tests := []struct {
		name    string
		backend display.Backend
		face    *font.Font
		value   string
		x, y    int16
		scratch []byte
	}{
		{"nil backend", nil, valid, "a", 0, 0, make([]byte, 2)},
		{"nil font", &fakeBackend{}, nil, "a", 0, 0, make([]byte, 2)},
		{"invalid UTF-8", &fakeBackend{}, valid, string([]byte{0xff}), 0, 0, make([]byte, 2)},
		{"missing", &fakeBackend{}, valid, "z", 0, 0, make([]byte, 2)},
		{"negative width", &fakeBackend{}, newFace([]font.GlyphInfo{{Rune: 'a', Width: -1, Height: 1}}, ""), "a", 0, 0, nil},
		{"negative height", &fakeBackend{}, newFace([]font.GlyphInfo{{Rune: 'a', Width: 1, Height: -1}}, ""), "a", 0, 0, nil},
		{"bitmap short", &fakeBackend{}, newFace([]font.GlyphInfo{{Rune: 'a', Width: 8, Height: 2}}, "\x80"), "a", 0, 0, make([]byte, 32)},
		{"scratch short", &fakeBackend{}, valid, "a", 0, 0, make([]byte, 1)},
		{"X overflow", &fakeBackend{}, newFace([]font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, XOffset: 1}}, "\x80"), "a", 32767, 0, make([]byte, 2)},
		{"Y overflow", &fakeBackend{}, newFace([]font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1, YOffset: -1}}, "\x80"), "a", 0, -32768, make([]byte, 2)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DrawString(tt.backend, tt.face, tt.x, tt.y, tt.value, 1, 0, tt.scratch)
			if err == nil {
				t.Fatal("expected error")
			}
			if (tt.name == "missing" || strings.HasPrefix(tt.name, "negative") || tt.name == "bitmap short" || strings.HasSuffix(tt.name, "overflow")) && !strings.Contains(err.Error(), "U+") {
				t.Fatalf("error lacks rune: %v", err)
			}
		})
	}
}

func TestDrawStringBackendErrors(t *testing.T) {
	sentinel := errors.New("backend failure")
	face := newFace([]font.GlyphInfo{{Rune: 'a', Width: 1, Height: 1}}, "\x80")
	tests := []struct {
		name      string
		configure func(*fakeBackend)
	}{
		{"BeginRect", func(b *fakeBackend) { b.beginErr = sentinel }},
		{"WritePixels", func(b *fakeBackend) { b.writeErr = sentinel }},
		{"EndRect", func(b *fakeBackend) { b.endErr = sentinel }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &fakeBackend{}
			tt.configure(b)
			_, err := DrawString(b, face, 0, 0, "a", 1, 0, make([]byte, 2))
			if !errors.Is(err, sentinel) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestDrawStringReusesScratch(t *testing.T) {
	face := newFace([]font.GlyphInfo{
		{Rune: 'a', Width: 1, Height: 1, Advance: 1},
		{Rune: 'b', BitmapOffset: 1, Width: 1, Height: 1, Advance: 1},
	}, "\x80\x00")
	b := &fakeBackend{}
	scratch := []byte{0xee, 0xee}
	advance, err := DrawString(b, face, 0, 0, "ab", 0xffff, 0, scratch)
	if err != nil || advance != 2 || len(b.writes) != 2 {
		t.Fatalf("advance=%d err=%v writes=%v", advance, err, b.writes)
	}
	if string(b.writes[0]) != "\xff\xff" || string(b.writes[1]) != "\x00\x00" || string(scratch) != "\x00\x00" {
		t.Fatalf("writes=%x scratch=%x", b.writes, scratch)
	}
}

func TestDrawStringAdvanceOverflow(t *testing.T) {
	face := newFace([]font.GlyphInfo{{Rune: 'a', Width: 0, Height: 0, Advance: 32767}}, "")
	_, err := DrawString(&fakeBackend{}, face, 0, 0, strings.Repeat("a", 65539), 0, 0, nil)
	if err == nil {
		t.Fatal("expected advance overflow")
	}
}
