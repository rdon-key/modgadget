package cardputeradv

import "errors"

// VolumeLevel selects a software PCM gain level.
type VolumeLevel uint8

const (
	// VolumeMute advances playback while emitting zero PCM.
	VolumeMute VolumeLevel = iota
	// VolumeLow applies 25 percent PCM gain.
	VolumeLow
	// VolumeMedium applies 50 percent PCM gain.
	VolumeMedium
	// VolumeHigh applies 100 percent PCM gain.
	VolumeHigh
)

// ErrInvalidVolume reports a VolumeLevel outside the supported range.
var ErrInvalidVolume = errors.New("cardputeradv audio: invalid volume")

// Next returns the next volume level, wrapping from high to mute.
func (level VolumeLevel) Next() VolumeLevel {
	if level >= VolumeHigh {
		return VolumeMute
	}
	return level + 1
}

// String returns the short display name of level.
func (level VolumeLevel) String() string {
	switch level {
	case VolumeMute:
		return "MUTE"
	case VolumeLow:
		return "LOW"
	case VolumeMedium:
		return "MEDIUM"
	case VolumeHigh:
		return "HIGH"
	default:
		return "UNKNOWN"
	}
}

func (level VolumeLevel) gain256() int32 {
	switch level {
	case VolumeLow:
		return 64
	case VolumeMedium:
		return 128
	case VolumeHigh:
		return 256
	default:
		return 0
	}
}

func applyVolumePCM(data []byte, level VolumeLevel) {
	if level == VolumeHigh {
		return
	}
	gain := level.gain256()
	for index := 0; index+1 < len(data); index += 2 {
		value := int16(uint16(data[index]) | uint16(data[index+1])<<8)
		scaled := int16(int32(value) * gain / 256)
		data[index] = byte(scaled)
		data[index+1] = byte(uint16(scaled) >> 8)
	}
}
