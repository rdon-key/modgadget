# ModGadget

> 🚧 Work in progress. API and package structure will change.

ModGadget is a TinyGo display/UI toolkit for small embedded devices.

See [Public API](docs/public-api.md) for the current high-level API.
See [Keyboard API](docs/keyboard.md) for direct key-event input.
With a volume controller, the standard Cardputer ADV keys are Fn+=, Fn+-, and Fn+M.

Small, API-focused examples are kept in [`examples`](examples/). More practical
examples that combine several ModGadget features are in
[`modgadget-examples`](https://github.com/rdon-key/modgadget-examples). For a
complete application, see
[`rdon-type100`](https://github.com/rdon-key/rdon-type100).

Generated fonts are distributed separately by
[`modgadget-fonts`](https://github.com/rdon-key/modgadget-fonts). Applications
can import packages such as `github.com/rdon-key/modgadget-fonts/efont16` and
`github.com/rdon-key/modgadget-fonts/efont24`. ModGadget provides the public
`Font` API together with font loading, measurement, layout, and drawing. It can
load embedded MGF data with `OpenMGF`, inspect coverage with `Font.HasGlyph`,
combine fonts with `NewFontStack`, and measure markup with `MeasureText`.
Glyph bitmaps and custom Font engine implementations are intentionally not
public APIs.

It currently renders RGB565 graphics directly to an ST7789 without requiring a full framebuffer.

## What works

- M5Stack Stamp-S3A + Cardputer ADV
- ST7789 initialization and rotation
- Rectangle streaming with `BeginRect` / `WritePixels` / `EndRect`
- Solid rectangle fills
- Stride-aware RGB565 bitmap transfer
- Small reusable transfer buffers

## Planned

- Widgets and dirty-region rendering
- Hardware scrolling
- Panel-specific color profiles

## Build

A local TinyGo tree containing the `m5stamp-s3a` target is currently required.

    tinygo flash -target=m5stamp-s3a -monitor .

## Status

Experimental prototype. Not ready as a stable library.

## License

BSD 3-Clause.

Parts of the ST7789 implementation are derived from `tinygo.org/x/drivers/st7789`.
See `LICENSES/tinygo-drivers-BSD-3-Clause.txt`.
