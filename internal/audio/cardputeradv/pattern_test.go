package cardputeradv

import "testing"

func TestStartupPattern(t *testing.T) {
	got := patternSteps(PatternStartup)
	want := []patternStep{
		{
			kind:      stepTone,
			frequency: StartupLowFrequencyHz,
			frames:    StartupFirstToneFrames,
		},
		{
			kind:   stepSilence,
			frames: StartupGapFrames,
		},
		{
			kind:      stepTone,
			frequency: StartupHighFrequencyHz,
			frames:    StartupSecondToneFrames,
		},
	}

	if len(got) != len(want) {
		t.Fatalf("patternSteps(PatternStartup) length = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("step %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestStartupPatternFrames(t *testing.T) {
	tests := []struct {
		name string
		got  uint32
		want uint32
	}{
		{
			name: "first tone",
			got:  StartupFirstToneFrames,
			want: uint32(SampleRate * 400 / 1000),
		},
		{
			name: "gap",
			got:  StartupGapFrames,
			want: uint32(SampleRate * 120 / 1000),
		},
		{
			name: "second tone",
			got:  StartupSecondToneFrames,
			want: uint32(SampleRate * 600 / 1000),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("frames = %d, want %d", test.got, test.want)
			}
		})
	}
}

func TestPatternStepsRejectsUnknownPattern(t *testing.T) {
	tests := []Pattern{
		0,
		Pattern(255),
	}

	for _, pattern := range tests {
		if got := patternSteps(pattern); got != nil {
			t.Errorf("patternSteps(%d) = %+v, want nil", pattern, got)
		}
	}
}
