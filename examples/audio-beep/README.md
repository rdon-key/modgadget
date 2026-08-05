# Cardputer ADV San-San-Nana Audio Experiment

This is a Cardputer ADV-only experiment, not a public ModGadget Audio API. It
initializes the onboard ES8311 codec and sends signed 16-bit stereo PCM through
ESP32-S3 I2S1/GDMA to the speaker path. The NS4150B amplifier is controlled by
the board's headphone-detect circuit; inserting a 3.5 mm plug disables the
built-in speaker amplifier.

The example waits two seconds and plays the Japanese 3-3-7 cheering rhythm with
thirteen 880 Hz sine tones. Each tone and within-group gap is 65 ms, each of the
two group gaps is 180 ms, and the final release silence is 50 ms. `PlayPattern`
only initializes fixed sequence state and returns. Gaps are zero PCM, not codec
mute operations.

The main loop repeatedly calls `Update`; each call submits at most 48 frames,
or 1 ms at 48 kHz, and never sleeps or waits for DMA completion. DMA uses one
finite descriptor at a time. `Update` reuses its fixed buffer only after the
ESP32-S3 GDMA OUT EOF flag reports completion. A final sequence of finite
zero-filled descriptors is sent after the pattern; there is no self-linked DMA
chain.

The format is 48 kHz, two identical channels, signed 16-bit samples in Philips
I2S format. The player has one 192-byte generation buffer and the device has one
reused 192-byte DMA buffer. The PCM peak is 4096 (one eighth of full scale),
with a 4 ms attack and release. ES8311 DAC volume register `0xef` is +24 dB
relative to its `0xbf` 0 dB reference; this deliberately high codec setting is
for Cardputer ADV bench testing while PCM peak remains limited to 4096.
Use caution with headphones and speakers even though this experiment uses a
short tone and conservative levels.

TinyGo 0.40.1 has no public ESP32-S3 `machine.I2S`. The internal implementation
therefore uses the generated `device/esp` I2S1 and GDMA register definitions,
following Espressif's ESP32-S3 low-level I2S implementation. It is intentionally
fixed to the Cardputer ADV pins and is not a generic I2S driver.

Build with the same local target used by the other Cardputer ADV examples:

```sh
tinygo build -target=m5stamp-s3a ./examples/audio-beep
```

Do not flash until the target definition and board wiring have been reviewed.
