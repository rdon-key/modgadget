package main

import (
	"fmt"
	"time"

	"github.com/rdon-key/modgadget"
	audio "github.com/rdon-key/modgadget/internal/audio/cardputeradv"
)

const (
	displayWidth      int16 = 240
	displayHeight     int16 = 135
	titleText               = "Rdon Type 100"
	titleMarkup             = "<style=title>Rdon <b>Type 100</b></style>"
	titleBoldInkExtra int16 = 1
	titleY            int16 = 0
	titleHeight       int16 = 18
	menuX             int16 = 20
	menuWidth         int16 = 200
	menuY             int16 = 21
	menuStep          int16 = 17
	menuHeight        int16 = 16
	menuFrameX        int16 = 16
	menuFrameY        int16 = 19
	menuFrameWidth    int16 = 208
	menuFrameHeight   int16 = 89
	guideY            int16 = 111
	guideHeight       int16 = 24
	statusTimeX       int16 = 0
	statusTimeY       int16 = 0
	statusTimeWidth   int16 = 176
	statusNumberY     int16 = 0
	statusHeight      int16 = 24
	romanX            int16 = 0
	romanY            int16 = 26
	romanWidth        int16 = 240
	romanHeight       int16 = 16
	promptX           int16 = 0
	promptY           int16 = 44
	promptWidth       int16 = 240
	promptHeight      int16 = 24
	inputFrameX       int16 = 0
	inputFrameY       int16 = 70
	inputFrameWidth   int16 = 240
	inputFrameHeight  int16 = 38
	inputX            int16 = 4
	inputY            int16 = 77
	inputWidth        int16 = 232
	inputHeight       int16 = 24
	resultX           int16 = 20
	resultWidth       int16 = 220
	resultCompleteY   int16 = 4
	resultTimeY       int16 = 34
	resultMissY       int16 = 62
	resultHeight      int16 = 24
	resultInstructY   int16 = 94
	resultInstructH   int16 = 16
)

const (
	guideSpeed       = 24
	guideGap   int16 = 32

	japanesePlayingGuide = "DELで強制終了 ◆ Fn+Mで消音 ◆ Fn++/Fn+-で音量調整 ◆ ローマ字で入力して下さい。"
	englishPlayingGuide  = "DEL: Quit ◆ Fn+M: Mute ◆ Fn++/Fn+-: Volume ◆ Type the shown letters."
	chinesePlayingGuide  = "DEL强制结束 ◆ Fn+M静音 ◆ Fn++/Fn+-调节音量 ◆ 请使用拼音输入。"
	koreanPlayingGuide   = "DEL로 강제 종료 ◆ Fn+M 음소거 ◆ Fn++/Fn+- 음량 조절 ◆ 로마자로 입력하세요."
	playingGuideDivider  = " ◇ "
	allPlayingGuide      = japanesePlayingGuide + playingGuideDivider + englishPlayingGuide + playingGuideDivider + chinesePlayingGuide + playingGuideDivider + koreanPlayingGuide
)

func playingGuideForCourse(id courseID) string {
	switch id {
	case courseJapanese:
		return japanesePlayingGuide
	case courseEnglish:
		return englishPlayingGuide
	case courseChinese:
		return chinesePlayingGuide
	case courseKorean:
		return koreanPlayingGuide
	case courseAll:
		return allPlayingGuide
	default:
		return ""
	}
}

func setGuide(view *modgadget.Viewport, text string) error {
	if err := view.SetText("<style=" + styleGuide + ">" + text + "</style>"); err != nil {
		return err
	}
	view.SetHorizontalScroll(
		modgadget.ScrollSpeed(guideSpeed),
		modgadget.ScrollGap(guideGap),
		modgadget.ScrollLoop(),
		modgadget.ScrollFromRight(),
	)
	return nil
}

const (
	styleTitle    = "title"
	styleSelected = "selected"
	styleReady    = "ready"
	styleGuide    = "guide"
	styleLarge    = "large"
	styleInput    = "input"
)

func makeStyles(menuFont, guideFont modgadget.Font) modgadget.StyleSet {
	return modgadget.StyleSet{
		Default: modgadget.Style{Font: menuFont, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack},
		Entries: []modgadget.StyleEntry{
			{Name: styleTitle, Style: modgadget.Style{Font: menuFont, Foreground: modgadget.RGB565(80, 220, 255), Background: modgadget.ColorBlack}},
			{Name: styleSelected, Style: modgadget.Style{Font: menuFont, Foreground: modgadget.RGB565(255, 220, 0), Background: modgadget.ColorBlack}},
			{Name: styleReady, Style: modgadget.Style{Font: menuFont, Foreground: modgadget.ColorGreen, Background: modgadget.ColorBlack}},
			{Name: styleGuide, Style: modgadget.Style{Font: guideFont, Foreground: modgadget.ColorRed, Background: modgadget.ColorBlack}},
			{Name: styleLarge, Style: modgadget.Style{Font: guideFont, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlack}},
			{Name: styleInput, Style: modgadget.Style{Font: guideFont, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlue}},
		},
	}
}

