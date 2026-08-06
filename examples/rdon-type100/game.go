package main

import (
	"fmt"
	"time"

	"github.com/rdon-key/modgadget"
)

type playOutcome uint8

const (
	playIgnored playOutcome = iota
	playInput
	playMiss
	playNextQuestion
	playComplete
)

type playState struct {
	course         courseID
	questions      []question
	questionIndex  int
	inputIndex     int
	misses         int
	startedAt      time.Time
	finishedAt     time.Time
	pressedLetters uint32
	active         bool
}

func (play *playState) start(course courseID, now time.Time) bool {
	questions := questionsForCourse(course)
	if len(questions) == 0 {
		play.reset()
		return false
	}
	*play = playState{course: course, questions: questions, startedAt: now, active: true}
	return true
}

func (play *playState) reset() {
	*play = playState{}
}

func (play *playState) currentQuestion() question {
	if !play.active || play.questionIndex < 0 || play.questionIndex >= len(play.questions) {
		return question{}
	}
	return play.questions[play.questionIndex]
}

func (play *playState) typed() string {
	current := play.currentQuestion()
	if play.inputIndex < 0 || play.inputIndex > len(current.roman) {
		return ""
	}
	return current.roman[:play.inputIndex]
}

func letterBit(code modgadget.KeyCode) (uint32, bool) {
	if code < modgadget.KeyA || code > modgadget.KeyZ {
		return 0, false
	}
	return uint32(1) << uint32(code-modgadget.KeyA), true
}

func asciiLower(r rune) (byte, bool) {
	switch {
	case r >= 'a' && r <= 'z':
		return byte(r), true
	case r >= 'A' && r <= 'Z':
		return byte(r + ('a' - 'A')), true
	default:
		return 0, false
	}
}

func (play *playState) handleKey(event modgadget.KeyEvent, now time.Time) playOutcome {
	bit, letter := letterBit(event.Code)
	if event.Action == modgadget.KeyUp {
		if letter {
			play.pressedLetters &^= bit
		}
		return playIgnored
	}
	if event.Action != modgadget.KeyDown || !play.active || !letter {
		return playIgnored
	}
	if play.pressedLetters&bit != 0 {
		return playIgnored
	}
	play.pressedLetters |= bit
	input, ok := asciiLower(event.Rune)
	if !ok {
		return playIgnored
	}
	current := play.currentQuestion()
	if play.inputIndex >= len(current.roman) || input != current.roman[play.inputIndex] {
		play.misses++
		return playMiss
	}
	play.inputIndex++
	if play.inputIndex < len(current.roman) {
		return playInput
	}
	if play.questionIndex+1 == len(play.questions) {
		play.finishedAt = now
		play.active = false
		return playComplete
	}
	play.questionIndex++
	play.inputIndex = 0
	return playNextQuestion
}

func (play *playState) elapsedTenths(now time.Time) int64 {
	if play.startedAt.IsZero() {
		return 0
	}
	end := now
	if !play.finishedAt.IsZero() {
		end = play.finishedAt
	}
	elapsed := end.Sub(play.startedAt)
	if elapsed < 0 {
		return 0
	}
	return int64(elapsed / (100 * time.Millisecond))
}

func formatTenths(tenths int64) string {
	if tenths < 0 {
		tenths = 0
	}
	return fmt.Sprintf("%d.%ds", tenths/10, tenths%10)
}

func formatQuestionNumber(index int) string {
	return fmt.Sprintf("%02d/%02d", index+1, 20)
}

func resultTimeText(play *playState) string {
	return "TIME  " + formatTenths(play.elapsedTenths(play.finishedAt))
}

func resultMissText(play *playState) string {
	return fmt.Sprintf("MISS  %d", play.misses)
}
