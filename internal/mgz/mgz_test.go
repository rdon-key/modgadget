package mgz

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"strings"
	"testing"
)

type testGlyph struct {
	cp            uint32
	width, height byte
	bitmap        string
}

func makeFont(t *testing.T, compressed bool) string {
	t.Helper()
	gs := []testGlyph{{'A', 8, 1, "aaaaaaaaaaaaaaaa"}, {'B', 8, 1, "bbbbbbbbbbbbbbbb"}, {0x3042, 8, 1, "cccccccccccccccc"}}
	// Deliberately use bitmap lengths larger than dimensions, then describe a
	// matching 128x1 bitmap so streams compress well.
	for i := range gs {
		gs[i].width = 128
	}
	gpb := uint16(2)
	raws := []string{gs[0].bitmap + gs[1].bitmap, gs[2].bitmap}
	stored := make([][]byte, 2)
	flags := make([]uint32, 2)
	for i, raw := range raws {
		stored[i] = []byte(raw)
		if compressed {
			var b bytes.Buffer
			w, _ := flate.NewWriter(&b, flate.BestCompression)
			_, _ = w.Write([]byte(raw))
			_ = w.Close()
			stored[i] = b.Bytes()
			flags[i] = 1
		}
	}
	gt := 64
	bt := gt + len(gs)*16
	bd := bt + len(raws)*16
	size := bd + len(stored[0]) + len(stored[1])
	d := make([]byte, size)
	copy(d, "MGZ1")
	le := binary.LittleEndian
	le.PutUint16(d[4:], 1)
	le.PutUint16(d[6:], 64)
	copy(d[8:], "testfullJP")
	le.PutUint32(d[20:], uint32(size))
	le.PutUint32(d[24:], uint32(len(gs)))
	le.PutUint32(d[28:], 2)
	le.PutUint16(d[32:], gpb)
	le.PutUint16(d[34:], 1)
	d[36], d[37], d[38], d[39], d[40] = 10, 2, 1, 128, 1
	le.PutUint32(d[44:], uint32(gt))
	le.PutUint32(d[48:], uint32(bt))
	le.PutUint32(d[52:], uint32(bd))
	offs := []uint16{0, 16, 0}
	for i, g := range gs {
		p := gt + i*16
		le.PutUint32(d[p:], g.cp)
		d[p+4], d[p+5] = g.width, g.height
		le.PutUint16(d[p+6:], uint16(16))
		le.PutUint16(d[p+12:], 16)
		le.PutUint16(d[p+14:], offs[i])
	}
	dataOff := bd
	for i := range raws {
		p := bt + i*16
		le.PutUint32(d[p:], uint32(dataOff))
		le.PutUint32(d[p+4:], uint32(len(stored[i])))
		le.PutUint32(d[p+8:], uint32(len(raws[i])))
		le.PutUint32(d[p+12:], flags[i])
		copy(d[dataOff:], stored[i])
		dataOff += len(stored[i])
	}
	return string(d)
}