func makeInputStyles(font modgadget.Font) modgadget.StyleSet {
	input := modgadget.Style{Font: font, Foreground: modgadget.ColorWhite, Background: modgadget.ColorBlue}
	return modgadget.StyleSet{
		Default: input,
		Entries: []modgadget.StyleEntry{{Name: styleInput, Style: input}},
	}
}

func textAdvance(font modgadget.Font, value string) (int16, error) {
	var width int32
	for _, r := range value {
		glyph, ok := font.Lookup(r)
		if !ok {
			return 0, fmt.Errorf("missing glyph %q", r)
		}
		width += int32(glyph.AdvanceX)
		if width > int32(displayWidth) {
			return 0, fmt.Errorf("text width %d exceeds display", width)
		}
	}
	return int16(width), nil
}

func centeredTitleX(titleWidth int16) int16 {
	return (displayWidth - titleWidth) / 2
}

type screenState uint8

const (
	stateSplash screenState = iota
	stateMenu
	statePlaying
	stateResult
)

type courseID uint8

const (
	courseJapanese courseID = iota
	courseEnglish
	courseChinese
	courseKorean
	courseAll
)

type course struct {
	id         courseID
	menuLabel  string
	guide      string
	startLabel string
}

const (
	japaneseGuide = "Fn+▲▼で言語を選択して、Enterで開始します。"
	englishGuide  = "Use Fn+▲▼ to select a language, then press Enter to start."
	chineseGuide  = "使用Fn+▲▼选择语言，然后按Enter键开始。"
	koreanGuide   = "Fn+▲▼로 언어를 선택하고 Enter를 눌러 시작합니다."
	guideDivider  = " ◆ "
	allGuide      = japaneseGuide + guideDivider + englishGuide + guideDivider + chineseGuide + guideDivider + koreanGuide
)

var courses = [...]course{
	{id: courseJapanese, menuLabel: "日本語", guide: japaneseGuide, startLabel: "日本語 course selected"},
	{id: courseEnglish, menuLabel: "English", guide: englishGuide, startLabel: "English course selected"},
	{id: courseChinese, menuLabel: "中文", guide: chineseGuide, startLabel: "中文 course selected"},
	{id: courseKorean, menuLabel: "한국어", guide: koreanGuide, startLabel: "한국어 course selected"},
	{id: courseAll, menuLabel: "All Languages", guide: allGuide, startLabel: "All Languages selected"},
}

type appState struct {
	screen          screenState
	selection       int
	selected        courseID
	waitingEnterUp  bool
	pressedControls uint8
	waitingDeleteUp bool
	play            playState
}

const (
	pressedUp uint8 = 1 << iota
	pressedDown
	pressedEnter
	pressedDelete
)

func controlKeyBit(code modgadget.KeyCode) uint8 {
	switch code {
	case modgadget.KeyArrowUp:
		return pressedUp
	case modgadget.KeyArrowDown:
		return pressedDown
	case modgadget.KeyEnter:
		return pressedEnter
	case modgadget.KeyDelete:
		return pressedDelete
	default:
		return 0
	}
}

type soundEffect uint8

const (
	soundNone soundEffect = iota
	soundCursorMove
	soundCourseConfirm
	soundMiss
	soundQuestionCorrect
	soundCourseComplete
)

type patternPlayer interface {
	PlayPattern(audio.Pattern) error
	Busy() bool
	Stop() error
}

type startupSound struct {
	phase uint8
}

func (sound *startupSound) start(player patternPlayer) error {
	if sound.phase != 0 {
		return nil
	}
	if err := player.PlayPattern(audio.PatternStartup); err != nil {
		return err
	}
	sound.phase = 1
	return nil
}

func (sound *startupSound) update(player patternPlayer) error {
	if sound.phase == 1 && !player.Busy() {
		if err := player.PlayPattern(audio.PatternClick); err != nil {
			return err
		}
		sound.phase = 2
	} else if sound.phase == 2 && !player.Busy() {
		sound.phase = 3
	}
	return nil
}

