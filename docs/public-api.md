# ModGadget Public API

> Status: Experimental

The API is under active development. Compatibility between revisions is not
yet guaranteed.

`Display` is the display-output interface that receives RGB565 pixels produced
by ModGadget and transfers them to an LCD or another rectangular pixel target.
It does not represent server-side application infrastructure.

## First `main.go`

The following program can be used from an external Go module. It uses the
public Cardputer ADV device package together with a font distributed by
[ModGadget Fonts](https://github.com/rdon-key/modgadget-fonts).

```go
//go:build tinygo

package main

import (
	"time"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget-fonts/efont24"
	board "github.com/rdon-key/modgadget/device/cardputeradv"
)

func main() {
	time.Sleep(3 * time.Second)

	panel, err := board.ConfigureDisplay()
	if err != nil {
		panic(err)
	}

	font := efont24.Font
	styles := modgadget.StyleSet{
		Default: modgadget.Style{
			Font:       font,
			Foreground: modgadget.ColorWhite,
			Background: modgadget.ColorBlack,
		},
		Entries: []modgadget.StyleEntry{
			{
				Name: "message",
				Style: modgadget.Style{
					Font:       font,
					Foreground: modgadget.RGB565(255, 220, 0),
					Background: modgadget.ColorBlack,
				},
			},
		},
	}

	gadget := modgadget.New(panel, modgadget.WithStyles(styles))
	if err := gadget.Clear(); err != nil {
		panic(err)
	}

	view := gadget.Viewport(modgadget.Bounds(0, 0, 240, 24))
	if err := view.SetText(
		"<style=message>日本語の文字を表示します。</style>",
	); err != nil {
		panic(err)
	}

	if err := gadget.Render(); err != nil {
		panic(err)
	}

	for {
		time.Sleep(time.Second)
	}
}
```

## How the first program works

1. **Configure the display.** The public Cardputer ADV device package configures
   the board-specific ST7789 display and returns a value satisfying
   `modgadget.Display`.
2. **Prepare a Font.** `efont24.Font` is a ready-to-use `modgadget.Font`
   distributed by ModGadget Fonts.
3. **Define Styles.** `Default` supplies the normal font and colors, while
   `message` is selected by markup.
4. **Create the Gadget.** `New` stores the Display and StyleSet.
5. **Clear the screen.** `Gadget.Clear` explicitly fills the whole Display with
   the default background.
6. **Place a Viewport.** `Bounds` uses physical Display coordinates.
7. **Set Text.** `SetText` parses markup and prepares the text layout.
8. **Render and keep the program running.** `Render` draws dirty Viewports, and
   the final wait loop keeps the embedded program alive without redrawing.

## Display

The public interface has the following method set:

```go
type Display interface {
	Size() (width, height int16)
	BeginRect(x, y, width, height int16) error
	WritePixels(data []byte) error
	EndRect() error
}
```

A Display receives row-major RGB565 pixel data for rectangular regions. Each
pixel is transferred as its high byte followed by its low byte. Implementations
can target an LCD, a memory display, a framebuffer sink, or a virtual display
in an emulator.

Display is output-only. It does not include a keyboard, buttons, touch input,
a pointer, audio, storage, or networking.

Applications can implement their own Display using the four methods above.
For the M5Stack Cardputer ADV, the public
`github.com/rdon-key/modgadget/device/cardputeradv` package provides
board-specific display configuration through `ConfigureDisplay`.

## Preparing a Font

Ready-to-use generated fonts are distributed separately by
[ModGadget Fonts](https://github.com/rdon-key/modgadget-fonts).

Only packages imported by the application are linked. For example:

```go
import "github.com/rdon-key/modgadget-fonts/efont24"

font := efont24.Font
```

Available packages include:

- Efont 16dot
- Efont 24dot
- Shinonome 12dot
- Spleen 8×16

Applications can embed their own validated MGF asset without copying it:


```go
//go:embed fonts/custom.mgf
var fontData string

font, err := modgadget.OpenMGF(fontData)
```

`Font` is an opaque value handle. `Valid`, `HasGlyph`, and `Metrics` expose
validation, character coverage, and line metrics without exposing glyph
bitmaps or MGF internals. The zero Font is invalid and safe to inspect.
External Font engine implementations are not currently a public extension
point.

`NewFontStack(primary, fallbacks...)` searches the primary first and then up
to three fallbacks in argument order. Its metrics are the component-wise
maximum of participating fonts.

```go
if !font.HasGlyph('あ') {
	// choose another font or report missing coverage
}

measurement, err := modgadget.MeasureText(value, styles)
```

## Style

The public style structures are conceptually:

```go
type StyleSet struct {
	Default Style
	Entries []StyleEntry
}

type Style struct {
	Font       Font
	Foreground Color565
	Background Color565
	Bold       bool
}

type StyleEntry struct {
	Name  string
	Style Style
}
```

`StyleSet.Default` applies to untagged text, supplies the Viewport clear color,
and supplies the color used by `Gadget.Clear`. `Entries` contains named Styles.
Names are case-sensitive and searched from the beginning of the slice, so the
first entry wins when names are duplicated.

Example markup:

```text
<style=message>日本語の文字を表示します。</style>
```

`Foreground` is used for set glyph pixels. `Background` is used within glyph
rectangles. The whole Viewport is first filled with the Default background.
`Bold` uses synthetic bold rendering: set bitmap pixels are extended one pixel
to the right without changing the font asset or glyph advance. Its zero value
is false, so existing text rendering is unchanged.

## Gadget

### Construction

```go
func New(display Display, options ...Option) *Gadget
func WithStyles(styles StyleSet) Option
```

`New` stores the Display and applies Options in order. A nil Display is accepted
at construction time, but `Clear` and `Render` return an error when used.

### Clear

```go
func (g *Gadget) Clear() error
```

`Clear` obtains the complete area from `Display.Size` and fills it with
`StyleSet.Default.Background`. It is explicit and is never called implicitly by
`Render`. It does not change Viewport text or dirty state.

The operation streams pixels with a reusable 64-byte scratch buffer rather than
allocating a screen-sized framebuffer. Steady-state `Clear` has been tested at
zero allocations. Display transfer errors are returned.

### Viewport, Update, and Render

```go
func (g *Gadget) Viewport(options ...ViewportOption) *Viewport
func (g *Gadget) Update(now time.Time)
func (g *Gadget) Render() error
```

`Viewport` creates and registers a Viewport. `Update` advances time-dependent
scroll state using the supplied time. `Render` processes dirty Viewports in
registration order. A successful Viewport is marked clean. On the first error,
that Viewport remains dirty and processing stops for that call.

## Placing Viewports

`Bounds(x, y, width, height)` selects an absolute rectangle on the physical
Display. Drawing inside the Viewport uses local coordinates and is clipped to
that rectangle. Omitting Bounds uses the complete Display area.

Example layout for a 240×135 Display:

```go
seconds := gadget.Viewport(
	modgadget.Bounds(0, 0, 240, 24),
)

japanese := gadget.Viewport(
	modgadget.Bounds(0, 27, 240, 24),
)

chinese := gadget.Viewport(
	modgadget.Bounds(0, 54, 240, 24),
)

english := gadget.Viewport(
	modgadget.Bounds(0, 81, 240, 24),
)

korean := gadget.Viewport(
	modgadget.Bounds(0, 108, 240, 24),
)
```

Viewport resize and deletion are not available. There is no z-index or
compositor. If Viewports overlap, later drawing can overwrite earlier pixels,
and unchanged overlapping Viewports are not automatically recomposed.

## Text and markup

```go
func (v *Viewport) SetText(value string) error
func (v *Viewport) Clear()
```

`SetText` accepts Unicode text, parses markup, prepares a layout, measures the
text, and marks the Viewport dirty. Supported markup is:

| Syntax | Meaning |
| --- | --- |
| `<style=name>` | Select a named Style |
| `</style>` | Restore the surrounding Style |
| `<b>` | Enable synthetic bold while preserving the current Style |
| `</b>` | Restore the Style in effect before `<b>` |
| `<br>` | Line break |
| `<br/>` | Line break |
| `<<` | Literal `<` |

Invalid UTF-8, malformed or unknown tags, unknown Style names, nil selected
fonts, and missing glyphs return errors. `Viewport.Clear` sets the text to the
empty string and returns no error.

Setting the same string normally does nothing. In a one-shot scroll mode, it
instead resets and restarts that animation.

Arbitrary HTML, inline CSS, images, links, alignment, scroll tags, and alpha
composition are not supported.

## Right-to-left one-shot scroll

The current one-shot example starts outside the right edge and moves left:

```go
view.SetHorizontalScroll(
	modgadget.ScrollSpeed(24),
	modgadget.ScrollFromRight(),
)
```

Drive it with an externally supplied clock:

```go
for {
	now := time.Now()

	gadget.Update(now)

	if err := gadget.Render(); err != nil {
		panic(err)
	}

	time.Sleep(16 * time.Millisecond)
}
```

The position is calculated from absolute elapsed time rather than by adding a
fixed number of pixels per frame:

```text
distance = floor(elapsedSeconds × pixelsPerSecond)
x = viewportWidth - distance
```

The text begins fully outside the right edge. It finishes after traveling
`viewportWidth + textWidth`, when it is fully outside the left edge. The final
dirty frame transfers a background-only Surface. After that successful transfer
the Viewport stays clean and sends no additional frames. Short text moves too.

## Other scroll modes

| Setting | Start position | Direction | Loop | After completion | Short text |
| --- | ---: | --- | --- | --- | --- |
| No scroll | `0` | None | No | Static | Static |
| `ScrollSpeed` | `0` | Left | No | Stops on the final visible portion | Does not move |
| `ScrollSpeed` + `ScrollLoop` | `0` | Left | Yes | Never completes | Does not move |
| `ScrollSpeed` + `ScrollFromLeft` | `-textWidth` | Right | No | Clears and stops | Moves |
| `ScrollSpeed` + `ScrollFromRight` | `viewportWidth` | Left | No | Clears and stops | Moves |
| `ScrollSpeed` + `ScrollFromRight` + `ScrollLoop` | `viewportWidth` | Left | Yes | Never completes | Moves |

Loop example:

```go
view.SetHorizontalScroll(
	modgadget.ScrollSpeed(30),
	modgadget.ScrollGap(32),
	modgadget.ScrollLoop(),
)
```

Looping draws the current copy at `-offset` and the next copy after
`textWidth + gap`. `ScrollFromRight` and `ScrollLoop` can be combined: the first
copy starts outside the right edge and subsequent copies use `ScrollGap`.
`ScrollFromLeft` remains a one-shot mode. Gap is unused by one-shot modes. A
speed of zero or less disables scrolling.
Negative Gap values are currently accepted without validation and can produce
overlapping or unusual loop cycles.

`ScrollFromLeft` means that text appears from the left and travels right.
`ScrollFromRight` means that text appears from the right and travels left.

## Buffer and memory behavior

Static Viewports use direct drawing. A normal horizontal scroll uses a Surface
when speed is positive and the text is wider than the Viewport. A one-shot
scroll uses a Surface even for short text.

A Surface requires:

```text
width × height × 2 bytes
```

For a 240×24 Viewport:

```text
240 × 24 × 2 = 11,520 bytes
```

For four such Viewports:

```text
11,520 × 4 = 46,080 bytes
```

No screen-sized framebuffer is required. A Surface is allocated on the first
buffered Render and retained for reuse, including across text changes. Tests
verify zero allocations during steady-state buffered Render. `SetText` may
allocate while parsing markup, constructing the layout, and sizing glyph
scratch storage.

## Keyboard

ModGadget drains and dispatches direct physical key events during
`Gadget.Update`. The current API is Key mode only; it does not implement text
composition or IME conversion. Keyboard input is independent of `Display`.

See [Keyboard API](keyboard.md) for event fields, handler and listener rules,
and the Cardputer ADV adapter.

An optional `VolumeController` enables standard shortcuts before application
handlers: Fn+= raises volume, Fn+- lowers it, and Fn+M toggles mute. The KeyDown
performs the operation; both it and its captured KeyUp are consumed, including
when Fn is released first. Without a controller both remain ordinary key events.

## Public API reference

Only identifiers currently exported by the root package are listed here.

### Display

| Signature | Description | Error and notes |
| --- | --- | --- |
| `type Display interface { ... }` | Rectangular RGB565 display output | Users implement its four methods |
| `Size() (width, height int16)` | Reports the complete display area | No error |
| `BeginRect(x, y, width, height int16) error` | Begins a rectangular transfer | Returns implementation errors |
| `WritePixels(data []byte) error` | Writes RGB565 bytes | High byte then low byte |
| `EndRect() error` | Completes the transfer | Returns implementation errors |

### Gadget

| Signature | Description | Error and notes |
| --- | --- | --- |
| `type Gadget struct` | Root display object | Fields are private |
| `func New(display Display, options ...Option) *Gadget` | Creates a Gadget | No error; nil Display fails later |
| `func (g *Gadget) Clear() error` | Clears the whole Display | Returns Display transfer errors |
| `func (g *Gadget) Viewport(options ...ViewportOption) *Viewport` | Registers a Viewport | No error |
| `func (g *Gadget) Update(now time.Time)` | Updates scroll state | No transfer and no error |
| `func (g *Gadget) Render() error` | Renders dirty Viewports | Stops at first error |

### Viewport

| Signature | Description | Error and notes |
| --- | --- | --- |
| `type Viewport struct` | Text display region | Fields are private |
| `func Bounds(x, y, width, height int16) ViewportOption` | Sets physical bounds | Invalid dimensions fail during Render |
| `func (v *Viewport) SetText(value string) error` | Parses and lays out text | Markup and glyph errors are returned |
| `func (v *Viewport) SetHorizontalScroll(options ...ScrollOption)` | Replaces scroll settings | Resets start and completion state |
| `func (v *Viewport) Clear()` | Empties Viewport text | Does not return an error |
| `func (v *Viewport) ScrollTo(pixel int16)` | Sets horizontal offset | Negative values clamp to zero |

### Scroll

| Signature | Description | Error and notes |
| --- | --- | --- |
| `type ScrollOption func(*horizontalScroll)` | Opaque scroll option | Use supplied constructors |
| `func ScrollSpeed(pixelsPerSecond float64) ScrollOption` | Sets speed | Non-positive disables scroll |
| `func ScrollGap(pixels int16) ScrollOption` | Sets loop gap | Negative values are accepted |
| `func ScrollLoop() ScrollOption` | Enables a gap-separated leftward loop | Combines with `ScrollFromRight` |
| `func ScrollFromLeft() ScrollOption` | Moves right once from the left | Short text moves |
| `func ScrollFromRight() ScrollOption` | Starts at the right and moves left | One-shot alone; loops with `ScrollLoop` |

### Style

| Signature | Description | Error and notes |
| --- | --- | --- |
| `type StyleSet` | Default and named Styles | First duplicate name wins |
| `type Style` | Font, RGB565 colors, and Bold | An invalid Font is an error |
| `type StyleEntry` | Name-to-Style entry | Names are case-sensitive |
| `func (styles StyleSet) Lookup(name string) (Style, bool)` | Finds the first exact name | No error; false if absent |

### Font

| Signature | Description | Error and notes |
| --- | --- | --- |
| `type Font` | Opaque, copyable font handle | Zero value is invalid and safe |
| `type FontMetrics` | Baseline-relative Ascent, Descent, and LineGap | Values define the font's line metrics |
| `func (Font) Valid() bool` | Reports whether a Font is usable | False for zero Font |
| `func (Font) HasGlyph(r rune) bool` | Reports character coverage | Does not expose bitmap data |
| `func (Font) Metrics() FontMetrics` | Returns line metrics | Zero metrics for zero Font |
| `func OpenMGF(data string) (Font, error)` | Validates an MGF string without copying it | Rejects malformed data |
| `func MustOpenMGF(data string) Font` | Opens static trusted MGF data | Panics on malformed data |
| `func NewFontStack(primary Font, fallbacks ...Font) (Font, error)` | Creates ordered fallback lookup | Up to three fallbacks |
| `type TextMeasurement` | Width and LineCount | Uses Viewport layout rules |
| `func MeasureText(value string, styles StyleSet) (TextMeasurement, error)` | Measures markup without rendering | Returns markup and glyph errors |

### Color

| Signature | Description | Error and notes |
| --- | --- | --- |
| `type Color565` | RGB565 color value | No error |
| `func RGB565(red, green, blue uint8) Color565` | Converts 8-bit RGB | Quantizes to 5/6/5 bits |
| `ColorBlack` | Black | Constant |
| `ColorWhite` | White | Constant |
| `ColorRed` | Red | Constant |
| `ColorGreen` | Green | Constant |
| `ColorBlue` | Blue | Constant |

### Constructor and option types

| Signature | Description | Error and notes |
| --- | --- | --- |
| `type Option func(*Gadget)` | Gadget option | Nil options are ignored |
| `func WithStyles(styles StyleSet) Option` | Supplies Styles | Applied during New |
| `func WithKeyboard(keyboard Keyboard) Option` | Supplies a key-event source | Nil means no Keyboard |
| `func WithVolumeController(controller VolumeController) Option` | Enables standard volume shortcuts | Nil disables system volume handling |
| `type ViewportOption func(*Viewport)` | Viewport option | Nil options are ignored |

### Keyboard

| Signature | Description | Error and notes |
| --- | --- | --- |
| `type Keyboard interface { ReadKeyEvent() (KeyEvent, bool) }` | Queued direct key source | Empty is false, not an error |
| `type VolumeController interface { VolumeUp(); VolumeDown(); ToggleMute() }` | Minimal system volume target | Independent from a concrete audio device |
| `type KeyEvent struct { Code KeyCode; Rune rune; Action KeyAction; Modifiers Modifiers }` | Logical key transition after layer mapping | Rune is one direct printable character |
| `type KeyCode uint16` | Layer-applied logical key identity | Standard values follow USB HID usages |
| `type KeyAction uint8` | Unknown, press, or release | Zero is unknown; no repeat action |
| `type Modifiers uint16` | Modifier bit set | Shift, Control, Alt, Meta, and Fn |
| `func (m Modifiers) Has(value Modifiers) bool` | Tests all requested bits | No error |
| `type KeyHandler func(KeyEvent) bool` | Handles or consumes an event | True stops later handlers |
| `type ListenerID uint16` | Listener identity | Zero is invalid |
| `func (g *Gadget) OnKey(handler KeyHandler) ListenerID` | Registers in call order | Nil returns zero |
| `func (g *Gadget) RemoveListener(id ListenerID) bool` | Removes by ID | Reports whether it removed a live listener |

## Current limitations

- Types needed to implement a custom Font engine are not exported by the root
  package.
- There are no general UI Widgets.
- Keyboard provides direct Key events only; there is no Text input or IME API.
- There is no Touch or Pointer interface.
- There is no focus or Widget event-propagation model.
- There is no z-index or compositor.
- There is no automatic vertical scroll.
- Viewports cannot be resized or removed.
- API compatibility is not guaranteed while the package is experimental.
