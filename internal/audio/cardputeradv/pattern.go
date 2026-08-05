package cardputeradv

// Pattern identifies one built-in sound pattern.
type Pattern uint8

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

type stepKind uint8

const (
	stepTone stepKind = iota
	stepSilence
)

type patternStep struct {
	kind      stepKind
	frequency uint16
	frames    uint32
}

const (
	StartupLowFrequencyHz  = uint16(660)
	StartupHighFrequencyHz = uint16(880)

	StartupFirstToneFrames  = uint32(SampleRate * 400 / 1000)
	StartupGapFrames        = uint32(SampleRate * 120 / 1000)
	StartupSecondToneFrames = uint32(SampleRate * 600 / 1000)

	ClickFrequencyHz = uint16(1400)
	ClickToneFrames  = uint32(SampleRate * 100 / 1000)

	WrongFrequencyHz = uint16(330)
	WrongToneFrames  = uint32(SampleRate * 500 / 1000)

	CorrectFirstFrequencyHz  = uint16(1047)
	CorrectSecondFrequencyHz = uint16(784)

	CorrectFirstToneFrames  = uint32(SampleRate * 300 / 1000)
	CorrectGapFrames        = uint32(SampleRate * 100 / 1000)
	CorrectSecondToneFrames = uint32(SampleRate * 2600 / 1000)
)

var startupPattern = [...]patternStep{
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

var clickPattern = [...]patternStep{
	{
		kind:      stepTone,
		frequency: ClickFrequencyHz,
		frames:    ClickToneFrames,
	},
}

var wrongPattern = [...]patternStep{
	{
		kind:      stepTone,
		frequency: WrongFrequencyHz,
		frames:    WrongToneFrames,
	},
}

var correctPattern = [...]patternStep{
	{
		kind:      stepTone,
		frequency: CorrectFirstFrequencyHz,
		frames:    CorrectFirstToneFrames,
	},
	{
		kind:   stepSilence,
		frames: CorrectGapFrames,
	},
	{
		kind:      stepTone,
		frequency: CorrectSecondFrequencyHz,
		frames:    CorrectSecondToneFrames,
	},
}

func patternSteps(pattern Pattern) []patternStep {
	switch pattern {
	case PatternStartup:
		return startupPattern[:]
	case PatternClick:
		return clickPattern[:]
	case PatternWrong:
		return correctPattern[:]
	case PatternCorrect:
		return wrongPattern[:]
	default:
		return nil
	}
}
