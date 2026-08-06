package main

import (
	"strings"
	"testing"
	"time"

	"github.com/rdon-key/modgadget"
	"github.com/rdon-key/modgadget/font/efont16"
	"github.com/rdon-key/modgadget/font/efont24"
)

var expectedEnglish = [...]question{
	{"hello", "hello"}, {"thanks", "thanks"}, {"morning", "morning"}, {"evening", "evening"},
	{"goodbye", "goodbye"}, {"sorry", "sorry"}, {"sleep", "sleep"}, {"japan", "japan"},
	{"cherry", "cherry"}, {"mountain", "mountain"}, {"cat", "cat"}, {"dog", "dog"},
	{"bird", "bird"}, {"train", "train"}, {"airplane", "airplane"}, {"sky", "sky"},
	{"sun", "sun"}, {"moon", "moon"}, {"star", "star"}, {"computer", "computer"},
}

var expectedChinese = [...]question{
	{"你好", "nihao"}, {"谢谢", "xiexie"}, {"早上好", "zaoshanghao"}, {"晚上好", "wanshanghao"},
	{"再见", "zaijian"}, {"中国", "zhongguo"}, {"北京", "beijing"}, {"上海", "shanghai"},
	{"朋友", "pengyou"}, {"老师", "laoshi"}, {"学生", "xuesheng"}, {"电脑", "diannao"},
	{"手机", "shouji"}, {"天气", "tianqi"}, {"太阳", "taiyang"}, {"月亮", "yueliang"},
	{"星星", "xingxing"}, {"小猫", "xiaomao"}, {"小狗", "xiaogou"}, {"飞机", "feiji"},
}

var expectedKorean = [...]question{
	{"안녕하세요", "annyeonghaseyo"}, {"감사합니다", "gamsahamnida"}, {"좋은 아침", "joeunachim"}, {"안녕히 가세요", "annyeonghigaseyo"},
	{"한국", "hanguk"}, {"서울", "seoul"}, {"친구", "chingu"}, {"선생님", "seonsaengnim"},
	{"학생", "haksaeng"}, {"컴퓨터", "keompyuteo"}, {"휴대폰", "hyudaepon"}, {"날씨", "nalssi"},
	{"하늘", "haneul"}, {"태양", "taeyang"}, {"달", "dal"}, {"별", "byeol"},
	{"고양이", "goyangi"}, {"강아지", "gangaji"}, {"기차", "gicha"}, {"비행기", "bihaenggi"},
}

var expectedAllLanguages = [...]question{
	{"こんにちは", "konnitiha"}, {"hello", "hello"}, {"你好", "nihao"}, {"안녕하세요", "annyeonghaseyo"},
	{"ありがとう", "arigatou"}, {"thanks", "thanks"}, {"谢谢", "xiexie"}, {"감사합니다", "gamsahamnida"},
	{"にほん", "nihon"}, {"japan", "japan"}, {"中国", "zhongguo"}, {"한국", "hanguk"},
	{"ねこ", "neko"}, {"cat", "cat"}, {"小猫", "xiaomao"}, {"고양이", "goyangi"},
	{"ひこうき", "hikouki"}, {"airplane", "airplane"}, {"飞机", "feiji"}, {"비행기", "bihaenggi"},
}

