package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget"
	audio "github.com/rdon-key/modgadget/audio/cardputeradv"
	"github.com/rdon-key/modgadget/font/efont16"
	"github.com/rdon-key/modgadget/font/efont24"
)

func keyDown(code modgadget.KeyCode) modgadget.KeyEvent {
	return modgadget.KeyEvent{Code: code, Action: modgadget.KeyDown}
}

func TestInitialCourseAndSelectionMovement(t *testing.T) {
	app := newAppState()
	if app.screen != stateSplash || app.currentCourse().id != courseJapanese {
		t.Fatalf("initial state=%v course=%v", app.screen, app.currentCourse().id)
	}
	app.showMenu()
	handled, effect := app.handleKey(keyDown(modgadget.KeyArrowDown))
	if !handled || effect != soundCursorMove || app.currentCourse().id != courseEnglish || app.currentCourse().guide != englishGuide {
		t.Fatalf("down course=%+v", app.currentCourse())
	}
	handled, effect = app.handleKey(keyDown(modgadget.KeyArrowUp))
	if !handled || effect != soundCursorMove || app.currentCourse().id != courseJapanese || app.currentCourse().guide != japaneseGuide {
		t.Fatalf("up course=%+v", app.currentCourse())
	}
}

func TestCourseSelectionWraps(t *testing.T) {
	app := newAppState()
	app.showMenu()
	_, _ = app.handleKey(keyDown(modgadget.KeyArrowUp))
	if app.currentCourse().id != courseAll {
		t.Fatalf("up wrap=%v", app.currentCourse().id)
	}
	_, _ = app.handleKey(keyDown(modgadget.KeyArrowDown))
	if app.currentCourse().id != courseJapanese {
		t.Fatalf("down wrap=%v", app.currentCourse().id)
	}
}

func TestEnterConfirmsCourseAndWaitsForRelease(t *testing.T) {
	app := newAppState()
	app.showMenu()
	_, _ = app.handleKey(keyDown(modgadget.KeyArrowDown))
	player := &fakePatternPlayer{}
	handled, err := handleKeyWithSound(&app, player, keyDown(modgadget.KeyEnter))
	if !handled || err != nil || app.screen != statePlaying || app.selected != courseEnglish || !app.waitingEnterUp {
		t.Fatalf("confirmed=%+v", app)
	}
	handled, effect := app.handleKey(modgadget.KeyEvent{Code: modgadget.KeyEnter, Action: modgadget.KeyUp})
	if !handled || effect != soundNone || app.waitingEnterUp {
		t.Fatalf("Enter release=%+v", app)
	}
	if len(player.patterns) != 1 || player.patterns[0] != audio.PatternCorrect {
		t.Fatalf("confirm patterns=%v", player.patterns)
	}
}

type fakePatternPlayer struct {
	patterns []audio.Pattern
	current  audio.Pattern
	busy     bool
	stops    int
	replaced int
}

func (player *fakePatternPlayer) PlayPattern(pattern audio.Pattern) error {
	if player.busy {
		player.replaced++
	}
	player.patterns = append(player.patterns, pattern)
	player.current = pattern
	player.busy = true
	return nil
}
func (player *fakePatternPlayer) Busy() bool { return player.busy }
func (player *fakePatternPlayer) Stop() error {
	player.stops++
	player.current = 0
	player.busy = true // Real Player remains Busy until finite silence EOF.
	return nil
}
func (player *fakePatternPlayer) complete() { player.current, player.busy = 0, false }

