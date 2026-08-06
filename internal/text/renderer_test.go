package text

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget/internal/display"
)

type fakeBackend struct {
	rects                            []display.Rect
	writes                           [][]byte
	beginErr, writeErr, endErr       error
	beginCalls, writeCalls, endCalls int
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

func newFace(glyphs []testGlyphInfo, bitmap string) Font {
	return spanFace(FontMetrics{}, glyphs, bitmap)
}

func TestDrawStringEmptyReturnsInitialPen(t *testing.T) {
	b := &fakeBackend{}
	pen, err := DrawString(b, newFace(nil, ""), -7, 11, "", 0, 0, nil)
	if err != nil || pen != -7 || b.beginCalls != 0 {
		t.Fatalf("pen=%d err=%v calls=%d", pen, err, b.beginCalls)
	}
}

func TestDrawStringVariableGlyphsBearingsAndBaseline(t *testing.T) {
	face := newFace([]testGlyphInfo{
		{Rune: 'A', Width: 3, Height: 2, AdvanceX: 5, BearingX: 2, BearingY: 3},
		{Rune: 'B', BitmapOffset: 2, Width: 1, Height: 3, AdvanceX: 2, BearingX: -1},
		{Rune: 'C', BitmapOffset: 5, Width: 2, Height: 1, AdvanceX: 4, BearingY: -2},
	}, "\xa0\x40\x80\x00\x80\x80")
	b := &fakeBackend{}
	pen, err := DrawString(b, face, 10, 20, "ABC", 0x1234, 0xabcd, make([]byte, 12))
	if err != nil || pen != 21 {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
	want := []display.Rect{{X: 12, Y: 17, Width: 3, Height: 2}, {X: 14, Y: 20, Width: 1, Height: 3}, {X: 17, Y: 22, Width: 2, Height: 1}}
	if len(b.rects) != len(want) {
		t.Fatalf("rects=%v", b.rects)
	}
	for i := range want {
		if b.rects[i] != want[i] {
			t.Errorf("rect %d=%v want %v", i, b.rects[i], want[i])
		}
	}
	wantRows := [][]byte{
		{0x12, 0x34, 0xab, 0xcd, 0x12, 0x34},
		{0xab, 0xcd, 0x12, 0x34, 0xab, 0xcd},
		{0x12, 0x34},
		{0xab, 0xcd},
		{0x12, 0x34},
		{0x12, 0x34, 0xab, 0xcd},
	}
	if len(b.writes) != len(wantRows) {
		t.Fatalf("writes=%d want %d", len(b.writes), len(wantRows))
	}
	for i := range wantRows {
		if string(b.writes[i]) != string(wantRows[i]) {
			t.Fatalf("row %d=%x want %x", i, b.writes[i], wantRows[i])
		}
	}
}

func TestDrawStringIgnoresUnusedBits(t *testing.T) {
	b := &fakeBackend{}
	face := newFace([]testGlyphInfo{{Rune: 'x', Width: 9, Height: 1, AdvanceX: 9}}, "\x80\x7f")
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
	face := newFace([]testGlyphInfo{{Rune: ' ', Width: 0, Height: 3, AdvanceX: 5, BearingX: -2, BearingY: -1}}, "")
	pen, err := DrawString(b, face, 4, 6, " ", 0, 0, nil)
	if err != nil || pen != 9 || b.beginCalls != 0 {
		t.Fatalf("pen=%d err=%v calls=%d", pen, err, b.beginCalls)
	}
}

func TestDrawStringValidationAndOverflow(t *testing.T) {
	valid := newFace([]testGlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 1}}, "\x80")
	tests := []struct {
		name    string
		backend display.Backend
		face    Font
		value   string
		x, y    int16
		scratch []byte
	}{
		{"nil backend", nil, valid, "a", 0, 0, make([]byte, 2)}, {"nil font", &fakeBackend{}, nil, "a", 0, 0, make([]byte, 2)},
		{"invalid UTF-8", &fakeBackend{}, valid, string([]byte{0xff}), 0, 0, make([]byte, 2)}, {"missing glyph", &fakeBackend{}, valid, "z", 0, 0, make([]byte, 2)},
		{"scratch short", &fakeBackend{}, valid, "a", 0, 0, make([]byte, 1)},
		{"pen plus bearing overflow", &fakeBackend{}, newFace([]testGlyphInfo{{Rune: 'a', Width: 1, Height: 1, BearingX: 1}}, "\x80"), "a", math.MaxInt16, 0, make([]byte, 2)},
		{"baseline minus bearing overflow", &fakeBackend{}, newFace([]testGlyphInfo{{Rune: 'a', Width: 1, Height: 1, BearingY: 1}}, "\x80"), "a", 0, math.MinInt16, make([]byte, 2)},
		{"pen plus advance overflow", &fakeBackend{}, newFace([]testGlyphInfo{{Rune: 'a', AdvanceX: 1}}, ""), "a", math.MaxInt16, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DrawString(tt.backend, tt.face, tt.x, tt.y, tt.value, 1, 0, tt.scratch); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDrawStringMissingGlyphReturnsCurrentPen(t *testing.T) {
	face := newFace([]testGlyphInfo{{Rune: 'a', AdvanceX: 3}}, "")
	pen, err := DrawString(&fakeBackend{}, face, 7, 0, "az", 0, 0, nil)
	if err == nil || pen != 10 || !strings.Contains(err.Error(), "U+007A") {
		t.Fatalf("pen=%d err=%v", pen, err)
	}
}

func TestDrawStringBackendErrors(t *testing.T) {
	sentinel := errors.New("backend failure")
	face := newFace([]testGlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 2}}, "\x80")
	tests := []struct {
		name      string
		configure func(*fakeBackend)
	}{{"BeginRect", func(b *fakeBackend) { b.beginErr = sentinel }}, {"WritePixels", func(b *fakeBackend) { b.writeErr = sentinel }}, {"EndRect", func(b *fakeBackend) { b.endErr = sentinel }}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &fakeBackend{}
			tt.configure(b)
			pen, err := DrawString(b, face, 5, 0, "a", 1, 0, make([]byte, 2))
			if pen != 5 || !errors.Is(err, sentinel) {
				t.Fatalf("pen=%d err=%v", pen, err)
			}
		})
	}
}

func TestDrawStringReusesScratch(t *testing.T) {
	face := newFace([]testGlyphInfo{{Rune: 'a', Width: 1, Height: 1, AdvanceX: 1}, {Rune: 'b', BitmapOffset: 1, Width: 1, Height: 1, AdvanceX: 1}}, "\x80\x00")
	b := &fakeBackend{}
	scratch := []byte{0xee, 0xee}
	pen, err := DrawString(b, face, 0, 0, "ab", 0xffff, 0, scratch)
	if err != nil || pen != 2 || len(b.writes) != 2 {
		t.Fatalf("pen=%d err=%v writes=%v", pen, err, b.writes)
	}
	if string(b.writes[0]) != "\xff\xff" || string(b.writes[1]) != "\x00\x00" || string(scratch) != "\x00\x00" {
		t.Fatalf("writes=%x scratch=%x", b.writes, scratch)
	}
}
