package display

import (
	"bytes"
	"errors"
	"testing"
)

func TestSurfaceConstructorAndSize(t *testing.T) {
	var _ Backend = (*Surface)(nil)
	pixels := make([]byte, 14)
	surface, err := NewSurface(3, 2, pixels)
	if err != nil {
		t.Fatal(err)
	}
	if width, height := surface.Size(); width != 3 || height != 2 || len(surface.pixels) != 12 {
		t.Fatalf("size=%dx%d pixels=%d", width, height, len(surface.pixels))
	}
	if err := surface.BeginRect(0, 0, 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := surface.WritePixels([]byte{0x12, 0x34}); err != nil {
		t.Fatal(err)
	}
	if err := surface.EndRect(); err != nil {
		t.Fatal(err)
	}
	if pixels[0] != 0x12 || pixels[1] != 0x34 || pixels[12] != 0 || pixels[13] != 0 {
		t.Fatalf("caller pixels=%x", pixels)
	}

	for _, dimensions := range [][2]int16{{0, 2}, {2, 0}, {0, 0}} {
		zero, err := NewSurface(dimensions[0], dimensions[1], nil)
		if err != nil || zero == nil {
			t.Fatalf("zero %v: surface=%v err=%v", dimensions, zero, err)
		}
	}
	if _, err := NewSurface(1, 1, nil); err == nil {
		t.Fatal("nil short buffer succeeded")
	}
	if _, err := NewSurface(2, 2, make([]byte, 7)); err == nil {
		t.Fatal("short buffer succeeded")
	}
	if _, err := NewSurface(-1, 0, nil); err == nil {
		t.Fatal("negative width succeeded")
	}
	if _, err := NewSurface(0, -1, nil); err == nil {
		t.Fatal("negative height succeeded")
	}
}

func TestSurfaceWritesRectAndPreservesOutside(t *testing.T) {
	pixels := bytes.Repeat([]byte{0xee}, 5*4*2)
	surface, err := NewSurface(5, 4, pixels)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte{
		0x00, 0x01, 0x00, 0x02, 0x00, 0x03,
		0x01, 0x01, 0x01, 0x02, 0x01, 0x03,
	}
	original := append([]byte(nil), data...)
	if err := surface.BeginRect(1, 1, 3, 2); err != nil {
		t.Fatal(err)
	}
	if err := surface.WritePixels(data); err != nil {
		t.Fatal(err)
	}
	if err := surface.EndRect(); err != nil {
		t.Fatal(err)
	}
	want := bytes.Repeat([]byte{0xee}, len(pixels))
	copy(want[12:18], data[:6])
	copy(want[22:28], data[6:])
	if !bytes.Equal(pixels, want) {
		t.Fatalf("pixels=%x want=%x", pixels, want)
	}
	if !bytes.Equal(data, original) {
		t.Fatalf("input changed: %x", data)
	}
}

func TestSurfaceArbitraryWriteSplits(t *testing.T) {
	data := []byte{
		0x00, 0x01, 0x00, 0x02, 0x00, 0x03,
		0x01, 0x01, 0x01, 0x02, 0x01, 0x03,
	}
	tests := []struct {
		name   string
		splits []int
	}{
		{"single write", []int{12}},
		{"one byte", []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
		{"pixel split", []int{1, 3, 8}},
		{"row split", []int{4, 5, 3}},
		{"multiple rows", []int{9, 3}},
		{"odd writes", []int{5, 7}},
	}
	var reference []byte
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixels := bytes.Repeat([]byte{0xee}, 5*4*2)
			surface, err := NewSurface(5, 4, pixels)
			if err != nil {
				t.Fatal(err)
			}
			if err := surface.BeginRect(1, 1, 3, 2); err != nil {
				t.Fatal(err)
			}
			offset := 0
			for _, size := range tt.splits {
				if err := surface.WritePixels(data[offset : offset+size]); err != nil {
					t.Fatal(err)
				}
				offset += size
			}
			if err := surface.EndRect(); err != nil {
				t.Fatal(err)
			}
			if reference == nil {
				reference = append([]byte(nil), pixels...)
			} else if !bytes.Equal(pixels, reference) {
				t.Fatalf("pixels=%x want=%x", pixels, reference)
			}
		})
	}
}

