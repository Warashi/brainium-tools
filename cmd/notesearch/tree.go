package main

import (
	"github.com/Warashi/brainium-tools/internal/notesearch/outline"
	"github.com/Warashi/brainium-tools/internal/notesearch/presentation"
)

// withOutline は item に、一致した抜粋ごとの位置と第 1 階層の目次、冒頭
// の文を付ける。丸ごと読むほうが安いノートは、notetree <path> と同じ基準
// で畳んだ見出しを KeyItems に出し、目次は出さない。
func withOutline(item *presentation.Item, source string) {
	p := outline.Prune(source, outline.DefaultRouteBytes)
	item.Bytes, item.Tree = p.Bytes, !p.Collapsed
	item.Opening = outline.Opening(source, outline.DefaultOpeningBytes)
	item.KeyItems = p.KeyItems

	nodes := outline.Sized(source)
	sections := make([]outline.Section, len(nodes))
	for i, n := range nodes {
		sections[i] = n.Section
	}
	top := map[int]int{} // nodes の添字 -> item.Outline の添字
	for i, n := range nodes {
		if !p.Collapsed && parentOf(sections, i) < 0 {
			top[i] = len(item.Outline)
			item.Outline = append(item.Outline, presentation.Section{
				Level: n.Level, Title: n.Title, StartLine: n.StartLine, EndLine: n.EndLine, Bytes: n.Bytes,
			})
		}
	}

	for _, snippet := range item.Snippets {
		if snippet.Kind != "match" {
			continue
		}
		m := presentation.MatchedSection{StartLine: snippet.StartLine, EndLine: snippet.EndLine}
		i := outline.Innermost(sections, snippet.StartLine)
		if i >= 0 {
			m.SectionStart, m.SectionEnd, m.Bytes = nodes[i].StartLine, nodes[i].EndLine, nodes[i].Bytes
			for j := i; j >= 0; j = parentOf(sections, j) {
				m.Chain = append([]presentation.Section{{Level: sections[j].Level, Title: sections[j].Title}}, m.Chain...)
				if k, ok := top[j]; ok {
					item.Outline[k].Matched = true
				}
			}
		}
		item.Matched = append(item.Matched, m)
	}
}

// parentOf は sections[i] を配下に含む直近の節の添字を返す。無ければ -1。
func parentOf(sections []outline.Section, i int) int {
	for j := i - 1; j >= 0; j-- {
		if sections[j].Level < sections[i].Level && sections[i].EndLine <= sections[j].EndLine {
			return j
		}
	}
	return -1
}
