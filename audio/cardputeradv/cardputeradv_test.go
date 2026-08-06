package cardputeradv

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/rdon-key/modgadget"
	internalaudio "github.com/rdon-key/modgadget/internal/audio/cardputeradv"
)

var _ modgadget.VolumeController = (*Player)(nil)

func TestPublicPatternsAreValidAndDistinct(t *testing.T) {
	patterns := []Pattern{PatternStartup, PatternClick, PatternCorrect, PatternWrong}
	seen := make(map[Pattern]bool, len(patterns))
	for _, pattern := range patterns {
		if pattern == 0 || seen[pattern] {
			t.Fatalf("invalid or duplicate pattern %d", pattern)
		}
		seen[pattern] = true
	}
}

func TestPublicValuesMatchImplementation(t *testing.T) {
	patterns := map[Pattern]internalaudio.Pattern{
		PatternStartup: internalaudio.PatternStartup,
		PatternClick:   internalaudio.PatternClick,
		PatternWrong:   internalaudio.PatternWrong,
		PatternCorrect: internalaudio.PatternCorrect,
	}
	for public, internal := range patterns {
		if uint8(public) != uint8(internal) {
			t.Fatalf("pattern %d != internal %d", public, internal)
		}
	}
	volumes := map[VolumeLevel]internalaudio.VolumeLevel{
		VolumeMute:   internalaudio.VolumeMute,
		VolumeLow:    internalaudio.VolumeLow,
		VolumeMedium: internalaudio.VolumeMedium,
		VolumeHigh:   internalaudio.VolumeHigh,
	}
	for public, internal := range volumes {
		if uint8(public) != uint8(internal) {
			t.Fatalf("volume %d != internal %d", public, internal)
		}
	}
}

func TestPlayerHasNoPublicFields(t *testing.T) {
	typeOfPlayer := reflect.TypeOf(Player{})
	for index := 0; index < typeOfPlayer.NumField(); index++ {
		if typeOfPlayer.Field(index).IsExported() {
			t.Fatalf("Player exposes field %s", typeOfPlayer.Field(index).Name)
		}
	}
}

func TestPlayerPublicMethodSet(t *testing.T) {
	var player *Player
	_ = player.PlayTone
	_ = player.PlayPattern
	_ = player.Update
	_ = player.Busy
	_ = player.Stop
	_ = player.VolumeUp
	_ = player.VolumeDown
	_ = player.ToggleMute
}

func TestZeroAndNilPlayerAreSafe(t *testing.T) {
	players := []*Player{{}, nil}
	for _, player := range players {
		errorMethods := []struct {
			name string
			call func() error
		}{
			{"PlayTone", func() error { return player.PlayTone(440, time.Second) }},
			{"PlayPattern", func() error { return player.PlayPattern(PatternClick) }},
			{"Update", player.Update},
			{"Stop", player.Stop},
			{"SetVolume", func() error { return player.SetVolume(VolumeHigh) }},
		}
		for _, method := range errorMethods {
			if err := method.call(); !errors.Is(err, ErrNotConfigured) {
				t.Errorf("%s error=%v, want ErrNotConfigured", method.name, err)
			}
		}
		if player.Busy() {
			t.Error("Busy=true, want false")
		}
		if got := player.Volume(); got != VolumeMute {
			t.Errorf("Volume=%v, want VolumeMute", got)
		}
		if !player.Muted() {
			t.Error("Muted=false, want true")
		}
		player.VolumeUp()
		player.VolumeDown()
		player.Mute()
		player.Unmute()
		player.ToggleMute()
	}
}