func (sound *startupSound) finish(player patternPlayer) error {
	if sound.phase < 3 && player.Busy() {
		if err := player.Stop(); err != nil {
			return err
		}
	}
	sound.phase = 3
	return nil
}

func playEffect(player patternPlayer, effect soundEffect) error {
	switch effect {
	case soundCursorMove:
		return player.PlayPattern(audio.PatternClick)
	case soundCourseConfirm:
		return player.PlayPattern(audio.PatternCorrect)
	case soundMiss:
		return player.PlayPattern(audio.PatternWrong)
	case soundQuestionCorrect:
		return player.PlayPattern(audio.PatternCorrect)
	case soundCourseComplete:
		return player.PlayPattern(audio.PatternStartup)
	default:
		return nil
	}
}

func handleKeyWithSound(app *appState, player patternPlayer, event modgadget.KeyEvent) (bool, error) {
	return handleKeyWithSoundAt(app, player, event, time.Time{})
}

func handleKeyWithSoundAt(app *appState, player patternPlayer, event modgadget.KeyEvent, now time.Time) (bool, error) {
	handled, effect := app.handleKeyAt(event, now)
	if !handled {
		return false, nil
	}
	if err := playEffect(player, effect); err != nil {
		return true, err
	}
	return true, nil
}

func newAppState() appState {
	return appState{screen: stateSplash, selection: int(courseJapanese), selected: courseJapanese}
}

func (app *appState) showMenu() {
	app.screen = stateMenu
	app.play.reset()
}

func (app *appState) currentCourse() course {
	return courses[app.selection]
}

func (app *appState) handleKey(event modgadget.KeyEvent) (bool, soundEffect) {
	return app.handleKeyAt(event, time.Time{})
}

func (app *appState) handleKeyAt(event modgadget.KeyEvent, now time.Time) (bool, soundEffect) {
	bit := controlKeyBit(event.Code)
	if event.Action == modgadget.KeyUp && bit != 0 {
		app.pressedControls &^= bit
	}
	if app.waitingEnterUp {
		if event.Code == modgadget.KeyEnter && event.Action == modgadget.KeyUp {
			app.waitingEnterUp = false
			return true, soundNone
		}
		if event.Code == modgadget.KeyEnter && event.Action == modgadget.KeyDown {
			return true, soundNone
		}
		return false, soundNone
	}
	if app.waitingDeleteUp {
		if event.Code == modgadget.KeyDelete && event.Action == modgadget.KeyUp {
			app.waitingDeleteUp = false
			return true, soundNone
		}
		if event.Code == modgadget.KeyDelete && event.Action == modgadget.KeyDown {
			return true, soundNone
		}
	}
	if app.screen == statePlaying {
		if event.Code == modgadget.KeyDelete && event.Action == modgadget.KeyDown {
			if bit != 0 && app.pressedControls&bit != 0 {
				return true, soundNone
			}
			app.pressedControls |= bit
			app.waitingDeleteUp = true
			app.showMenu()
			return true, soundNone
		}
		switch app.play.handleKey(event, now) {
		case playInput:
			return true, soundNone
		case playMiss:
			return true, soundMiss
		case playNextQuestion:
			return true, soundQuestionCorrect
		case playComplete:
			app.screen = stateResult
			return true, soundCourseComplete
		default:
			if _, letter := letterBit(event.Code); letter {
				return true, soundNone
			}
			return false, soundNone
		}
	}
	if app.screen == stateResult {
		if event.Action == modgadget.KeyDown && event.Code == modgadget.KeyEnter {
			if bit != 0 && app.pressedControls&bit != 0 {
				return true, soundNone
			}
			app.pressedControls |= bit
			app.waitingEnterUp = true
			app.showMenu()
			return true, soundCourseConfirm
		}
		return false, soundNone
	}
	if app.screen != stateMenu || event.Action != modgadget.KeyDown {
		return false, soundNone
	}
	if bit != 0 {
		if app.pressedControls&bit != 0 {
			return true, soundNone
		}
		app.pressedControls |= bit
	}
	switch event.Code {
	case modgadget.KeyArrowUp:
		app.selection--
		if app.selection < 0 {
			app.selection = len(courses) - 1
		}
		return true, soundCursorMove
	case modgadget.KeyArrowDown:
		app.selection++
		if app.selection == len(courses) {
			app.selection = 0
		}
		return true, soundCursorMove
	case modgadget.KeyEnter:
		app.selected = app.currentCourse().id
		app.waitingEnterUp = true
		app.screen = statePlaying
		app.play.start(app.selected, now)
		return true, soundCourseConfirm
	default:
		return false, soundNone
	}
}