func makeBlockFont(t *testing.T, blocks int) string {
	t.Helper()
	const rawLen = 16
	gt := HeaderSize
	bt := gt + blocks*GlyphEntrySize
	bd := bt + blocks*BlockEntrySize
	stored := make([][]byte, blocks)
	size := bd
	for i := range stored {
		var compressed bytes.Buffer
		w, err := flate.NewWriter(&compressed, flate.BestCompression)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(strings.Repeat(string(rune('a'+i)), rawLen))); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		stored[i] = compressed.Bytes()
		size += len(stored[i])
	}

	d := make([]byte, size)
	copy(d, "MGZ1")
	le := binary.LittleEndian
	le.PutUint16(d[4:], Version1)
	le.PutUint16(d[6:], HeaderSize)
	copy(d[8:], "testfullJP")
	le.PutUint32(d[20:], uint32(size))
	le.PutUint32(d[24:], uint32(blocks))
	le.PutUint32(d[28:], uint32(blocks))
	le.PutUint16(d[32:], 1)
	le.PutUint16(d[34:], CodecDeflate)
	d[36], d[37], d[38], d[39], d[40] = 10, 2, 1, 128, 1
	le.PutUint32(d[44:], uint32(gt))
	le.PutUint32(d[48:], uint32(bt))
	le.PutUint32(d[52:], uint32(bd))

	dataOff := bd
	for i := 0; i < blocks; i++ {
		gp := gt + i*GlyphEntrySize
		le.PutUint32(d[gp:], uint32('A'+i))
		d[gp+4], d[gp+5] = 128, 1
		le.PutUint16(d[gp+6:], rawLen)
		le.PutUint16(d[gp+12:], rawLen)

		bp := bt + i*BlockEntrySize
		le.PutUint32(d[bp:], uint32(dataOff))
		le.PutUint32(d[bp+4:], uint32(len(stored[i])))
		le.PutUint32(d[bp+8:], rawLen)
		le.PutUint32(d[bp+12:], blockDeflated)
		copy(d[dataOff:], stored[i])
		dataOff += len(stored[i])
	}
	return string(d)
}

func TestOpenLookupRawAndDeflate(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		f, err := Open(makeFont(t, compressed))
		if err != nil {
			t.Fatal(err)
		}
		if err := f.ValidateAll(); err != nil {
			t.Fatal(err)
		}
		g, ok := f.Lookup('A')
		if !ok || g.Bitmap != "aaaaaaaaaaaaaaaa" || g.AdvanceX != 16 {
			t.Fatalf("glyph=%+v ok=%v", g, ok)
		}
		if _, ok = f.Lookup('Z'); ok {
			t.Fatal("missing glyph found")
		}
		if g, ok = f.Lookup('あ'); !ok || g.Bitmap != "cccccccccccccccc" {
			t.Fatalf("Japanese glyph=%+v %v", g, ok)
		}
	}
}

func TestCacheReuseAndBitmapLifetime(t *testing.T) {
	f := MustOpen(makeFont(t, true))
	a, _ := f.Lookup('A')
	if f.inflations != 1 {
		t.Fatalf("inflations=%d", f.inflations)
	}
	_, _ = f.Lookup('B')
	if f.inflations != 1 {
		t.Fatalf("same block inflations=%d", f.inflations)
	}
	_, _ = f.Lookup('あ')
	if f.inflations != 2 {
		t.Fatalf("new block inflations=%d", f.inflations)
	}
	if a.Bitmap != "aaaaaaaaaaaaaaaa" {
		t.Fatalf("expired bitmap %q", a.Bitmap)
	}
}

func TestCacheAlternatingBlocksDoesNotReinflate(t *testing.T) {
	f := MustOpen(makeBlockFont(t, 2))
	for i := 0; i < 2; i++ {
		if _, ok := f.Lookup(rune('A' + i%2)); !ok {
			t.Fatalf("lookup %d failed", i)
		}
	}
	lookup := 0
	allocations := testing.AllocsPerRun(100, func() {
		if _, ok := f.Lookup(rune('A' + lookup%2)); !ok {
			t.Fatalf("lookup %d failed", lookup)
		}
		lookup++
	})
	if allocations != 0 {
		t.Fatalf("cache-hit allocations=%v, want 0", allocations)
	}
	if f.inflations != 2 {
		t.Fatalf("inflations=%d, want 2", f.inflations)
	}
}

func TestCacheLRUEviction(t *testing.T) {
	f := MustOpen(makeBlockFont(t, 5))
	var evicted Glyph
	for r := 'A'; r <= 'D'; r++ {
		g, ok := f.Lookup(r)
		if !ok {
			t.Fatalf("lookup %q failed", r)
		}
		if r == 'B' {
			evicted = g
		}
	}
	_, _ = f.Lookup('A') // Make A most recently used; B is now the LRU block.
	_, _ = f.Lookup('E') // Evict B.
	if evicted.Bitmap != strings.Repeat("b", 16) {
		t.Fatalf("expired bitmap %q", evicted.Bitmap)
	}
	_, _ = f.Lookup('A') // A must still be cached.
	if f.inflations != 5 {
		t.Fatalf("inflations before evicted lookup=%d, want 5", f.inflations)
	}
	_, _ = f.Lookup('B')
	if f.inflations != 6 {
		t.Fatalf("inflations after evicted lookup=%d, want 6", f.inflations)
	}
}

