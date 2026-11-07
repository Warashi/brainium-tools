package orglint

import (
	"slices"

	"github.com/Warashi/brainium-tools/internal/org"
)

// verbatimSkips は行ごとに、コード表記の候補を探さない範囲をバイト単位で
// 昇順に返す。parser がコード表記と解析した範囲と、リンクのうち説明を
// 除いた部分(記法と行き先)。org-element はリンクを先に取るので、行き先
// にある = や ~ はコード表記にならない。
func verbatimSkips(root *org.Node, lines []string) [][][]int {
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
	walkObjects(root, func(n *org.Node) {
		end := n.End - n.PostBlank
		switch {
		case n.Type == org.TypeVerbatim || n.Type == org.TypeCode:
			add(n.Begin, end)
		case n.Type == org.TypeLink && n.ContentsBegin < 0:
			add(n.Begin, end)
		case n.Type == org.TypeLink:
			add(n.Begin, n.ContentsBegin)
			add(n.ContentsEnd, end)
		}
	})
	for _, s := range spans {
		slices.SortFunc(s, func(a, b []int) int { return a[0] - b[0] })
	}
	return spans
}

// danglingVerbatim は line のうち skip の外にあるコード表記の候補を返す。
// 候補は skip の間の区間ごとに探す。行全体で探して skip と重なる候補を
// 捨てると、skip の中から始まる候補が skip の外の開き記号まで飲み込み、
// その先の候補を取りこぼす。
func danglingVerbatim(line string, skip [][]int) [][]int {
	var dangling [][]int
	from := 0
	for _, gap := range append(slices.Clone(skip), []int{len(line), len(line)}) {
		if from < gap[0] {
			for _, m := range verbatimRe.FindAllStringIndex(line[from:gap[0]], -1) {
				dangling = append(dangling, []int{from + m[0], from + m[1]})
			}
		}
		from = max(from, gap[1])
	}
	return dangling
}

// typesetAt は line を parser で解析したとき、[start, end) がちょうど
// コード表記になるかを返す。
func typesetAt(line string, start, end int) bool {
	found := false
	walkObjects(org.Parse(line), func(n *org.Node) {
		if (n.Type == org.TypeVerbatim || n.Type == org.TypeCode) && n.Begin == start && n.End-n.PostBlank == end {
			found = true
		}
	})
	return found
}

// walkObjects は n 以下の節点を、見出しや item tag の object も含めて
// 前順にたどる。
func walkObjects(n *org.Node, f func(*org.Node)) {
	f(n)
	for _, c := range n.Secondary {
		walkObjects(c, f)
	}
	for _, c := range n.Children {
		walkObjects(c, f)
	}
}
