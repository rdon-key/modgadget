package cardputeradv

import (
	"errors"
	"testing"
	"time"
)

type fakePCMDevice struct {
	configured bool
	ready      bool
	writes     int
	stopCalls  int
	last       [BufferBytes]byte
	err        error
}

type finitePCMDevice struct {
	ready          bool
	writes         int
	writesNotReady int
	last           [BufferBytes]byte
}

func (d *finitePCMDevice) Configure() error {
	// Configure has submitted one finite startup-silence descriptor.
	d.ready = false
	d.writes = 0
	d.writesNotReady = 0
	return nil
}
func (d *finitePCMDevice) Ready() bool { return d.ready }
func (d *finitePCMDevice) WritePCM(data []byte) error {
	if !d.ready {
		d.writesNotReady++
		return errors.New("DMA buffer reused before EOF")
	}
	copy(d.last[:], data)
	d.writes++
	d.ready = false
	return nil
}
func (d *finitePCMDevice) Stop() error { d.ready = false; return nil }
func (d *finitePCMDevice) complete()   { d.ready = true }

func (d *fakePCMDevice) Configure() error { d.configured = true; d.ready = true; return d.err }
func (d *fakePCMDevice) Ready() bool      { return d.ready }
func (d *fakePCMDevice) WritePCM(data []byte) error {
	if d.err != nil {
		return d.err
	}
	copy(d.last[:], data)
	d.writes++
	return nil
}
func (d *fakePCMDevice) Stop() error { d.stopCalls++; d.ready = true; return d.err }

