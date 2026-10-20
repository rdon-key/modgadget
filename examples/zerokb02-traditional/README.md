# Traditional Chinese on Zero-KB02

This example renders the text **恭喜進軍 / 台灣！** on the 128x64
SSD1306 OLED of the TinyGo Keeb Tour Zero-KB02 (Waveshare RP2040-Zero).
It uses the MGZ-compressed 24-pixel Efont font. The font is supplied by
`modgadget-fonts`, not copied into this repository.

From the ModGadget repository root:

```sh
go get github.com/rdon-key/modgadget-fonts/efont24
tinygo build -target=waveshare-rp2040-zero -size=short -o zerokb02.uf2 ./examples/zerokb02-traditional
tinygo flash -target=waveshare-rp2040-zero ./examples/zerokb02-traditional
```

The `go get` step adds the font module to the local `go.mod` and `go.sum`.
A TinyGo version supporting the local `go.mod` Go version is required.

**Flash budget:** The `efont24` MGZ asset alone is approximately 1.52 MB.
The RP2040-Zero has 2 MB of flash, so check the full firmware size before
flashing. If it does not fit, generate a smaller subset MGZ containing the
required characters and import that instead.

The display backend uses I2C0 at 400 kHz, SDA GPIO12, SCL GPIO13 and address
0x3C. It initializes the OLED in the physical 180-degree orientation.
Pixels are monochrome: black is off, any other RGB565 color is on.
Full OLED buffers are transferred at the end of each rectangle; this is
intentionally optimized for simplicity rather than high refresh rates.

This example supports the OLED only. The keyboard matrix, joystick,
rotary encoder and LEDs are not initialized.
