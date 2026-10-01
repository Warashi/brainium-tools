// Package tokenize は BM25 の索引とクエリで共通に使う語の切り出しを行う。
//
// 英数字は空白と記号で切って語単位に、日本語は連続する区間を 1-gram から
// 5-gram まで展開する。日本語を語単位にすると形態素解析器が要るが、n-gram
// なら辞書を持たずに済む。英数字は空白で切れるので n-gram にする理由がない。
package tokenize

import (
	"strings"
	"unicode"
)

// MaxGram は日本語を展開する n-gram の最大長。notesearch-eval が -maxgram で振れる
// よう変数にしてある。上限を伸ばす候補は hold-out で閉じているので、既定は
// 5 のまま動かさない。
var MaxGram = 5

// Tokens は text から語を切り出す。同じ語が複数回現れれば、その回数だけ返る。
func Tokens(text string) []string {
	groups := Groups(text)
	var tokens []string
	for _, group := range groups {
		tokens = append(tokens, group...)
	}
	return tokens
}

// Groups は正規化と境界判定を Tokens と共有し、独立した英数字語または
// 日本語の連続区間ごとに token をまとめて返す。各 group を順に連結した
// 結果は Tokens と同一になる。
func Groups(text string) [][]string {
	var groups [][]string
	var run []rune
	runKind := 0 // 1: Latin/digit word, 2: Japanese run
	flush := func() {
		if len(run) == 0 {
			return
		}
		if runKind == 1 {
			groups = append(groups, []string{string(run)})
		} else {
			groups = append(groups, grams(run))
		}
		run = nil
		runKind = 0
	}

	for _, r := range fold(text) {
		kind := 0
		switch {
		case isWordRune(r):
			kind = 1
			r = unicode.ToLower(r)
		case isJapanese(r):
			kind = 2
		}
		if kind == 0 {
			flush()
			continue
		}
		if runKind != 0 && runKind != kind {
			flush()
		}
		runKind = kind
		run = append(run, r)
	}
	flush()

	return groups
}

// isWordRune は英数字として語にまとめる文字かどうかを返す。
func isWordRune(r rune) bool {
	return unicode.IsDigit(r) || unicode.Is(unicode.Latin, r)
}

// isJapanese は n-gram に展開する文字かどうかを返す。
func isJapanese(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		r == 'ー'
}

// grams は日本語の連続区間を 1-gram から MaxGram まで展開する。
func grams(segment []rune) []string {
	var out []string
	for n := 1; n <= MaxGram && n <= len(segment); n++ {
		for i := 0; i+n <= len(segment); i++ {
			out = append(out, string(segment[i:i+n]))
		}
	}
	return out
}

// fold は全角英数を半角に、半角カナを全角に畳む。
//
// 正規の NFKC は golang.org/x/text に頼ることになり、flake の vendorHash
// を捨てる必要がある。検索の一致に効くのはこの 2 つだけなので、スパイクの
// 間は標準ライブラリだけで済ませる。
func fold(text string) string {
	var b strings.Builder
	b.Grow(len(text))

	var pending rune // 濁点・半濁点の合成を待っている 1 文字
	writePending := func() {
		if pending != 0 {
			b.WriteRune(pending)
			pending = 0
		}
	}

	for _, r := range text {
		// 全角英数は合成の相手にならないので、保留せずその場で書き出す
		if '！' <= r && r <= '～' {
			writePending()
			b.WriteRune(r - 0xFEE0)
			continue
		}

		// 濁点・半濁点は半角形と全角形のどちらでも来るので、畳んだ後に判定する
		c := r
		if '｡' <= r && r <= 'ﾟ' {
			c = halfwidthKatakana[r-'｡']
		}

		switch {
		case c == '゛' && voiced[pending] != 0:
			pending = voiced[pending]
			writePending()
		case c == '゜' && semiVoiced[pending] != 0:
			pending = semiVoiced[pending]
			writePending()
		default:
			writePending()
			pending = c
		}
	}
	writePending()

	return b.String()
}

// halfwidthKatakana は U+FF61 から U+FF9F までの半角形に対応する全角の文字。
// 末尾の 2 つは単独の濁点・半濁点で、直前の文字と合成して 1 文字にする。
var halfwidthKatakana = []rune("。「」、・ヲァィゥェォャュョッーアイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワン゛゜")

var voiced = map[rune]rune{
	'ウ': 'ヴ',
	'カ': 'ガ', 'キ': 'ギ', 'ク': 'グ', 'ケ': 'ゲ', 'コ': 'ゴ',
	'サ': 'ザ', 'シ': 'ジ', 'ス': 'ズ', 'セ': 'ゼ', 'ソ': 'ゾ',
	'タ': 'ダ', 'チ': 'ヂ', 'ツ': 'ヅ', 'テ': 'デ', 'ト': 'ド',
	'ハ': 'バ', 'ヒ': 'ビ', 'フ': 'ブ', 'ヘ': 'ベ', 'ホ': 'ボ',
}

var semiVoiced = map[rune]rune{
	'ハ': 'パ', 'ヒ': 'ピ', 'フ': 'プ', 'ヘ': 'ペ', 'ホ': 'ポ',
}
