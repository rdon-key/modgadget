//go:build tinygo

package main

import (
	"time"

	"github.com/rdon-key/modgadget"
	board "github.com/rdon-key/modgadget/device/cardputeradv"
	"github.com/rdon-key/modgadget/font/efont16"
	"github.com/rdon-key/modgadget/font/efont24"
)

const (
	splashDuration = 1500 * time.Millisecond
)

type menuViews struct {
	title *modgadget.Viewport
	menu  [len(courses)]*modgadget.Viewport
	guide *modgadget.Viewport
}

type playViews struct {
	time   *modgadget.Viewport
	number *modgadget.Viewport
	roman  *modgadget.Viewport
	prompt *modgadget.Viewport
	guide  *modgadget.Viewport
	input  *modgadget.Viewport
}

type resultViews struct {
	complete *modgadget.Viewport
	time     *modgadget.Viewport
	misses   *modgadget.Viewport
	instruct *modgadget.Viewport
}

func main() {
	time.Sleep(3 * time.Second)
	panel, err := board.ConfigureDisplay()
	if err != nil {
		panic(err)
	}
	player, err := board.ConfigureAudio()
	if err != nil {
		panic(err)
	}
	// Configure the keyboard last because audio and keyboard share I2C0.
	keyboard, err := board.ConfigureKeyboard()
	if err != nil {
		panic(err)
	}

	menuFont := efont16.Font
	largeFont := efont24.Font
	styles := makeStyles(menuFont, largeFont)
	inputStyles := makeInputStyles(largeFont)
	titleWidth := mustTextAdvance(menuFont, titleText) + titleBoldInkExtra
	numberWidth := mustTextAdvance(largeFont, "00/00")

	app := newAppState()
	startup := startupSound{}
	var gadget *modgadget.Gadget
	var inputGadget *modgadget.Gadget
	var menu menuViews
	var play playViews
	var result resultViews
	var frameScratch [64]byte
	var loopNow time.Time

	newGadget := func() *modgadget.Gadget {
		return modgadget.New(panel,
			modgadget.WithStyles(styles),
			modgadget.WithKeyboard(keyboard),
			modgadget.WithVolumeController(player),
		)
	}
	installHandler := func(target *modgadget.Gadget) {
		target.OnKey(func(event modgadget.KeyEvent) bool {
			handled, err := handleKeyWithSoundAt(&app, player, event, loopNow)
			if err != nil {
				panic(err)
			}
			return handled
		})
	}

	gadget = newGadget()
	installHandler(gadget)
	menu = newMenuViews(gadget, titleWidth)
	if err := gadget.Clear(); err != nil {
		panic(err)
	}
	if err := renderSplash(menu); err != nil {
		panic(err)
	}
	if err := startup.start(player); err != nil {
		panic(err)
	}

	renderedScreen := stateSplash
	renderedSelection := app.selection
	renderedQuestion, renderedInput := -1, -1
	renderedTenths := int64(-1)
	splashUntil := time.Now().Add(splashDuration)

	for {
		if err := player.Update(); err != nil {
			_ = player.Stop()
			panic(err)
		}
		now := time.Now()
		if app.screen == stateSplash {
			if err := startup.update(player); err != nil {
				panic(err)
			}
			if !now.Before(splashUntil) {
				if err := startup.finish(player); err != nil {
					panic(err)
				}
				app.showMenu()
			}
		}

		loopNow = now
		gadget.Update(now)
		if err := keyboard.Err(); err != nil {
			panic(err)
		}

		if app.screen != renderedScreen {
			gadget = newGadget()
			installHandler(gadget)
			inputGadget = nil
			if err := gadget.Clear(); err != nil {
				panic(err)
			}
			switch app.screen {
			case stateMenu:
				menu = newMenuViews(gadget, titleWidth)
				if err := drawFrame(panel, menuFrameBounds, styles.Default.Foreground, &frameScratch); err != nil {
					panic(err)
				}
				if err := renderMenu(menu, &app); err != nil {
					panic(err)
				}
				renderedSelection = app.selection
			case statePlaying:
				play, inputGadget = newPlayViews(gadget, panel, inputStyles, numberWidth)
				if err := fillSolidRect(panel, inputInteriorBounds, modgadget.ColorBlue, &frameScratch); err != nil {
					panic(err)
				}
				if err := drawFrame(panel, inputFrameBounds, modgadget.ColorWhite, &frameScratch); err != nil {
					panic(err)
				}
				if err := renderPlaying(play, &app.play, now); err != nil {
					panic(err)
				}
				renderedQuestion = app.play.questionIndex
				renderedInput = app.play.inputIndex
				renderedTenths = app.play.elapsedTenths(now)
			case stateResult:
				result = newResultViews(gadget)
				if err := renderResult(result, &app.play); err != nil {
					panic(err)
				}
			}
			renderedScreen = app.screen
		} else {
			switch app.screen {
			case stateMenu:
				if app.selection != renderedSelection {
					if err := renderMenu(menu, &app); err != nil {
						panic(err)
					}
					renderedSelection = app.selection
				}
			case statePlaying:
				if app.play.questionIndex != renderedQuestion {
					if err := renderQuestion(play, &app.play); err != nil {
						panic(err)
					}
					renderedQuestion = app.play.questionIndex
					renderedInput = app.play.inputIndex
				} else if app.play.inputIndex != renderedInput {
					if err := renderTyped(play.input, app.play.typed()); err != nil {
						panic(err)
					}
					renderedInput = app.play.inputIndex
				}
				tenths := app.play.elapsedTenths(now)
				if tenths != renderedTenths {
					if err := renderElapsed(play.time, tenths); err != nil {
						panic(err)
					}
					renderedTenths = tenths
				}
			}
		}

		if err := gadget.Render(); err != nil {
			panic(err)
		}
		if inputGadget != nil {
			if err := inputGadget.Render(); err != nil {
				panic(err)
			}
		}
		time.Sleep(time.Millisecond)
	}
}

