//go:build tinygo

package main

import (
	"runtime"
	"time"

	"github.com/rdon-key/modgadget/internal/audio/cardputeradv"
)

type soundTest struct {
	name    string
	pattern cardputeradv.Pattern
}

var soundTests = [...]soundTest{
	{
		name:    "startup",
		pattern: cardputeradv.PatternStartup,
	},
	{
		name:    "click",
		pattern: cardputeradv.PatternClick,
	},
	{
		name:    "correct",
		pattern: cardputeradv.PatternCorrect,
	},
	{
		name:    "wrong",
		pattern: cardputeradv.PatternWrong,
	},
}

func main() {
	player := cardputeradv.New()
	if err := player.Configure(); err != nil {
		panic(err)
	}
	println("audio: configured")

	time.Sleep(2 * time.Second)

	for loop := 1; loop <= 2; loop++ {
		println("audio: test loop", loop, "started")

		for _, test := range soundTests {
			println("audio:", test.name, "started")

			if err := player.PlayPattern(test.pattern); err != nil {
				panic(err)
			}

			for player.Busy() {
				if err := player.Update(); err != nil {
					_ = player.Stop()
					panic(err)
				}
				runtime.Gosched()
			}

			println("audio:", test.name, "complete")
			time.Sleep(1 * time.Second)
		}

		println("audio: test loop", loop, "complete")
		time.Sleep(2 * time.Second)
	}

	println("audio: test complete")

	for {
		runtime.Gosched()
	}
}
