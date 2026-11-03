package org

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const TypeLink Type = "link"

// linkTypes は emacs -Q で org-id を読んだときの org-link-types。
const linkTypes = `bbdb|bibtex|docview|doi|elisp|eww|file\+emacs|file\+sys|file|ftp|gnus|help|https?|id|info|irc|mailto|mhe|news|rmail|shell|shortdoc|w3m`

var (
	// linkBracketRe は org-link-bracket-re。説明は改行も含めて最短で取る。
	linkBracketRe = regexp.MustCompile(`^\[\[(?:[^\]\[\\]|\\(?:\\\\)*[\]\[]|\\+[^\]\[])+\](?:\[((?s:.)+?)\])?\]`)
	// org は型の大文字と小文字を区別しない
	linkAngleRe = regexp.MustCompile(`(?i)^<(?:` + linkTypes + `):[^>\n]*(?:\n[ \t]*[^> \t\n][^>\n]*)*>`)
	linkPlainRe = regexp.MustCompile(`(?i)^(?:` + linkTypes + `):`)
)

// link は pos から始まる link を、括弧・平文・山括弧の順に解析する
// (org-element-link-parser)。radio target への link は扱わない。
func (s objectSpan) link(pos int) *Node {
	text := s.p.src[pos:s.end]
	n := &Node{Type: TypeLink, Begin: pos, ContentsBegin: -1, ContentsEnd: -1}
	var end int
	if m := linkBracketRe.FindStringSubmatchIndex(text); m != nil {
		end = pos + m[1]
		if m[2] >= 0 {
			n.ContentsBegin, n.ContentsEnd = pos+m[2], pos+m[3]
		}
	} else if e := s.plainLinkEnd(pos); e >= 0 {
		end = e
	} else if m := linkAngleRe.FindStringIndex(text); m != nil {
		end = pos + m[1]
	} else {
		return nil
	}
	n.PostBlank, n.End = s.skipSpaces(end)
	return n
}

// plainLinkEnd は pos から始まる平文の link の終わりを返す。無ければ -1。
// org-link-plain-re は、括弧を含まない文字か対になった括弧の並びを最長
// で取り、最後が英数字のような区切りでない文字か / - か括弧の並びになる
// ところまで戻る。
func (s objectSpan) plainLinkEnd(pos int) int {
	if pos > s.begin {
		if prev, _ := utf8.DecodeLastRuneInString(s.p.src[s.begin:pos]); joinsLatin(prev) {
			return -1
		}
	}
	m := linkPlainRe.FindStringIndex(s.p.src[pos:s.end])
	if m == nil {
		return -1
	}
	end := -1
	units := 0
	for i := pos + m[1]; i < s.end; {
		next, final := s.linkPathUnit(i)
		if next < 0 {
			break
		}
		if final && units > 0 {
			end = next
		}
		units++
		i = next
	}
	return end
}

// linkPathUnit は i から始まる path の 1 単位(括弧以外の 1 文字か、対に
// なった括弧の並び)の終わりと、それが path の最後になれるかを返す。単
// 位にならなければ -1。
func (s objectSpan) linkPathUnit(i int) (int, bool) {
	r, size := utf8.DecodeRuneInString(s.p.src[i:s.end])
	if strings.ContainsRune("([<", r) {
		return s.linkParens(i, 2), true
	}
	if !isLinkPathChar(r) {
		return -1, false
	}
	final := r == '/' || r == '-' || (r < utf8.RuneSelf && !isASCIIPunct(r) && !isSpace(r)) ||
		(r >= utf8.RuneSelf && isWord(r))
	return i + size, final
}

// linkParens は i の開き括弧から閉じ括弧までの終わりを返す。括弧は depth
// 段まで入れ子にでき、対にならなければ -1。開きと閉じの括弧の種類は問
// わない。
func (s objectSpan) linkParens(i, depth int) int {
	i++
	for i < s.end {
		r, size := utf8.DecodeRuneInString(s.p.src[i:s.end])
		switch {
		case strings.ContainsRune(")]>", r):
			return i + 1
		case strings.ContainsRune("([<", r):
			if depth == 1 {
				return -1
			}
			if i = s.linkParens(i, depth-1); i < 0 {
				return -1
			}
		case !isLinkPathChar(r):
			return -1
		default:
			i += size
		}
	}
	return -1
}

// isLinkPathChar は括弧の並びの外で 1 文字の単位になれる文字かを返す。
func isLinkPathChar(r rune) bool {
	return !strings.ContainsRune("[] \t\n()<>", r)
}

func isASCIIPunct(r rune) bool {
	return strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", r)
}
