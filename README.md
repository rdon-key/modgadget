# ModGadget

> 🚧 Work in progress. API and package structure will change.

ModGadget is a TinyGo display/UI toolkit for small embedded devices.

See [Public API](docs/public-api.md) for the current high-level API.

It currently renders RGB565 graphics directly to an ST7789 without requiring a full framebuffer.

## What works

- M5Stack Stamp-S3A + Cardputer ADV
- ST7789 initialization and rotation
- Rectangle streaming with `BeginRect` / `WritePixels` / `EndRect`
- Solid rectangle fills
- Stride-aware RGB565 bitmap transfer
- Small reusable transfer buffers

## Planned

- Clipping and viewports
- Packed and CJK fonts
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