func TestStartupSoundStartsOnceCompletesAndStopsAtMenu(t *testing.T) {
	player := &fakePatternPlayer{}
	sound := startupSound{}
	if err := sound.start(player); err != nil {
		t.Fatal(err)
	}
	if err := sound.start(player); err != nil {
		t.Fatal(err)
	}
	if len(player.patterns) != 1 || player.patterns[0] != audio.PatternStartup {
		t.Fatalf("startup patterns=%v", player.patterns)
	}
	if err := sound.update(player); err != nil || len(player.patterns) != 1 {
		t.Fatalf("busy startup restarted: patterns=%v err=%v", player.patterns, err)
	}
	player.complete()
	if err := sound.update(player); err != nil {
		t.Fatal(err)
	}
	if len(player.patterns) != 2 || player.patterns[1] != audio.PatternClick {
		t.Fatalf("three-tone completion patterns=%v", player.patterns)
	}
	player.complete()
	if err := sound.update(player); err != nil || sound.phase != 3 || player.busy {
		t.Fatalf("completed startup phase=%d busy=%v err=%v", sound.phase, player.busy, err)
	}
	if err := sound.update(player); err != nil || len(player.patterns) != 2 {
		t.Fatalf("completed startup restarted: patterns=%v err=%v", player.patterns, err)
	}

	interruptedPlayer := &fakePatternPlayer{}
	interrupted := startupSound{}
	_ = interrupted.start(interruptedPlayer)
	if err := interrupted.finish(interruptedPlayer); err != nil {
		t.Fatal(err)
	}
	if interruptedPlayer.stops != 1 || !interruptedPlayer.busy || interruptedPlayer.current != 0 || interrupted.phase != 3 {
		t.Fatalf("menu finish stops=%d busy=%v phase=%d", interruptedPlayer.stops, interruptedPlayer.busy, interrupted.phase)
	}
	interruptedPlayer.complete()
	if interruptedPlayer.busy {
		t.Fatal("finite stop silence did not complete")
	}
}

func TestBusySoundReplacementPriority(t *testing.T) {
	app := newAppState()
	app.showMenu()
	player := &fakePatternPlayer{}

	// A second cursor edge replaces the first cursor sound without losing the
	// selection operation or returning an error.
	if handled, err := handleKeyWithSound(&app, player, keyDown(modgadget.KeyArrowDown)); !handled || err != nil {
		t.Fatalf("first cursor handled=%v err=%v", handled, err)
	}
	_, _ = app.handleKey(modgadget.KeyEvent{Code: modgadget.KeyArrowDown, Action: modgadget.KeyUp})
	if handled, err := handleKeyWithSound(&app, player, keyDown(modgadget.KeyArrowDown)); !handled || err != nil {
		t.Fatalf("second cursor handled=%v err=%v", handled, err)
	}
	if player.current != audio.PatternClick || player.replaced != 1 || app.currentCourse().id != courseChinese {
		t.Fatalf("cursor replacement current=%v replaced=%d course=%v", player.current, player.replaced, app.currentCourse().id)
	}

	// Enter has priority because its Correct pattern replaces an in-flight Click.
	if handled, err := handleKeyWithSound(&app, player, keyDown(modgadget.KeyEnter)); !handled || err != nil {
		t.Fatalf("confirm handled=%v err=%v", handled, err)
	}
	if player.current != audio.PatternCorrect || player.replaced != 2 || app.screen != statePlaying {
		t.Fatalf("confirm priority current=%v replaced=%d screen=%v", player.current, player.replaced, app.screen)
	}
	count := len(player.patterns)
	if handled, err := handleKeyWithSound(&app, player, keyDown(modgadget.KeyArrowUp)); handled || err != nil || len(player.patterns) != count {
		t.Fatalf("playing key interrupted confirm: handled=%v patterns=%v err=%v", handled, player.patterns, err)
	}

	// A cursor sound may replace Stop's finite silence submission after splash;
	// the hardware Player waits for the in-flight silence descriptor before
	// submitting the new pattern's first chunk.
	stopped := &fakePatternPlayer{busy: true, current: audio.PatternStartup}
	startup := startupSound{phase: 1}
	if err := startup.finish(stopped); err != nil {
		t.Fatal(err)
	}
	menu := newAppState()
	menu.showMenu()
	if handled, err := handleKeyWithSound(&menu, stopped, keyDown(modgadget.KeyArrowDown)); !handled || err != nil {
		t.Fatalf("cursor after stop handled=%v err=%v", handled, err)
	}
	if stopped.current != audio.PatternClick || stopped.replaced != 1 {
		t.Fatalf("cursor after stop current=%v replaced=%d", stopped.current, stopped.replaced)
	}
}

