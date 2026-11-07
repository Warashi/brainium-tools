package orglint

import (
	"slices"

	"github.com/Warashi/brainium-tools/internal/org"
)

// linkPathSpans は行ごとに、リンクのうち説明を除いた部分(記法と行き先)
// の範囲をバイト単位で昇順に返す。org-element はリンクを先に取るので、
// そこにある = や ~ はコード表記にならない。
func linkPathSpans(root *org.Node, lines []string) [][][]int {
	starts := make([]int, len(lines))
	for i, off := 0, 0; i < len(lines); i++ {
		starts[i] = off
		off += len(lines[i]) + 1
	}
	spans := make([][][]int, len(lines))
	add := func(begin, end int) {
		for i := range lines {
			lo, hi := max(begin, starts[i]), min(end, starts[i]+len(lines[i]))
			if lo < hi {
				spans[i] = append(spans[i], []int{lo - starts[i], hi - starts[i]})
			}
		}
	}
	for _, l := range org.Links(root) {
		end := l.End - l.PostBlank
		if l.ContentsBegin < 0 {
			add(l.Begin, end)
			continue
		}
		add(l.Begin, l.ContentsBegin)
		add(l.ContentsEnd, end)
	}
	for _, s := range spans {
		slices.SortFunc(s, func(a, b []int) int { return a[0] - b[0] })
	}
	return spans
}