func TestValidateAllDoesNotChangeCache(t *testing.T) {
	f := MustOpen(makeBlockFont(t, 5))
	_, _ = f.Lookup('C')
	want := f.cache
	wantLen, wantInflations := f.cacheLen, f.inflations
	if err := f.ValidateAll(); err != nil {
		t.Fatal(err)
	}
	if f.cache != want || f.cacheLen != wantLen || f.inflations != wantInflations {
		t.Fatalf("ValidateAll changed cache or inflation count")
	}
}

func TestInvalidHeaderAndOffsets(t *testing.T) {
	for _, change := range []func([]byte){func(d []byte) { d[0] = 'X' }, func(d []byte) { binary.LittleEndian.PutUint32(d[48:], 65) }, func(d []byte) { d[41] = 1 }} {
		d := []byte(makeFont(t, false))
		change(d)
		if _, err := Open(string(d)); err == nil {
			t.Fatal("invalid file opened")
		}
	}
}

func TestBrokenDeflateIsDeferredUntilLookup(t *testing.T) {
	d := []byte(makeFont(t, true))
	bt := 64 + 3*16
	n := binary.LittleEndian.Uint32(d[bt+4:])
	d = d[:len(d)-1]
	binary.LittleEndian.PutUint32(d[20:], uint32(len(d))) // Truncate the last block, whose entry follows the first.
	p := bt + 16
	binary.LittleEndian.PutUint32(d[p+4:], binary.LittleEndian.Uint32(d[p+4:])-1)
	_ = n
	f, err := Open(string(d))
	if err != nil {
		t.Fatalf("Open expanded DEFLATE: %v", err)
	}
	if f.inflations != 0 {
		t.Fatalf("Open inflations=%d", f.inflations)
	}
	if _, ok := f.Lookup('あ'); ok {
		t.Fatal("Lookup accepted broken DEFLATE")
	}
	if err := f.ValidateAll(); err == nil || !strings.Contains(err.Error(), "inflate block") {
		t.Fatalf("ValidateAll err=%v", err)
	}
}

func replaceLastDeflate(t *testing.T, raw string) string {
	t.Helper()
	d := []byte(makeFont(t, true))
	var compressed bytes.Buffer
	w, err := flate.NewWriter(&compressed, flate.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	entry := 64 + 3*16 + 16
	offset := int(binary.LittleEndian.Uint32(d[entry:]))
	d = append(d[:offset], compressed.Bytes()...)
	binary.LittleEndian.PutUint32(d[entry+4:], uint32(compressed.Len()))
	binary.LittleEndian.PutUint32(d[20:], uint32(len(d)))
	return string(d)
}

func TestDeflateOutputLength(t *testing.T) {
	for _, tt := range []struct{ name, raw, errorText string }{
		{"short", strings.Repeat("c", 15), "want 16"},
		{"long", strings.Repeat("c", 17), "exceeds raw length"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Open(replaceLastDeflate(t, tt.raw))
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := f.Lookup(rune(0x3042)); ok {
				t.Fatal("Lookup accepted invalid output length")
			}
			if err := f.ValidateAll(); err == nil || !strings.Contains(err.Error(), tt.errorText) {
				t.Fatalf("ValidateAll err=%v", err)
			}
		})
	}
}

func TestOpenDoesNotExpandDeflate(t *testing.T) {
	f, err := Open(makeFont(t, true))
	if err != nil {
		t.Fatal(err)
	}
	if f.cacheLen != 0 || f.inflations != 0 {
		t.Fatalf("cache entries=%d inflations=%d", f.cacheLen, f.inflations)
	}
}
