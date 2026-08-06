// Package cardputeradv provides cooperative audio playback for M5Stack
// Cardputer ADV devices.
package cardputeradv

import (
	"time"

	internalaudio "github.com/rdon-key/modgadget/internal/audio/cardputeradv"
)

// Player cooperatively generates and submits audio for a Cardputer ADV.
// Its fields and hardware implementation are private. The zero value and a nil
// receiver are safe but unconfigured: methods returning an error return
// ErrNotConfigured, Busy reports false, Volume reports VolumeMute, Muted reports
// true, and volume controls without an error result are no-ops.
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
	ErrNotConfigured = internalaudio.ErrNotConfigured
	// ErrInvalidFrequency reports an unsupported tone frequency.
	ErrInvalidFrequency = internalaudio.ErrInvalidFrequency
	// ErrInvalidDuration reports a non-positive tone duration.
	ErrInvalidDuration = internalaudio.ErrInvalidDuration
	// ErrInvalidPattern reports an unknown built-in Pattern.
	ErrInvalidPattern = internalaudio.ErrInvalidPattern
	// ErrInvalidVolume reports a VolumeLevel outside the supported range.
	ErrInvalidVolume = internalaudio.ErrInvalidVolume
)

// Configure initializes the Cardputer ADV codec and I2S transmitter and
// returns a ready Player. It returns nil and an error if initialization fails.
func Configure() (*Player, error) {
	impl := internalaudio.New()
	if err := impl.Configure(); err != nil {
		return nil, err
	}
	return &Player{impl: impl}, nil
}

func (p *Player) internal() *internalaudio.Player {
	if p == nil {
		return nil
	}
	return p.impl
}

// PlayTone replaces current playback with a tone and returns without sending PCM.
func (p *Player) PlayTone(frequencyHz uint16, duration time.Duration) error {
	if p.internal() == nil {
		return ErrNotConfigured
	}
	return p.impl.PlayTone(frequencyHz, duration)
}

// PlayPattern replaces current playback with a built-in fixed pattern.
func (p *Player) PlayPattern(pattern Pattern) error {
	if p.internal() == nil {
		return ErrNotConfigured
	}
	return p.impl.PlayPattern(internalaudio.Pattern(pattern))
}

// Update performs at most one non-blocking audio submission.
func (p *Player) Update() error {
	if p.internal() == nil {
		return ErrNotConfigured
	}
	return p.impl.Update()
}

// Busy reports whether audio or its final silence buffer is pending. It reports
// false for a nil receiver or zero Player.
func (p *Player) Busy() bool {
	return p.internal() != nil && p.impl.Busy()
}

// Stop cancels playback and submits a finite silence chunk.
func (p *Player) Stop() error {
	if p.internal() == nil {
		return ErrNotConfigured
	}
	return p.impl.Stop()
}

// SetVolume changes the software gain used by subsequent audio submissions.
func (p *Player) SetVolume(level VolumeLevel) error {
	if p.internal() == nil {
		return ErrNotConfigured
	}
	return p.impl.SetVolume(internalaudio.VolumeLevel(level))
}

// Volume returns the current software gain level. It returns VolumeMute for a
// nil receiver or zero Player.
func (p *Player) Volume() VolumeLevel {
	if p.internal() == nil {
		return VolumeMute
	}
	return VolumeLevel(p.impl.Volume())
}

// VolumeUp raises software volume by one level and stops at high. It is a no-op
// for a nil receiver or zero Player.
func (p *Player) VolumeUp() {
	if p.internal() != nil {
		p.impl.VolumeUp()
	}
}

// VolumeDown lowers software volume by one level and stops at mute. It is a
// no-op for a nil receiver or zero Player.
func (p *Player) VolumeDown() {
	if p.internal() != nil {
		p.impl.VolumeDown()
	}
}

// Mute stores the current non-mute level and selects mute. It is a no-op for a
// nil receiver or zero Player.
func (p *Player) Mute() {
	if p.internal() != nil {
		p.impl.Mute()
	}
}

// Unmute restores the most recently selected non-mute level. It is a no-op for
// a nil receiver or zero Player.
func (p *Player) Unmute() {
	if p.internal() != nil {
		p.impl.Unmute()
	}
}

// ToggleMute switches between mute and the previous non-mute level. It is a
// no-op for a nil receiver or zero Player.
func (p *Player) ToggleMute() {
	if p.internal() != nil {
		p.impl.ToggleMute()
	}
}

// Muted reports whether software volume is mute. It reports true for a nil
// receiver or zero Player.
func (p *Player) Muted() bool {
	return p.internal() == nil || p.impl.Muted()
}

// String returns the short display name of level.
func (level VolumeLevel) String() string {
	return internalaudio.VolumeLevel(level).String()
}
