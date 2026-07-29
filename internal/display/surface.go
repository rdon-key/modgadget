package display

import (
	"errors"
	"fmt"
)

var (
	errSurfaceActive     = errors.New("display: surface transfer already active")
	errSurfaceInactive   = errors.New("display: surface has no active transfer")
	errSurfaceTooMuch    = errors.New("display: surface pixel data exceeds rectangle")
	errSurfaceIncomplete = errors.New("display: surface pixel data is incomplete")
)

// Surface is a caller-backed, row-major RGB565 big-endian image Backend.
type Surface struct {
	width  int16
	height int16
	pixels []byte

	active   bool
	rect     Rect
	expected int64
	consumed int64
}

var _ Backend = (*Surface)(nil)

// NewSurface constructs a Surface over the required prefix of pixels.
func NewSurface(width, height int16, pixels []byte) (*Surface, error) {
	if width < 0 {
		return nil, fmt.Errorf("display: surface width must not be negative")
	}
	if height < 0 {
		return nil, fmt.Errorf("display: surface height must not be negative")
	}
	required := int64(width) * int64(height) * 2
	maxInt := int64(^uint(0) >> 1)
	if required > maxInt {
		return nil, fmt.Errorf("display: surface byte size overflows int")
	}
	if int64(len(pixels)) < required {
		return nil, fmt.Errorf("display: surface pixel buffer is too short: need %d bytes", required)
	}
	return &Surface{width: width, height: height, pixels: pixels[:int(required)]}, nil
}

// Size returns the Surface dimensions.
func (surface *Surface) Size() (int16, int16) {
	if surface == nil {
		return 0, 0
	}
	return surface.width, surface.height
}

// BeginRect starts a row-major RGB565 byte transfer into a Surface rectangle.
func (surface *Surface) BeginRect(x, y, width, height int16) error {
	if surface == nil {
		return fmt.Errorf("display: surface is nil")
	}
	if surface.active {
		return errSurfaceActive
	}
	if width < 0 {
		return fmt.Errorf("display: surface rectangle width must not be negative")
	}
	if height < 0 {
		return fmt.Errorf("display: surface rectangle height must not be negative")
	}
	if x < 0 {
		return fmt.Errorf("display: surface rectangle X must not be negative")
	}
	if y < 0 {
		return fmt.Errorf("display: surface rectangle Y must not be negative")
	}
	right := int64(x) + int64(width)
	bottom := int64(y) + int64(height)
	if right > int64(surface.width) {
		return fmt.Errorf("display: surface rectangle right edge exceeds width")
	}
	if bottom > int64(surface.height) {
		return fmt.Errorf("display: surface rectangle bottom edge exceeds height")
	}
	expected := int64(width) * int64(height) * 2
	if expected < 0 || expected > int64(len(surface.pixels)) {
		return fmt.Errorf("display: surface rectangle byte size is invalid")
	}
	surface.active = true
	surface.rect = Rect{X: x, Y: y, Width: width, Height: height}
	surface.expected = expected
	surface.consumed = 0
	return nil
}

// WritePixels writes the next arbitrary byte segment of the active rectangle.
func (surface *Surface) WritePixels(data []byte) error {
	if surface == nil || !surface.active {
		return errSurfaceInactive
	}
	if int64(len(data)) > surface.expected-surface.consumed {
		return errSurfaceTooMuch
	}
	if len(data) == 0 {
		return nil
	}

	rowBytes := int64(surface.rect.Width) * 2
	surfaceRowBytes := int64(surface.width) * 2
	sourceOffset := int64(0)
	streamOffset := surface.consumed
	for sourceOffset < int64(len(data)) {
		row := streamOffset / rowBytes
		byteInRow := streamOffset % rowBytes
		remainingInRow := rowBytes - byteInRow
		segmentBytes := int64(len(data)) - sourceOffset
		if segmentBytes > remainingInRow {
			segmentBytes = remainingInRow
		}
		destination := (int64(surface.rect.Y)+row)*surfaceRowBytes + int64(surface.rect.X)*2 + byteInRow
		copy(
			surface.pixels[int(destination):int(destination+segmentBytes)],
			data[int(sourceOffset):int(sourceOffset+segmentBytes)],
		)
		sourceOffset += segmentBytes
		streamOffset += segmentBytes
	}
	surface.consumed = streamOffset
	return nil
}

// EndRect completes the active transfer and always returns the Surface to idle.
func (surface *Surface) EndRect() error {
	if surface == nil || !surface.active {
		return errSurfaceInactive
	}
	complete := surface.consumed == surface.expected
	surface.reset()
	if !complete {
		return errSurfaceIncomplete
	}
	return nil
}

// BlitTo streams the complete Surface to backend in one WritePixels call.
func (surface *Surface) BlitTo(backend Backend, x, y int16) error {
	if surface == nil {
		return fmt.Errorf("display: surface is nil")
	}
	if surface.active {
		return fmt.Errorf("display: surface blit: %w", errSurfaceActive)
	}
	if backend == nil {
		return fmt.Errorf("display: surface blit: %w", ErrNilBackend)
	}
	if surface.width == 0 || surface.height == 0 {
		return nil
	}
	if err := backend.BeginRect(x, y, surface.width, surface.height); err != nil {
		return fmt.Errorf("display: surface blit begin: %w", err)
	}
	if err := backend.WritePixels(surface.pixels); err != nil {
		return fmt.Errorf("display: surface blit write: %w", err)
	}
	if err := backend.EndRect(); err != nil {
		return fmt.Errorf("display: surface blit end: %w", err)
	}
	return nil
}

func (surface *Surface) reset() {
	surface.active = false
	surface.rect = Rect{}
	surface.expected = 0
	surface.consumed = 0
}
