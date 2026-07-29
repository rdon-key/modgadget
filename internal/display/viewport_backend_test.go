package display

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

var (
	errPhysicalBegin = errors.New("physical begin failure")
	errPhysicalWrite = errors.New("physical write failure")
	errPhysicalEnd   = errors.New("physical end failure")
)

type viewportPhysicalBackend struct {
	width, height int16
	active        bool
	remaining     int
	beginErr      error
	writeErr      error
	endErr        error
	beginCalls    int
	writeCalls    int
	endCalls      int
	rects         []Rect
	pixels        []byte
}

func (backend *viewportPhysicalBackend) Size() (int16, int16) { return backend.width, backend.height }
func (backend *viewportPhysicalBackend) BeginRect(x, y, width, height int16) error {
	backend.beginCalls++
	if backend.beginErr != nil {
		return backend.beginErr
	}
	if backend.active {
		return errors.New("physical transfer already active")
	}
	backend.active = true
	backend.remaining = int(width) * int(height) * 2
	backend.rects = append(backend.rects, Rect{X: x, Y: y, Width: width, Height: height})
	return nil
}
func (backend *viewportPhysicalBackend) WritePixels(data []byte) error {
	backend.writeCalls++
	if !backend.active {
		return errors.New("physical transfer inactive")
	}
	if len(data)&1 != 0 {
		return errors.New("physical write is odd")
	}
	if backend.writeErr != nil {
		backend.active = false
		backend.remaining = 0
		return backend.writeErr
	}
	if len(data) > backend.remaining {
		return errors.New("physical write exceeds rectangle")
	}
	backend.pixels = append(backend.pixels, data...)
	backend.remaining -= len(data)
	return nil
}
func (backend *viewportPhysicalBackend) EndRect() error {
	backend.endCalls++
	if !backend.active {
		return errors.New("physical transfer inactive")
	}
	complete := backend.remaining == 0
	backend.active = false
	backend.remaining = 0
	if backend.endErr != nil {
		return backend.endErr
	}
	if !complete {
		return errors.New("physical transfer incomplete")
	}
	return nil
}

func TestViewportBackendImplementsBackend(t *testing.T) {
	var _ Backend = (*ViewportBackend)(nil)
}

func TestNewViewportBackend(t *testing.T) {
	viewport, err := NewViewport(Rect{X: 10, Y: 20, Width: 4, Height: 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewViewportBackend(nil, viewport); err == nil || !errors.Is(err, ErrNilBackend) {
		t.Fatalf("nil backend err=%v", err)
	}
	physical := &viewportPhysicalBackend{}
	backend, err := NewViewportBackend(physical, viewport)
	if err != nil {
		t.Fatal(err)
	}
	if width, height := backend.Size(); width != 4 || height != 3 {
		t.Fatalf("size=%dx%d", width, height)
	}
}

func TestViewportBackendCrop(t *testing.T) {
	viewport, err := NewViewport(Rect{X: 10, Y: 20, Width: 4, Height: 3})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		local Rect
		dst   Rect
		sx    int
		sy    int
		vw    int
		vh    int
	}{
		{"fully visible", Rect{Width: 4, Height: 3}, Rect{X: 10, Y: 20, Width: 4, Height: 3}, 0, 0, 4, 3},
		{"negative local and left clip", Rect{X: -1, Width: 4, Height: 3}, Rect{X: 10, Y: 20, Width: 3, Height: 3}, 1, 0, 3, 3},
		{"right clip", Rect{X: 1, Width: 4, Height: 3}, Rect{X: 11, Y: 20, Width: 3, Height: 3}, 0, 0, 3, 3},
		{"top clip", Rect{Y: -1, Width: 4, Height: 3}, Rect{X: 10, Y: 20, Width: 4, Height: 2}, 0, 1, 4, 2},
		{"bottom clip", Rect{Y: 1, Width: 4, Height: 3}, Rect{X: 10, Y: 21, Width: 4, Height: 2}, 0, 0, 4, 2},
		{"left top", Rect{X: -1, Y: -1, Width: 4, Height: 3}, Rect{X: 10, Y: 20, Width: 3, Height: 2}, 1, 1, 3, 2},
		{"right bottom", Rect{X: 1, Y: 1, Width: 4, Height: 3}, Rect{X: 11, Y: 21, Width: 3, Height: 2}, 0, 0, 3, 2},
		{"all sides", Rect{X: -1, Y: -1, Width: 6, Height: 5}, Rect{X: 10, Y: 20, Width: 4, Height: 3}, 1, 1, 4, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			physical := &viewportPhysicalBackend{}
			backend, err := NewViewportBackend(physical, viewport)
			if err != nil {
				t.Fatal(err)
			}
			pixels := identifiedPixels(int(tt.local.Width), int(tt.local.Height))
			original := append([]byte(nil), pixels...)
			if err := backend.BeginRect(tt.local.X, tt.local.Y, tt.local.Width, tt.local.Height); err != nil {
				t.Fatal(err)
			}
			if err := backend.WritePixels(pixels); err != nil {
				t.Fatal(err)
			}
			if err := backend.EndRect(); err != nil {
				t.Fatal(err)
			}
			wantPixels := croppedPixels(pixels, int(tt.local.Width), tt.sx, tt.sy, tt.vw, tt.vh)
			if !reflect.DeepEqual(physical.rects, []Rect{tt.dst}) || !reflect.DeepEqual(physical.pixels, wantPixels) {
				t.Fatalf("rects=%v pixels=%x want=%v/%x", physical.rects, physical.pixels, tt.dst, wantPixels)
			}
			if !reflect.DeepEqual(pixels, original) {
				t.Fatalf("input changed: got=%x want=%x", pixels, original)
			}
		})
	}
}

