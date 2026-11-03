// Package org は org 文書を、Emacs の org-element と同じ形の木に解析す
// る。種類・範囲・入れ子が org-element と一致することを差分テストで確か
// める。
package org

import "strings"

// Type は節点の種類。org-element の型名と同じ文字列を使う。
type Type string

const (
	TypeOrgData   Type = "org-data"
	TypeHeadline  Type = "headline"
	TypeSection   Type = "section"
	TypeParagraph Type = "paragraph"
)

// Node は木の節点 1 つ。位置は source のバイト位置で、End は直後の空行
// (PostBlank 行)を含む。中身を持たない節点の ContentsBegin と
// ContentsEnd は -1。
type Node struct {
	Type          Type
	Begin, End    int
	ContentsBegin int
	ContentsEnd   int
	PostBlank     int
	Children      []*Node
}

// Parse は source 全体を解析して org-data の節点を返す。
func Parse(source string) *Node {
	p := parser{src: source}
	root := &Node{Type: TypeOrgData, Begin: 0, End: len(source), ContentsBegin: 0, ContentsEnd: len(source)}
	root.Children = p.headlineContents(0, len(source))
	return root
}

type parser struct {
	src string
}

// lineEnd は pos を含む行の、改行を含めた終わりを返す。
func (p *parser) lineEnd(pos int) int {
	if i := strings.IndexByte(p.src[pos:], '\n'); i >= 0 {
		return pos + i + 1
	}
	return len(p.src)
}

func (p *parser) line(pos int) string {
	return strings.TrimSuffix(p.src[pos:p.lineEnd(pos)], "\n")
}

func (p *parser) blank(pos int) bool {
	return strings.TrimSpace(p.line(pos)) == ""
}

// skipBlank は pos から続く空行を飛ばし、飛ばした行数と次の位置を返す。
func (p *parser) skipBlank(pos, limit int) (int, int) {
	n := 0
	for pos < limit && p.blank(pos) {
		pos = p.lineEnd(pos)
		n++
	}
	return n, pos
}

// headlineLevel は pos の行が見出しならその深さを、でなければ 0 を返す。
func (p *parser) headlineLevel(pos int) int {
	line := p.line(pos)
	level := 0
	for level < len(line) && line[level] == '*' {
		level++
	}
	if level == 0 || level >= len(line) || (line[level] != ' ' && line[level] != '\t') {
		return 0
	}
	return level
}

// nextHeadline は [pos, limit) で深さ maxLevel 以下の最初の見出しの位置を
// 返す。maxLevel が 0 なら深さを問わない。無ければ limit。
func (p *parser) nextHeadline(pos, limit, maxLevel int) int {
	for pos < limit {
		if l := p.headlineLevel(pos); l > 0 && (maxLevel == 0 || l <= maxLevel) {
			return pos
		}
		pos = p.lineEnd(pos)
	}
	return limit
}

// headlineContents は見出しの中身(先頭の section と子の見出し)を返す。
func (p *parser) headlineContents(begin, end int) []*Node {
	var nodes []*Node
	if first := p.nextHeadline(begin, end, 0); first > begin {
		nodes = append(nodes, p.section(begin, first))
		begin = first
	}
	for begin < end {
		next := p.nextHeadline(p.lineEnd(begin), end, p.headlineLevel(begin))
		nodes = append(nodes, p.headline(begin, next))
		begin = next
	}
	return nodes
}

func (p *parser) headline(begin, end int) *Node {
	n := &Node{Type: TypeHeadline, Begin: begin, End: end, ContentsBegin: -1, ContentsEnd: -1}
	blanks, contents := p.skipBlank(p.lineEnd(begin), end)
	if contents == end {
		n.PostBlank = blanks
		return n
	}
	n.ContentsBegin, n.ContentsEnd = contents, end
	n.Children = p.headlineContents(contents, end)
	return n
}

func (p *parser) section(begin, end int) *Node {
	n := &Node{Type: TypeSection, Begin: begin, End: end, ContentsBegin: begin, ContentsEnd: end}
	n.Children = p.elements(begin, end)
	return n
}

// elements は section の中の要素を並べる。
func (p *parser) elements(begin, end int) []*Node {
	var nodes []*Node
	_, pos := p.skipBlank(begin, end)
	for pos < end {
		n := p.paragraph(pos, end)
		nodes = append(nodes, n)
		pos = n.End
	}
	return nodes
}

func (p *parser) paragraph(begin, limit int) *Node {
	pos := begin
	for pos < limit && !p.blank(pos) {
		pos = p.lineEnd(pos)
	}
	blanks, end := p.skipBlank(pos, limit)
	return &Node{Type: TypeParagraph, Begin: begin, End: end, ContentsBegin: begin, ContentsEnd: pos, PostBlank: blanks}
}
