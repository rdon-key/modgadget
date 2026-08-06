package main

type question struct {
	prompt string
	roman  string
}

var japaneseQuestions = [...]question{
	{prompt: "こんにちは", roman: "konnitiwa"},
	{prompt: "ありがとう", roman: "arigatou"},
	{prompt: "おはよう", roman: "ohayou"},
	{prompt: "こんばんは", roman: "konbanwa"},
	{prompt: "さようなら", roman: "sayounara"},
	{prompt: "すみません", roman: "sumimasen"},
	{prompt: "おやすみ", roman: "oyasumi"},
	{prompt: "いただきます", roman: "itadakimasu"},
	{prompt: "ごちそうさま", roman: "gotisousama"},
	{prompt: "にほん", roman: "nihon"},
	{prompt: "さくら", roman: "sakura"},
	{prompt: "ふじさん", roman: "fujisan"},
	{prompt: "ねこ", roman: "neko"},
	{prompt: "いぬ", roman: "inu"},
	{prompt: "ことり", roman: "kotori"},
	{prompt: "でんしゃ", roman: "densya"},
	{prompt: "ひこうき", roman: "hikouki"},
	{prompt: "あおぞら", roman: "aozora"},
	{prompt: "たいよう", roman: "taiyou"},
	{prompt: "ほし", roman: "hoshi"},
}

var englishQuestions = [...]question{
	{prompt: "hello", roman: "hello"},
	{prompt: "thanks", roman: "thanks"},
	{prompt: "morning", roman: "morning"},
	{prompt: "evening", roman: "evening"},
	{prompt: "goodbye", roman: "goodbye"},
	{prompt: "sorry", roman: "sorry"},
	{prompt: "sleep", roman: "sleep"},
	{prompt: "japan", roman: "japan"},
	{prompt: "cherry", roman: "cherry"},
	{prompt: "mountain", roman: "mountain"},
	{prompt: "cat", roman: "cat"},
	{prompt: "dog", roman: "dog"},
	{prompt: "bird", roman: "bird"},
	{prompt: "train", roman: "train"},
	{prompt: "airplane", roman: "airplane"},
	{prompt: "sky", roman: "sky"},
	{prompt: "sun", roman: "sun"},
	{prompt: "moon", roman: "moon"},
	{prompt: "star", roman: "star"},
	{prompt: "computer", roman: "computer"},
}

var chineseQuestions = [...]question{
	{prompt: "你好", roman: "nihao"},
	{prompt: "谢谢", roman: "xiexie"},
	{prompt: "早上好", roman: "zaoshanghao"},
	{prompt: "晚上好", roman: "wanshanghao"},
	{prompt: "再见", roman: "zaijian"},
	{prompt: "中国", roman: "zhongguo"},
	{prompt: "北京", roman: "beijing"},
	{prompt: "上海", roman: "shanghai"},
	{prompt: "朋友", roman: "pengyou"},
	{prompt: "老师", roman: "laoshi"},
	{prompt: "学生", roman: "xuesheng"},
	{prompt: "电脑", roman: "diannao"},
	{prompt: "手机", roman: "shouji"},
	{prompt: "天气", roman: "tianqi"},
	{prompt: "太阳", roman: "taiyang"},
	{prompt: "月亮", roman: "yueliang"},
	{prompt: "星星", roman: "xingxing"},
	{prompt: "小猫", roman: "xiaomao"},
	{prompt: "小狗", roman: "xiaogou"},
	{prompt: "飞机", roman: "feiji"},
}

var koreanQuestions = [...]question{
	{prompt: "안녕하세요", roman: "annyeonghaseyo"},
	{prompt: "감사합니다", roman: "gamsahamnida"},
	{prompt: "좋은 아침", roman: "joeunachim"},
	{prompt: "안녕히 가세요", roman: "annyeonghigaseyo"},
	{prompt: "한국", roman: "hanguk"},
	{prompt: "서울", roman: "seoul"},
	{prompt: "친구", roman: "chingu"},
	{prompt: "선생님", roman: "seonsaengnim"},
	{prompt: "학생", roman: "haksaeng"},
	{prompt: "컴퓨터", roman: "keompyuteo"},
	{prompt: "휴대폰", roman: "hyudaepon"},
	{prompt: "날씨", roman: "nalssi"},
	{prompt: "하늘", roman: "haneul"},
	{prompt: "태양", roman: "taeyang"},
	{prompt: "달", roman: "dal"},
	{prompt: "별", roman: "byeol"},
	{prompt: "고양이", roman: "goyangi"},
	{prompt: "강아지", roman: "gangaji"},
	{prompt: "기차", roman: "gicha"},
	{prompt: "비행기", roman: "bihaenggi"},
}

var allLanguagesQuestions = [...]question{
	{prompt: "こんにちは", roman: "konnitiwa"},
	{prompt: "hello", roman: "hello"},
	{prompt: "你好", roman: "nihao"},
	{prompt: "안녕하세요", roman: "annyeonghaseyo"},
	{prompt: "ありがとう", roman: "arigatou"},
	{prompt: "thanks", roman: "thanks"},
	{prompt: "谢谢", roman: "xiexie"},
	{prompt: "감사합니다", roman: "gamsahamnida"},
	{prompt: "にほん", roman: "nihon"},
	{prompt: "japan", roman: "japan"},
	{prompt: "中国", roman: "zhongguo"},
	{prompt: "한국", roman: "hanguk"},
	{prompt: "ねこ", roman: "neko"},
	{prompt: "cat", roman: "cat"},
	{prompt: "小猫", roman: "xiaomao"},
	{prompt: "고양이", roman: "goyangi"},
	{prompt: "ひこうき", roman: "hikouki"},
	{prompt: "airplane", roman: "airplane"},
	{prompt: "飞机", roman: "feiji"},
	{prompt: "비행기", roman: "bihaenggi"},
}

func questionsForCourse(id courseID) []question {
	switch id {
	case courseJapanese:
		return japaneseQuestions[:]
	case courseEnglish:
		return englishQuestions[:]
	case courseChinese:
		return chineseQuestions[:]
	case courseKorean:
		return koreanQuestions[:]
	case courseAll:
		return allLanguagesQuestions[:]
	default:
		return nil
	}
}
