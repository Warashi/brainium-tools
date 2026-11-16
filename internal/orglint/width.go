package orglint

import (
	"github.com/Warashi/brainium-tools/internal/org"
	"github.com/Warashi/brainium-tools/internal/textwidth"
)

// runeWidths は text の rune ごとの表示幅を返す。org の parser がブラケット
// リンクと解析した部分は、説明があれば説明だけ、なければパスだけを数え、
// 残りの記法を 0 桁とする。コード表記の中は parser がリンクを解析しない
// ので記法もそのまま数える。head は行頭の fill-prefix の rune 数で、
// コメントの # の後ろもリンクを隠すため、その前置きを外して解析する。
func runeWidths(text string, head int) []int {
	rs := []rune(text)
	ws := make([]int, len(rs))
	for i, r := range rs {
		ws[i] = textwidth.Rune(r)
	}
	body := string(rs[head:])
	index := runeIndex(body)
	hide := func(begin, end int) {
		for i := index[begin]; i < index[end]; i++ {
			ws[head+i] = 0
		}
	}
	for _, l := range org.Links(org.Parse(body)) {
		if body[l.Begin] != '[' {
			continue
		}
		end := l.End - l.PostBlank
		if l.ContentsBegin < 0 {
			hide(l.Begin, l.Begin+2)
			hide(end-2, end)
			continue
		}
		hide(l.Begin, l.ContentsBegin)
		hide(l.ContentsEnd, end)
	}
	return ws
}

// runeIndex は s のバイト位置から rune 位置への対応を返す。
func runeIndex(s string) map[int]int {
	index := make(map[int]int, len(s)+1)
	n := 0
	for b := range s {
		index[b] = n
		n++
	}
	index[len(s)] = n
	return index
}
