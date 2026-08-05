package cardputeradv

import (
	"math"
	"testing"
)

func TestToneDurationStereoAndBounds(t *testing.T) {
	var generator tone
	generator.start(880, 80*SampleRate/1000)
	var buffer [BufferBytes]byte
	frames := 0
	peak := int16(0)
	for generator.active() {
		written := generator.fill(buffer[:])
		frames += written
		for i := 0; i < written; i++ {
			left := decodeSample(buffer[i*BytesPerFrame:])
			right := decodeSample(buffer[i*BytesPerFrame+2:])
			if left != right {
				t.Fatalf("frame %d channels differ: %d != %d", frames-written+i, left, right)
			}
			if abs16(left) > peak {
				peak = abs16(left)
			}
		}
	}
	if frames != 3840 {
		t.Fatalf("frames = %d, want 3840", frames)
	}
	if peak == 0 || peak > int16(maxAmplitude) {
		t.Fatalf("peak = %d, want 1..%d", peak, maxAmplitude)
	}
	if peak == math.MaxInt16 {
		t.Fatal("tone reached full scale")
	}
}

func TestTonePhaseAndEnvelope(t *testing.T) {
	var generator tone
	generator.start(880, 80*SampleRate/1000)
	wantStep := uint32((uint64(880) << 32) / SampleRate)
	if generator.phaseStep != wantStep {
		t.Fatalf("phase step = %d, want %d", generator.phaseStep, wantStep)
	}
	var buffer [BufferBytes]byte
	levels := make([]int64, 0, 80)
	for generator.active() {
		written := generator.fill(buffer[:])
		var sum int64
		for i := 0; i < written; i++ {
			sum += int64(abs16(decodeSample(buffer[i*BytesPerFrame:])))
		}
		levels = append(levels, sum)
	}
	if levels[0] >= levels[4] {
		t.Fatalf("attack did not rise: first=%d later=%d", levels[0], levels[4])
	}
	if levels[len(levels)-1] >= levels[len(levels)-5] {
		t.Fatalf("release did not fall: earlier=%d last=%d", levels[len(levels)-5], levels[len(levels)-1])
	}
	if last := abs16(decodeSample(buffer[(FramesPerBuffer-1)*BytesPerFrame:])); last > 64 {
		t.Fatalf("last sample magnitude = %d, want near zero", last)
	}
}

func TestVeryShortToneDoesNotPanic(t *testing.T) {
	var generator tone
	generator.start(880, 1)
	var buffer [BufferBytes]byte
	if got := generator.fill(buffer[:]); got != 1 {
		t.Fatalf("frames = %d, want 1", got)
	}
	if generator.active() {
		t.Fatal("one-frame tone remains active")
	}
}

func TestToneFillAllocations(t *testing.T) {
	var generator tone
	var buffer [BufferBytes]byte
	allocs := testing.AllocsPerRun(1000, func() {
		generator.start(880, FramesPerBuffer)
		generator.fill(buffer[:])
	})
	if allocs != 0 {
		t.Fatalf("allocations = %v, want 0", allocs)
	}
}

func decodeSample(data []byte) int16 { return int16(uint16(data[0]) | uint16(data[1])<<8) }

func abs16(value int16) int16 {
	if value < 0 {
		return -value
	}
	return value
}