func TestSurfaceFullSingleRowAndFillRect(t *testing.T) {
	pixels := make([]byte, 3*2*2)
	surface, err := NewSurface(3, 2, pixels)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.BeginRect(0, 0, 3, 2); err != nil {
		t.Fatal(err)
	}
	full := []byte{0, 1, 0, 2, 0, 3, 1, 1, 1, 2, 1, 3}
	if err := surface.WritePixels(full); err != nil {
		t.Fatal(err)
	}
	if err := surface.EndRect(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pixels, full) {
		t.Fatalf("full=%x", pixels)
	}
	if err := FillRect(surface, Rect{X: 1, Y: 1, Width: 2, Height: 1}, ColorRed, make([]byte, 4)); err != nil {
		t.Fatal(err)
	}
	want := []byte{0, 1, 0, 2, 0, 3, 1, 1, 0xf8, 0, 0xf8, 0}
	if !bytes.Equal(pixels, want) {
		t.Fatalf("fill=%x want=%x", pixels, want)
	}
}

func TestSurfaceValidationAndState(t *testing.T) {
	var nilSurface *Surface
	if width, height := nilSurface.Size(); width != 0 || height != 0 {
		t.Fatalf("nil size=%dx%d", width, height)
	}
	if err := nilSurface.BeginRect(0, 0, 0, 0); err == nil {
		t.Fatal("nil BeginRect succeeded")
	}
	if err := nilSurface.WritePixels(nil); !errors.Is(err, errSurfaceInactive) {
		t.Fatalf("nil WritePixels=%v", err)
	}
	if err := nilSurface.EndRect(); !errors.Is(err, errSurfaceInactive) {
		t.Fatalf("nil EndRect=%v", err)
	}
	if err := nilSurface.BlitTo(&surfacePhysicalBackend{}, 0, 0); err == nil {
		t.Fatal("nil BlitTo succeeded")
	}

	surface, err := NewSurface(4, 3, make([]byte, 24))
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.WritePixels(nil); !errors.Is(err, errSurfaceInactive) {
		t.Fatalf("write before begin=%v", err)
	}
	if err := surface.EndRect(); !errors.Is(err, errSurfaceInactive) {
		t.Fatalf("end before begin=%v", err)
	}
	invalid := []Rect{
		{X: -1, Width: 1, Height: 1},
		{Y: -1, Width: 1, Height: 1},
		{Width: -1, Height: 1},
		{Width: 1, Height: -1},
		{X: 3, Width: 2, Height: 1},
		{Y: 2, Width: 1, Height: 2},
	}
	for _, rect := range invalid {
		if err := surface.BeginRect(rect.X, rect.Y, rect.Width, rect.Height); err == nil {
			t.Fatalf("invalid rect succeeded: %+v", rect)
		}
	}
	if err := surface.BeginRect(1, 1, 2, 1); err != nil {
		t.Fatal(err)
	}
	if err := surface.BeginRect(0, 0, 1, 1); !errors.Is(err, errSurfaceActive) {
		t.Fatalf("double begin=%v", err)
	}
	if err := surface.WritePixels(make([]byte, 5)); !errors.Is(err, errSurfaceTooMuch) {
		t.Fatalf("too much=%v", err)
	}
	if err := surface.WritePixels([]byte{1, 2}); err != nil {
		t.Fatal(err)
	}
	if err := surface.EndRect(); !errors.Is(err, errSurfaceIncomplete) {
		t.Fatalf("incomplete=%v", err)
	}
	if err := surface.BeginRect(0, 0, 1, 1); err != nil {
		t.Fatalf("begin after incomplete=%v", err)
	}
	if err := surface.WritePixels([]byte{3, 4}); err != nil {
		t.Fatal(err)
	}
	if err := surface.EndRect(); err != nil {
		t.Fatal(err)
	}
	if err := surface.BeginRect(4, 3, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := surface.WritePixels(nil); err != nil {
		t.Fatal(err)
	}
	if err := surface.EndRect(); err != nil {
		t.Fatalf("zero area=%v", err)
	}
}

var (
	errSurfaceTestBegin = errors.New("surface test begin")
	errSurfaceTestWrite = errors.New("surface test write")
	errSurfaceTestEnd   = errors.New("surface test end")
)

type surfacePhysicalBackend struct {
	beginErr, writeErr, endErr error
	beginCalls, writeCalls     int
	endCalls                   int
	rect                       Rect
	pixels                     []byte
}

func (backend *surfacePhysicalBackend) Size() (int16, int16) { return 240, 135 }
func (backend *surfacePhysicalBackend) BeginRect(x, y, width, height int16) error {
	backend.beginCalls++
	backend.rect = Rect{X: x, Y: y, Width: width, Height: height}
	return backend.beginErr
}
func (backend *surfacePhysicalBackend) WritePixels(data []byte) error {
	backend.writeCalls++
	if backend.writeErr == nil {
		backend.pixels = append(backend.pixels, data...)
	}
	return backend.writeErr
}
func (backend *surfacePhysicalBackend) EndRect() error {
	backend.endCalls++
	return backend.endErr
}

func TestSurfaceBlitTo(t *testing.T) {
	pixels := []byte{0, 1, 0, 2, 1, 1, 1, 2}
	surface, err := NewSurface(2, 2, pixels)
	if err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), pixels...)
	physical := &surfacePhysicalBackend{}
	if err := surface.BlitTo(physical, 10, 20); err != nil {
		t.Fatal(err)
	}
	if physical.rect != (Rect{X: 10, Y: 20, Width: 2, Height: 2}) || physical.beginCalls != 1 || physical.writeCalls != 1 || physical.endCalls != 1 || !bytes.Equal(physical.pixels, pixels) {
		t.Fatalf("physical=%+v", physical)
	}
	if !bytes.Equal(pixels, original) {
		t.Fatalf("surface changed: %x", pixels)
	}
	if err := surface.BlitTo(nil, 0, 0); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("nil backend=%v", err)
	}
	if err := surface.BeginRect(0, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := surface.BlitTo(physical, 0, 0); !errors.Is(err, errSurfaceActive) {
		t.Fatalf("active=%v", err)
	}
	if err := surface.EndRect(); err != nil {
		t.Fatal(err)
	}

	zero, err := NewSurface(0, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	unused := &surfacePhysicalBackend{}
	if err := zero.BlitTo(unused, 0, 0); err != nil || unused.beginCalls+unused.writeCalls+unused.endCalls != 0 {
		t.Fatalf("zero blit backend=%+v err=%v", unused, err)
	}
}

func TestSurfaceBlitErrorsPreserveCause(t *testing.T) {
	surface, err := NewSurface(1, 1, []byte{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		backend    *surfacePhysicalBackend
		cause      error
		beginCalls int
		writeCalls int
		endCalls   int
	}{
		{"begin", &surfacePhysicalBackend{beginErr: errSurfaceTestBegin}, errSurfaceTestBegin, 1, 0, 0},
		{"write", &surfacePhysicalBackend{writeErr: errSurfaceTestWrite}, errSurfaceTestWrite, 1, 1, 0},
		{"end", &surfacePhysicalBackend{endErr: errSurfaceTestEnd}, errSurfaceTestEnd, 1, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := surface.BlitTo(tt.backend, 0, 0)
			if !errors.Is(err, tt.cause) || tt.backend.beginCalls != tt.beginCalls || tt.backend.writeCalls != tt.writeCalls || tt.backend.endCalls != tt.endCalls {
				t.Fatalf("backend=%+v err=%v", tt.backend, err)
			}
		})
	}
}
