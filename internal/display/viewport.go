package display

import (
	"fmt"
	"math"
)

// Viewport is an absolute display region used to translate and clip local
// rectangles.
type Viewport struct {
	bounds Rect
}

// ClipResult describes a visible destination and its offset in the source
// rectangle.
type ClipResult struct {
	Destination Rect
	SourceX     int16
	SourceY     int16
}

// NewViewport constructs a viewport with non-negative dimensions.
func NewViewport(bounds Rect) (Viewport, error) {
	if err := validateViewportBounds(bounds); err != nil {
		return Viewport{}, err
	}
	return Viewport{bounds: bounds}, nil
}

// Bounds returns the viewport's absolute bounds by value.
func (viewport Viewport) Bounds() Rect {
	return viewport.bounds
}

// Clip translates local into display coordinates and intersects it with the
// viewport. Rectangles and viewport bounds use half-open intervals.
func (viewport Viewport) Clip(local Rect) (ClipResult, bool, error) {
	if err := validateViewportBounds(viewport.bounds); err != nil {
		return ClipResult{}, false, err
	}
	if local.Width < 0 {
		return ClipResult{}, false, fmt.Errorf("display: local rectangle width is negative")
	}
	if local.Height < 0 {
		return ClipResult{}, false, fmt.Errorf("display: local rectangle height is negative")
	}
	if viewport.bounds.Width == 0 || viewport.bounds.Height == 0 || local.Width == 0 || local.Height == 0 {
		return ClipResult{}, false, nil
	}

	viewportLeft := int32(viewport.bounds.X)
	viewportTop := int32(viewport.bounds.Y)
	viewportRight := viewportLeft + int32(viewport.bounds.Width)
	viewportBottom := viewportTop + int32(viewport.bounds.Height)
	screenLeft := viewportLeft + int32(local.X)
	screenTop := viewportTop + int32(local.Y)
	screenRight := screenLeft + int32(local.Width)
	screenBottom := screenTop + int32(local.Height)

	clippedLeft := maxInt32(viewportLeft, screenLeft)
	clippedTop := maxInt32(viewportTop, screenTop)
	clippedRight := minInt32(viewportRight, screenRight)
	clippedBottom := minInt32(viewportBottom, screenBottom)
	if clippedLeft >= clippedRight || clippedTop >= clippedBottom {
		return ClipResult{}, false, nil
	}

	width := clippedRight - clippedLeft
	height := clippedBottom - clippedTop
	sourceX := clippedLeft - screenLeft
	sourceY := clippedTop - screenTop
	values := []struct {
		name  string
		value int32
	}{
		{"destination X", clippedLeft},
		{"destination Y", clippedTop},
		{"destination width", width},
		{"destination height", height},
		{"source X", sourceX},
		{"source Y", sourceY},
	}
	for _, value := range values {
		if value.value < math.MinInt16 || value.value > math.MaxInt16 {
			return ClipResult{}, false, fmt.Errorf("display: viewport clip %s is outside int16", value.name)
		}
	}

	return ClipResult{
		Destination: Rect{X: int16(clippedLeft), Y: int16(clippedTop), Width: int16(width), Height: int16(height)},
		SourceX:     int16(sourceX),
		SourceY:     int16(sourceY),
	}, true, nil
}

func validateViewportBounds(bounds Rect) error {
	if bounds.Width < 0 {
		return fmt.Errorf("display: viewport width is negative")
	}
	if bounds.Height < 0 {
		return fmt.Errorf("display: viewport height is negative")
	}
	return nil
}

func minInt32(left, right int32) int32 {
	if left < right {
		return left
	}
	return right
}

func maxInt32(left, right int32) int32 {
	if left > right {
		return left
	}
	return right
}
