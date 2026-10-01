// Package translit はカタカナ語と英単語を、同じ子音の骨格に落として
// 突き合わせられるようにする。
//
// カタカナ外来語は英語の音の転写なので、規則で骨格に落とせば辞書も統計
// も要らない。共起統計が潰れたのは corpus が 41 文書しかないからで、
// 規則は corpus の大きさに依存しない。
package translit

import (
	"strings"
	"unicode"
)

// KatakanaRuns は text の中のカタカナだけの連続区間を返す。
// 1 文字の区間は骨格が短すぎて何にでも当たるので落とす。
func KatakanaRuns(text string) []string {
	var out []string
	var run []rune
	flush := func() {
		if len(run) > 1 {
			out = append(out, string(run))
		}
		run = run[:0]
	}
	for _, r := range text {
		if unicode.Is(unicode.Katakana, r) || r == 'ー' {
			run = append(run, r)
			continue
		}
		flush()
	}
	flush()
	return out
}

// KatakanaSkeleton はカタカナ語をヘボン式ローマ字に直してから骨格にする。
func KatakanaSkeleton(s string) string {
	return skeleton(romaji(s))
}

// LatinSkeleton は英単語をそのまま骨格にする。
func LatinSkeleton(s string) string {
	return skeleton(s)
}

// Distance は 2 つの骨格のレーベンシュタイン距離。
func Distance(a, b string) int {
	x, y := []rune(a), []rune(b)
	prev := make([]int, len(y)+1)
	cur := make([]int, len(y)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(x); i++ {
		cur[0] = i
		for j := 1; j <= len(y); j++ {
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(y)]
}

// skeleton はラテン文字列から子音の骨格を作る。
//
// 二重字を先に 1 つの子音へ畳んでから母音を落とす。ヘボン式の chi / tsu
// / shi は英語側の t / s に対応するので、c や s のまま落とすと元の子音を
// 見失う。
func skeleton(s string) string {
	s = strings.ToLower(s)
	for _, d := range digraphs {
		s = strings.ReplaceAll(s, d.from, d.to)
	}

	var b strings.Builder
	var last rune
	for _, r := range s {
		c, ok := consonants[r]
		if !ok {
			continue
		}
		// 促音や英語の重ね字で子音が 2 つ続くのは表記の都合なので畳む
		if c == last {
			continue
		}
		b.WriteRune(c)
		last = c
	}
	return b.String()
}

var digraphs = []struct{ from, to string }{
	{"ch", "t"}, {"ts", "t"}, {"sh", "s"}, {"ph", "f"}, {"th", "t"},
	{"ck", "k"}, {"qu", "k"}, {"x", "ks"},
}

// consonants は骨格に残す文字と、その畳み先。ここに無い文字 (母音と
// h / w / y) は落とす。字母の違いでしかない対 (c/k・z/s・l/r・v/b・
// j/g) は同じ音として畳む。
var consonants = map[rune]rune{
	'b': 'b', 'v': 'b',
	'c': 'k', 'k': 'k', 'q': 'k',
	'd': 'd',
	'f': 'f',
	'g': 'g', 'j': 'g',
	'l': 'r', 'r': 'r',
	'm': 'm',
	'n': 'n',
	'p': 'p',
	's': 's', 'z': 's',
	't': 't',
}
