//go:build tinygo

package main

import (
	"machine"
	"strconv"
	"time"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget/internal/fontdata/mgf/efont24"
	"github.com/rdon-key/modgadget/internal/st7789"
)

const (
	displayWidth int16 = 240
	rowHeight    int16 = 24
)

type tickerConfig struct {
	y      int16
	markup string
	speed  float64
}

func main() {
	time.Sleep(3 * time.Second)
	if err := machine.SPI1.Configure(machine.SPIConfig{Frequency: 40_000_000, Mode: 0, SCK: machine.DISPLAY_SCK, SDO: machine.DISPLAY_MOSI, SDI: machine.NoPin}); err != nil {
		panic(err)
	}
	machine.DISPLAY_BL.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.DISPLAY_BL.High()
	panel := st7789.New(machine.SPI1, machine.DISPLAY_CS, machine.DISPLAY_DC, machine.DISPLAY_RST)
	if err := panel.Configure(st7789.Config{Width: 135, Height: 240, Rotation: st7789.Rotation90, RowOffset: 40, ColumnOffset: 52, Invert: true}); err != nil {
		panic(err)
	}
	font := modgadget.NewMGFFont(efont24.Font)
	styles := modgadget.StyleSet{
		Default: modgadget.Style{Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack},
		Entries: []modgadget.StyleEntry{
			{Name: "seconds", Style: modgadget.Style{Font: font, Foreground: modgadget.RGB565(255, 160, 80), Background: modgadget.ColorBlack}},
			{Name: "japanese", Style: modgadget.Style{Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack}},
			{Name: "chinese", Style: modgadget.Style{Font: font, Foreground: modgadget.RGB565(255, 220, 0), Background: modgadget.ColorBlack}},
			{Name: "english", Style: modgadget.Style{Font: font, Foreground: modgadget.RGB565(80, 220, 255), Background: modgadget.ColorBlack}},
			{Name: "korean", Style: modgadget.Style{Font: font, Foreground: modgadget.RGB565(80, 255, 120), Background: modgadget.ColorBlack}},
		},
	}
	gadget := modgadget.New(panel, modgadget.WithStyles(styles))
	if err := gadget.Clear(); err != nil {
		panic(err)
	}
	secondsView := gadget.Viewport(modgadget.Bounds(0, 0, displayWidth, rowHeight))
	tickers := [...]tickerConfig{
		{y: 27, markup: "<style=japanese>日本語の文字が、小さな画面を静かに流れています。（新聞　電気　新闻　电子）も正しく表示されます。</style>", speed: 24},
		{y: 54, markup: "<style=chinese>中文文字正在小屏幕上缓缓滚动。（新聞　電気　新闻　电子）也能正确显示。</style>", speed: 28},
		{y: 81, markup: "<style=english>English text is moving smoothly across the small screen. The characters “新聞　電気　新闻　电子” are also displayed correctly.</style>", speed: 32},
		{y: 108, markup: "<style=korean>한글 문자가 작은 화면 위를 천천히 흐릅니다.（新聞　電気　新闻　电子）도 올바르게 표시됩니다.</style>", speed: 36},
	}
	for _, ticker := range tickers {
		view := gadget.Viewport(modgadget.Bounds(0, ticker.y, displayWidth, rowHeight))
		if err := view.SetText(ticker.markup); err != nil {
			panic(err)
		}
		view.SetHorizontalScroll(
			modgadget.ScrollSpeed(ticker.speed),
			modgadget.ScrollGap(32),
			modgadget.ScrollLoop(),
		)
	}
	start := time.Now()
	lastSecond := int64(-1)
	for {
		now := time.Now()
		second := int64(now.Sub(start) / time.Second)
		if second != lastSecond {
			lastSecond = second
			value := "<style=seconds>Seconds: " + strconv.FormatInt(second, 10) + "</style>"
			if err := secondsView.SetText(value); err != nil {
				panic(err)
			}
		}
		gadget.Update(now)
		if err := gadget.Render(); err != nil {
			panic(err)
		}
		time.Sleep(16 * time.Millisecond)
	}
}
