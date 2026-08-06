package main

import (
	"strings"
	"testing"
	"time"

	"github.com/rdon-key/modgadget"
	audio "github.com/rdon-key/modgadget/audio/cardputeradv"
	"github.com/rdon-key/modgadget/font/efont16"
	"github.com/rdon-key/modgadget/font/efont24"
)

func letterEvent(code modgadget.KeyCode, r rune, action modgadget.KeyAction) modgadget.KeyEvent {
	return modgadget.KeyEvent{Code: code, Rune: r, Action: action}
}

func TestJapaneseQuestionData(t *testing.T) {
	if len(japaneseQuestions) != 20 {
		t.Fatalf("questions=%d want=20", len(japaneseQuestions))
	}
	seenPrompt, seenRoman := map[string]bool{}, map[string]bool{}
	promptStyles := modgadget.StyleSet{Default: modgadget.Style{Font: efont24.Font}}
	for index, item := range japaneseQuestions {
		if item.prompt == "" || item.roman == "" {
			t.Fatalf("question %d is empty: %+v", index, item)
		}
		if seenPrompt[item.prompt] || seenRoman[item.roman] {
			t.Fatalf("duplicate question %d: %+v", index, item)
		}
		seenPrompt[item.prompt], seenRoman[item.roman] = true, true
		for _, r := range item.roman {
			if r < 'a' || r > 'z' {
				t.Errorf("question %d roman contains %q", index, r)
			}
			if !efont16.Font.HasGlyph(r) {
				t.Errorf("question %d Efont16 missing %q", index, r)
			}
			if !efont24.Font.HasGlyph(r) {
				t.Errorf("question %d Efont24 missing %q", index, r)
			}
		}
		if _, err := modgadget.MeasureText(item.prompt, promptStyles); err != nil {
			t.Errorf("question %d prompt markup: %v", index, err)
		}
		if width := fontWidth(t, efont16.Font, item.roman); width > romanWidth {
			t.Errorf("question %d roman width=%d > %d", index, width, romanWidth)
		}
		if width := fontWidth(t, efont24.Font, item.roman); width > inputWidth {
			t.Errorf("question %d input width=%d > %d", index, width, inputWidth)
		}
	}
}

func fontWidth(t *testing.T, font modgadget.Font, value string) int16 {
	t.Helper()
	measurement, err := modgadget.MeasureText(value, modgadget.StyleSet{Default: modgadget.Style{Font: font}})
	if err != nil {
		t.Fatal(err)
	}
	return measurement.Width
}

func TestPlayInputCorrectMissUppercaseAndDuplicateDown(t *testing.T) {
	start := time.Unix(100, 0)
	play := playState{}
	play.start(courseJapanese, start)
	first := play.currentQuestion().roman
	code := modgadget.KeyK
	if first[0] != 'k' {
		t.Fatalf("test expects first roman to start k: %q", first)
	}
	if got := play.handleKey(letterEvent(code, 'K', modgadget.KeyDown), start); got != playInput || play.inputIndex != 1 {
		t.Fatalf("uppercase outcome=%v input=%d", got, play.inputIndex)
	}
	if got := play.handleKey(letterEvent(code, 'K', modgadget.KeyDown), start); got != playIgnored || play.inputIndex != 1 {
		t.Fatalf("duplicate down outcome=%v input=%d", got, play.inputIndex)
	}
	if got := play.handleKey(letterEvent(code, 0, modgadget.KeyUp), start); got != playIgnored {
		t.Fatalf("keyup outcome=%v", got)
	}
	misses := play.misses
	if got := play.handleKey(letterEvent(modgadget.KeyA, 'a', modgadget.KeyDown), start); got != playMiss || play.inputIndex != 1 || play.misses != misses+1 {
		t.Fatalf("miss outcome=%v input=%d misses=%d", got, play.inputIndex, play.misses)
	}
	play.handleKey(letterEvent(modgadget.KeyA, 0, modgadget.KeyUp), start)
	if got := play.handleKey(modgadget.KeyEvent{Code: modgadget.KeyF12, Action: modgadget.KeyDown, Modifiers: modgadget.ModFn}, start); got != playIgnored {
		t.Fatalf("system key outcome=%v", got)
	}
}

func typeRemaining(t *testing.T, play *playState, now time.Time) playOutcome {
	t.Helper()
	current := play.currentQuestion()
	last := playIgnored
	for index := play.inputIndex; index < len(current.roman); index++ {
		letter := current.roman[index]
		code := modgadget.KeyA + modgadget.KeyCode(letter-'a')
		last = play.handleKey(letterEvent(code, rune(letter), modgadget.KeyDown), now)
		play.handleKey(letterEvent(code, 0, modgadget.KeyUp), now)
	}
	return last
}

