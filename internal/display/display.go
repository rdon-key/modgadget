// Package display defines device-independent RGB565 rectangle drawing.
package display

import "errors"

// Backend streams RGB565 pixels into rectangular display regions.
type Backend interface {
	Size() (width, height int16)
	BeginRect(x, y, width, height int16) error
	WritePixels(data []byte) error
	EndRect() error
}

// Rect describes a rectangular region in display coordinates.
type Rect struct {
	X      int16
	Y      int16
	Width  int16
	Height int16
}

// Empty reports whether the rectangle has no positive area.
func (r Rect) Empty() bool {
	return r.Width <= 0 || r.Height <= 0
}

// PixelCount returns the rectangle's pixel count, or zero for an empty rectangle.
func (r Rect) PixelCount() int32 {
	if r.Empty() {
		return 0
	}
	return int32(r.Width) * int32(r.Height)
}

// Color565 is a 16-bit RGB565 color value.
type Color565 uint16

const (
	// ColorBlack is RGB565 black.
	ColorBlack Color565 = 0x0000
	// ColorWhite is RGB565 white.
	ColorWhite Color565 = 0xffff
	// ColorRed is RGB565 red.
	ColorRed Color565 = 0xf800
	// ColorGreen is RGB565 green.
	ColorGreen Color565 = 0x07e0
	// ColorBlue is RGB565 blue.
	ColorBlue Color565 = 0x001f
)

// RGB565 converts 8-bit red, green, and blue components to RGB565.
func RGB565(r, g, b uint8) Color565 {
	return Color565(uint16(r&0xf8)<<8 | uint16(g&0xfc)<<3 | uint16(b)>>3)
}

var (
	// ErrNilBackend indicates that no drawing backend was supplied.
	ErrNilBackend = errors.New("display: backend is nil")
	// ErrInvalidRect indicates that a rectangle has a non-positive dimension.
	ErrInvalidRect = errors.New("display: rectangle width and height must be positive")
	// ErrScratchTooSmall indicates that a fill buffer cannot hold one RGB565 pixel.
	ErrScratchTooSmall = errors.New("display: scratch buffer must contain at least two bytes")
	// ErrInvalidStride indicates that an image row is shorter than its pixel data.
	ErrInvalidStride = errors.New("display: stride is shorter than the RGB565 row")
	// ErrPixelDataTooShort indicates that an image does not contain all requested rows.
	ErrPixelDataTooShort = errors.New("display: RGB565 pixel data is too short")
)
