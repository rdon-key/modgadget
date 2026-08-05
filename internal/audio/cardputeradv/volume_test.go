package cardputeradv

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/rdon-key/modgadget"
)

var _ modgadget.VolumeController = (*Player)(nil)

func TestVolumeDefaultsSetNextAndString(t *testing.T) {
	player := New()
	if player.Volume() != VolumeMedium {
		t.Fatalf("New volume=%v want MEDIUM", player.Volume())
	}
	levels := []VolumeLevel{VolumeMute, VolumeLow, VolumeMedium, VolumeHigh}
	for _, level := range levels {
		if err := player.SetVolume(level); err != nil {
			t.Fatalf("SetVolume(%v): %v", level, err)
		}
		if player.Volume() != level {
			t.Fatalf("volume=%v want=%v", player.Volume(), level)
		}
	}
	if !errors.Is(player.SetVolume(VolumeLevel(4)), ErrInvalidVolume) {
		t.Fatal("invalid volume was accepted")
	}
	for index, level := range levels {
		want := levels[(index+1)%len(levels)]
		if got := level.Next(); got != want {
			t.Errorf("%s.Next()=%s want=%s", level, got, want)
		}
	}
	wantNames := []string{"MUTE", "LOW", "MEDIUM", "HIGH"}
	for index, want := range wantNames {
		if got := VolumeLevel(index).String(); got != want {
			t.Errorf("VolumeLevel(%d).String()=%q want=%q", index, got, want)
		}
	}
}

func TestVolumeNilReceiver(t *testing.T) {
	var player *Player
	if !errors.Is(player.SetVolume(VolumeHigh), ErrNotConfigured) {
		t.Fatal("nil SetVolume did not follow existing ErrNotConfigured policy")
	}
	if player.Volume() != VolumeMute {
		t.Fatalf("nil Volume=%v want MUTE zero value", player.Volume())
	}
}

func TestInvalidVolumeDoesNotChangeCurrentLevel(t *testing.T) {
	player := newPlayer(&fakePCMDevice{})
	if err := player.SetVolume(VolumeHigh); err != nil {
		t.Fatal(err)
	}
	if err := player.SetVolume(VolumeLevel(255)); !errors.Is(err, ErrInvalidVolume) {
		t.Fatalf("invalid volume error=%v", err)
	}
	if player.Volume() != VolumeHigh {
		t.Fatalf("invalid SetVolume changed level to %v", player.Volume())
	}
}

func TestVolumeUpDownClamp(t *testing.T) {
	player := newPlayer(&fakePCMDevice{})
	_ = player.SetVolume(VolumeMute)
	for _, want := range []VolumeLevel{VolumeLow, VolumeMedium, VolumeHigh, VolumeHigh} {
		player.VolumeUp()
		if player.Volume() != want {
			t.Fatalf("VolumeUp=%v want=%v", player.Volume(), want)
		}
	}
	for _, want := range []VolumeLevel{VolumeMedium, VolumeLow, VolumeMute, VolumeMute} {
		player.VolumeDown()
		if player.Volume() != want {
			t.Fatalf("VolumeDown=%v want=%v", player.Volume(), want)
		}
	}
}

func TestMuteUnmuteRestore(t *testing.T) {
	for _, level := range []VolumeLevel{VolumeLow, VolumeMedium, VolumeHigh} {
		player := newPlayer(&fakePCMDevice{})
		_ = player.SetVolume(level)
		player.Mute()
		player.Mute()
		if !player.Muted() {
			t.Fatalf("%v did not mute", level)
		}
		player.Unmute()
		player.Unmute()
		if player.Muted() || player.Volume() != level {
			t.Fatalf("%v restored as %v", level, player.Volume())
		}
	}

	player := &Player{volume: VolumeMute}
	player.Unmute()
	if player.Volume() != VolumeMedium {
		t.Fatalf("missing restore level returned %v", player.Volume())
	}
}

func TestToggleMuteAndSetVolumeRestore(t *testing.T) {
	player := newPlayer(&fakePCMDevice{})
	for _, level := range []VolumeLevel{VolumeHigh, VolumeLow} {
		_ = player.SetVolume(level)
		player.ToggleMute()
		if !player.Muted() {
			t.Fatalf("ToggleMute from %v did not mute", level)
		}
		player.ToggleMute()
		if player.Volume() != level {
			t.Fatalf("ToggleMute restored %v want=%v", player.Volume(), level)
		}
	}

	_ = player.SetVolume(VolumeHigh)
	_ = player.SetVolume(VolumeMute)
	player.Unmute()
	if player.Volume() != VolumeHigh {
		t.Fatalf("SetVolume(MUTE) restored %v want HIGH", player.Volume())
	}
	_ = player.SetVolume(VolumeLow)
	beforeVolume, beforePrevious, beforeHas := player.volume, player.previousVolume, player.hasPreviousVolume
	if err := player.SetVolume(VolumeLevel(255)); !errors.Is(err, ErrInvalidVolume) {
		t.Fatalf("invalid volume error=%v", err)
	}
	if player.volume != beforeVolume || player.previousVolume != beforePrevious || player.hasPreviousVolume != beforeHas {
		t.Fatal("invalid volume changed current or restore state")
	}
}

