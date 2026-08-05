package cardputeradv

// ESP32-S3 I2S1 uses the 160 MHz PLL and an exact 13+1/48 fractional
// divider. ES8311 is configured to derive its master clock from BCLK.
const (
	i2sSourceClockHz = uint32(160_000_000)
	i2sClockDivInt   = uint32(13)
	i2sClockDivNum   = uint32(1)
	i2sClockDivDen   = uint32(48)
	i2sBCLKDivider   = uint32(8)
)

func configuredSampleRate() uint32 {
	moduleClock := uint64(i2sSourceClockHz) * uint64(i2sClockDivDen) /
		uint64(i2sClockDivInt*i2sClockDivDen+i2sClockDivNum)
	return uint32(moduleClock / uint64(i2sBCLKDivider*Channels*BitsPerSample))
}
