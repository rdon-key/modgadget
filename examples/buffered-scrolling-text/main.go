package main

import (
	"fmt"
	"machine"
	"math"
	"time"

	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/spleen8x16"
	"github.com/rdon-key/modgadget/internal/st7789"
	textdraw "github.com/rdon-key/modgadget/internal/text"
)

const (
	displayWidth      int16 = 240
	displayHeight     int16 = 135
	viewportX         int16 = 10
	viewportY         int16 = 10
	viewportWidth     int16 = 220
	viewportHeight    int16 = 115
	horizontalPadding int16 = 6
	verticalPadding   int16 = 6
	frameInterval           = 0
	endPause                = 500 * time.Millisecond
)

var (
	viewportPixels   [220 * 115 * 2]byte
	fillScratch      [1024]byte
	glyphScratch     [288]byte
	physicalViewport = display.Rect{X: viewportX, Y: viewportY, Width: viewportWidth, Height: viewportHeight}
)

var _ display.Backend = (*st7789.Device)(nil)

func main() {
	time.Sleep(3 * time.Second)
	println("boot")

	if err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 40_000_000,
		Mode:      0,
		SCK:       machine.DISPLAY_SCK,
		SDO:       machine.DISPLAY_MOSI,
		SDI:       machine.NoPin,
	}); err != nil {
		println("SPI configure failed:", err.Error())
		return
	}

	machine.DISPLAY_BL.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.DISPLAY_BL.High()

	panel := st7789.New(machine.SPI1, machine.DISPLAY_CS, machine.DISPLAY_DC, machine.DISPLAY_RST)
	if err := panel.Configure(st7789.Config{
		Width:        135,
		Height:       240,
		Rotation:     st7789.Rotation90,
		RowOffset:    40,
		ColumnOffset: 52,
		Invert:       true,
	}); err != nil {
		println("display configure failed:", err.Error())
		return
	}

	var backend display.Backend = panel
	if err := runBufferedScrollingText(backend); err != nil {
		println("buffered scrolling text failed:", err.Error())
	}
}