func TestQuestionAdvanceAndFinalResultWithoutOutOfBounds(t *testing.T) {
	now := time.Unix(200, 0)
	play := playState{}
	play.start(courseJapanese, now)
	if got := typeRemaining(t, &play, now.Add(time.Second)); got != playNextQuestion || play.questionIndex != 1 || play.inputIndex != 0 {
		t.Fatalf("first completion outcome=%v question=%d input=%d", got, play.questionIndex, play.inputIndex)
	}
	for play.questionIndex < len(play.questions)-1 {
		if got := typeRemaining(t, &play, now.Add(time.Second)); got != playNextQuestion {
			t.Fatalf("question %d outcome=%v", play.questionIndex, got)
		}
	}
	if got := typeRemaining(t, &play, now.Add(42300*time.Millisecond)); got != playComplete {
		t.Fatalf("last outcome=%v", got)
	}
	if play.active || play.questionIndex != 19 || play.finishedAt.IsZero() {
		t.Fatalf("final state=%+v", play)
	}
	if got := play.handleKey(letterEvent(modgadget.KeyA, 'a', modgadget.KeyDown), now.Add(time.Minute)); got != playIgnored {
		t.Fatalf("post-complete outcome=%v", got)
	}
}

func TestElapsedTenthsStartsClampsAndFreezes(t *testing.T) {
	start := time.Unix(300, 0)
	play := playState{}
	play.start(courseJapanese, start)
	if got := play.elapsedTenths(start.Add(349 * time.Millisecond)); got != 3 || formatTenths(got) != "0.3s" {
		t.Fatalf("tenths=%d text=%q", got, formatTenths(got))
	}
	if got := play.elapsedTenths(start.Add(-time.Second)); got != 0 {
		t.Fatalf("negative time=%d", got)
	}
	play.finishedAt = start.Add(12340 * time.Millisecond)
	if got := play.elapsedTenths(start.Add(time.Hour)); got != 123 || formatTenths(got) != "12.3s" {
		t.Fatalf("frozen=%d text=%q", got, formatTenths(got))
	}
	if formatTenths(1234) != "123.4s" {
		t.Fatalf("long time=%q", formatTenths(1234))
	}
	if formatQuestionNumber(0) != "01/20" || formatQuestionNumber(19) != "20/20" {
		t.Fatalf("question labels=%q/%q", formatQuestionNumber(0), formatQuestionNumber(19))
	}
	if resultTimeText(&play) != "TIME  12.3s" {
		t.Fatalf("result time=%q", resultTimeText(&play))
	}
	play.misses = 7
	if resultMissText(&play) != "MISS  7" {
		t.Fatalf("result misses=%q", resultMissText(&play))
	}
}

func TestAppDeleteAbortsAndResultEnterReturnsToMenu(t *testing.T) {
	start := time.Unix(400, 0)
	app := newAppState()
	app.showMenu()
	_, _ = app.handleKeyAt(keyDown(modgadget.KeyEnter), start)
	_, _ = app.handleKeyAt(modgadget.KeyEvent{Code: modgadget.KeyEnter, Action: modgadget.KeyUp}, start)
	app.play.inputIndex, app.play.misses = 2, 3
	if handled, effect := app.handleKeyAt(keyDown(modgadget.KeyDelete), start); !handled || effect != soundNone || app.screen != stateMenu {
		t.Fatalf("delete handled=%v effect=%v app=%+v", handled, effect, app)
	}
	if app.selection != int(courseJapanese) || app.selected != courseJapanese || app.play.active || app.play.inputIndex != 0 || app.play.misses != 0 {
		t.Fatalf("delete did not discard play or preserve course: %+v", app)
	}
	if handled, _ := app.handleKeyAt(keyDown(modgadget.KeyDelete), start); !handled || app.screen != stateMenu {
		t.Fatalf("held delete handled=%v screen=%v", handled, app.screen)
	}
	_, _ = app.handleKeyAt(modgadget.KeyEvent{Code: modgadget.KeyDelete, Action: modgadget.KeyUp}, start)

	app.screen = stateResult
	app.play = playState{misses: 7, startedAt: start, finishedAt: start.Add(42300 * time.Millisecond)}
	if app.play.elapsedTenths(time.Now()) != 423 || formatTenths(app.play.elapsedTenths(time.Now())) != "42.3s" {
		t.Fatal("result time is not fixed")
	}
	if handled, effect := app.handleKeyAt(keyDown(modgadget.KeyEnter), start); !handled || effect != soundCourseConfirm || app.screen != stateMenu || !app.waitingEnterUp {
		t.Fatalf("result enter handled=%v effect=%v app=%+v", handled, effect, app)
	}
}