func mustTextAdvance(font modgadget.Font, value string) int16 {
	width, err := textAdvance(font, value)
	if err != nil {
		panic(err)
	}
	return width
}

func newMenuViews(gadget *modgadget.Gadget, titleWidth int16) menuViews {
	views := menuViews{
		title: gadget.Viewport(modgadget.Bounds(centeredTitleX(titleWidth), titleY, titleWidth, titleHeight)),
		guide: gadget.Viewport(modgadget.Bounds(0, guideY, displayWidth, guideHeight)),
	}
	for index := range views.menu {
		views.menu[index] = gadget.Viewport(modgadget.Bounds(menuX, menuY+int16(index)*menuStep, menuWidth, menuHeight))
	}
	return views
}

func newPlayViews(gadget *modgadget.Gadget, panel modgadget.Display, inputStyles modgadget.StyleSet, numberWidth int16) (playViews, *modgadget.Gadget) {
	inputGadget := modgadget.New(panel, modgadget.WithStyles(inputStyles))
	return playViews{
		time:   gadget.Viewport(modgadget.Bounds(statusTimeX, statusTimeY, statusTimeWidth, statusHeight)),
		number: gadget.Viewport(modgadget.Bounds(displayWidth-numberWidth, statusNumberY, numberWidth, statusHeight)),
		roman:  gadget.Viewport(modgadget.Bounds(romanX, romanY, romanWidth, romanHeight)),
		prompt: gadget.Viewport(modgadget.Bounds(promptX, promptY, promptWidth, promptHeight)),
		guide:  gadget.Viewport(modgadget.Bounds(0, guideY, displayWidth, guideHeight)),
		input:  inputGadget.Viewport(modgadget.Bounds(inputX, inputY, inputWidth, inputHeight)),
	}, inputGadget
}

func newResultViews(gadget *modgadget.Gadget) resultViews {
	return resultViews{
		complete: gadget.Viewport(modgadget.Bounds(resultX, resultCompleteY, resultWidth, resultHeight)),
		time:     gadget.Viewport(modgadget.Bounds(resultX, resultTimeY, resultWidth, resultHeight)),
		misses:   gadget.Viewport(modgadget.Bounds(resultX, resultMissY, resultWidth, resultHeight)),
		instruct: gadget.Viewport(modgadget.Bounds(resultX, resultInstructY, resultWidth, resultInstructH)),
	}
}

func renderSplash(views menuViews) error {
	if err := views.title.SetText(titleMarkup); err != nil {
		return err
	}
	for _, view := range views.menu {
		view.Clear()
	}
	views.guide.SetHorizontalScroll()
	views.guide.Clear()
	return nil
}

func renderMenu(views menuViews, app *appState) error {
	if err := views.title.SetText(titleMarkup); err != nil {
		return err
	}
	for index, item := range courses {
		value := "  " + item.menuLabel
		if index == app.selection {
			value = "<style=" + styleSelected + ">⇒ " + item.menuLabel + "</style>"
		}
		if err := views.menu[index].SetText(value); err != nil {
			return err
		}
	}
	return setGuide(views.guide, app.currentCourse().guide)
}

func renderPlaying(views playViews, play *playState, now time.Time) error {
	if err := renderElapsed(views.time, play.elapsedTenths(now)); err != nil {
		return err
	}
	if err := renderQuestion(views, play); err != nil {
		return err
	}
	return setGuide(views.guide, playingGuideForCourse(play.course))
}

func renderElapsed(view *modgadget.Viewport, tenths int64) error {
	return view.SetText("<style=" + styleLarge + ">" + formatTenths(tenths) + "</style>")
}

func renderQuestion(views playViews, play *playState) error {
	current := play.currentQuestion()
	if err := views.number.SetText("<style=" + styleLarge + ">" + formatQuestionNumber(play.questionIndex) + "</style>"); err != nil {
		return err
	}
	if err := views.roman.SetText(current.roman); err != nil {
		return err
	}
	if err := views.prompt.SetText("<style=" + styleLarge + ">" + current.prompt + "</style>"); err != nil {
		return err
	}
	return renderTyped(views.input, play.typed())
}

func renderTyped(view *modgadget.Viewport, typed string) error {
	if typed == "" {
		typed = " "
	}
	return view.SetText("<style=" + styleInput + ">" + typed + "</style>")
}

func renderResult(views resultViews, play *playState) error {
	if err := views.complete.SetText("<style=" + styleLarge + ">COMPLETE!</style>"); err != nil {
		return err
	}
	if err := views.time.SetText("<style=" + styleLarge + ">" + resultTimeText(play) + "</style>"); err != nil {
		return err
	}
	if err := views.misses.SetText("<style=" + styleLarge + ">" + resultMissText(play) + "</style>"); err != nil {
		return err
	}
	return views.instruct.SetText("Press Enter")
}