func TestMuteRestoreSurvivesConfigureAndStop(t *testing.T) {
	player := newPlayer(&fakePCMDevice{})
	_ = player.SetVolume(VolumeHigh)
	player.Mute()
	if err := player.Configure(); err != nil {
		t.Fatal(err)
	}
	player.Unmute()
	if player.Volume() != VolumeHigh {
		t.Fatalf("Configure lost restore level: %v", player.Volume())
	}
	player.Mute()
	if err := player.Stop(); err != nil {
		t.Fatal(err)
	}
	player.Unmute()
	if player.Volume() != VolumeHigh {
		t.Fatalf("Stop lost restore level: %v", player.Volume())
	}
}

func TestVolumeOperationsDoNotAffectPlaybackOrWritePCM(t *testing.T) {
	device := &fakePCMDevice{}
	player := newPlayer(device)
	_ = player.Configure()
	_ = player.PlayPattern(PatternStartup)
	pattern, index, remaining, frame, writes := player.pattern, player.stepIndex, player.stepRemaining, player.tone.frame, device.writes
	player.VolumeUp()
	player.VolumeDown()
	player.Mute()
	player.Unmute()
	player.ToggleMute()
	player.ToggleMute()
	if player.pattern != pattern || player.stepIndex != index || player.stepRemaining != remaining || player.tone.frame != frame || device.writes != writes {
		t.Fatal("volume operation changed pattern state or wrote PCM")
	}

	_ = player.PlayTone(880, 3*time.Millisecond)
	frame, writes = player.tone.frame, device.writes
	player.VolumeUp()
	player.Mute()
	player.Unmute()
	if player.tone.frame != frame || device.writes != writes {
		t.Fatal("volume operation changed tone state or wrote PCM")
	}
	player.Mute()
	_ = player.Update()
	if player.tone.frame != FramesPerBuffer || !allZero(device.last[:]) {
		t.Fatal("muted playback did not advance silently")
	}
}

func TestVolumeOperationsAllocations(t *testing.T) {
	player := newPlayer(&fakePCMDevice{})
	if allocs := testing.AllocsPerRun(1000, func() {
		player.VolumeUp()
		player.VolumeDown()
		player.ToggleMute()
		player.ToggleMute()
	}); allocs != 0 {
		t.Fatalf("volume operation allocations=%v", allocs)
	}
}

func TestVolumeSurvivesPlaybackLifecycle(t *testing.T) {
	device := &fakePCMDevice{}
	player := newPlayer(device)
	if err := player.SetVolume(VolumeLow); err != nil {
		t.Fatal(err)
	}
	if device.writes != 0 {
		t.Fatal("SetVolume submitted PCM")
	}
	if err := player.Configure(); err != nil {
		t.Fatal(err)
	}
	assertVolume(t, player, VolumeLow, "Configure")
	_ = player.PlayTone(880, time.Millisecond)
	assertVolume(t, player, VolumeLow, "PlayTone")
	_ = player.PlayPattern(PatternClick)
	assertVolume(t, player, VolumeLow, "PlayPattern")
	_ = player.Stop()
	assertVolume(t, player, VolumeLow, "Stop")

	_ = player.PlayTone(880, time.Millisecond)
	for player.Busy() {
		if err := player.Update(); err != nil {
			t.Fatal(err)
		}
	}
	assertVolume(t, player, VolumeLow, "completion")
}

func TestApplyVolumePCM(t *testing.T) {
	original := samplesPCM(12000, -8000, 32760, -32768)
	tests := []struct {
		level VolumeLevel
		want  []int16
	}{
		{VolumeMute, []int16{0, 0, 0, 0}},
		{VolumeLow, []int16{3000, -2000, 8190, -8192}},
		{VolumeMedium, []int16{6000, -4000, 16380, -16384}},
		{VolumeHigh, []int16{12000, -8000, 32760, -32768}},
	}
	for _, test := range tests {
		data := append([]byte(nil), original...)
		applyVolumePCM(data, test.level)
		for index, want := range test.want {
			if got := decodeSample(data[index*2:]); got != want {
				t.Errorf("%s sample[%d]=%d want=%d", test.level, index, got, want)
			}
		}
		if test.level == VolumeHigh && !bytes.Equal(data, original) {
			t.Fatal("HIGH modified original PCM")
		}
	}
}