func TestPlayerCooperativePlayback(t *testing.T) {
	device := &fakePCMDevice{}
	player := newPlayer(device)
	if err := player.Configure(); err != nil {
		t.Fatal(err)
	}
	if err := player.PlayTone(880, 3*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !player.Busy() {
		t.Fatal("player is not busy after PlayTone")
	}
	if device.writes != 0 {
		t.Fatalf("PlayTone wrote %d buffers", device.writes)
	}
	updates := 0
	for player.Busy() && updates < 20 {
		if err := player.Update(); err != nil {
			t.Fatal(err)
		}
		updates++
	}
	if player.Busy() {
		t.Fatal("playback did not finish")
	}
	if device.writes != 4 { // three tone chunks and one silence flush
		t.Fatalf("writes = %d, want 4", device.writes)
	}
	for _, value := range device.last {
		if value != 0 {
			t.Fatal("final buffer is not silent")
		}
	}
}

func TestFiniteStartupSilenceCannotBlockToneForever(t *testing.T) {
	device := &finitePCMDevice{}
	player := newPlayer(device)
	if err := player.Configure(); err != nil {
		t.Fatal(err)
	}
	if err := player.PlayTone(880, 3*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Startup silence is still in flight: no generation and no buffer reuse.
	for i := 0; i < 10; i++ {
		if err := player.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if device.writes != 0 || player.tone.frame != 0 {
		t.Fatalf("startup DMA advanced tone: writes=%d frame=%d", device.writes, player.tone.frame)
	}

	device.complete()
	if err := player.Update(); err != nil {
		t.Fatal(err)
	}
	if device.writes != 1 || player.tone.frame != FramesPerBuffer {
		t.Fatalf("first EOF did not queue tone: writes=%d frame=%d", device.writes, player.tone.frame)
	}
	// Repeated Update before EOF must neither reuse nor advance the buffer.
	frame := player.tone.frame
	for i := 0; i < 10; i++ {
		if err := player.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if device.writes != 1 || player.tone.frame != frame || device.writesNotReady != 0 {
		t.Fatalf("in-flight buffer reused: writes=%d frame=%d rejected=%d",
			device.writes, player.tone.frame, device.writesNotReady)
	}

	for player.Busy() {
		device.complete()
		before := device.writes
		if err := player.Update(); err != nil {
			t.Fatal(err)
		}
		if device.writes-before > 1 {
			t.Fatalf("one Update queued %d chunks", device.writes-before)
		}
	}
	if device.writes != 4 { // three tone chunks and one finite release silence
		t.Fatalf("writes=%d, want 4", device.writes)
	}
	for _, value := range device.last {
		if value != 0 {
			t.Fatal("last finite descriptor was not silence")
		}
	}
}

func TestFiniteSilenceAllowsPlaybackAfterReconfigure(t *testing.T) {
	device := &finitePCMDevice{}
	player := newPlayer(device)
	for attempt := 0; attempt < 2; attempt++ {
		if err := player.Configure(); err != nil {
			t.Fatal(err)
		}
		if err := player.PlayTone(880, time.Millisecond); err != nil {
			t.Fatal(err)
		}
		device.complete()
		if err := player.Update(); err != nil {
			t.Fatal(err)
		}
		if player.tone.frame == 0 {
			t.Fatalf("attempt %d did not advance", attempt)
		}
	}
}

func TestStopWaitsForFiniteSilenceCompletion(t *testing.T) {
	device := &finitePCMDevice{}
	player := newPlayer(device)
	_ = player.Configure()
	device.complete()
	_ = player.PlayTone(880, time.Second)
	_ = player.Update()
	if err := player.Stop(); err != nil {
		t.Fatal(err)
	}
	if !player.Busy() {
		t.Fatal("Stop did not retain Busy during finite silence")
	}
	_ = player.Update()
	if !player.Busy() {
		t.Fatal("Stop completed before silence EOF")
	}
	device.complete()
	_ = player.Update()
	if player.Busy() {
		t.Fatal("Stop remained busy after silence EOF")
	}
}

func TestPlayerUpdateIsBoundedByDeviceReadiness(t *testing.T) {
	device := &fakePCMDevice{}
	player := newPlayer(device)
	_ = player.Configure()
	_ = player.PlayTone(880, 80*time.Millisecond)
	device.ready = false
	for i := 0; i < 1000; i++ {
		if err := player.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if device.writes != 0 || player.tone.frame != 0 {
		t.Fatalf("blocked device advanced playback: writes=%d frame=%d", device.writes, player.tone.frame)
	}
	device.ready = true
	_ = player.Update()
	if device.writes != 1 || player.tone.frame > FramesPerBuffer {
		t.Fatalf("one update wrote=%d frame=%d", device.writes, player.tone.frame)
	}
}

func TestPlayerStopAndReplayReplacement(t *testing.T) {
	device := &fakePCMDevice{}
	player := newPlayer(device)
	_ = player.Configure()
	_ = player.PlayTone(440, time.Second)
	_ = player.Update()
	if err := player.PlayTone(880, 2*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	wantStep := uint32((uint64(880) << 32) / SampleRate)
	if player.tone.phaseStep != wantStep || player.tone.frame != 0 {
		t.Fatal("replay did not replace current generator state")
	}
	if err := player.Stop(); err != nil {
		t.Fatal(err)
	}
	if device.stopCalls != 1 || !player.Busy() {
		t.Fatalf("stop calls=%d busy=%v", device.stopCalls, player.Busy())
	}
	_ = player.Update()
	if player.Busy() {
		t.Fatal("Stop silence did not drain")
	}
	for _, value := range player.buffer {
		if value != 0 {
			t.Fatal("Stop did not end with silence")
		}
	}
}

func TestConfigureResetsStateAndErrorsPropagate(t *testing.T) {
	device := &fakePCMDevice{}
	player := newPlayer(device)
	if !errors.Is(player.PlayTone(880, time.Millisecond), ErrNotConfigured) {
		t.Fatal("PlayTone before Configure did not fail")
	}
	_ = player.Configure()
	_ = player.PlayTone(880, time.Second)
	if err := player.Configure(); err != nil {
		t.Fatal(err)
	}
	if player.Busy() {
		t.Fatal("Configure did not reset playback")
	}
	device.err = errors.New("output failed")
	if err := player.PlayTone(880, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := player.Update(); !errors.Is(err, device.err) {
		t.Fatalf("Update error = %v", err)
	}
	if player.Busy() || device.stopCalls != 1 {
		t.Fatalf("output error did not stop safely: busy=%v stops=%d", player.Busy(), device.stopCalls)
	}
}

func TestPlayerRejectsInvalidTone(t *testing.T) {
	player := newPlayer(&fakePCMDevice{})
	_ = player.Configure()
	for _, frequency := range []uint16{0, SampleRate / 2, SampleRate/2 + 1} {
		if !errors.Is(player.PlayTone(frequency, time.Millisecond), ErrInvalidFrequency) {
			t.Fatalf("frequency %d accepted", frequency)
		}
	}
	if !errors.Is(player.PlayTone(880, 0), ErrInvalidDuration) {
		t.Fatal("zero duration accepted")
	}
}

func TestPlayerUpdateAllocations(t *testing.T) {
	device := &fakePCMDevice{}
	player := newPlayer(device)
	_ = player.Configure()
	allocs := testing.AllocsPerRun(1000, func() {
		_ = player.PlayTone(880, time.Millisecond)
		_ = player.Update()
	})
	if allocs != 0 {
		t.Fatalf("active Update allocations = %v", allocs)
	}
	for player.Busy() {
		_ = player.Update()
	}
	allocs = testing.AllocsPerRun(1000, func() { _ = player.Update() })
	if allocs != 0 {
		t.Fatalf("idle Update allocations = %v", allocs)
	}
}
