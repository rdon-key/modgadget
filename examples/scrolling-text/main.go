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
	horizontalPadding int16 = 6
	verticalPadding   int16 = 6
	//frameInterval           = 60 * time.Millisecond
	frameInterval = 0
	endPause      = 500 * time.Millisecond
)

var viewportBounds = display.Rect{X: 10, Y: 10, Width: 220, Height: 115}

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

	panel := st7789.New(
		machine.SPI1,
		machine.DISPLAY_CS,
		machine.DISPLAY_DC,
		machine.DISPLAY_RST,
	)
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
	if err := runScrollingText(backend); err != nil {
		println("scrolling text failed:", err.Error())
		return
	}
}

func runScrollingText(backend display.Backend) error {
	var fillScratch [64]byte
	var glyphScratch [288]byte

	if err := display.FillRect(
		backend,
		display.Rect{Width: displayWidth, Height: displayHeight},
		display.RGB565(16, 16, 24),
		fillScratch[:],
	); err != nil {
		return fmt.Errorf("fill display: %w", err)
	}

	border := display.Rect{
		X:      viewportBounds.X - 1,
		Y:      viewportBounds.Y - 1,
		Width:  viewportBounds.Width + 2,
		Height: viewportBounds.Height + 2,
	}
	if err := display.FillRect(
		backend,
		border,
		display.ColorWhite,
		fillScratch[:],
	); err != nil {
		return fmt.Errorf("draw viewport border: %w", err)
	}

	viewport, err := display.NewViewport(viewportBounds)
	if err != nil {
		return fmt.Errorf("create viewport: %w", err)
	}

	viewportBackend, err := display.NewViewportBackend(backend, viewport)
	if err != nil {
		return fmt.Errorf("create viewport backend: %w", err)
	}

	maxAdvanceX := viewportBounds.Width - horizontalPadding*2
	layout, err := buildScrollingLayout(maxAdvanceX)
	if err != nil {
		return fmt.Errorf("build scrolling layout: %w", err)
	}

	measurement := layout.Measurement()
	if !measurement.HasInk {
		return fmt.Errorf("scrolling layout has no ink")
	}

	contentHeight :=
		int32(measurement.Bounds.MaxY) -
			int32(measurement.Bounds.MinY)

	visibleHeight :=
		int32(viewportBounds.Height) -
			int32(verticalPadding)*2

	maxScroll := contentHeight - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}

	initialBaseline :=
		int32(verticalPadding) -
			int32(measurement.Bounds.MinY)

	printLayoutInfo(
		&layout,
		maxAdvanceX,
		contentHeight,
		visibleHeight,
		maxScroll,
		initialBaseline,
	)

	scrollY := int32(0)
	direction := int32(1)
	frameCount := uint32(0)

	println("scroll at top")
	time.Sleep(endPause)

	for {
		baseline := initialBaseline - scrollY
		if baseline < math.MinInt16 || baseline > math.MaxInt16 {
			return fmt.Errorf(
				"baseline %d is outside int16",
				baseline,
			)
		}

		frameStart := time.Now()

		clearStart := time.Now()
		if err := display.FillRect(
			backend,
			viewportBounds,
			display.ColorBlack,
			fillScratch[:],
		); err != nil {
			return fmt.Errorf("clear viewport: %w", err)
		}
		clearElapsed := time.Since(clearStart)

		drawStart := time.Now()
		if _, err := layout.Draw(
			viewportBackend,
			horizontalPadding,
			int16(baseline),
			glyphScratch[:],
		); err != nil {
			return fmt.Errorf("draw scrolling layout: %w", err)
		}
		drawElapsed := time.Since(drawStart)

		totalElapsed := time.Since(frameStart)

		frameCount++
		if frameCount%10 == 0 {
			println(
				"clear ms:", clearElapsed.Milliseconds(),
				"draw ms:", drawElapsed.Milliseconds(),
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