func TestGeneratedToneVolumePeaks(t *testing.T) {
	peaks := [4]int16{}
	for level := VolumeMute; level <= VolumeHigh; level++ {
		var generator tone
		generator.start(880, 80*SampleRate/1000)
		var buffer [BufferBytes]byte
		for generator.active() {
			written := generator.fill(buffer[:])
			applyVolumePCM(buffer[:], level)
			for frame := 0; frame < written; frame++ {
				for channel := 0; channel < Channels; channel++ {
					value := abs16(decodeSample(buffer[frame*BytesPerFrame+channel*2:]))
					if value > peaks[level] {
						peaks[level] = value
					}
				}
			}
		}
	}
	if peaks[VolumeMute] != 0 {
		t.Fatalf("MUTE peak=%d", peaks[VolumeMute])
	}
	if peaks[VolumeHigh] == 0 || peaks[VolumeHigh] > 2048 {
		t.Fatalf("HIGH peak=%d want=1..2048", peaks[VolumeHigh])
	}
	if difference(peaks[VolumeMedium], peaks[VolumeHigh]/2) > 1 {
		t.Fatalf("MEDIUM peak=%d HIGH=%d", peaks[VolumeMedium], peaks[VolumeHigh])
	}
	if difference(peaks[VolumeLow], peaks[VolumeHigh]/4) > 1 {
		t.Fatalf("LOW peak=%d HIGH=%d", peaks[VolumeLow], peaks[VolumeHigh])
	}
}

func TestMuteAdvancesAndVolumeChangeAffectsNextWrite(t *testing.T) {
	device := &fakePCMDevice{}
	player := newPlayer(device)
	_ = player.Configure()
	_ = player.SetVolume(VolumeMute)
	_ = player.PlayTone(880, 3*time.Millisecond)
	_ = player.Update()
	if player.tone.frame != FramesPerBuffer || !allZero(device.last[:]) {
		t.Fatalf("muted tone did not advance silently: frame=%d", player.tone.frame)
	}
	frame := player.tone.frame
	writes := device.writes
	_ = player.SetVolume(VolumeHigh)
	if player.tone.frame != frame || device.writes != writes {
		t.Fatal("SetVolume restarted playback or submitted PCM")
	}
	_ = player.Update()
	if player.tone.frame != frame+FramesPerBuffer || allZero(device.last[:]) {
		t.Fatal("next tone chunk did not use HIGH volume")
	}

	_ = player.SetVolume(VolumeMute)
	_ = player.PlayPattern(PatternStartup)
	before := player.stepRemaining
	_ = player.Update()
	if player.stepRemaining != before-FramesPerBuffer || !allZero(device.last[:]) {
		t.Fatal("muted pattern did not advance silently")
	}
}

func TestPatternAndReleaseSilenceRemainZero(t *testing.T) {
	device := &fakePCMDevice{}
	player := newPlayer(device)
	_ = player.Configure()
	_ = player.SetVolume(VolumeHigh)
	player.pattern = PatternStartup
	player.stepIndex = 1
	player.loadPatternStep()
	for index := range player.buffer {
		player.buffer[index] = 0xff
	}
	_ = player.Update()
	if !allZero(device.last[:]) {
		t.Fatal("pattern silence contains nonzero PCM")
	}

	player.pattern = 0
	player.flushPending = true
	player.releaseFrames = FramesPerBuffer
	for index := range player.buffer {
		player.buffer[index] = 0xff
	}
	_ = player.Update()
	if !allZero(device.last[:]) {
		t.Fatal("final silence contains nonzero PCM")
	}
}

func TestVolumeProcessingAllocations(t *testing.T) {
	data := samplesPCM(12000, -8000)
	allocs := testing.AllocsPerRun(1000, func() {
		applyVolumePCM(data, VolumeMedium)
	})
	if allocs != 0 {
		t.Fatalf("volume allocations=%v", allocs)
	}
}

func assertVolume(t *testing.T, player *Player, want VolumeLevel, operation string) {
	t.Helper()
	if got := player.Volume(); got != want {
		t.Fatalf("volume after %s=%v want=%v", operation, got, want)
	}
}

func samplesPCM(values ...int16) []byte {
	data := make([]byte, len(values)*2)
	for index, value := range values {
		data[index*2] = byte(value)
		data[index*2+1] = byte(uint16(value) >> 8)
	}
	return data
}

func allZero(data []byte) bool {
	for _, value := range data {
		if value != 0 {
			return false
		}
	}
	return true
}

func difference(left, right int16) int16 {
	if left < right {
		return right - left
	}
	return left - right
}
