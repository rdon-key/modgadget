//go:build tinygo

package main

import (
	"fmt"
	"machine"
	"math"
	"time"

	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/st7789"
	"github.com/rdon-key/modgadget/internal/text"
)

const (
	displayWidth      int16 = 240
	displayHeight     int16 = 135
	viewportX         int16 = 10
	viewportY         int16 = 10
	viewportWidth     int16 = 220
	viewportHeight    int16 = 115
	horizontalPadding int16 = 6
	frameInterval           = 0
	endPause                = 500 * time.Millisecond
)

var (
	viewportPixels   [220 * 115 * 2]byte
	fillScratch      [1024]byte
	rowScratch       [24 * 2]byte
	spanStorage      [32]text.Span
	lineStorage      [4]text.Line
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
	if err := runMarkupJapanese(backend); err != nil {
		println("markup Japanese failed:", err.Error())
	}
}

func runMarkupJapanese(physical display.Backend) error {
	if err := display.FillRect(
		physical,
		display.Rect{Width: displayWidth, Height: displayHeight},
		display.RGB565(16, 16, 24),
		fillScratch[:],
	); err != nil {
		return fmt.Errorf("fill display: %w", err)
	}
	border := display.Rect{
		X: physicalViewport.X - 1, Y: physicalViewport.Y - 1,
		Width: physicalViewport.Width + 2, Height: physicalViewport.Height + 2,
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
	clippedSurface, err := display.NewViewportBackend(surface, localViewport)
	if err != nil {
		return fmt.Errorf("create surface viewport backend: %w", err)
	}

	parseStart := time.Now()
	spans, err := messageParser.ParseInto(spanStorage[:0], message)
	if err != nil {
		return fmt.Errorf("parse markup: %w", err)
	}
	if err := validateGlyphs(spans); err != nil {
		return err
	}
	lines, err := splitLines(lineStorage[:0], spans)
	if err != nil {
		return err
	}
	measurement, err := text.MeasureLines(lines)
	if err != nil {
		return fmt.Errorf("measure markup lines: %w", err)
	}
	if !measurement.HasInk {
		return fmt.Errorf("markup block has no ink")
	}
	parseElapsed := time.Since(parseStart)

	blockWidth := int32(measurement.MaxAdvanceX)
	blockHeight := int32(measurement.Bounds.MaxY) - int32(measurement.Bounds.MinY)
	visibleWidth := int32(viewportWidth) - int32(horizontalPadding)*2
	maxScroll := blockWidth - visibleWidth
	if maxScroll < 0 {
		maxScroll = 0
	}
	blockTop := (int32(viewportHeight) - blockHeight) / 2
	firstBaseline := blockTop - int32(measurement.Bounds.MinY)
	if firstBaseline < math.MinInt16 || firstBaseline > math.MaxInt16 {
		return fmt.Errorf("first baseline %d is outside int16", firstBaseline)
	}
	println("parse us:", parseElapsed.Microseconds(), "spans:", len(spans), "lines:", len(lines))
	println("width:", blockWidth, "height:", blockHeight)
	println("MaxAdvanceX:", measurement.MaxAdvanceX, "AdvanceY:", measurement.AdvanceY)
	println("row scratch bytes:", len(rowScratch), "max scroll:", maxScroll)

	scrollX := int32(0)
	direction := int32(1)
	frameCount := uint32(0)
	time.Sleep(endPause)
	for {
		penX := int32(horizontalPadding) - scrollX
		if penX < math.MinInt16 || penX > math.MaxInt16 {
			return fmt.Errorf("pen X %d is outside int16", penX)
		}

		totalStart := time.Now()
		composeStart := time.Now()
		if err := display.FillRect(surface, display.Rect{Width: viewportWidth, Height: viewportHeight}, display.ColorBlack, fillScratch[:]); err != nil {
			return fmt.Errorf("clear surface: %w", err)
		}
		if _, err := text.DrawLines(clippedSurface, lines, int16(penX), int16(firstBaseline), rowScratch[:]); err != nil {
			return fmt.Errorf("draw markup lines: %w", err)
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
			println("compose ms:", composeElapsed.Milliseconds(), "blit ms:", blitElapsed.Milliseconds(), "total ms:", totalElapsed.Milliseconds())
		}
		time.Sleep(frameInterval)
		if maxScroll == 0 {
			continue
		}
		if direction > 0 {
			if scrollX == maxScroll {
				direction = -1
				time.Sleep(endPause)
			} else {
				scrollX++
			}
		} else if scrollX == 0 {
			direction = 1
			time.Sleep(endPause)
		} else {
			scrollX--
		}
	}
}

func validateGlyphs(spans []text.Span) error {
	for spanIndex := range spans {
		for _, r := range spans[spanIndex].Value {
			if r == '\n' {
				continue
			}
			if _, ok := spans[spanIndex].Font.Lookup(r); !ok {
				return fmt.Errorf("markup span %d font is missing U+%04X", spanIndex, r)
			}
		}
	}
	return nil
}
