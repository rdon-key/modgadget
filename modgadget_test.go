package modgadget

import (
	"errors"
	"testing"
	"time"

	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/efont24"
	"github.com/rdon-key/modgadget/internal/text"
)

const testBitmap = "\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff"

type testFont struct {
	advance int16
	tooWide *bool
}

func (f testFont) Lookup(r rune) (text.Glyph, bool) {
	width := f.advance
	if f.tooWide != nil && *f.tooWide {
		width *= 2
	}
	bitmap := testBitmap
	if r == 'b' {
		bitmap = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
	}
	return text.Glyph{Width: width, Height: 8, AdvanceX: f.advance, BearingY: 8, Bitmap: bitmap}, true
}
func (testFont) Metrics() text.FontMetrics { return text.FontMetrics{Ascent: 8} }

type testBackend struct {
	width, height int16
	beginErr      error
	writeErr      error
	endErr        error
	begins        []display.Rect
	remaining     int
	writes        int
	written       int
	ends          int
	capture       []byte
}

func (b *testBackend) Size() (int16, int16) { return b.width, b.height }
func (b *testBackend) BeginRect(x, y, w, h int16) error {
	if b.beginErr != nil {
		return b.beginErr
	}
	b.begins = append(b.begins, display.Rect{X: x, Y: y, Width: w, Height: h})
	b.remaining = int(w) * int(h) * 2
	return nil
}
func (b *testBackend) WritePixels(p []byte) error {
	if b.writeErr != nil {
		return b.writeErr
	}
	b.writes++
	b.written += len(p)
	b.remaining -= len(p)
	if b.capture != nil {
		b.capture = append(b.capture, p...)
	}
	return nil
}
func (b *testBackend) EndRect() error { b.ends++; return b.endErr }

func testStyles() StyleSet {
	return StyleSet{Default: Style{Font: testFont{advance: 10}, Foreground: ColorWhite, Background: ColorBlack}, Entries: []StyleEntry{{Name: "news", Style: Style{Font: testFont{advance: 10}, Foreground: ColorGreen, Background: ColorBlack}}}}
}

func TestViewportBounds(t *testing.T) {
	b := &testBackend{width: 240, height: 135}
	g := New(b, WithStyles(testStyles()))
	whole := g.Viewport()
	if whole.bounds != (display.Rect{Width: 240, Height: 135}) {
		t.Fatalf("whole bounds = %+v", whole.bounds)
	}
	part := g.Viewport(Bounds(2, 3, 40, 12))
	if part.bounds != (display.Rect{X: 2, Y: 3, Width: 40, Height: 12}) {
		t.Fatalf("part bounds = %+v", part.bounds)
	}
	if err := part.SetText("a"); err != nil {
		t.Fatal(err)
	}
	if err := g.Render(); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range b.begins {
		if r.X == 2 && r.Y == 3 {
			found = true
		}
	}
	if !found {
		t.Fatal("bounded viewport was not forwarded to backend")
	}
}

func TestSetTextDirtyAndSameValue(t *testing.T) {
	v := New(&testBackend{width: 20, height: 10}, WithStyles(testStyles())).Viewport()
	v.dirty = false
	if err := v.SetText("ab"); err != nil {
		t.Fatal(err)
	}
	if !v.dirty {
		t.Fatal("SetText did not mark dirty")
	}
	v.dirty = false
	if err := v.SetText("ab"); err != nil {
		t.Fatal(err)
	}
	if v.dirty {
		t.Fatal("same text unexpectedly marked dirty")
	}
}

func TestHorizontalScrollTiming(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newView := func(width int16) *Viewport {
		v := New(&testBackend{width: width, height: 10}, WithStyles(testStyles())).Viewport()
		if err := v.SetText("abcd"); err != nil {
			t.Fatal(err)
		}
		return v
	}
	v := newView(20)
	v.UpdateForTest(base)
	v.UpdateForTest(base.Add(time.Second))
	if v.offset != 0 {
		t.Fatalf("disabled offset=%d", v.offset)
	}
	short := newView(50)
	short.SetHorizontalScroll(ScrollSpeed(30))
	short.UpdateForTest(base)
	short.UpdateForTest(base.Add(time.Second))
	if short.offset != 0 {
		t.Fatalf("short offset=%d", short.offset)
	}
	v.SetHorizontalScroll(ScrollSpeed(30))
	v.UpdateForTest(base)
	v.UpdateForTest(base.Add(500 * time.Millisecond))
	if v.offset != 15 {
		t.Fatalf("offset=%d want 15", v.offset)
	}
	loop := newView(20)
	loop.SetHorizontalScroll(ScrollSpeed(30), ScrollGap(5), ScrollLoop())
	loop.UpdateForTest(base)
	loop.UpdateForTest(base.Add(2 * time.Second))
	if loop.offset != 15 {
		t.Fatalf("loop offset=%d want 15", loop.offset)
	}
	a := newView(20)
	a.SetHorizontalScroll(ScrollSpeed(10), ScrollGap(7), ScrollLoop())
	a.UpdateForTest(base)
	a.UpdateForTest(base.Add(300 * time.Millisecond))
	a.UpdateForTest(base.Add(1700 * time.Millisecond))
	c := newView(20)
	c.SetHorizontalScroll(ScrollSpeed(10), ScrollGap(7), ScrollLoop())
	c.UpdateForTest(base)
	c.UpdateForTest(base.Add(1700 * time.Millisecond))
	if a.offset != c.offset {
		t.Fatalf("irregular=%d direct=%d", a.offset, c.offset)
	}
}