func TestCursorAndConfirmSoundsOnlyForHandledKeyDown(t *testing.T) {
	app := newAppState()
	app.showMenu()
	player := &fakePatternPlayer{}

	for _, code := range []modgadget.KeyCode{modgadget.KeyArrowUp, modgadget.KeyArrowDown} {
		before := app.selection
		handled, err := handleKeyWithSound(&app, player, keyDown(code))
		if !handled || err != nil || app.selection == before || player.patterns[len(player.patterns)-1] != audio.PatternClick {
			t.Fatalf("cursor code=%v selection=%d patterns=%v err=%v", code, app.selection, player.patterns, err)
		}
	}
	count := len(player.patterns)
	handled, err := handleKeyWithSound(&app, player, keyDown(modgadget.KeyArrowDown))
	if !handled || err != nil || len(player.patterns) != count {
		t.Fatalf("held cursor repeated sound: handled=%v patterns=%v err=%v", handled, player.patterns, err)
	}
	// KeyUp and unrelated keys do not move or start a sound.
	count, selection := len(player.patterns), app.selection
	for _, event := range []modgadget.KeyEvent{
		{Code: modgadget.KeyArrowDown, Action: modgadget.KeyUp},
		keyDown(modgadget.KeyA),
	} {
		handled, err := handleKeyWithSound(&app, player, event)
		if handled || err != nil {
			t.Fatalf("unexpected handled event=%+v err=%v", event, err)
		}
	}
	if len(player.patterns) != count || app.selection != selection {
		t.Fatalf("ignored input changed selection/sound: selection=%d patterns=%v", app.selection, player.patterns)
	}

	handled, err = handleKeyWithSound(&app, player, keyDown(modgadget.KeyEnter))
	if !handled || err != nil || app.screen != statePlaying || player.patterns[len(player.patterns)-1] != audio.PatternCorrect {
		t.Fatalf("confirm screen=%v patterns=%v err=%v", app.screen, player.patterns, err)
	}
	count = len(player.patterns)
	handled, err = handleKeyWithSound(&app, player, keyDown(modgadget.KeyEnter))
	if !handled || err != nil || len(player.patterns) != count {
		t.Fatalf("held Enter repeated sound: handled=%v patterns=%v err=%v", handled, player.patterns, err)
	}
}

func TestAllLanguagesGuideContainsEveryLanguage(t *testing.T) {
	for _, guide := range []string{japaneseGuide, englishGuide, chineseGuide, koreanGuide} {
		if !strings.Contains(allGuide, guide) {
			t.Fatalf("all-language guide is missing %q", guide)
		}
	}
}

func TestGuideTextMatchesSpecification(t *testing.T) {
	want := [...]string{
		"Fn+▲▼で言語を選択して、Enterで開始します。",
		"Use Fn+▲▼ to select a language, then press Enter to start.",
		"使用Fn+▲▼选择语言，然后按Enter键开始。",
		"Fn+▲▼로 언어를 선택하고 Enter를 눌러 시작합니다.",
		"Fn+▲▼で言語を選択して、Enterで開始します。 ◆ Use Fn+▲▼ to select a language, then press Enter to start. ◆ 使用Fn+▲▼选择语言，然后按Enter键开始。 ◆ Fn+▲▼로 언어를 선택하고 Enter를 눌러 시작합니다.",
	}
	for index := range courses {
		if courses[index].guide != want[index] {
			t.Errorf("course %d guide=%q want=%q", index, courses[index].guide, want[index])
		}
	}
	for _, r := range "▲▼" {
		if !efont24.Font.HasGlyph(r) {
			t.Errorf("efont24 is missing guide arrow %q", r)
		}
	}
}

