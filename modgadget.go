// Package modgadget provides a small high-level API for displaying styled text.
package modgadget

import (
	"fmt"
	"math"
	"time"

	displaypkg "github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/mgf"
	"github.com/rdon-key/modgadget/internal/text"
	"github.com/rdon-key/modgadget/internal/text/markup"
)

const maxKeyEventsPerUpdate = 64

// Display receives row-major RGB565 pixel data for rectangular display regions.
type Display = displaypkg.Backend
type Color565 = displaypkg.Color565
type Font = text.Font
type FontStack = text.FontStack
type MGFFont = text.MGFFont
type Style = text.Style
type StyleEntry = text.StyleEntry
type StyleSet = text.StyleSet

func NewMGFFont(source mgf.Font) MGFFont { return text.NewMGFFont(source) }

const (
	ColorBlack = displaypkg.ColorBlack
	ColorWhite = displaypkg.ColorWhite
	ColorRed   = displaypkg.ColorRed
	ColorGreen = displaypkg.ColorGreen
	ColorBlue  = displaypkg.ColorBlue
)

func RGB565(red, green, blue uint8) Color565 { return displaypkg.RGB565(red, green, blue) }

type Option func(*Gadget)

func WithStyles(styles StyleSet) Option { return func(g *Gadget) { g.styles = styles } }

type Gadget struct {
	display            Display
	styles             text.StyleSet
	viewports          []*Viewport
	clearScratch       [64]byte
	keyboard           Keyboard
	volumeController   VolumeController
	capturedSystemKeys uint8
	keyListeners       []keyListener
	nextListener       ListenerID
	keyDispatchDepth   int
	keyListenersDirty  bool
}

// Clear fills the entire Display, as reported by Display.Size, with the default
// background color. It does not change Viewport content or dirty state, and it
// returns Display transfer errors.
func (g *Gadget) Clear() error {
	if g == nil || g.display == nil {
		return fmt.Errorf("modgadget: display is nil")
	}
	width, height := g.display.Size()
	if err := displaypkg.FillRect(
		g.display,
		displaypkg.Rect{Width: width, Height: height},
		g.styles.Default.Background,
		g.clearScratch[:],
	); err != nil {
		return fmt.Errorf("modgadget: clear display: %w", err)
	}
	return nil
}

// New creates a Gadget that renders to display.
func New(display Display, options ...Option) *Gadget {
	g := &Gadget{display: display}
	for _, option := range options {
		if option != nil {
			option(g)
		}
	}
	return g
}

type ViewportOption func(*Viewport)

func Bounds(x, y, width, height int16) ViewportOption {
	return func(v *Viewport) { v.bounds = displaypkg.Rect{X: x, Y: y, Width: width, Height: height} }
}

type ScrollOption func(*horizontalScroll)

type horizontalScroll struct {
	speed     float64
	gap       int16
	loop      bool
	fromLeft  bool
	fromRight bool
}

func ScrollSpeed(pixelsPerSecond float64) ScrollOption {
	return func(s *horizontalScroll) { s.speed = pixelsPerSecond }
}
func ScrollGap(pixels int16) ScrollOption { return func(s *horizontalScroll) { s.gap = pixels } }
func ScrollLoop() ScrollOption            { return func(s *horizontalScroll) { s.loop = true } }

// ScrollFromLeft starts the text outside the left edge and moves it right once.
func ScrollFromLeft() ScrollOption {
	return func(s *horizontalScroll) { s.fromLeft, s.fromRight = true, false }
}

// ScrollFromRight starts the text outside the right edge and moves it left once.
func ScrollFromRight() ScrollOption {
	return func(s *horizontalScroll) { s.fromRight, s.fromLeft = true, false }
}

type Viewport struct {
	owner          *Gadget
	bounds         displaypkg.Rect
	text           string
	layout         text.TextLayout
	textWidth      int16
	scratch        []byte
	fillScratch    [64]byte
	buffer         []byte
	surface        *displaypkg.Surface
	surfaceBackend *displaypkg.ViewportBackend
	dirty          bool
	parseErr       error
	scroll         horizontalScroll
	scrollEnabled  bool
	started        bool
	finished       bool
	start          time.Time
	offset         int16
}

func (g *Gadget) Viewport(options ...ViewportOption) *Viewport {
	v := &Viewport{owner: g, dirty: true}
	if g != nil && g.display != nil {
		width, height := g.display.Size()
		v.bounds = displaypkg.Rect{Width: width, Height: height}
	}
	for _, option := range options {
		if option != nil {
			option(v)
		}
	}
	if g != nil {
		g.viewports = append(g.viewports, v)
	}
	return v
}

