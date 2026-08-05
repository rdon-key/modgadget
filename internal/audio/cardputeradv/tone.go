package cardputeradv

const (
	// maxAmplitude leaves PCM headroom for the Cardputer ADV codec's +24 dB
	// bench setting. Software LOW/MEDIUM/HIGH then apply 25/50/100 percent.
	maxAmplitude   = int32(2048)
	envelopeFrames = uint32(SampleRate * 4 / 1000)
)

// Quarter-wave Q15 sine table. Symmetry reconstructs a 256-step cycle.
var sineQuarter = [...]int16{
	0, 804, 1608, 2410, 3212, 4011, 4808, 5602,
	6393, 7179, 7962, 8739, 9512, 10278, 11039, 11793,
	12539, 13279, 14010, 14732, 15446, 16151, 16846, 17530,
	18204, 18868, 19519, 20159, 20787, 21403, 22005, 22594,
	23170, 23731, 24279, 24811, 25329, 25832, 26319, 26790,
	27245, 27683, 28105, 28510, 28898, 29268, 29621, 29956,
	30273, 30571, 30852, 31113, 31356, 31580, 31785, 31971,
	32137, 32285, 32412, 32521, 32609, 32678, 32728, 32757,
	32767,
}

type tone struct {
	phase       uint32
	phaseStep   uint32
	totalFrames uint32
	frame       uint32
}

func (t *tone) start(frequency, frames uint32) {
	t.phase = 0
	t.phaseStep = uint32((uint64(frequency) << 32) / SampleRate)
	t.totalFrames = frames
	t.frame = 0
}

func (t *tone) reset() { *t = tone{} }

func (t *tone) active() bool { return t.frame < t.totalFrames }

func (t *tone) fill(dst []byte) int {
	frames := len(dst) / BytesPerFrame
	written := 0
	for written < frames && t.active() {
		sample := t.sample()
		putSample(dst[written*BytesPerFrame:], sample)
		t.phase += t.phaseStep
		t.frame++
		written++
	}
	for i := written * BytesPerFrame; i < len(dst); i++ {
		dst[i] = 0
	}
	return written
}

func (t *tone) sample() int16 {
	index := uint8(t.phase >> 24)
	quadrant := index >> 6
	offset := index & 63
	var q uint8
	if quadrant&1 == 0 {
		q = offset
	} else {
		q = 64 - offset
	}
	value := int32(sineQuarter[q])
	if quadrant >= 2 {
		value = -value
	}

	gain := envelopeFrames
	if t.totalFrames < gain*2 {
		gain = t.totalFrames / 2
	}
	if gain > 0 {
		if t.frame < gain {
			value = value * int32(t.frame) / int32(gain)
		}
		remaining := t.totalFrames - t.frame - 1
		if remaining < gain {
			value = value * int32(remaining) / int32(gain)
		}
	}
	return int16(value * maxAmplitude / 32767)
}

func putSample(dst []byte, sample int16) {
	value := uint16(sample)
	// ESP32-S3 GDMA feeds little-endian 16-bit slots to the I2S shifter.
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
	dst[2] = byte(value)
	dst[3] = byte(value >> 8)
}