func TestPlayingLayoutAndStyles(t *testing.T) {
	numberWidth := fontWidth(t, efont24.Font, "00/00")
	regions := []uiRect{
		{x: statusTimeX, y: statusTimeY, width: statusTimeWidth, height: statusHeight},
		{x: displayWidth - numberWidth, y: statusNumberY, width: numberWidth, height: statusHeight},
		{x: romanX, y: romanY, width: romanWidth, height: romanHeight},
		{x: promptX, y: promptY, width: promptWidth, height: promptHeight},
		inputFrameBounds,
		{x: 0, y: guideY, width: displayWidth, height: guideHeight},
		{x: resultX, y: resultCompleteY, width: resultWidth, height: resultHeight},
		{x: resultX, y: resultTimeY, width: resultWidth, height: resultHeight},
		{x: resultX, y: resultMissY, width: resultWidth, height: resultHeight},
		{x: resultX, y: resultInstructY, width: resultWidth, height: resultInstructH},
	}
	for _, region := range regions {
		if region.x < 0 || region.y < 0 || region.x+region.width > displayWidth || region.y+region.height > displayHeight {
			t.Errorf("region outside display: %+v", region)
		}
	}
	if inputFrameY+inputFrameHeight > guideY {
		t.Fatalf("input overlaps guide: inputBottom=%d guideY=%d", inputFrameY+inputFrameHeight, guideY)
	}
	if romanHeight != 16 || statusHeight != 24 || promptHeight != 24 || inputHeight != 24 || guideHeight != 24 || promptX != 0 {
		t.Fatal("playing font/layout heights or prompt alignment changed")
	}
	styles := makeStyles(efont16.Font, efont24.Font)
	input, ok := styles.Lookup(styleInput)
	if !ok || input.Foreground != modgadget.ColorWhite || input.Background != modgadget.ColorBlue || input.Font.Metrics().LineHeight() != 24 {
		t.Fatalf("input style=%+v found=%v", input, ok)
	}
	guide, _ := styles.Lookup(styleGuide)
	if guide.Foreground != modgadget.ColorRed || guide.Background != modgadget.ColorBlack || guide.Font.Metrics().LineHeight() != 24 {
		t.Fatalf("guide style=%+v", guide)
	}
	if guideSpeed != 24 || guideGap != 32 {
		t.Fatalf("guide scroll speed=%v gap=%v", guideSpeed, guideGap)
	}
}

func TestPlayingGuidesByCourseAndGlyphCoverage(t *testing.T) {
	tests := []struct {
		id   courseID
		want string
	}{
		{courseJapanese, "DELで強制終了 ◆ Fn+Mで消音 ◆ Fn++/Fn+-で音量調整 ◆ ローマ字で入力して下さい。"},
		{courseEnglish, "DEL: Quit ◆ Fn+M: Mute ◆ Fn++/Fn+-: Volume ◆ Type the shown letters."},
		{courseChinese, "DEL强制结束 ◆ Fn+M静音 ◆ Fn++/Fn+-调节音量 ◆ 请使用拼音输入。"},
		{courseKorean, "DEL로 강제 종료 ◆ Fn+M 음소거 ◆ Fn++/Fn+- 음량 조절 ◆ 로마자로 입력하세요."},
		{courseAll, "DELで強制終了 ◆ Fn+Mで消音 ◆ Fn++/Fn+-で音量調整 ◆ ローマ字で入力して下さい。 ◇ DEL: Quit ◆ Fn+M: Mute ◆ Fn++/Fn+-: Volume ◆ Type the shown letters. ◇ DEL强制结束 ◆ Fn+M静音 ◆ Fn++/Fn+-调节音量 ◆ 请使用拼音输入。 ◇ DEL로 강제 종료 ◆ Fn+M 음소거 ◆ Fn++/Fn+- 음량 조절 ◆ 로마자로 입력하세요."},
	}
	font := efont24.Font
	for _, test := range tests {
		got := playingGuideForCourse(test.id)
		if got != test.want {
			t.Errorf("course %v guide=%q want=%q", test.id, got, test.want)
		}
		for _, r := range got {
			if !font.HasGlyph(r) {
				t.Errorf("course %v guide missing Efont24 rune %q", test.id, r)
			}
		}
	}
	if strings.Contains(englishPlayingGuide, "強制終了") {
		t.Fatal("English guide contains Japanese quit text")
	}
	if strings.Contains(chinesePlayingGuide, "ローマ字で入力") {
		t.Fatal("Chinese guide contains Japanese input text")
	}
	if strings.Contains(koreanPlayingGuide, "音量調整") {
		t.Fatal("Korean guide contains Japanese volume text")
	}
	for _, guide := range []string{japanesePlayingGuide, englishPlayingGuide, chinesePlayingGuide, koreanPlayingGuide} {
		if !strings.Contains(allPlayingGuide, guide) {
			t.Errorf("All Languages guide is missing %q", guide)
		}
	}
	for _, value := range []string{"DEL", "Fn+M", "Fn++", "Fn+-", ":", "。", "◇"} {
		for _, r := range value {
			if !font.HasGlyph(r) {
				t.Errorf("required guide token %q missing rune %q", value, r)
			}
		}
	}
	if got := playingGuideForCourse(courseID(255)); got != "" {
		t.Fatalf("invalid course guide=%q want empty", got)
	}
}

