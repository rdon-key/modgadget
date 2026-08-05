// Package cardputeradv contains an experimental Cardputer ADV audio player.
// It is internal and is not part of ModGadget's public API.
package cardputeradv

import (
	"errors"
	"fmt"
	"time"
)

const (
	SampleRate      = 48_000
	Channels        = 2
	BitsPerSample   = 16
	FramesPerBuffer = 48
	BytesPerFrame   = Channels * BitsPerSample / 8
	BufferBytes     = FramesPerBuffer * BytesPerFrame
)

var (
	ErrNotConfigured    = errors.New("cardputeradv audio: player is not configured")
	ErrInvalidFrequency = errors.New("cardputeradv audio: invalid frequency")
	ErrInvalidDuration  = errors.New("cardputeradv audio: invalid duration")
	ErrInvalidPattern   = errors.New("cardputeradv audio: invalid pattern")
)

type pcmDevice interface {
	Configure() error
	Ready() bool
	WritePCM([]byte) error
	Stop() error
}

// Player cooperatively generates and submits tone PCM.
type Player struct {
	device            pcmDevice
	buffer            [BufferBytes]byte
	tone              tone
	configured        bool
	flushPending      bool
	releaseFrames     uint32
	draining          bool
	pattern           Pattern
	stepIndex         uint8
	stepRemaining     uint32
	volume            VolumeLevel
	previousVolume    VolumeLevel
	hasPreviousVolume bool
}

// New returns a Player connected to the Cardputer ADV audio hardware.
func New() *Player { return newPlayer(newPCMDevice()) }

func newPlayer(device pcmDevice) *Player {
	return &Player{device: device, volume: VolumeMedium}
}

// SetVolume changes the software gain used by subsequent PCM submissions.
func (p *Player) SetVolume(level VolumeLevel) error {
	if p == nil {
		return ErrNotConfigured
	}
	if level > VolumeHigh {
		return ErrInvalidVolume
	}
	if level == VolumeMute {
		if p.volume != VolumeMute {
			p.previousVolume = p.volume
			p.hasPreviousVolume = true
		}
	} else {
		p.previousVolume = level
		p.hasPreviousVolume = true
	}
	p.volume = level
	return nil
}

// Volume returns the current software gain level.
func (p *Player) Volume() VolumeLevel {
	if p == nil {
		return VolumeMute
	}
	return p.volume
}

// VolumeUp raises software volume by one level and stops at HIGH.
func (p *Player) VolumeUp() {
	if p == nil || p.volume == VolumeHigh {
		return
	}
	_ = p.SetVolume(p.volume + 1)
}

// VolumeDown lowers software volume by one level and stops at MUTE.
func (p *Player) VolumeDown() {
	if p == nil || p.volume == VolumeMute {
		return
	}
	_ = p.SetVolume(p.volume - 1)
}

// Mute stores the current non-mute level and selects MUTE.
func (p *Player) Mute() {
	if p == nil || p.volume == VolumeMute {
		return
	}
	_ = p.SetVolume(VolumeMute)
}

// Unmute restores the most recently selected non-mute level, or MEDIUM when
// no such level has been selected.
func (p *Player) Unmute() {
	if p == nil || p.volume != VolumeMute {
		return
	}
	level := VolumeMedium
	if p.hasPreviousVolume && p.previousVolume > VolumeMute && p.previousVolume <= VolumeHigh {
		level = p.previousVolume
	}
	_ = p.SetVolume(level)
}

// ToggleMute switches between MUTE and the previous non-mute level.
func (p *Player) ToggleMute() {
	if p == nil {
		return
	}
	if p.Muted() {
		p.Unmute()
	} else {
		p.Mute()
	}
}

// Muted reports whether software volume is MUTE.
func (p *Player) Muted() bool { return p != nil && p.volume == VolumeMute }

// Configure initializes the codec and I2S transmitter and resets playback.
func (p *Player) Configure() error {
	if p == nil || p.device == nil {
		return fmt.Errorf("cardputeradv audio: device is nil")
	}
	p.reset()
	if err := p.device.Configure(); err != nil {
		return fmt.Errorf("cardputeradv audio: configure: %w", err)
	}
	p.configured = true
	return nil
}

// PlayTone replaces any current playback and returns without sending PCM.
func (p *Player) PlayTone(frequencyHz uint16, duration time.Duration) error {
	if p == nil || !p.configured {
		return ErrNotConfigured
	}
	if frequencyHz == 0 || uint32(frequencyHz) >= SampleRate/2 {
		return ErrInvalidFrequency
	}
	if duration <= 0 {
		return ErrInvalidDuration
	}

	frames := duration.Nanoseconds() * SampleRate / int64(time.Second)
	if frames <= 0 {
		frames = 1
	}

	p.tone.start(uint32(frequencyHz), uint32(frames))
	p.pattern = 0
	p.stepIndex = 0
	p.stepRemaining = 0
	p.flushPending = false
	p.releaseFrames = 0
	p.draining = false
	return nil
}

