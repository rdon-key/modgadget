// Package cardputeradv provides cooperative audio playback for M5Stack
// Cardputer ADV devices.
package cardputeradv

import (
	"time"

	internalaudio "github.com/rdon-key/modgadget/internal/audio/cardputeradv"
)

// Player cooperatively generates and submits audio for a Cardputer ADV.
// Its fields and hardware implementation are private.
type Player struct{ impl *internalaudio.Player }

// Pattern identifies one built-in Cardputer ADV sound pattern.
type Pattern uint8

// VolumeLevel selects a software PCM gain level.
type VolumeLevel uint8

const (
	// PatternStartup is a short rising two-tone startup sound.
	PatternStartup Pattern = iota + 1
	// PatternClick is a short UI input sound.
	PatternClick
	// PatternWrong is a low error sound.
	PatternWrong
	// PatternCorrect is a descending two-tone success sound.
	PatternCorrect
)

const (
	// VolumeMute advances playback while emitting zero PCM.
	VolumeMute VolumeLevel = iota
	// VolumeLow applies 25 percent PCM gain.
	VolumeLow
	// VolumeMedium applies 50 percent PCM gain and is the initial level.
	VolumeMedium
	// VolumeHigh applies 100 percent PCM gain.
	VolumeHigh
)

var (
	// ErrNotConfigured reports an operation on an unconfigured Player.
	ErrNotConfigured error
	// ErrInvalidFrequency reports an unsupported tone frequency.
	ErrInvalidFrequency error
	// ErrInvalidDuration reports a non-positive tone duration.
	ErrInvalidDuration error
	// ErrInvalidPattern reports an unknown built-in Pattern.
	ErrInvalidPattern error
	// ErrInvalidVolume reports a VolumeLevel outside the supported range.
	ErrInvalidVolume error
)

func init() {
	ErrNotConfigured = internalaudio.ErrNotConfigured
	ErrInvalidFrequency = internalaudio.ErrInvalidFrequency
	ErrInvalidDuration = internalaudio.ErrInvalidDuration
	ErrInvalidPattern = internalaudio.ErrInvalidPattern
	ErrInvalidVolume = internalaudio.ErrInvalidVolume
}

// New returns an unconfigured Player connected to Cardputer ADV audio
// hardware. Applications normally obtain a configured player from
// device/cardputeradv.ConfigureAudio.
func New() *Player { return &Player{impl: internalaudio.New()} }

func (p *Player) internal() *internalaudio.Player {
	if p == nil {
		return nil
	}
	return p.impl
}

// Configure initializes the Cardputer ADV codec and I2S transmitter.
func (p *Player) Configure() error { return p.internal().Configure() }

// PlayTone replaces current playback with a tone and returns without sending PCM.
func (p *Player) PlayTone(frequencyHz uint16, duration time.Duration) error {
	return p.internal().PlayTone(frequencyHz, duration)
}

// PlayPattern replaces current playback with a built-in fixed pattern.
func (p *Player) PlayPattern(pattern Pattern) error {
	return p.internal().PlayPattern(internalaudio.Pattern(pattern))
}

// Update performs at most one non-blocking audio submission.
func (p *Player) Update() error { return p.internal().Update() }

// Busy reports whether audio or its final silence buffer is pending.
func (p *Player) Busy() bool { return p.internal().Busy() }

// Stop cancels playback and submits a finite silence chunk.
func (p *Player) Stop() error { return p.internal().Stop() }

// SetVolume changes the software gain used by subsequent audio submissions.
func (p *Player) SetVolume(level VolumeLevel) error {
	return p.internal().SetVolume(internalaudio.VolumeLevel(level))
}

// Volume returns the current software gain level.
func (p *Player) Volume() VolumeLevel { return VolumeLevel(p.internal().Volume()) }

// VolumeUp raises software volume by one level and stops at high.
func (p *Player) VolumeUp() { p.internal().VolumeUp() }

// VolumeDown lowers software volume by one level and stops at mute.
func (p *Player) VolumeDown() { p.internal().VolumeDown() }

// Mute stores the current non-mute level and selects mute.
func (p *Player) Mute() { p.internal().Mute() }

// Unmute restores the most recently selected non-mute level.
func (p *Player) Unmute() { p.internal().Unmute() }

// ToggleMute switches between mute and the previous non-mute level.
func (p *Player) ToggleMute() { p.internal().ToggleMute() }

// Muted reports whether software volume is mute.
func (p *Player) Muted() bool { return p.internal().Muted() }

// Next returns the next volume level, wrapping from high to mute.
func (level VolumeLevel) Next() VolumeLevel {
	return VolumeLevel(internalaudio.VolumeLevel(level).Next())
}

// String returns the short display name of level.
func (level VolumeLevel) String() string {
	return internalaudio.VolumeLevel(level).String()
}
