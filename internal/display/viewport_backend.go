package display

import (
	"errors"
	"fmt"
)

var (
	ErrViewportBackendActive     = errors.New("display: viewport backend transfer already active")
	ErrViewportBackendInactive   = errors.New("display: viewport backend has no active transfer")
	ErrViewportBackendTooMuch    = errors.New("display: viewport backend pixel data exceeds rectangle")
	ErrViewportBackendIncomplete = errors.New("display: viewport backend pixel data is incomplete")
)

const (
	viewportBackendIdle uint8 = iota
	viewportBackendVisible
	viewportBackendDiscard
)

// ViewportBackend adapts a Backend to viewport-local coordinates and crops its
// row-major RGB565 byte stream.
type ViewportBackend struct {
	backend  Backend
	viewport Viewport

	state         uint8
	sourceWidth   int64
	sourceBytes   int64
	consumedBytes int64
	clip          ClipResult

	pending     byte
	hasPending  bool
	pendingPair [2]byte
}

var _ Backend = (*ViewportBackend)(nil)

// NewViewportBackend constructs a local-coordinate adapter.
func NewViewportBackend(backend Backend, viewport Viewport) (*ViewportBackend, error) {
	if backend == nil {
		return nil, fmt.Errorf("display: viewport backend: %w", ErrNilBackend)
	}
	return &ViewportBackend{backend: backend, viewport: Viewport{bounds: viewport.Bounds()}}, nil
}

// Size returns the viewport's local dimensions.
func (backend *ViewportBackend) Size() (width, height int16) {
	if backend == nil {
		return 0, 0
	}
	bounds := backend.viewport.Bounds()
	return bounds.Width, bounds.Height
}

// BeginRect begins a local rectangle transfer.
func (backend *ViewportBackend) BeginRect(x, y, width, height int16) error {
	if backend.state != viewportBackendIdle {
		return ErrViewportBackendActive
	}
	local := Rect{X: x, Y: y, Width: width, Height: height}
	clip, visible, err := backend.viewport.Clip(local)
	if err != nil {
		return fmt.Errorf("display: viewport backend clip: %w", err)
	}

	backend.sourceWidth = int64(width)
	backend.sourceBytes = int64(width) * int64(height) * 2
	backend.consumedBytes = 0
	backend.clip = clip
	backend.hasPending = false
	if !visible {
		backend.state = viewportBackendDiscard
		return nil
	}
	if err := backend.backend.BeginRect(clip.Destination.X, clip.Destination.Y, clip.Destination.Width, clip.Destination.Height); err != nil {
		backend.reset()
		return fmt.Errorf("display: viewport backend begin: %w", err)
	}
	backend.state = viewportBackendVisible
	return nil
}

// WritePixels consumes bytes for the full local rectangle and forwards only
// intersections with visible row segments.
func (backend *ViewportBackend) WritePixels(data []byte) error {
	if backend.state == viewportBackendIdle {
		return ErrViewportBackendInactive
	}
	if int64(len(data)) > backend.sourceBytes-backend.consumedBytes {
		return ErrViewportBackendTooMuch
	}
	if backend.state == viewportBackendDiscard {
		backend.consumedBytes += int64(len(data))
		return nil
	}

	start := backend.consumedBytes
	end := start + int64(len(data))
	position := start
	rowBytes := backend.sourceWidth * 2
	visibleFirstRow := int64(backend.clip.SourceY)
	visibleLastRow := visibleFirstRow + int64(backend.clip.Destination.Height)
	visibleByteX := int64(backend.clip.SourceX) * 2
	visibleRowBytes := int64(backend.clip.Destination.Width) * 2
	for position < end {
		row := position / rowBytes
		rowStart := row * rowBytes
		rowEnd := rowStart + rowBytes
		if row < visibleFirstRow || row >= visibleLastRow {
			position = minInt64(end, rowEnd)
			continue
		}
		visibleStart := rowStart + visibleByteX
		visibleEnd := visibleStart + visibleRowBytes
		if position < visibleStart {
			position = minInt64(end, visibleStart)
			continue
		}
		if position >= visibleEnd {
			position = minInt64(end, rowEnd)
			continue
		}
		segmentEnd := minInt64(end, visibleEnd)
		from := int(position - start)
		to := int(segmentEnd - start)
		if err := backend.writeVisible(data[from:to]); err != nil {
			backend.reset()
			return fmt.Errorf("display: viewport backend write: %w", err)
		}
		position = segmentEnd
	}
	backend.consumedBytes = end
	return nil
}

// EndRect ends the current visible transfer or completes discard mode.
func (backend *ViewportBackend) EndRect() error {
	if backend.state == viewportBackendIdle {
		return ErrViewportBackendInactive
	}
	state := backend.state
	complete := backend.consumedBytes == backend.sourceBytes && !backend.hasPending
	backend.reset()
	if state == viewportBackendVisible {
		if err := backend.backend.EndRect(); err != nil {
			if !complete {
				return fmt.Errorf("display: viewport backend end: %w: %w", ErrViewportBackendIncomplete, err)
			}
			return fmt.Errorf("display: viewport backend end: %w", err)
		}
	}
	if !complete {
		return ErrViewportBackendIncomplete
	}
	return nil
}

func (backend *ViewportBackend) writeVisible(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if backend.hasPending {
		backend.pendingPair[0] = backend.pending
		backend.pendingPair[1] = data[0]
		if err := backend.backend.WritePixels(backend.pendingPair[:]); err != nil {
			return err
		}
		backend.hasPending = false
		data = data[1:]
	}
	even := len(data) &^ 1
	if even != 0 {
		if err := backend.backend.WritePixels(data[:even]); err != nil {
			return err
		}
	}
	if even != len(data) {
		backend.pending = data[even]
		backend.hasPending = true
	}
	return nil
}

func (backend *ViewportBackend) reset() {
	backend.state = viewportBackendIdle
	backend.sourceWidth = 0
	backend.sourceBytes = 0
	backend.consumedBytes = 0
	backend.clip = ClipResult{}
	backend.hasPending = false
}

func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}