func TestCourseGlyphCoverageAndGuideWidths(t *testing.T) {
	menuText := "Rdon Type 100⇒ Readycourse selected"
	for _, item := range courses {
		menuText += item.menuLabel + item.startLabel
	}
	for _, r := range menuText {
		if !efont16.Font.HasGlyph(r) {
			t.Errorf("efont16 is missing %q", r)
		}
	}
	for _, item := range courses {
		measurement, err := modgadget.MeasureText(item.guide, modgadget.StyleSet{Default: modgadget.Style{Font: efont24.Font}})
		if err != nil {
			t.Errorf("guide %v: %v", item.id, err)
			continue
		}
		width := measurement.Width
		if width <= 240 {
			t.Errorf("guide %v width=%d, want scrolling text", item.id, width)
		}
	}
}

func TestViewportLayoutAndFontMetricsFitDisplay(t *testing.T) {
	menuBottom := menuY + int16(len(courses)-1)*menuStep + menuHeight
	if titleY+titleHeight > menuY || menuBottom > guideY || guideY+guideHeight > displayHeight {
		t.Fatalf("layout titleBottom=%d menuBottom=%d guide=%d..%d displayHeight=%d",
			titleY+titleHeight, menuBottom, guideY, guideY+guideHeight, displayHeight)
	}
	menuMetrics, guideMetrics := efont16.Font.Metrics(), efont24.Font.Metrics()
	if menuMetrics.LineHeight() > menuHeight {
		t.Fatalf("efont16 metrics exceed menu height: %+v", menuMetrics)
	}
	if guideMetrics.LineHeight() > guideHeight {
		t.Fatalf("efont24 metrics exceed guide height: %+v", guideMetrics)
	}
}

func TestTitleCenteringFromMeasuredWidth(t *testing.T) {
	font := efont16.Font
	width, err := textAdvance(font, titleText)
	if err != nil {
		t.Fatal(err)
	}
	width += titleBoldInkExtra
	x := centeredTitleX(width)
	left, right := x, displayWidth-(x+width)
	if x < 0 || x+width > displayWidth || left-right < -1 || left-right > 1 {
		t.Fatalf("title width=%d x=%d margins=%d/%d", width, x, left, right)
	}
	t.Logf("title width=%d x=%d margins=%d/%d", width, x, left, right)
}

func TestTitleMarksOnlyType100Bold(t *testing.T) {
	if titleMarkup != "<style=title>Rdon <b>Type 100</b></style>" {
		t.Fatalf("title markup=%q", titleMarkup)
	}
	if _, err := modgadget.MeasureText(titleMarkup, makeStyles(efont16.Font, efont24.Font)); err != nil {
		t.Fatal(err)
	}
}

func TestMenuFrameContainsRowsWithoutOverlap(t *testing.T) {
	frame := menuFrameBounds
	if frame.x < 0 || frame.y < 0 || frame.x+frame.width > displayWidth || frame.y+frame.height > displayHeight {
		t.Fatalf("frame outside display: %+v", frame)
	}
	if titleY+titleHeight > frame.y || frame.y+frame.height > guideY {
		t.Fatalf("frame overlaps title or guide: frame=%+v", frame)
	}
	var previousBottom int16
	for index := range courses {
		row := uiRect{x: menuX, y: menuY + int16(index)*menuStep, width: menuWidth, height: menuHeight}
		if row.x <= frame.x || row.x+row.width >= frame.x+frame.width || row.y <= frame.y || row.y+row.height >= frame.y+frame.height {
			t.Fatalf("row %d is not inside frame: row=%+v frame=%+v", index, row, frame)
		}
		if index > 0 && row.y < previousBottom {
			t.Fatalf("row %d overlaps prior row: y=%d priorBottom=%d", index, row.y, previousBottom)
		}
		previousBottom = row.y + row.height
	}
}