func (v *Viewport) SetText(value string) error {
	if v.text == value {
		if v.scroll.oneShot() {
			v.started, v.finished, v.offset, v.dirty = false, false, v.initialScrollOffset(), true
		}
		return v.parseErr
	}
	v.text, v.dirty, v.started, v.finished, v.offset = value, true, false, false, 0
	if v.owner == nil {
		v.parseErr = fmt.Errorf("modgadget: viewport has no gadget")
		return v.parseErr
	}
	if value == "" {
		v.layout, v.textWidth, v.scratch, v.parseErr = text.TextLayout{}, 0, nil, nil
		return nil
	}
	spans, err := (markup.Parser{Styles: v.owner.styles}).Parse(value)
	if err != nil {
		v.parseErr = fmt.Errorf("modgadget: parse text: %w", err)
		return v.parseErr
	}
	layout, err := text.NewTextLayout(spans)
	if err != nil {
		v.parseErr = fmt.Errorf("modgadget: layout text: %w", err)
		return v.parseErr
	}
	v.layout, v.textWidth, v.parseErr = layout, layout.Measurement().MaxAdvanceX, nil
	if v.scroll.oneShot() {
		v.offset = v.initialScrollOffset()
	}
	maxGlyphWidth := int16(1)
	for i := range spans {
		for _, r := range spans[i].Value {
			if r != '\n' {
				if glyph, ok := spans[i].Font.Lookup(r); ok && glyph.Width > maxGlyphWidth {
					maxGlyphWidth = glyph.Width
				}
			}
		}
	}
	v.scratch = make([]byte, int(maxGlyphWidth)*2)
	return nil
}

func (v *Viewport) SetHorizontalScroll(options ...ScrollOption) {
	s := horizontalScroll{}
	for _, option := range options {
		if option != nil {
			option(&s)
		}
	}
	offset := int16(0)
	if s.fromLeft {
		offset = v.textWidth
	} else if s.fromRight {
		offset = -v.bounds.Width
	}
	v.scroll, v.scrollEnabled, v.started, v.finished, v.offset, v.dirty = s, s.speed > 0, false, false, offset, true
}

func (v *Viewport) Clear() { _ = v.SetText("") }
func (v *Viewport) ScrollTo(pixel int16) {
	if pixel < 0 {
		pixel = 0
	}
	v.offset, v.started, v.dirty = pixel, false, true
}

func (g *Gadget) Update(now time.Time) {
	if g == nil {
		return
	}
	if g.keyboard != nil {
		for count := 0; count < maxKeyEventsPerUpdate; count++ {
			event, ok := g.keyboard.ReadKeyEvent()
			if !ok {
				break
			}
			if !g.handleSystemKey(event) {
				g.dispatchKey(event)
			}
		}
	}
	for _, v := range g.viewports {
		v.update(now)
	}
}

func (v *Viewport) update(now time.Time) {
	if !v.scrollEnabled || v.scroll.speed <= 0 || (!v.scroll.oneShot() && v.textWidth <= v.bounds.Width) {
		return
	}
	if v.scroll.oneShot() && v.finished {
		return
	}
	if !v.started {
		v.start, v.started = now, true
		return
	}
	pixels := math.Floor(now.Sub(v.start).Seconds() * v.scroll.speed)
	if pixels < 0 {
		pixels = 0
	}
	var next int64
	if pixels > math.MaxInt64 {
		next = math.MaxInt64
	} else {
		next = int64(pixels)
	}
	if v.scroll.fromLeft {
		distance := next
		travel := int64(v.textWidth) + int64(v.bounds.Width)
		if distance >= travel {
			v.offset = -v.bounds.Width
			v.finished = true
			v.dirty = true
			return
		}
		next = int64(v.textWidth) - distance
		if next < math.MinInt16 {
			next = math.MinInt16
		}
		if int16(next) != v.offset {
			v.offset, v.dirty = int16(next), true
		}
		return
	}
	if v.scroll.fromRight {
		distance := next
		travel := int64(v.bounds.Width) + int64(v.textWidth)
		if distance >= travel {
			v.offset = v.textWidth
			v.finished = true
			v.dirty = true
			return
		}
		next = distance - int64(v.bounds.Width)
		if next < math.MinInt16 {
			next = math.MinInt16
		}
		if next > math.MaxInt16 {
			next = math.MaxInt16
		}
		if int16(next) != v.offset {
			v.offset, v.dirty = int16(next), true
		}
		return
	}
	if v.scroll.loop {
		cycle := int64(v.textWidth) + int64(v.scroll.gap)
		if cycle > 0 {
			next %= cycle
		} else {
			next = 0
		}
	} else {
		maximum := int64(v.textWidth) - int64(v.bounds.Width)
		if next > maximum {
			next = maximum
		}
	}
	if next > math.MaxInt16 {
		next = math.MaxInt16
	}
	if int16(next) != v.offset {
		v.offset, v.dirty = int16(next), true
	}
}