func TestPlayingGuideKeepsLoopingScrollConfiguration(t *testing.T) {
	display := &frameDisplay{color: modgadget.ColorBlack}
	styles := makeStyles(efont16.Font, efont24.Font)
	gadget := modgadget.New(display, modgadget.WithStyles(styles))
	view := gadget.Viewport(modgadget.Bounds(0, 0, 100, 24))
	if err := setGuide(view, englishPlayingGuide); err != nil {
		t.Fatal(err)
	}
	start := time.Unix(900, 0)
	for _, now := range []time.Time{start, start.Add(10 * time.Second), start.Add(11 * time.Second)} {
		gadget.Update(now)
		if err := gadget.Render(); err != nil {
			t.Fatal(err)
		}
	}
	if len(display.rects) != 3 {
		t.Fatalf("looping guide transfers=%d want=3", len(display.rects))
	}
}

func TestInputFrameUsesOneWhiteBorderAroundBlueInterior(t *testing.T) {
	display := &frameDisplay{color: modgadget.ColorBlue}
	var scratch [64]byte
	if err := fillSolidRect(display, inputInteriorBounds, modgadget.ColorBlue, &scratch); err != nil {
		t.Fatal(err)
	}
	display.color = modgadget.ColorWhite
	if err := drawFrame(display, inputFrameBounds, modgadget.ColorWhite, &scratch); err != nil {
		t.Fatal(err)
	}
	if display.badData || len(display.rects) != 5 || display.rects[0] != inputInteriorBounds {
		t.Fatalf("input drawing rects=%v badData=%v", display.rects, display.badData)
	}
}

func TestTypingSoundEffects(t *testing.T) {
	now := time.Unix(500, 0)
	app := newAppState()
	app.showMenu()
	player := &fakePatternPlayer{}
	if _, err := handleKeyWithSoundAt(&app, player, keyDown(modgadget.KeyEnter), now); err != nil {
		t.Fatal(err)
	}
	_, _ = handleKeyWithSoundAt(&app, player, modgadget.KeyEvent{Code: modgadget.KeyEnter, Action: modgadget.KeyUp}, now)
	if _, err := handleKeyWithSoundAt(&app, player, letterEvent(modgadget.KeyA, 'a', modgadget.KeyDown), now); err != nil {
		t.Fatal(err)
	}
	if player.current != audio.PatternWrong || app.play.misses != 1 {
		t.Fatalf("miss pattern=%v misses=%d", player.current, app.play.misses)
	}
	_, _ = handleKeyWithSoundAt(&app, player, letterEvent(modgadget.KeyA, 0, modgadget.KeyUp), now)
	for _, letter := range app.play.currentQuestion().roman {
		code := modgadget.KeyA + modgadget.KeyCode(letter-'a')
		_, _ = handleKeyWithSoundAt(&app, player, letterEvent(code, letter, modgadget.KeyDown), now)
		_, _ = handleKeyWithSoundAt(&app, player, letterEvent(code, 0, modgadget.KeyUp), now)
	}
	if player.current != audio.PatternCorrect || app.play.questionIndex != 1 {
		t.Fatalf("correct pattern=%v question=%d", player.current, app.play.questionIndex)
	}
	app.play.questionIndex, app.play.inputIndex = len(japaneseQuestions)-1, 0
	for _, letter := range app.play.currentQuestion().roman {
		code := modgadget.KeyA + modgadget.KeyCode(letter-'a')
		_, _ = handleKeyWithSoundAt(&app, player, letterEvent(code, letter, modgadget.KeyDown), now.Add(time.Second))
		_, _ = handleKeyWithSoundAt(&app, player, letterEvent(code, 0, modgadget.KeyUp), now.Add(time.Second))
	}
	if player.current != audio.PatternStartup || app.screen != stateResult {
		t.Fatalf("complete pattern=%v screen=%v", player.current, app.screen)
	}
}
