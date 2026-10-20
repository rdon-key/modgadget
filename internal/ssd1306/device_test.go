package ssd1306

import (
	"errors"
	"testing"
)

type testBus struct {
	writes [][]byte
	failAt int
}

func (b *testBus) Tx(addr uint16, write, read []byte) error {
	if addr != Address || read != nil {
		return errors.New("unexpected I2C transaction")
	}
	b.writes = append(b.writes, append([]byte(nil), write...))
	if b.failAt != 0 && len(b.writes) == b.failAt {
		return errors.New("I2C failed")
	}
	return nil
}

func configured(t *testing.T) (*Device, *testBus) {
	t.Helper()
	bus := &testBus{}
	d := New(bus)
	if err := d.Configure(); err != nil {
		t.Fatal(err)
	}
	if len(bus.writes) != 3 || bus.writes[0][0] != 0x00 ||
		len(bus.writes[1]) != 7 || bus.writes[1][1] != 0x21 ||
		len(bus.writes[2]) != 1+frameBytes || bus.writes[2][0] != 0x40 {
		t.Fatalf("unexpected initialization writes: %d", len(bus.writes))
	}
	return d, bus
}

func pixelOn(d *Device, x, y int) bool {
	return d.frame[1+(y/8)*int(Width)+x]&(1<<uint(y&7)) != 0
}

func TestRectStreamsRowsAndSplitPixels(t *testing.T) {
	d, bus := configured(t)
	if w, h := d.Size(); w != Width || h != Height {
		t.Fatalf("Size = %dx%d", w, h)
	}
	if err := d.BeginRect(2, 9, 2, 2); err != nil {
		t.Fatal(err)
	}
	// White, black, black, white in row-major big-endian RGB565.
	for _, chunk := range [][]byte{
		{0xff},
		{0xff, 0, 0, 0},
		{0, 0xff, 0xff},
	} {
		if err := d.WritePixels(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.EndRect(); err != nil {
		t.Fatal(err)
	}
	if len(bus.writes) != 5 {
		t.Fatalf("writes = %d, want 5", len(bus.writes))
	}
	for _, tc := range []struct {
		x, y int
		on bool
	} {
		{2, 9, true}, {3, 9, false}, {2, 10, false},
		{3, 10, true}, {2, 8, false}, {4, 9, false},
	} {
		if got := pixelOn(d, tc.x, tc.y); got != tc.on {
			t.Errorf("pixel (%d,%d) = %v, want %v", tc.x, tc.y, got, tc.on)
		}
	}
}

func TestPartialUpdatePreservesOtherPixelsAndErasesBlack(t *testing.T) {
	d, _ := configured(t)
	if err := d.BeginRect(0, 0, 2, 1); err != nil {
		t.Fatal(err)
	}
	if err := d.WritePixels([]byte{0xf8, 0, 0, 0x1f}); err != nil {
		t.Fatal(err)
	}
	if err := d.EndRect(); err != nil {
		t.Fatal(err)
	}
	if !pixelOn(d, 0, 0) || !pixelOn(d, 1, 0) {
		t.Fatal("nonblack red and blue must both be on")
	}
	if err := d.BeginRect(0, 0, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := d.WritePixels([]byte{0, 0}); err != nil {
		t.Fatal(err)
	}
	if err := d.EndRect(); err != nil {
		t.Fatal(err)
	}
	if pixelOn(d, 0, 0) || !pixelOn(d, 1, 0) {
		t.Fatal("erasing one pixel damaged its neighbor")
	}
}

func TestInvalidAndIncompleteTransfers(t *testing.T) {
	d := New(&testBus{})
	if err := d.BeginRect(0, 0, 1, 1); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("unconfigured error = %v", err)
	}
	if err := d.Configure(); err != nil {
		t.Fatal(err)
	}
	for _, r := range [][4]int16{{-1, 0, 1, 1}, {0, 0, 0, 1},
		{128, 0, 1, 1}, {0, 63, 1, 2}, {0, 0, 129, 1}} {
		if err := d.BeginRect(r[0], r[1], r[2], r[3]); !errors.Is(err, ErrInvalidRect) {
			t.Fatalf("rect %v error = %v", r, err)
		}
	}
	if err := d.BeginRect(127, 63, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := d.BeginRect(0, 0, 1, 1); !errors.Is(err, ErrActive) {
		t.Fatalf("active error = %v", err)
	}
	if err := d.WritePixels([]byte{0xff, 0xff, 0x00}); !errors.Is(err, ErrTooMuchData) {
		t.Fatalf("overflow error = %v", err)
	}
	if err := d.WritePixels([]byte{0xff}); err != nil {
		t.Fatal(err)
	}
	if err := d.EndRect(); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("incomplete error = %v", err)
	}
	if err := d.WritePixels([]byte{0}); !errors.Is(err, ErrInactive) {
		t.Fatalf("inactive error = %v", err)
	}
	if err := d.BeginRect(127, 63, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := d.WritePixels([]byte{0xff, 0xff}); err != nil {
		t.Fatal(err)
	}
	if err := d.EndRect(); err != nil {
		t.Fatal(err)
	}
	if !pixelOn(d, 127, 63) {
		t.Fatal("right-bottom pixel missing")
	}
}

func TestI2CErrors(t *testing.T) {
	b := &testBus{failAt: 1}
	d := New(b)
	if err := d.Configure(); err == nil {
		t.Fatal("expected init error")
	}
	if err := d.BeginRect(0, 0, 1, 1); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("after init failure: %v", err)
	}

	b = &testBus{failAt: 2}
	d = New(b)
	if err := d.Configure(); err == nil {
		t.Fatal("expected initial flush error")
	}

	d, b = configured(t)
	b.failAt = 5
	if err := d.BeginRect(0, 0, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := d.WritePixels([]byte{0xff, 0xff}); err != nil {
		t.Fatal(err)
	}
	if err := d.EndRect(); err == nil {
		t.Fatal("expected draw flush error")
	}
	if err := d.BeginRect(0, 0, 1, 1); err != nil {
		t.Fatalf("device remained active after flush error: %v", err)
	}
}