func (v *Viewport) UpdateForTest(now time.Time) { v.update(now) }

func TestErrorsReturned(t *testing.T) {
	g := New(&testBackend{width: 20, height: 10}, WithStyles(testStyles()))
	v := g.Viewport()
	if err := v.SetText("<bad>"); err == nil {
		t.Fatal("expected markup error")
	}
	if err := g.Render(); err == nil {
		t.Fatal("Render did not return parse error")
	}
	want := errors.New("backend failure")
	g = New(&testBackend{width: 20, height: 10, beginErr: want}, WithStyles(testStyles()))
	v = g.Viewport()
	if err := v.SetText("a"); err != nil {
		t.Fatal(err)
	}
	if err := g.Render(); !errors.Is(err, want) {
		t.Fatalf("Render error=%v", err)
	}
}

func TestDirectAndBufferedRendering(t *testing.T) {
	staticBackend := &testBackend{width: 20, height: 10}
	static := New(staticBackend, WithStyles(testStyles())).Viewport()
	if err := static.SetText("a"); err != nil {
		t.Fatal(err)
	}
	if err := static.owner.Render(); err != nil {
		t.Fatal(err)
	}
	if static.surface != nil {
		t.Fatal("static viewport allocated a surface")
	}
	if len(staticBackend.begins) < 2 {
		t.Fatalf("direct BeginRect count = %d", len(staticBackend.begins))
	}

	backend := &testBackend{width: 80, height: 30, capture: make([]byte, 0, 20*10*2)}
	v := New(backend, WithStyles(testStyles())).Viewport(Bounds(7, 9, 20, 10))
	if err := v.SetText("aaaa"); err != nil {
		t.Fatal(err)
	}
	v.SetHorizontalScroll(ScrollSpeed(30), ScrollGap(5), ScrollLoop())
	if err := v.owner.Render(); err != nil {
		t.Fatal(err)
	}
	if v.surface == nil {
		t.Fatal("scrolling viewport did not allocate a surface")
	}
	if len(v.buffer) != 20*10*2 {
		t.Fatalf("buffer length = %d", len(v.buffer))
	}
	if len(backend.begins) != 1 || backend.begins[0] != (display.Rect{X: 7, Y: 9, Width: 20, Height: 10}) {
		t.Fatalf("blit rectangles = %+v", backend.begins)
	}
	if backend.writes != 1 || backend.written != len(v.buffer) || backend.ends != 1 {
		t.Fatalf("writes=%d bytes=%d ends=%d", backend.writes, backend.written, backend.ends)
	}
	if len(backend.capture) != len(v.buffer) {
		t.Fatalf("captured bytes = %d", len(backend.capture))
	}
	for i := range v.buffer {
		if backend.capture[i] != v.buffer[i] {
			t.Fatalf("physical transfer differs from completed buffer at byte %d", i)
		}
	}
	hasForeground, hasBackground := false, false
	for i := 0; i < len(v.buffer); i += 2 {
		color := Color565(uint16(v.buffer[i])<<8 | uint16(v.buffer[i+1]))
		hasForeground = hasForeground || color == ColorWhite
		hasBackground = hasBackground || color == ColorBlack
	}
	if !hasForeground || !hasBackground {
		t.Fatalf("completed buffer foreground=%v background=%v", hasForeground, hasBackground)
	}
	bufferAddress := &v.buffer[0]
	v.dirty = true
	if err := v.owner.Render(); err != nil {
		t.Fatal(err)
	}
	if &v.buffer[0] != bufferAddress {
		t.Fatal("viewport buffer was not reused")
	}
}

func TestBufferedLoopDrawsNextCopy(t *testing.T) {
	b := &testBackend{width: 20, height: 10}
	v := New(b, WithStyles(testStyles())).Viewport()
	if err := v.SetText("aaaa"); err != nil {
		t.Fatal(err)
	}
	v.SetHorizontalScroll(ScrollSpeed(1), ScrollGap(5), ScrollLoop())
	v.offset = 35
	if err := v.owner.Render(); err != nil {
		t.Fatal(err)
	}
	offset := (10 * 2) // second copy begins at x = 40 + 5 - 35
	color := Color565(uint16(v.buffer[offset])<<8 | uint16(v.buffer[offset+1]))
	if color != ColorWhite {
		t.Fatalf("second copy pixel = %#04x", color)
	}
}