// PlayPattern replaces current playback with a built-in fixed pattern and
// returns without sending PCM.
func (p *Player) PlayPattern(pattern Pattern) error {
	if p == nil || !p.configured {
		return ErrNotConfigured
	}
	if len(patternSteps(pattern)) == 0 {
		return ErrInvalidPattern
	}

	p.pattern = pattern
	p.stepIndex = 0
	p.stepRemaining = 0
	p.flushPending = false
	p.releaseFrames = 0
	p.draining = false
	p.loadPatternStep()
	return nil
}

// Update performs at most one non-blocking 48-frame DMA submission.
func (p *Player) Update() error {
	if p == nil || !p.configured {
		return ErrNotConfigured
	}
	if !p.device.Ready() {
		return nil
	}

	switch {
	case p.pattern != 0:
		steps := patternSteps(p.pattern)
		if int(p.stepIndex) >= len(steps) {
			p.stopAfterError()
			return ErrInvalidPattern
		}

		written := p.fillPatternChunk(steps[p.stepIndex])
		if err := p.writeBuffer(); err != nil {
			p.stopAfterError()
			return fmt.Errorf("cardputeradv audio: write pattern: %w", err)
		}

		p.stepRemaining -= uint32(written)
		if p.stepRemaining == 0 {
			p.stepIndex++
			if int(p.stepIndex) == len(steps) {
				p.pattern = 0
				p.tone.reset()
				p.flushPending = true
				p.releaseFrames = FramesPerBuffer
			} else {
				p.loadPatternStep()
			}
		}

	case p.tone.active():
		written := p.tone.fill(p.buffer[:])
		if err := p.writeBuffer(); err != nil {
			p.stopAfterError()
			return fmt.Errorf("cardputeradv audio: write tone: %w", err)
		}

		if p.tone.frame == uint32(written) || !p.tone.active() {
			audioDebugToneQueued(written, p.tone.totalFrames-p.tone.frame)
		}
		if !p.tone.active() {
			p.flushPending = true
			p.releaseFrames = FramesPerBuffer
		}

	case p.flushPending:
		if p.releaseFrames == FramesPerBuffer {
			audioDebugText("audio: release silence started")
		}

		p.silenceBuffer()
		if err := p.writeBuffer(); err != nil {
			p.stopAfterError()
			return fmt.Errorf("cardputeradv audio: flush silence: %w", err)
		}

		if p.releaseFrames > FramesPerBuffer {
			p.releaseFrames -= FramesPerBuffer
		} else {
			p.releaseFrames = 0
			p.flushPending = false
			p.draining = true
		}

	case p.draining:
		audioDebugText("audio: silence complete")
		p.draining = false
	}

	return nil
}

// Busy reports whether tone PCM or its final silence buffer is pending.
func (p *Player) Busy() bool {
	return p != nil &&
		(p.pattern != 0 ||
			p.tone.active() ||
			p.flushPending ||
			p.draining)
}

// Stop cancels generation and submits one finite silence DMA chunk.
func (p *Player) Stop() error {
	if p == nil || !p.configured {
		return ErrNotConfigured
	}

	p.tone.reset()
	p.pattern = 0
	p.stepIndex = 0
	p.stepRemaining = 0
	p.flushPending = false
	p.releaseFrames = 0
	p.draining = true
	p.silenceBuffer()

	if err := p.device.Stop(); err != nil {
		return fmt.Errorf("cardputeradv audio: stop: %w", err)
	}
	return nil
}

func (p *Player) stopAfterError() {
	p.tone.reset()
	p.pattern = 0
	p.stepIndex = 0
	p.stepRemaining = 0
	p.flushPending = false
	p.releaseFrames = 0
	p.draining = false
	p.silenceBuffer()
	_ = p.device.Stop()
}

func (p *Player) reset() {
	p.tone.reset()
	p.pattern = 0
	p.stepIndex = 0
	p.stepRemaining = 0
	p.flushPending = false
	p.releaseFrames = 0
	p.draining = false
	p.configured = false
	p.silenceBuffer()
}

func (p *Player) loadPatternStep() {
	step := patternSteps(p.pattern)[p.stepIndex]
	p.stepRemaining = step.frames

	if step.kind == stepTone {
		p.tone.start(uint32(step.frequency), step.frames)
	} else {
		p.tone.reset()
	}
}

func (p *Player) fillPatternChunk(step patternStep) int {
	frames := FramesPerBuffer
	if p.stepRemaining < FramesPerBuffer {
		frames = int(p.stepRemaining)
	}

	if step.kind == stepTone {
		return p.tone.fill(p.buffer[:])
	}

	p.silenceBuffer()
	return frames
}

func (p *Player) silenceBuffer() {
	for i := range p.buffer {
		p.buffer[i] = 0
	}
}

func (p *Player) writeBuffer() error {
	applyVolumePCM(p.buffer[:], p.volume)
	return p.device.WritePCM(p.buffer[:])
}
