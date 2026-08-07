# ModGadget

> 🚧 Experimental. API and package structure may change.

ModGadget is a lightweight UI and device toolkit for building structured TinyGo applications on small embedded systems.

https://github.com/user-attachments/assets/cea078a8-8ea5-4a3d-9d7e-98daf0f6bce5

It provides text layout, multilingual bitmap-font support, viewports, scrolling, direct keyboard events, display integration, and audio support without requiring a full-screen framebuffer.

ModGadget is currently developed primarily for the M5Stack Cardputer ADV and the TinyGo `m5stamp-s3a` target.

## What ModGadget provides

Current functionality includes:

* RGB565 display output through a small `Display` interface
* ST7789 support for the Cardputer ADV
* Unicode text layout and drawing
* Public bitmap-font API
* Named text styles and simple markup
* Synthetic bold text
* Viewports with clipping
* Horizontal scrolling
* Direct keyboard events
* Cardputer ADV keyboard integration
* Cooperative audio playback
* Software volume control
* Small reusable rendering buffers instead of a full-screen framebuffer

For the current high-level API, see [Public API](docs/public-api.md).

For keyboard-event details, see [Keyboard API](docs/keyboard.md).

## Showcase

### Rdon Type 100

[Rdon Type 100](https://github.com/rdon-key/rdon-type100) is a complete multilingual typing game for the M5Stack Cardputer ADV built entirely on ModGadget's public APIs.

It combines:

* Japanese, English, Simplified Chinese, and Korean text
* multilingual bitmap fonts
* styled text layout
* Viewports and scrolling
* direct keyboard input
* Cardputer ADV display integration
* cooperative audio playback
* software volume control

A prebuilt firmware image is available from the [Rdon Type 100 Releases](https://github.com/rdon-key/rdon-type100/releases/latest) page.

## Examples

Practical examples that demonstrate individual or combined ModGadget features are available in:

* [ModGadget Examples](https://github.com/rdon-key/modgadget-examples)

The examples cover multilingual text rendering, horizontal scrolling, application-level vertical document scrolling, audio, and keyboard-driven applications.

Small API-focused examples may also be kept in this repository when they are useful for documenting an individual API.

## Fonts

Generated fonts are distributed separately by:

* [ModGadget Fonts](https://github.com/rdon-key/modgadget-fonts)

Applications can import ready-to-use packages such as:

```
github.com/rdon-key/modgadget-fonts/efont16
github.com/rdon-key/modgadget-fonts/efont24
```

ModGadget provides the public `Font` API together with font validation, measurement, layout, and drawing.

Applications can:

* inspect character coverage with `Font.HasGlyph`
* inspect line metrics with `Font.Metrics`
* load embedded MGF data with `OpenMGF`
* combine fonts with `NewFontStack`
* measure styled text with `MeasureText`

Glyph bitmaps and the internal font engine are intentionally not public APIs.

Font sources, provenance, generation, and validation are maintained separately in [modgadget-font-assets](https://github.com/rdon-key/modgadget-font-assets).

## UI model

A ModGadget application creates a `Gadget` around a display and optional device services.

Text is normally placed inside rectangular `Viewport` regions.

A Viewport can contain styled Unicode text and can either be drawn statically or configured for horizontal scrolling.

`Gadget.Update` advances time-dependent state and processes input. `Gadget.Render` draws Viewports that need updating.

This keeps the application loop explicit and suitable for small cooperative embedded systems.

## Rendering and memory

ModGadget does not require a full-screen framebuffer.

Static content can be rendered directly to the display.

Operations that require buffering, such as scrolling Viewports, use buffers sized to the affected region rather than the entire screen.

For example, an RGB565 240×24 Viewport requires:

```
240 × 24 × 2 = 11,520 bytes
```

Reusable buffers are retained where practical to avoid repeated steady-state allocations.

See [Public API](docs/public-api.md) for detailed rendering and memory behavior.

## Hardware

The primary development platform is currently the M5Stack Cardputer ADV, based on the M5Stamp-S3A.

The current Cardputer ADV integration covers:

* ST7789 240×135 display
* Cardputer ADV keyboard
* Cardputer ADV audio hardware

Hardware-specific integration is kept separate from the generic display, text, and input APIs where practical.

## Build

A TinyGo development version containing the `m5stamp-s3a` target is currently required for Cardputer ADV applications.

Applications using ModGadget are built normally with TinyGo, for example:

```
tinygo build -target=m5stamp-s3a .
```

For working standalone programs, see [ModGadget Examples](https://github.com/rdon-key/modgadget-examples).

For complete TinyGo setup, firmware build, and flashing instructions, see [Rdon Type 100](https://github.com/rdon-key/rdon-type100).

## Project repositories

| Repository                                                                 | Purpose                                              |
| -------------------------------------------------------------------------- | ---------------------------------------------------- |
| [rdon-type100](https://github.com/rdon-key/rdon-type100)                   | Complete multilingual Cardputer ADV application      |
| [modgadget](https://github.com/rdon-key/modgadget)                         | UI, display, input, and device toolkit               |
| [modgadget-fonts](https://github.com/rdon-key/modgadget-fonts)             | Importable MGF bitmap-font packages                  |
| [modgadget-examples](https://github.com/rdon-key/modgadget-examples)       | Standalone ModGadget examples                        |
| [modgadget-font-assets](https://github.com/rdon-key/modgadget-font-assets) | Font sources, provenance, generation, and validation |

## Future work

Possible future work includes:

* General UI widgets
* Touch and pointer input
* Focus and widget event propagation
* Z-index and composition
* Automatic vertical scrolling
* Viewport resizing and removal
* Text input and IME support
* Hardware scrolling
* Panel-specific color profiles

These are not required for the current ModGadget application model and may evolve as the API develops.

## Status

ModGadget is experimental. API compatibility between revisions is not yet guaranteed.

The goal is not to reproduce a desktop GUI framework. ModGadget focuses on small, explicit UI primitives suitable for resource-constrained TinyGo applications.

## License

BSD 3-Clause.

Parts of the ST7789 implementation are derived from `tinygo.org/x/drivers/st7789`.

See `LICENSES/tinygo-drivers-BSD-3-Clause.txt`.

