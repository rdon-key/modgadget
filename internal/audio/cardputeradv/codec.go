package cardputeradv

import "fmt"

const codecAddress = uint16(0x18)

type i2cBus interface {
	Tx(address uint16, write, read []byte) error
}

type registerWrite struct {
	register byte
	value    byte
}

// The sequence follows M5Unified's Cardputer ADV callback. ES8311 volume 0xbf
// is the 0 dB reference and each register step is 0.5 dB, so 0xef requests
// +24 dB. This intentionally high setting is for Cardputer ADV bench testing;
// PCM peak remains limited separately. The DAC stays muted until zero PCM is
// queued.
var codecPrepareSequence = [...]registerWrite{
	{0x31, 0x60}, // DAC DSM and DEM mute.
	{0x00, 0x80}, // CSM power on/reset.
	{0x01, 0xb5}, // Use BCLK as MCLK.
	{0x02, 0x18}, // Clock multiplier/pre-divider used by M5Unified.
	{0x0d, 0x01}, // Power analog circuitry.
	{0x12, 0x00}, // Power DAC.
	{0x13, 0x10}, // Enable headphone/output driver.
	{0x32, 0xef}, // +24 dB relative to the 0xbf 0 dB reference.
	{0x37, 0x08}, // Bypass DAC equalizer.
	{0x31, 0x60}, // Keep muted while clocks and silence start.
}

func detectCodec(bus i2cBus) error {
	for _, check := range [...]struct{ register, want byte }{{0xfd, 0x83}, {0xfe, 0x11}} {
		write := [1]byte{check.register}
		read := [1]byte{}
		if err := bus.Tx(codecAddress, write[:], read[:]); err != nil {
			return fmt.Errorf("read ES8311 register 0x%02x: %w", check.register, err)
		}
		if read[0] != check.want {
			return fmt.Errorf("unexpected ES8311 register 0x%02x: got 0x%02x, want 0x%02x", check.register, read[0], check.want)
		}
	}
	return nil
}

func prepareCodec(bus i2cBus) error {
	for _, item := range codecPrepareSequence {
		if err := writeCodecRegister(bus, item.register, item.value); err != nil {
			return err
		}
	}
	return nil
}

func unmuteCodec(bus i2cBus) error { return writeCodecRegister(bus, 0x31, 0x00) }

func muteCodec(bus i2cBus) error { return writeCodecRegister(bus, 0x31, 0x60) }

func writeCodecRegister(bus i2cBus, register, value byte) error {
	data := [2]byte{register, value}
	if err := bus.Tx(codecAddress, data[:], nil); err != nil {
		return fmt.Errorf("write ES8311 register 0x%02x: %w", register, err)
	}
	return nil
}