// Render draws dirty Viewports to the Display in registration order and returns
// the first Display transfer or drawing error.
func (g *Gadget) Render() error {
	if g == nil || g.display == nil {
		return fmt.Errorf("modgadget: display is nil")
	}
	for _, v := range g.viewports {
		if v.dirty {
			if err := v.render(g.display); err != nil {
				return err
			}
			v.dirty = false
		}
	}
	return nil
}

func (v *Viewport) render(target Display) error {
	if v.parseErr != nil {
		return v.parseErr
	}
	if v.scrollEnabled && v.scroll.speed > 0 && (v.scroll.oneShot() || v.textWidth > v.bounds.Width) {
		return v.renderBuffered(target)
	}
	return v.renderDirect(target)
}

func (v *Viewport) renderDirect(target Display) error {
	low, err := displaypkg.NewViewport(v.bounds)
	if err != nil {
		return fmt.Errorf("modgadget: viewport: %w", err)
	}
	clipped, err := displaypkg.NewViewportBackend(target, low)
	if err != nil {
		return fmt.Errorf("modgadget: viewport display adapter: %w", err)
	}
	if err := displaypkg.FillRect(clipped, displaypkg.Rect{Width: v.bounds.Width, Height: v.bounds.Height}, v.owner.styles.Default.Background, v.fillScratch[:]); err != nil {
		return fmt.Errorf("modgadget: clear viewport: %w", err)
	}
	if v.text == "" {
		return nil
	}
	return v.drawText(clipped)
}

func (v *Viewport) renderBuffered(target Display) error {
	if err := v.ensureSurface(); err != nil {
		return err
	}
	if err := displaypkg.FillRect(v.surface, displaypkg.Rect{Width: v.bounds.Width, Height: v.bounds.Height}, v.owner.styles.Default.Background, v.fillScratch[:]); err != nil {
		return fmt.Errorf("modgadget: clear viewport buffer: %w", err)
	}
	if v.text != "" && !(v.scroll.oneShot() && v.finished) {
		if err := v.drawText(v.surfaceBackend); err != nil {
			return err
		}
	}
	if err := v.surface.BlitTo(target, v.bounds.X, v.bounds.Y); err != nil {
		return fmt.Errorf("modgadget: blit viewport buffer: %w", err)
	}
	return nil
}

func (v *Viewport) ensureSurface() error {
	if v.surface != nil {
		return nil
	}
	required := int(v.bounds.Width) * int(v.bounds.Height) * 2
	if required <= 0 {
		return fmt.Errorf("modgadget: viewport buffer dimensions must be positive")
	}
	v.buffer = make([]byte, required)
	surface, err := displaypkg.NewSurface(v.bounds.Width, v.bounds.Height, v.buffer)
	if err != nil {
		return fmt.Errorf("modgadget: create viewport surface: %w", err)
	}
	local, err := displaypkg.NewViewport(displaypkg.Rect{Width: v.bounds.Width, Height: v.bounds.Height})
	if err != nil {
		return fmt.Errorf("modgadget: create buffered viewport: %w", err)
	}
	clipped, err := displaypkg.NewViewportBackend(surface, local)
	if err != nil {
		return fmt.Errorf("modgadget: create buffered viewport display adapter: %w", err)
	}
	v.surface, v.surfaceBackend = surface, clipped
	return nil
}

func (v *Viewport) drawText(target Display) error {
	measurement := v.layout.Measurement()
	baseline := int16(-int32(measurement.Bounds.MinY))
	penX := -v.offset
	if _, err := v.layout.Draw(target, penX, baseline, v.scratch); err != nil {
		return fmt.Errorf("modgadget: draw text: %w", err)
	}
	if v.scroll.loop && !v.scroll.oneShot() && v.scrollEnabled && v.textWidth > v.bounds.Width {
		next := int32(penX) + int32(v.textWidth) + int32(v.scroll.gap)
		if next <= math.MaxInt16 && next >= math.MinInt16 {
			if _, err := v.layout.Draw(target, int16(next), baseline, v.scratch); err != nil {
				return fmt.Errorf("modgadget: draw loop text: %w", err)
			}
		}
	}
	return nil
}

func (scroll horizontalScroll) oneShot() bool { return scroll.fromLeft || scroll.fromRight }

func (v *Viewport) initialScrollOffset() int16 {
	if v.scroll.fromLeft {
		return v.textWidth
	}
	if v.scroll.fromRight {
		return -v.bounds.Width
	}
	return 0
}