func runBufferedScrollingText(physical display.Backend) error {
	if err := display.FillRect(
		physical,
		display.Rect{Width: displayWidth, Height: displayHeight},
		display.RGB565(16, 16, 24),
		fillScratch[:],
	); err != nil {
		return fmt.Errorf("fill display: %w", err)
	}
	border := display.Rect{
		X:      physicalViewport.X - 1,
		Y:      physicalViewport.Y - 1,
		Width:  physicalViewport.Width + 2,
		Height: physicalViewport.Height + 2,
	}
	if err := display.FillRect(physical, border, display.ColorWhite, fillScratch[:]); err != nil {
		return fmt.Errorf("draw viewport border: %w", err)
	}

	surface, err := display.NewSurface(viewportWidth, viewportHeight, viewportPixels[:])
	if err != nil {
		return fmt.Errorf("create surface: %w", err)
	}
	localViewport, err := display.NewViewport(display.Rect{Width: viewportWidth, Height: viewportHeight})
	if err != nil {
		return fmt.Errorf("create local viewport: %w", err)
	}
	surfaceViewportBackend, err := display.NewViewportBackend(surface, localViewport)
	if err != nil {
		return fmt.Errorf("create surface viewport backend: %w", err)
	}

	maxAdvanceX := viewportWidth - horizontalPadding*2
	layout, err := buildScrollingLayout(maxAdvanceX)
	if err != nil {
		return fmt.Errorf("build scrolling layout: %w", err)
	}
	measurement := layout.Measurement()
	if !measurement.HasInk {
		return fmt.Errorf("scrolling layout has no ink")
	}
	contentHeight := int32(measurement.Bounds.MaxY) - int32(measurement.Bounds.MinY)
	visibleHeight := int32(viewportHeight) - int32(verticalPadding)*2
	maxScroll := contentHeight - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	initialBaseline := int32(verticalPadding) - int32(measurement.Bounds.MinY)
	printLayoutInfo(&layout, maxAdvanceX, contentHeight, visibleHeight, maxScroll, initialBaseline)

	scrollY := int32(0)
	direction := int32(1)
	frameCount := uint32(0)
	println("scroll at top")
	time.Sleep(endPause)
	for {
		baseline := initialBaseline - scrollY
		if baseline < math.MinInt16 || baseline > math.MaxInt16 {
			return fmt.Errorf("baseline %d is outside int16", baseline)
		}

		totalStart := time.Now()
		composeStart := time.Now()
		if err := display.FillRect(
			surface,
			display.Rect{Width: viewportWidth, Height: viewportHeight},
			display.ColorBlack,
			fillScratch[:],
		); err != nil {
			return fmt.Errorf("clear surface: %w", err)
		}
		if _, err := layout.Draw(
			surfaceViewportBackend,
			horizontalPadding,
			int16(baseline),
			glyphScratch[:],
		); err != nil {
			return fmt.Errorf("draw layout to surface: %w", err)
		}
		composeElapsed := time.Since(composeStart)

		blitStart := time.Now()
		if err := surface.BlitTo(physical, viewportX, viewportY); err != nil {
			return fmt.Errorf("blit surface: %w", err)
		}
		blitElapsed := time.Since(blitStart)
		totalElapsed := time.Since(totalStart)

		frameCount++
		if frameCount%10 == 0 {
			println(
				"compose ms:", composeElapsed.Milliseconds(),
				"blit ms:", blitElapsed.Milliseconds(),
				"total ms:", totalElapsed.Milliseconds(),
			)
		}

		time.Sleep(frameInterval)
		if maxScroll == 0 {
			continue
		}
		if direction > 0 {
			if scrollY == maxScroll {
				direction = -1
				println("scroll at bottom")
				time.Sleep(endPause)
			} else {
				scrollY++
			}
		} else if scrollY == 0 {
			direction = 1
			println("scroll at top")
			time.Sleep(endPause)
		} else {
			scrollY--
		}
	}
}

func buildScrollingLayout(maxAdvanceX int16) (textdraw.TextLayout, error) {
	return textdraw.NewWrappedTextLayout([]textdraw.Span{
		{
			Font:       textdraw.NewMGFFont(spleen8x16.Font),
			Value:      "ModGadget scrolling text\n\n",
			Foreground: display.RGB565(255, 255, 0),
			Background: display.ColorBlack,
		},
		{
			Font: textdraw.NewMGFFont(spleen8x16.Font),
			Value: "This text is wrapped to the viewport width. " +
				"The prepared layout is built only once. " +
				"Each frame reuses the same glyph data. " +
				"The viewport clips pixels above and below.\n\n",
			Foreground: display.RGB565(0, 255, 255),
			Background: display.ColorBlack,
		},
		{
			Font: textdraw.NewMGFFont(spleen8x16.Font),
			Value: "No full screen text buffer is required. " +
				"Scrolling only changes the first baseline. " +
				"This is the foundation for logs and consoles. " +
				"The same prepared text is drawn every frame.",
			Foreground: display.ColorGreen,
			Background: display.ColorBlack,
		},
	}, maxAdvanceX)
}

func printLayoutInfo(layout *textdraw.TextLayout, maxAdvanceX int16, contentHeight, visibleHeight, maxScroll, initialBaseline int32) {
	measurement := layout.Measurement()
	println("layout line count:", layout.LineCount())
	println("layout MaxAdvanceX:", measurement.MaxAdvanceX)
	println("layout AdvanceY:", measurement.AdvanceY)
	println("layout bounds:", measurement.Bounds.MinX, measurement.Bounds.MinY, measurement.Bounds.MaxX, measurement.Bounds.MaxY)
	println("layout HasInk:", measurement.HasInk)
	println("wrap maxAdvanceX:", maxAdvanceX)
	println("content height:", contentHeight)
	println("visible height:", visibleHeight)
	println("max scroll:", maxScroll)
	println("initial baseline:", initialBaseline)
}
