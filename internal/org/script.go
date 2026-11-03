package org

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	TypeSubscript   Type = "subscript"
	TypeSuperscript Type = "superscript"
)

// scriptRe は org-match-substring-regexp の _ や ^ の後の部分。{} と ()
// は 3 段まで入れ子にできる。
var scriptRe = regexp.MustCompile(`^(?:\{(` + nested(`{`, `}`) + `)\}|\((?:` + nested(`(`, `)`) + `)\)|\*|[+-]?[` +
	alnumClass() + `.,\\]*[` + alnumClass() + `])`)

func nested(opening, closing string) string {
	o, c := regexp.QuoteMeta(opening), regexp.QuoteMeta(closing)
	other := `[^` + o + c + `]*?`
	return strings.Join([]string{
		other,
		`(?:` + other + o + other + c + `)+` + other,
		`(?:` + other + o + `(?:` + other + o + other + c + `)+` + other + c + `)+` + other,
	}, "|")
}

// script は pos の _ か ^ から始まる下付きか上付きを解析する
// (org-element-subscript-parser と org-element-superscript-parser)。
// 中身は {} の中か、{} が無ければ記号の後ろ全体。
func (s objectSpan) script(pos int, t Type) *Node {
	if s.bol(pos) {
		return nil
	}
	if prev, _ := utf8.DecodeLastRuneInString(s.p.src[s.begin:pos]); isSpace(prev) {
		return nil
	}
	m := scriptRe.FindStringSubmatchIndex(s.p.src[pos+1 : s.end])
	if m == nil {
		return nil
	}
	n := &Node{Type: t, Begin: pos, ContentsBegin: pos + 1 + m[0], ContentsEnd: pos + 1 + m[1]}
	if m[2] >= 0 {
		n.ContentsBegin, n.ContentsEnd = pos+1+m[2], pos+1+m[3]
	}
	n.PostBlank, n.End = s.skipSpaces(pos + 1 + m[1])
	return n
}

// startsSuperscript は ^ の次の文字が、org が上付きを試す文字かを返す
// (org-element--object-regexp)。
func (s objectSpan) startsSuperscript(pos int) bool {
	r, ok := s.runeAt(pos + 1)
	return ok && (strings.ContainsRune("-{(*+.,", r) || inRanges(alnumRanges, r))
}
