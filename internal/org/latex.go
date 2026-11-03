package org

import (
	"regexp"
	"strings"
)

const TypeLatexFragment Type = "latex-fragment"

var latexMacroRe = regexp.MustCompile(`^\\[a-zA-Z]+\*?(?:\[[^\]\[\n{}]*\]|\{[^{}\n]*\})*`)

// latexFragment は pos から始まる LaTeX の断片を解析する
// (org-element-latex-fragment-parser)。
func (s objectSpan) latexFragment(pos int) *Node {
	end := s.latexFragmentEnd(pos)
	if end < 0 {
		return nil
	}
	n := &Node{Type: TypeLatexFragment, Begin: pos, ContentsBegin: -1, ContentsEnd: -1}
	n.PostBlank, n.End = s.skipSpaces(end)
	return n
}

// latexFragmentEnd は pos から始まる断片の終わりを返す。無ければ -1。
func (s objectSpan) latexFragmentEnd(pos int) int {
	text := s.p.src[pos:s.end]
	switch {
	case strings.HasPrefix(text, `\(`):
		return s.after(pos, `\)`)
	case strings.HasPrefix(text, `\[`):
		return s.after(pos, `\]`)
	case text[0] == '\\':
		if m := latexMacroRe.FindStringIndex(text); m != nil {
			return pos + m[1]
		}
		return -1
	case strings.HasPrefix(text, "$$"):
		return s.after(pos+2, "$$")
	}
	// $ 1 つで囲む断片は、中身の両端が空白や句読点でなく、閉じの $ の後
	// が句読点・空白・括弧・引用符か行末のときに限る
	if pos > s.begin && s.p.src[pos-1] == '$' {
		return -1
	}
	if len(text) > 1 && strings.IndexByte(" \t\n,.;", text[1]) >= 0 {
		return -1
	}
	closing := strings.IndexByte(text[1:], '$')
	if closing < 0 {
		return -1
	}
	closing += pos + 1
	if strings.IndexByte(" \t\n,.", s.p.src[closing-1]) >= 0 {
		return -1
	}
	end := closing + 1
	if r, ok := s.runeAt(end); ok && r != '\n' && r != '\'' && !endsLatexFragment(r) {
		return -1
	}
	return end
}

// after は from から最初の needle の直後の位置を返す。無ければ -1。
func (s objectSpan) after(from int, needle string) int {
	i := strings.Index(s.p.src[from:s.end], needle)
	if i < 0 {
		return -1
	}
	return from + i + len(needle)
}
