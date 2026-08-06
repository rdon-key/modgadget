package main

type question struct {
	prompt string
	roman  string
}

var japaneseQuestions = [...]question{
	{prompt: "<b>こんにちは</b>(hello)", roman: "konnitiha"},
	{prompt: "<b>ありがとう</b>(thanks)", roman: "arigatou"},
	{prompt: "<b>おはよう</b>(morning)", roman: "ohayou"},
	{prompt: "<b>こんばんは</b>(evening)", roman: "konbanha"},
	{prompt: "<b>さようなら</b>(goodbye)", roman: "sayounara"},
	{prompt: "<b>すみません</b>(sorry)", roman: "sumimasen"},
	{prompt: "<b>おやすみ</b>(night)", roman: "oyasumi"},
	{prompt: "<b>いただきます</b>(meal)", roman: "itadakimasu"},
	{prompt: "<b>ごちそうさま</b>(meal)", roman: "gotisousama"},
	{prompt: "<b>にほん</b>(Japan)", roman: "nihon"},
	{prompt: "<b>さくら</b>(cherry)", roman: "sakura"},
	{prompt: "<b>ふじさん</b>(Fuji)", roman: "fujisan"},
	{prompt: "<b>ねこ</b>(cat)", roman: "neko"},
	{prompt: "<b>いぬ</b>(dog)", roman: "inu"},
	{prompt: "<b>とり</b>(bird)", roman: "tori"},
	{prompt: "<b>でんしゃ</b>(train)", roman: "densya"},
	{prompt: "<b>ひこうき</b>(plane)", roman: "hikouki"},
	{prompt: "<b>あおぞら</b>(sky)", roman: "aozora"},
	{prompt: "<b>たいよう</b>(sun)", roman: "taiyou"},
	{prompt: "<b>ほし</b>(star)", roman: "hoshi"},
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
	{prompt: "<b>你好</b>(hello)", roman: "nihao"},
	{prompt: "<b>谢谢</b>(thanks)", roman: "xiexie"},
	{prompt: "<b>早上好</b>(morning)", roman: "zaoshanghao"},
	{prompt: "<b>晚上好</b>(evening)", roman: "wanshanghao"},
	{prompt: "<b>再见</b>(goodbye)", roman: "zaijian"},
	{prompt: "<b>中国</b>(China)", roman: "zhongguo"},
	{prompt: "<b>北京</b>(Beijing)", roman: "beijing"},
	{prompt: "<b>上海</b>(Shanghai)", roman: "shanghai"},
	{prompt: "<b>朋友</b>(friend)", roman: "pengyou"},
	{prompt: "<b>老师</b>(teacher)", roman: "laoshi"},
	{prompt: "<b>学生</b>(student)", roman: "xuesheng"},
	{prompt: "<b>电脑</b>(computer)", roman: "diannao"},
	{prompt: "<b>手机</b>(phone)", roman: "shouji"},
	{prompt: "<b>天气</b>(weather)", roman: "tianqi"},
	{prompt: "<b>太阳</b>(sun)", roman: "taiyang"},
	{prompt: "<b>月亮</b>(moon)", roman: "yueliang"},
	{prompt: "<b>星星</b>(stars)", roman: "xingxing"},
	{prompt: "<b>小猫</b>(kitten)", roman: "xiaomao"},
	{prompt: "<b>小狗</b>(puppy)", roman: "xiaogou"},
	{prompt: "<b>飞机</b>(plane)", roman: "feiji"},
}

var koreanQuestions = [...]question{
	{prompt: "<b>안녕하세요</b>(hello)", roman: "annyeonghaseyo"},
	{prompt: "<b>감사합니다</b>(thanks)", roman: "gamsahamnida"},
	{prompt: "<b>좋은 아침</b>(morning)", roman: "joeunachim"},
	{prompt: "<b>안녕히 가세요</b>(bye)", roman: "annyeonghigaseyo"},
	{prompt: "<b>한국</b>(Korea)", roman: "hanguk"},
	{prompt: "<b>서울</b>(Seoul)", roman: "seoul"},
	{prompt: "<b>친구</b>(friend)", roman: "chingu"},
	{prompt: "<b>선생님</b>(teacher)", roman: "seonsaengnim"},
	{prompt: "<b>학생</b>(student)", roman: "haksaeng"},
	{prompt: "<b>컴퓨터</b>(computer)", roman: "keompyuteo"},
	{prompt: "<b>휴대폰</b>(phone)", roman: "hyudaepon"},
	{prompt: "<b>날씨</b>(weather)", roman: "nalssi"},
	{prompt: "<b>하늘</b>(sky)", roman: "haneul"},
	{prompt: "<b>태양</b>(sun)", roman: "taeyang"},
	{prompt: "<b>달</b>(moon)", roman: "dal"},
	{prompt: "<b>별</b>(star)", roman: "byeol"},
	{prompt: "<b>고양이</b>(cat)", roman: "goyangi"},
	{prompt: "<b>강아지</b>(dog)", roman: "gangaji"},
	{prompt: "<b>기차</b>(train)", roman: "gicha"},
	{prompt: "<b>비행기</b>(plane)", roman: "bihaenggi"},
}

var allLanguagesQuestions = [...]question{
	{prompt: "<b>こんにちは</b>(hello)", roman: "konnitiha"},
	{prompt: "hello", roman: "hello"},
	{prompt: "<b>你好</b>(hello)", roman: "nihao"},
	{prompt: "<b>안녕하세요</b>(hello)", roman: "annyeonghaseyo"},
	{prompt: "<b>ありがとう</b>(thanks)", roman: "arigatou"},
	{prompt: "thanks", roman: "thanks"},
	{prompt: "<b>谢谢</b>(thanks)", roman: "xiexie"},
	{prompt: "<b>감사합니다</b>(thanks)", roman: "gamsahamnida"},
	{prompt: "<b>にほん</b>(Japan)", roman: "nihon"},
	{prompt: "japan", roman: "japan"},
	{prompt: "<b>中国</b>(China)", roman: "zhongguo"},
	{prompt: "<b>한국</b>(Korea)", roman: "hanguk"},
	{prompt: "<b>ねこ</b>(cat)", roman: "neko"},
	{prompt: "cat", roman: "cat"},
	{prompt: "<b>小猫</b>(kitten)", roman: "xiaomao"},
	{prompt: "<b>고양이</b>(cat)", roman: "goyangi"},
	{prompt: "<b>ひこうき</b>(plane)", roman: "hikouki"},
	{prompt: "airplane", roman: "airplane"},
	{prompt: "<b>飞机</b>(plane)", roman: "feiji"},
	{prompt: "<b>비행기</b>(plane)", roman: "bihaenggi"},
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
