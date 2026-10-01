// Package outline は org 文書の見出しを、行範囲付きの節の並びとして取り出す。
//
// agent が目次をたどって読む範囲を選べるようにするためのもので、要約は
// 作らない。見出しの文字列と原文の行番号だけを返す。
package outline

import "strings"

// Section は見出し 1 つと、その配下を含む行範囲。行番号は 1 始まりで、
// EndLine は次の同じか浅い見出しの直前、無ければ最終行。
type Section struct {
	Level     int    `json:"level"`
	Title     string `json:"title"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Parse は source の見出しを出現順に返す。ブロックの中の行も見出しとし
// て扱う。org は見出しを跨ぐブロックを認めず、evidence の解析とも揃う。
func Parse(source string) []Section {
	lines := strings.Split(strings.TrimSuffix(source, "\n"), "\n")
	var sections []Section
	// open は閉じていない節の添字。浅い順に積む
	var open []int
	for i, line := range lines {
		level, title, ok := heading(strings.TrimSuffix(line, "\r"))
		if !ok {
			continue
		}
		for len(open) > 0 && sections[open[len(open)-1]].Level >= level {
			sections[open[len(open)-1]].EndLine = i
			open = open[:len(open)-1]
		}
		sections = append(sections, Section{Level: level, Title: title, StartLine: i + 1})
		open = append(open, len(sections)-1)
	}
	for _, j := range open {
		sections[j].EndLine = len(lines)
	}
	return sections
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

func heading(line string) (level int, title string, ok bool) {
	for level < len(line) && line[level] == '*' {
		level++
	}
	if level == 0 || level >= len(line) || (line[level] != ' ' && line[level] != '\t') {
		return 0, "", false
	}
	return level, strings.TrimSpace(line[level:]), true
}
