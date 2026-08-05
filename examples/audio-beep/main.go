//go:build tinygo

package main

import (
	"machine"
	"runtime"
	"time"

	"github.com/rdon-key/modgadget"
	audio "github.com/rdon-key/modgadget/internal/audio/cardputeradv"
	keyboarddriver "github.com/rdon-key/modgadget/internal/keyboard/cardputeradv"
)

const (
	initialDelay    = 2 * time.Second
	patternInterval = 750 * time.Millisecond
	loopInterval    = 1500 * time.Millisecond
	testLoopCount   = 5
)

type soundTest struct {
	name    string
	pattern audio.Pattern
}

var soundTests = [...]soundTest{
	{name: "startup", pattern: audio.PatternStartup},
	{name: "click", pattern: audio.PatternClick},
	{name: "correct", pattern: audio.PatternCorrect},
	{name: "wrong", pattern: audio.PatternWrong},
}

type debugVolumeController struct {
	player *audio.Player
}

var _ modgadget.VolumeController = (*debugVolumeController)(nil)

func (controller *debugVolumeController) VolumeUp() {
	controller.player.VolumeUp()
	println("audio: volume up", controller.player.Volume().String())
}

func (controller *debugVolumeController) VolumeDown() {
	controller.player.VolumeDown()
	println("audio: volume down", controller.player.Volume().String())
}

func (controller *debugVolumeController) ToggleMute() {
	controller.player.ToggleMute()
	if controller.player.Muted() {
		println("audio: mute on")
	} else {
		println("audio: mute off", controller.player.Volume().String())
	}
}

func main() {
	player := audio.New()
	if err := player.Configure(); err != nil {
		panic(err)
	}
	println("audio: configured")
	println("audio: volume", player.Volume().String())
	println("audio: Fn+= up, Fn+- down, Fn+M mute")

	// Audio Configure initially owns the shared I2C bus at 100 kHz. Configure
	// it for the TCA8418 last; ES8311 playback does not perform I2C transfers.
	if err := machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 400_000,
		SDA:       machine.GPIO8,
		SCL:       machine.GPIO9,
	}); err != nil {
		panic(err)
	}
	keyboard := keyboarddriver.New(machine.I2C0)
	if err := keyboard.Configure(); err != nil {
		panic(err)
	}
	volume := &debugVolumeController{player: player}
	gadget := modgadget.New(nil,
		modgadget.WithKeyboard(keyboard),
		modgadget.WithVolumeController(volume),
	)

	nextStart := time.Now().Add(initialDelay)
	loopIndex := 0
	testIndex := 0
	playing := false
	complete := false

	for {
		if err := player.Update(); err != nil {
			_ = player.Stop()
			panic(err)
		}

		now := time.Now()
		gadget.Update(now)
		if err := keyboard.Err(); err != nil {
			panic(err)
		}

		if playing && !player.Busy() {
			println("audio:", soundTests[testIndex].name, "complete")
			playing = false
			testIndex++
			if testIndex == len(soundTests) {
				testIndex = 0
				loopIndex++
				if loopIndex == testLoopCount {
					complete = true
					println("audio: test complete")
				} else {
					nextStart = now.Add(loopInterval)
				}
			} else {
				nextStart = now.Add(patternInterval)
			}
		}

		if !playing && !complete && !now.Before(nextStart) {
			test := soundTests[testIndex]
			if testIndex == 0 {
				println("audio: round", loopIndex+1, "/", testLoopCount)
			}
			if err := player.PlayPattern(test.pattern); err != nil {
				panic(err)
			}
			println("audio:", test.name, "started")
			playing = true
		}

		runtime.Gosched()
	}
}