func TestSeparatedStylesAndGuideRed(t *testing.T) {
	menuFont := efont16.Font
	guideFont := efont24.Font
	styles := makeStyles(menuFont, guideFont)
	guide, ok := styles.Lookup(styleGuide)
	if !ok || guide.Foreground != modgadget.ColorRed || guide.Background != modgadget.ColorBlack || guide.Font.Metrics().LineHeight() != 24 {
		t.Fatalf("guide style=%+v found=%v", guide, ok)
	}
	title, ok := styles.Lookup(styleTitle)
	if !ok || title.Foreground == modgadget.ColorRed || title.Font.Metrics().LineHeight() != 16 {
		t.Fatalf("title style=%+v found=%v", title, ok)
	}
	selected, ok := styles.Lookup(styleSelected)
	if !ok || selected.Foreground == modgadget.ColorRed || styles.Default.Foreground == modgadget.ColorRed {
		t.Fatalf("menu/default styles selected=%+v default=%+v", selected, styles.Default)
	}
	if styles.Default.Foreground != modgadget.ColorWhite {
		t.Fatalf("frame color=%#04x want white", styles.Default.Foreground)
	}
}

type frameDisplay struct {
	rects   []uiRect
	current uiRect
	color   modgadget.Color565
	badData bool
}

func (*frameDisplay) Size() (int16, int16) { return displayWidth, displayHeight }
func (display *frameDisplay) BeginRect(x, y, width, height int16) error {
	display.current = uiRect{x: x, y: y, width: width, height: height}
	display.rects = append(display.rects, display.current)
	return nil
}
func (display *frameDisplay) WritePixels(data []byte) error {
	want := []byte{byte(uint16(display.color) >> 8), byte(display.color)}
	for index := 0; index < len(data); index += 2 {
		if !bytes.Equal(data[index:index+2], want) {
			display.badData = true
		}
	}
	return nil
}
func (*frameDisplay) EndRect() error { return nil }

func TestDrawsOneOuterFrameWithoutRowRules(t *testing.T) {
	display := &frameDisplay{color: modgadget.ColorWhite}
	var scratch [64]byte
	if err := drawFrame(display, menuFrameBounds, display.color, &scratch); err != nil {
		t.Fatal(err)
	}
	want := []uiRect{
		{x: menuFrameX, y: menuFrameY, width: menuFrameWidth, height: 1},
		{x: menuFrameX, y: menuFrameY + menuFrameHeight - 1, width: menuFrameWidth, height: 1},
		{x: menuFrameX, y: menuFrameY + 1, width: 1, height: menuFrameHeight - 2},
		{x: menuFrameX + menuFrameWidth - 1, y: menuFrameY + 1, width: 1, height: menuFrameHeight - 2},
	}
	if len(display.rects) != 4 || display.badData {
		t.Fatalf("frame rects=%v badData=%v", display.rects, display.badData)
	}
	for index := range want {
		if display.rects[index] != want[index] {
			t.Fatalf("edge %d=%+v want=%+v", index, display.rects[index], want[index])
		}
	}
	// Cursor movement changes only selection state; it cannot draw another
	// frame because the state handler has no Display dependency.
	app := newAppState()
	app.showMenu()
	_, _ = app.handleKey(keyDown(modgadget.KeyArrowDown))
	if len(display.rects) != 4 || menuFrameBounds != (uiRect{x: menuFrameX, y: menuFrameY, width: menuFrameWidth, height: menuFrameHeight}) {
		t.Fatalf("cursor move changed frame: rects=%v bounds=%+v", display.rects, menuFrameBounds)
	}
}

func TestMenuShiftFitsDisplayWidth(t *testing.T) {
	if menuX != 20 || menuX+menuWidth >= menuFrameX+menuFrameWidth {
		t.Fatalf("menu bounds x=%d width=%d displayWidth=%d", menuX, menuWidth, displayWidth)
	}
	for _, item := range courses {
		measurement, err := modgadget.MeasureText("⇒ "+item.menuLabel, modgadget.StyleSet{Default: modgadget.Style{Font: efont16.Font}})
		if err != nil {
			t.Fatal(err)
		}
		width := measurement.Width
		if width > menuWidth {
			t.Errorf("menu %q width=%d exceeds viewport width=%d", item.menuLabel, width, menuWidth)
		}
		t.Logf("menu %q width=%d right=%d", item.menuLabel, width, menuX+width)
	}
}