func TestCourseQuestionTablesAndRenderingCoverage(t *testing.T) {
	menuFont := efont16.Font
	largeFont := efont24.Font
	promptStyles := modgadget.StyleSet{Default: modgadget.Style{Font: largeFont}}
	tests := []struct {
		id       courseID
		expected []question
	}{
		{courseJapanese, japaneseQuestions[:]},
		{courseEnglish, expectedEnglish[:]},
		{courseChinese, expectedChinese[:]},
		{courseKorean, expectedKorean[:]},
		{courseAll, expectedAllLanguages[:]},
	}
	for _, test := range tests {
		got := questionsForCourse(test.id)
		if len(got) != 20 || len(test.expected) != 20 {
			t.Fatalf("course %v lengths got=%d expected=%d", test.id, len(got), len(test.expected))
		}
		seenPrompts, seenInputs := map[string]bool{}, map[string]bool{}
		for index, item := range got {
			if primaryPrompt(t, item.prompt) != primaryPrompt(t, test.expected[index].prompt) || item.roman != test.expected[index].roman {
				t.Errorf("course %v question %d=%+v want=%+v", test.id, index, item, test.expected[index])
			}
			if item.prompt == "" || item.roman == "" {
				t.Errorf("course %v question %d is empty", test.id, index)
			}
			if seenPrompts[item.prompt] || seenInputs[item.roman] {
				t.Errorf("course %v duplicate question %d=%+v", test.id, index, item)
			}
			seenPrompts[item.prompt], seenInputs[item.roman] = true, true
			if _, err := modgadget.MeasureText(item.prompt, promptStyles); err != nil {
				t.Errorf("course %v prompt %q markup: %v", test.id, item.prompt, err)
			}
			for _, r := range item.roman {
				if r < 'a' || r > 'z' {
					t.Errorf("course %v input %q contains non-lowercase rune %q", test.id, item.roman, r)
				}
				if !menuFont.HasGlyph(r) {
					t.Errorf("course %v input %q missing Efont16 rune %q", test.id, item.roman, r)
				}
				if !largeFont.HasGlyph(r) {
					t.Errorf("course %v input %q missing Efont24 rune %q", test.id, item.roman, r)
				}
			}
			if width := fontWidth(t, menuFont, item.roman); width > romanWidth {
				t.Errorf("course %v input %q Efont16 width=%d > %d", test.id, item.roman, width, romanWidth)
			}
			if width := fontWidth(t, largeFont, item.roman); width > inputWidth {
				t.Errorf("course %v input %q Efont24 width=%d > %d", test.id, item.roman, width, inputWidth)
			}
		}
	}
	if questionsForCourse(courseID(255)) != nil {
		t.Fatal("invalid course returned questions")
	}
}

func primaryPrompt(t *testing.T, value string) string {
	t.Helper()
	rest, ok := strings.CutPrefix(value, "<b>")
	if !ok {
		return value
	}
	end := strings.Index(rest, "</b>")
	if end < 0 {
		t.Fatalf("prompt %q has no closing bold tag", value)
	}
	return rest[:end]
}

func TestAllLanguagesMatchesSourceQuestions(t *testing.T) {
	want := []question{
		japaneseQuestions[0], englishQuestions[0], chineseQuestions[0], koreanQuestions[0],
		japaneseQuestions[1], englishQuestions[1], chineseQuestions[1], koreanQuestions[1],
		japaneseQuestions[9], englishQuestions[7], chineseQuestions[5], koreanQuestions[4],
		japaneseQuestions[12], englishQuestions[10], chineseQuestions[17], koreanQuestions[16],
		japaneseQuestions[16], englishQuestions[14], chineseQuestions[19], koreanQuestions[19],
	}
	for index, item := range allLanguagesQuestions {
		if item != want[index] {
			t.Errorf("all-languages question %d=%+v want source=%+v", index, item, want[index])
		}
	}
}

