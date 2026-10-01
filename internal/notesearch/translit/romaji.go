package translit

import "strings"

// romaji はカタカナ語をヘボン式のローマ字にする。長音符と促音は落とす。
// 骨格を作る前段なので、母音の長短は結果に影響しない。
func romaji(s string) string {
	var b strings.Builder
	r := []rune(s)
	for i := 0; i < len(r); {
		if i+1 < len(r) {
			if v, ok := youon[string(r[i:i+2])]; ok {
				b.WriteString(v)
				i += 2
				continue
			}
		}
		if v, ok := kana[r[i]]; ok {
			b.WriteString(v)
		}
		i++
	}
	return b.String()
}

// youon は拗音と外来語の音。単独のカナより先に引く。
var youon = map[string]string{
	"キャ": "kya", "キュ": "kyu", "キョ": "kyo",
	"ギャ": "gya", "ギュ": "gyu", "ギョ": "gyo",
	"シャ": "sha", "シュ": "shu", "ショ": "sho", "シェ": "she",
	"ジャ": "ja", "ジュ": "ju", "ジョ": "jo", "ジェ": "je",
	"チャ": "cha", "チュ": "chu", "チョ": "cho", "チェ": "che",
	"ニャ": "nya", "ニュ": "nyu", "ニョ": "nyo",
	"ヒャ": "hya", "ヒュ": "hyu", "ヒョ": "hyo",
	"ビャ": "bya", "ビュ": "byu", "ビョ": "byo",
	"ピャ": "pya", "ピュ": "pyu", "ピョ": "pyo",
	"ミャ": "mya", "ミュ": "myu", "ミョ": "myo",
	"リャ": "rya", "リュ": "ryu", "リョ": "ryo",
	"ファ": "fa", "フィ": "fi", "フェ": "fe", "フォ": "fo", "フュ": "fyu",
	"ヴァ": "va", "ヴィ": "vi", "ヴェ": "ve", "ヴォ": "vo",
	"ティ": "ti", "ディ": "di", "トゥ": "tu", "ドゥ": "du",
	"ウィ": "wi", "ウェ": "we", "ウォ": "wo",
	"ツァ": "tsa", "ツィ": "tsi", "ツェ": "tse", "ツォ": "tso",
}

var kana = map[rune]string{
	'ア': "a", 'イ': "i", 'ウ': "u", 'エ': "e", 'オ': "o",
	'カ': "ka", 'キ': "ki", 'ク': "ku", 'ケ': "ke", 'コ': "ko",
	'ガ': "ga", 'ギ': "gi", 'グ': "gu", 'ゲ': "ge", 'ゴ': "go",
	'サ': "sa", 'シ': "shi", 'ス': "su", 'セ': "se", 'ソ': "so",
	'ザ': "za", 'ジ': "ji", 'ズ': "zu", 'ゼ': "ze", 'ゾ': "zo",
	'タ': "ta", 'チ': "chi", 'ツ': "tsu", 'テ': "te", 'ト': "to",
	'ダ': "da", 'ヂ': "ji", 'ヅ': "zu", 'デ': "de", 'ド': "do",
	'ナ': "na", 'ニ': "ni", 'ヌ': "nu", 'ネ': "ne", 'ノ': "no",
	'ハ': "ha", 'ヒ': "hi", 'フ': "fu", 'ヘ': "he", 'ホ': "ho",
	'バ': "ba", 'ビ': "bi", 'ブ': "bu", 'ベ': "be", 'ボ': "bo",
	'パ': "pa", 'ピ': "pi", 'プ': "pu", 'ペ': "pe", 'ポ': "po",
	'マ': "ma", 'ミ': "mi", 'ム': "mu", 'メ': "me", 'モ': "mo",
	'ヤ': "ya", 'ユ': "yu", 'ヨ': "yo",
	'ラ': "ra", 'リ': "ri", 'ル': "ru", 'レ': "re", 'ロ': "ro",
	'ワ': "wa", 'ヲ': "o", 'ン': "n", 'ヴ': "vu",
	'ァ': "a", 'ィ': "i", 'ゥ': "u", 'ェ': "e", 'ォ': "o",
	'ャ': "ya", 'ュ': "yu", 'ョ': "yo",
}
