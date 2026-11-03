// Package outline は org 文書の見出しを、行範囲付きの節の並びとして取り出す。
//
// agent が目次をたどって読む範囲を選べるようにするためのもので、要約は
// 作らない。見出しの文字列と原文の行番号だけを返す。
package outline

import (
	"strings"

	"github.com/Warashi/brainium-tools/internal/org"
)

// Section は見出し 1 つと、その配下を含む行範囲。行番号は 1 始まりで、
// EndLine は次の同じか浅い見出しの直前、無ければ最終行。
type Section struct {
	Level     int    `json:"level"`
	Title     string `json:"title"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Parse は source の見出しを出現順に返す。見出しの判定と節の範囲は
// org の parser に従う。
func Parse(source string) []Section {
	var sections []Section
	var walk func(n *org.Node)
	walk = func(n *org.Node) {
		for _, c := range n.Children {
			if c.Type != org.TypeHeadline {
				continue
			}
			sections = append(sections, Section{
				Level:     c.Level,
				Title:     strings.TrimSpace(firstLine(source[c.Begin+c.Level:])),
				StartLine: strings.Count(source[:c.Begin], "\n") + 1,
				EndLine:   lastLine(source, c.End),
			})
			walk(c)
		}
	}
	walk(org.Parse(source))
	return sections
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// lastLine は source の end の直前の文字がある行の番号を返す。
func lastLine(source string, end int) int {
	n := strings.Count(source[:end], "\n")
	if end == len(source) && !strings.HasSuffix(source, "\n") {
		n++
	}
	return n
}

// Innermost は line を含む最も深い節の添字を返す。最初の見出しより前な
// ら -1。
func Innermost(sections []Section, line int) int {
	found := -1
	for i, s := range sections {
		if s.StartLine > line {
			break
		}
		if line <= s.EndLine {
			found = i
		}
	}
	return found
}