func TestBufferedErrorsKeepDirty(t *testing.T) {
	want := errors.New("blit failed")
	b := &testBackend{width: 20, height: 10, writeErr: want}
	v := New(b, WithStyles(testStyles())).Viewport()
	if err := v.SetText("aaaa"); err != nil {
		t.Fatal(err)
	}
	v.SetHorizontalScroll(ScrollSpeed(1))
	if err := v.owner.Render(); !errors.Is(err, want) {
		t.Fatalf("blit error = %v", err)
	}
	if !v.dirty {
		t.Fatal("backend error cleared dirty")
	}

	tooWide := false
	styles := testStyles()
	styles.Default.Font = testFont{advance: 10, tooWide: &tooWide}
	v = New(&testBackend{width: 20, height: 10}, WithStyles(styles)).Viewport()
	if err := v.SetText("aaaa"); err != nil {
		t.Fatal(err)
	}
	v.SetHorizontalScroll(ScrollSpeed(1))
	tooWide = true
	if err := v.owner.Render(); err == nil {
		t.Fatal("expected buffered glyph draw error")
	}
	if !v.dirty {
		t.Fatal("buffer draw error cleared dirty")
	}
}

func TestBufferedSetTextUpdatesRetainedBuffer(t *testing.T) {
	v := New(&testBackend{width: 20, height: 10}, WithStyles(testStyles())).Viewport()
	if err := v.SetText("aaaa"); err != nil {
		t.Fatal(err)
	}
	v.SetHorizontalScroll(ScrollSpeed(1))
	if err := v.owner.Render(); err != nil {
		t.Fatal(err)
	}
	address := &v.buffer[0]
	before := append([]byte(nil), v.buffer...)
	if err := v.SetText("bbbb"); err != nil {
		t.Fatal(err)
	}
	if err := v.owner.Render(); err != nil {
		t.Fatal(err)
	}
	if &v.buffer[0] != address {
		t.Fatal("SetText replaced viewport buffer")
	}
	equal := true
	for i := range before {
		if before[i] != v.buffer[i] {
			equal = false
			break
		}
	}
	if equal {
		t.Fatal("SetText did not update buffer content")
	}
}

func TestShortScrolledTextDoesNotUseBuffer(t *testing.T) {
	b := &testBackend{width: 50, height: 10}
	v := New(b, WithStyles(testStyles())).Viewport()
	if err := v.SetText("a"); err != nil {
		t.Fatal(err)
	}
	v.SetHorizontalScroll(ScrollSpeed(30), ScrollLoop())
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v.owner.Update(base)
	if err := v.owner.Render(); err != nil {
		t.Fatal(err)
	}
	count := len(b.begins)
	v.owner.Update(base.Add(10 * time.Second))
	if err := v.owner.Render(); err != nil {
		t.Fatal(err)
	}
	if v.offset != 0 || v.surface != nil || len(b.begins) != count {
		t.Fatalf("offset=%d surface=%v transfers=%d want %d", v.offset, v.surface != nil, len(b.begins), count)
	}
}

type allocationBackend struct{ remaining int }

func (*allocationBackend) Size() (int16, int16) { return 20, 10 }
func (b *allocationBackend) BeginRect(_, _, width, height int16) error {
	b.remaining = int(width) * int(height) * 2
	return nil
}
func (b *allocationBackend) WritePixels(data []byte) error { b.remaining -= len(data); return nil }
func (*allocationBackend) EndRect() error                  { return nil }

func TestBufferedSteadyRenderAllocations(t *testing.T) {
	v := New(&allocationBackend{}, WithStyles(testStyles())).Viewport()
	if err := v.SetText("aaaa"); err != nil {
		t.Fatal(err)
	}
	v.SetHorizontalScroll(ScrollSpeed(30), ScrollGap(5), ScrollLoop())
	if err := v.owner.Render(); err != nil {
		t.Fatal(err)
	}
	allocations := testing.AllocsPerRun(100, func() {
		v.dirty = true
		if err := v.owner.Render(); err != nil {
			panic(err)
		}
	})
	if allocations != 0 {
		t.Fatalf("steady Render allocations = %v", allocations)
	}
}

func TestTickerFontMetrics(t *testing.T) {
	font := NewMGFFont(efont24.Font)
	metrics := font.Metrics()
	if metrics.Ascent != 22 || metrics.Descent != 2 || metrics.LineGap != 0 || metrics.LineHeight() != 24 {
		t.Fatalf("metrics = %+v", metrics)
	}
	measurement, err := text.MeasureString(font, "ModGadgetニュース：日本語表示に成功しました。")
	if err != nil {
		t.Fatal(err)
	}
	height := int(measurement.Bounds.MaxY) - int(measurement.Bounds.MinY)
	t.Logf("ticker measurement: advance=%d bounds=%+v inkHeight=%d", measurement.Advance, measurement.Bounds, height)
	if height > 24 {
		t.Fatalf("ticker ink height = %d", height)
	}
}
