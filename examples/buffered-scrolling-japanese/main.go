package main

import (
	"fmt"
	"machine"
	"math"
	"time"

	fontpkg "github.com/rdon-key/modgadget-fonts/font"
	"github.com/rdon-key/modgadget/internal/display"
	"github.com/rdon-key/modgadget/internal/fontdata/shinonome12"
	"github.com/rdon-key/modgadget/internal/fontdata/spleen8x16"
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

type styledText struct {
	value      string
	foreground display.Color565
}

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
	if err := runBufferedScrollingJapanese(backend); err != nil {
		println("buffered scrolling Japanese failed:", err.Error())
	}
}

func runBufferedScrollingJapanese(physical display.Backend) error {
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
	layout, primaryCount, fallbackCount, err := buildJapaneseLayout(maxAdvanceX)
	if err != nil {
		return fmt.Errorf("build Japanese layout: %w", err)
	}
	measurement := layout.Measurement()
	if !measurement.HasInk {
		return fmt.Errorf("Japanese layout has no ink")
	}
	contentHeight := int32(measurement.Bounds.MaxY) - int32(measurement.Bounds.MinY)
	visibleHeight := int32(viewportHeight) - int32(verticalPadding)*2
	maxScroll := contentHeight - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	initialBaseline := int32(verticalPadding) - int32(measurement.Bounds.MinY)
	printLayoutInfo(&layout, primaryCount, fallbackCount, maxAdvanceX, contentHeight, visibleHeight, maxScroll, initialBaseline)

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
			return fmt.Errorf("draw Japanese layout to surface: %w", err)
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

func buildJapaneseLayout(maxAdvanceX int16) (textdraw.TextLayout, int, int, error) {
	paragraphs := [...]styledText{
		{value: "ModGadget 日本語表示デモ\n\n", foreground: display.RGB565(255, 255, 0)},
		{value: "小さな画面に、読みやすい日本語の文章を表示します。文字は表示領域の幅に合わせて折り返され、画面の外へ出た部分は描画されません。\n\n", foreground: display.RGB565(0, 255, 255)},
		{value: "このデモでは、文章全体をいったんメモリ上の画面へ描きます。背景と文字を合成してから液晶へ転送するため、画面を消去する途中や、文字を一文字ずつ描く途中の状態は表示されません。\n\n", foreground: display.ColorGreen},
		{value: "表示内容は一ピクセルずつ上下に移動します。ひらがな、カタカナ、漢字、句読点、数字、英字を含む文章を使い、日本語の折り返しと滑らかなスクロールを確認します。\n\n", foreground: display.ColorWhite},
		{value: "画面転送にはRGB565形式と40MHzのSPI通信を使用しています。限られたメモリを活用しながら、小型端末、ログ表示器、ニュース端末、機器の操作画面に使える表示基盤を目指します。", foreground: display.RGB565(0, 255, 255)},
	}

	var spans []textdraw.Span
	primaryCount := 0
	fallbackCount := 0
	for _, paragraph := range paragraphs {
		var err error
		spans, primaryCount, fallbackCount, err = appendFontSpans(
			spans,
			paragraph.value,
			paragraph.foreground,
			primaryCount,
			fallbackCount,
		)
		if err != nil {
			return textdraw.TextLayout{}, primaryCount, fallbackCount, err
		}
	}
	layout, err := textdraw.NewWrappedTextLayout(spans, maxAdvanceX)
	if err != nil {
		return textdraw.TextLayout{}, primaryCount, fallbackCount, err
	}
	return layout, primaryCount, fallbackCount, nil
}

func appendFontSpans(spans []textdraw.Span, value string, foreground display.Color565, primaryCount, fallbackCount int) ([]textdraw.Span, int, int, error) {
	var currentFace *fontpkg.Font
	segmentStart := 0
	for runeStart, r := range value {
		face := currentFace
		if r != '\n' && r != '\r' && r != '\t' {
			if _, ok := shinonome12.Font.Lookup(r); ok {
				face = &shinonome12.Font
				primaryCount++
			} else if _, ok := spleen8x16.Font.Lookup(r); ok {
				face = &spleen8x16.Font
				fallbackCount++
			} else {
				return spans, primaryCount, fallbackCount, fmt.Errorf("Japanese demo fonts are missing U+%04X", r)
			}
		} else if face == nil {
			face = &shinonome12.Font
		}

		if currentFace == nil {
			currentFace = face
		} else if face != currentFace {
			spans = append(spans, textdraw.Span{
				Face:       currentFace,
				Value:      value[segmentStart:runeStart],
				Foreground: foreground,
				Background: display.ColorBlack,
			})
			segmentStart = runeStart
			currentFace = face
		}
	}
	if currentFace != nil {
		spans = append(spans, textdraw.Span{
			Face:       currentFace,
			Value:      value[segmentStart:],
			Foreground: foreground,
			Background: display.ColorBlack,
		})
	}
	return spans, primaryCount, fallbackCount, nil
}

func printLayoutInfo(layout *textdraw.TextLayout, primaryCount, fallbackCount int, maxAdvanceX int16, contentHeight, visibleHeight, maxScroll, initialBaseline int32) {
	measurement := layout.Measurement()
	println("primary font: shinonome12.Font")
	println("fallback font: spleen8x16.Font")
	println("glyph scratch bytes:", len(glyphScratch))
	println("primary glyph count:", primaryCount)
	println("fallback glyph count:", fallbackCount)
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