func TestEveryCourseStartsAdvancesCompletesAndRestarts(t *testing.T) {
	start := time.Unix(600, 0)
	for id := courseJapanese; id <= courseAll; id++ {
		app := newAppState()
		app.showMenu()
		app.selection = int(id)
		if handled, _ := app.handleKeyAt(keyDown(modgadget.KeyEnter), start); !handled || app.screen != statePlaying || app.selected != id {
			t.Fatalf("course %v did not start: %+v", id, app)
		}
		if app.play.course != id || app.play.questionIndex != 0 || app.play.inputIndex != 0 || app.play.misses != 0 || app.play.currentQuestion() != questionsForCourse(id)[0] {
			t.Fatalf("course %v initial play=%+v question=%+v", id, app.play, app.play.currentQuestion())
		}
		_, _ = app.handleKeyAt(modgadget.KeyEvent{Code: modgadget.KeyEnter, Action: modgadget.KeyUp}, start)
		first := app.play.currentQuestion()
		wrong := rune('a')
		if first.roman[0] == 'a' {
			wrong = 'b'
		}
		wrongCode := modgadget.KeyA + modgadget.KeyCode(wrong-'a')
		if handled, effect := app.handleKeyAt(letterEvent(wrongCode, wrong, modgadget.KeyDown), start); !handled || effect != soundMiss || app.play.inputIndex != 0 || app.play.misses != 1 {
			t.Fatalf("course %v miss handled=%v effect=%v play=%+v", id, handled, effect, app.play)
		}
		_, _ = app.handleKeyAt(letterEvent(wrongCode, 0, modgadget.KeyUp), start)
		if got := typeRemaining(t, &app.play, start.Add(time.Second)); got != playNextQuestion || app.play.questionIndex != 1 {
			t.Fatalf("course %v first completion=%v play=%+v", id, got, app.play)
		}
		app.play.questionIndex, app.play.inputIndex = 19, 0
		lastEffect := soundNone
		for _, letter := range app.play.currentQuestion().roman {
			code := modgadget.KeyA + modgadget.KeyCode(letter-'a')
			_, lastEffect = app.handleKeyAt(letterEvent(code, letter, modgadget.KeyDown), start.Add(2*time.Second))
			_, _ = app.handleKeyAt(letterEvent(code, 0, modgadget.KeyUp), start.Add(2*time.Second))
		}
		if lastEffect != soundCourseComplete || app.screen != stateResult || app.play.active {
			t.Fatalf("course %v final effect=%v screen=%v play=%+v", id, lastEffect, app.screen, app.play)
		}
		if handled, _ := app.handleKeyAt(keyDown(modgadget.KeyEnter), start.Add(3*time.Second)); !handled || app.screen != stateMenu || app.selection != int(id) || app.selected != id {
			t.Fatalf("course %v result return=%+v", id, app)
		}
		_, _ = app.handleKeyAt(modgadget.KeyEvent{Code: modgadget.KeyEnter, Action: modgadget.KeyUp}, start.Add(3*time.Second))
		if handled, _ := app.handleKeyAt(keyDown(modgadget.KeyEnter), start.Add(4*time.Second)); !handled || app.play.course != id || app.play.questionIndex != 0 || app.play.inputIndex != 0 || app.play.misses != 0 || !app.play.startedAt.Equal(start.Add(4*time.Second)) {
			t.Fatalf("course %v restart=%+v", id, app.play)
		}
	}
}

func TestEveryCourseDeletePreservesSelection(t *testing.T) {
	start := time.Unix(700, 0)
	for id := courseJapanese; id <= courseAll; id++ {
		app := newAppState()
		app.showMenu()
		app.selection = int(id)
		_, _ = app.handleKeyAt(keyDown(modgadget.KeyEnter), start)
		_, _ = app.handleKeyAt(modgadget.KeyEvent{Code: modgadget.KeyEnter, Action: modgadget.KeyUp}, start)
		app.play.inputIndex, app.play.misses = 1, 2
		if handled, _ := app.handleKeyAt(keyDown(modgadget.KeyDelete), start); !handled || app.screen != stateMenu || app.selection != int(id) || app.selected != id || app.play.active {
			t.Fatalf("course %v delete=%+v", id, app)
		}
	}
}

func TestAllLanguagesCyclesJapaneseEnglishChineseKorean(t *testing.T) {
	play := playState{}
	if !play.start(courseAll, time.Unix(800, 0)) {
		t.Fatal("all-languages did not start")
	}
	want := []string{"こんにちは", "hello", "你好", "안녕하세요"}
	for index, prompt := range want {
		if primaryPrompt(t, play.currentQuestion().prompt) != prompt {
			t.Fatalf("step %d prompt=%q want=%q", index, play.currentQuestion().prompt, prompt)
		}
		if index+1 < len(want) {
			if got := typeRemaining(t, &play, time.Unix(801+int64(index), 0)); got != playNextQuestion {
				t.Fatalf("step %d completion=%v", index, got)
			}
		}
	}
}
