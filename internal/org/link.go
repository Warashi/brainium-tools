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
	linkBracketRe = regexp.MustCompile(`^\[\[((?:[^\]\[\\]|\\(?:\\\\)*[\]\[]|\\+[^\]\[])+)\](?:\[((?s:.)+?)\])?\]`)
	// org は型の大文字と小文字を区別しない
	linkAngleRe = regexp.MustCompile(`(?i)^<(?:` + linkTypes + `):[^>\n]*(?:\n[ \t]*[^> \t\n][^>\n]*)*>`)
	linkPlainRe = regexp.MustCompile(`(?i)^(?:` + linkTypes + `):`)
	// linkTypeRe は org-link-types-re。括弧の link の行き先から型を取る。
	linkTypeRe = regexp.MustCompile(`(?i)^(` + linkTypes + `):`)
	// linkEscapeRe は org-link-unescape が半分に減らす \ の並び。
	linkEscapeRe = regexp.MustCompile(`(\\+)(?:$|[\[\]])`)
	// linkNewlineRe は括弧と山括弧の link の行き先の中の改行と字下げ。
	linkNewlineRe = regexp.MustCompile(`[ \t]*\n[ \t]*`)
	// fileTypeRe・searchRe・fileURIRe は file の link の型、path から除く
	// 検索語、file:/// の重なった / を見分ける。org は file の型の大文字
	// と小文字を区別しない
	fileTypeRe = regexp.MustCompile(`(?i)^file(?:\+.+)?$`)
	searchRe   = regexp.MustCompile(`::.*$`)
	fileURIRe  = regexp.MustCompile(`^///*(.:)?/`)
)

// link は pos から始まる link を、括弧・平文・山括弧の順に解析する
// (org-element-link-parser)。radio target への link は扱わない。
func (s objectSpan) link(pos int) *Node {
	text := s.p.src[pos:s.end]
	n := &Node{Type: TypeLink, Begin: pos, ContentsBegin: -1, ContentsEnd: -1}
	var end int
	if m := linkBracketRe.FindStringSubmatchIndex(text); m != nil {
		end = pos + m[1]
		if m[4] >= 0 {
			n.ContentsBegin, n.ContentsEnd = pos+m[4], pos+m[5]
		}
		n.LinkType, n.Path = bracketLinkTarget(text[m[2]:m[3]])
	} else if e := s.plainLinkEnd(pos); e >= 0 {
		end = e
		n.LinkType, n.Path, _ = strings.Cut(s.p.src[pos:end], ":")
	} else if m := linkAngleRe.FindStringIndex(text); m != nil {
		end = pos + m[1]
		var path string
		n.LinkType, path, _ = strings.Cut(text[1:m[1]-1], ":")
		n.Path = linkNewlineRe.ReplaceAllString(path, "")
	} else {
		return nil
	}
	if fileTypeRe.MatchString(n.LinkType) {
		n.LinkType = "file"
		n.Path = fileURIRe.ReplaceAllString(searchRe.ReplaceAllString(n.Path, ""), "$1/")
	}
	n.PostBlank, n.End = s.skipSpaces(end)
	return n
}

// bracketLinkTarget は括弧の link の行き先 raw から型と path を決める
// (org-element-link-parser)。link の略語(#+LINK)は扱わない。
func bracketLinkTarget(raw string) (string, string) {
	raw = linkNewlineRe.ReplaceAllString(raw, " ")
	raw = linkEscapeRe.ReplaceAllStringFunc(raw, func(m string) string {
		backslashes := len(m) - len(strings.TrimLeft(m, "\\"))
		return strings.Repeat("\\", backslashes/2) + m[backslashes:]
	})
	switch {
	case strings.HasPrefix(raw, "/") || raw == "~" || strings.HasPrefix(raw, "~/") ||
		strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../"):
		return "file", raw
	case linkTypeRe.MatchString(raw):
		m := linkTypeRe.FindStringSubmatchIndex(raw)
		return raw[m[2]:m[3]], raw[m[1]:]
	case strings.HasPrefix(raw, "(") && strings.HasSuffix(raw, ")"):
		return "coderef", raw[1 : len(raw)-1]
	case strings.HasPrefix(raw, "#"):
		return "custom-id", raw[1:]
	default:
		return "fuzzy", raw
	}
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

// Links は n 以下の link を文書の順に返す。見出しの title と説明リスト
// の語の link も含める。
func Links(n *Node) []*Node {
	var links []*Node
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Type == TypeLink {
			links = append(links, n)
		}
		for _, c := range n.Secondary {
			walk(c)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return links
}