func TestViewportBackendArbitraryWriteSplits(t *testing.T) {
	viewport, err := NewViewport(Rect{X: 10, Y: 20, Width: 3, Height: 2})
	if err != nil {
		t.Fatal(err)
	}
	local := Rect{X: -1, Y: -1, Width: 5, Height: 4}
	pixels := identifiedPixels(5, 4)
	want := croppedPixels(pixels, 5, 1, 1, 3, 2)
	tests := []struct {
		name   string
		chunks []int
	}{
		{"one write", []int{len(pixels)}},
		{"one byte writes", oneByteChunks(len(pixels))},
		{"pixel row and visible boundary splits", []int{1, 2, 4, 3, 7, 5, 6, 12}},
		{"multiple rows in writes", []int{13, 19, 8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			physical := &viewportPhysicalBackend{}
			backend, err := NewViewportBackend(physical, viewport)
			if err != nil {
				t.Fatal(err)
			}
			if err := backend.BeginRect(local.X, local.Y, local.Width, local.Height); err != nil {
				t.Fatal(err)
			}
			offset := 0
			for _, size := range tt.chunks {
				if offset+size > len(pixels) {
					size = len(pixels) - offset
				}
				if err := backend.WritePixels(pixels[offset : offset+size]); err != nil {
					t.Fatal(err)
				}
				offset += size
				if offset == len(pixels) {
					break
				}
			}
			if offset != len(pixels) {
				t.Fatalf("consumed=%d want=%d", offset, len(pixels))
			}
			if err := backend.EndRect(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(physical.pixels, want) {
				t.Fatalf("pixels=%x want=%x", physical.pixels, want)
			}
		})
	}
}

func TestViewportBackendInvisibleAndZeroArea(t *testing.T) {
	viewport, err := NewViewport(Rect{Width: 4, Height: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, local := range []Rect{{X: 5, Width: 2, Height: 2}, {Width: 0, Height: 2}, {Width: 2, Height: 0}} {
		physical := &viewportPhysicalBackend{}
		backend, err := NewViewportBackend(physical, viewport)
		if err != nil {
			t.Fatal(err)
		}
		if err := backend.BeginRect(local.X, local.Y, local.Width, local.Height); err != nil {
			t.Fatal(err)
		}
		pixels := identifiedPixels(int(local.Width), int(local.Height))
		if err := backend.WritePixels(pixels); err != nil {
			t.Fatal(err)
		}
		if err := backend.EndRect(); err != nil {
			t.Fatal(err)
		}
		if physical.beginCalls != 0 || physical.writeCalls != 0 || physical.endCalls != 0 {
			t.Fatalf("local=%+v calls=%d/%d/%d", local, physical.beginCalls, physical.writeCalls, physical.endCalls)
		}
	}
}

func TestViewportBackendStateAndMultipleSequences(t *testing.T) {
	viewport, err := NewViewport(Rect{Width: 2, Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	physical := &viewportPhysicalBackend{}
	backend, err := NewViewportBackend(physical, viewport)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.WritePixels(nil); !errors.Is(err, ErrViewportBackendInactive) {
		t.Fatalf("write before begin err=%v", err)
	}
	if err := backend.EndRect(); !errors.Is(err, ErrViewportBackendInactive) {
		t.Fatalf("end before begin err=%v", err)
	}
	if err := backend.BeginRect(0, 0, 2, 1); err != nil {
		t.Fatal(err)
	}
	if err := backend.BeginRect(0, 0, 2, 1); !errors.Is(err, ErrViewportBackendActive) {
		t.Fatalf("double begin err=%v", err)
	}
	if err := backend.WritePixels([]byte{0, 0, 0, 1, 0}); !errors.Is(err, ErrViewportBackendTooMuch) {
		t.Fatalf("too much err=%v", err)
	}
	if err := backend.WritePixels([]byte{0, 0}); err != nil {
		t.Fatal(err)
	}
	if err := backend.EndRect(); !errors.Is(err, ErrViewportBackendIncomplete) {
		t.Fatalf("incomplete err=%v", err)
	}
	for sequence := 0; sequence < 2; sequence++ {
		if err := backend.BeginRect(0, 0, 2, 1); err != nil {
			t.Fatal(err)
		}
		if err := backend.WritePixels([]byte{0, byte(sequence), 0, byte(sequence + 1)}); err != nil {
			t.Fatal(err)
		}
		if err := backend.EndRect(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestViewportBackendErrorsWrapCauses(t *testing.T) {
	viewport, err := NewViewport(Rect{Width: 1, Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("clip", func(t *testing.T) {
		backend, err := NewViewportBackend(&viewportPhysicalBackend{}, viewport)
		if err != nil {
			t.Fatal(err)
		}
		if err := backend.BeginRect(0, 0, -1, 1); err == nil || !strings.Contains(err.Error(), "viewport backend clip") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("begin", func(t *testing.T) {
		backend, _ := NewViewportBackend(&viewportPhysicalBackend{beginErr: errPhysicalBegin}, viewport)
		if err := backend.BeginRect(0, 0, 1, 1); !errors.Is(err, errPhysicalBegin) || !strings.Contains(err.Error(), "viewport backend begin") {
			t.Fatalf("err=%v", err)
		}
		if err := backend.WritePixels(nil); !errors.Is(err, ErrViewportBackendInactive) {
			t.Fatalf("state after begin error=%v", err)
		}
	})
	t.Run("write", func(t *testing.T) {
		backend, _ := NewViewportBackend(&viewportPhysicalBackend{writeErr: errPhysicalWrite}, viewport)
		if err := backend.BeginRect(0, 0, 1, 1); err != nil {
			t.Fatal(err)
		}
		if err := backend.WritePixels([]byte{0, 0}); !errors.Is(err, errPhysicalWrite) || !strings.Contains(err.Error(), "viewport backend write") {
			t.Fatalf("err=%v", err)
		}
		if err := backend.EndRect(); !errors.Is(err, ErrViewportBackendInactive) {
			t.Fatalf("state after write error=%v", err)
		}
	})
	t.Run("end", func(t *testing.T) {
		backend, _ := NewViewportBackend(&viewportPhysicalBackend{endErr: errPhysicalEnd}, viewport)
		if err := backend.BeginRect(0, 0, 1, 1); err != nil {
			t.Fatal(err)
		}
		if err := backend.WritePixels([]byte{0, 0}); err != nil {
			t.Fatal(err)
		}
		if err := backend.EndRect(); !errors.Is(err, errPhysicalEnd) || !strings.Contains(err.Error(), "viewport backend end") {
			t.Fatalf("err=%v", err)
		}
		if err := backend.EndRect(); !errors.Is(err, ErrViewportBackendInactive) {
			t.Fatalf("state after end error=%v", err)
		}
	})
}

func identifiedPixels(width, height int) []byte {
	pixels := make([]byte, 0, width*height*2)
	for row := 0; row < height; row++ {
		for column := 0; column < width; column++ {
			pixels = append(pixels, byte(row), byte(column))
		}
	}
	return pixels
}

func croppedPixels(source []byte, sourceWidth, sourceX, sourceY, width, height int) []byte {
	pixels := make([]byte, 0, width*height*2)
	for row := 0; row < height; row++ {
		start := ((sourceY+row)*sourceWidth + sourceX) * 2
		pixels = append(pixels, source[start:start+width*2]...)
	}
	return pixels
}

func oneByteChunks(length int) []int {
	chunks := make([]int, length)
	for index := range chunks {
		chunks[index] = 1
	}
	return chunks
}
