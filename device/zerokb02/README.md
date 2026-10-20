# Zero-KB02 OLED backend

`ConfigureDisplay()` returns a `modgadget.Display` backed by the 128x64
SSD1306 OLED on the Waveshare RP2040-Zero used by TinyGo Keeb Tour.

- I2C0: GPIO12 (SDA), GPIO13 (SCL), 400 kHz
- SSD1306 address: 0x3C
- Display rotation: 180 degrees (matching the upside-down mounting)
- Pixel conversion: RGB565 black -> off, any nonblack value -> on
- RAM used by the panel framebuffer: 1024 bytes (plus transfer state)
- Display updates: whole 1024-byte frame after every completed rectangle

Use `modgadget.New(display, ...)` as for Cardputer ADV; text layout,
MGF and MGZ fonts are unchanged.

The implementation deliberately covers **only the display**. Zero-KB02's
key matrix, joystick, rotary encoder, LEDs and USB HID are separate
features, not implemented by this package.

See [the Traditional Chinese greeting example](../../examples/zerokb02-traditional/).
