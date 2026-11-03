// Package org は org 文書を、Emacs の org-element と同じ形の木に解析す
// る。種類・範囲・入れ子が org-element と一致することを差分テストで確か
// める。
package org

import (
	"regexp"
	"strings"
)

// Type は節点の種類。org-element の型名と同じ文字列を使う。
type Type string

const (
	TypeOrgData   Type = "org-data"
	TypeHeadline  Type = "headline"
	TypeSection   Type = "section"
	TypeParagraph Type = "paragraph"

	TypeDrawer         Type = "drawer"
	TypePropertyDrawer Type = "property-drawer"
	TypeNodeProperty   Type = "node-property"
	TypeKeyword        Type = "keyword"
	TypePlanning       Type = "planning"
	TypeComment        Type = "comment"
	TypeFixedWidth     Type = "fixed-width"
	TypeHorizontalRule Type = "horizontal-rule"
	TypeTable          Type = "table"
	TypeTableRow       Type = "table-row"

	TypeExampleBlock Type = "example-block"
	TypeSrcBlock     Type = "src-block"
	TypeCommentBlock Type = "comment-block"
	TypeQuoteBlock   Type = "quote-block"
	TypeSpecialBlock Type = "special-block"
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
	// Level は headline の深さ。
	Level int
	// Key と Value は node-property と keyword のキーと値。keyword の Key
	// は org-element と同じく大文字にする。
	Key, Value string
	// LinkType と Path は link の種類と行き先。
	LinkType, Path string
	// Secondary は headline の title と item の tag の object。org-element
	// と同じく Children には入れない。
	Secondary []*Node
	Children  []*Node
}

// Parse は source 全体を解析して org-data の節点を返す。
func Parse(source string) *Node {
	p := parser{src: source, listItems: map[int]listItem{}}
	root := &Node{Type: TypeOrgData, Begin: 0, End: len(source), ContentsBegin: 0, ContentsEnd: len(source)}
	root.Children = p.headlineContents(0, len(source), modeTop)
	p.headlineTitles(root)
	return root
}

type parser struct {
	src string
	// listItems は解析済みのリストの item を開始位置で引く。
	listItems map[int]listItem
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
// m は先頭の section の最初の要素に使う mode。
func (p *parser) headlineContents(begin, end int, m mode) []*Node {
	var nodes []*Node
	if first := p.nextHeadline(begin, end, 0); first > begin {
		nodes = append(nodes, p.section(begin, first, m))
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
	n := &Node{Type: TypeHeadline, Begin: begin, End: end, ContentsBegin: -1, ContentsEnd: -1, Level: p.headlineLevel(begin)}
	blanks, contents := p.skipBlank(p.lineEnd(begin), end)
	if contents == end {
		n.PostBlank = blanks
		return n
	}
	n.ContentsBegin, n.ContentsEnd = contents, end
	// 見出しの行と空行で離れた計画行と PROPERTIES ドロワーは、org では
	// 見出しのものとみなされない
	m := modeNone
	if blanks == 0 {
		m = modePlanning
	}
	n.Children = p.headlineContents(contents, end, m)
	return n
}

func (p *parser) section(begin, end int, m mode) *Node {
	n := &Node{Type: TypeSection, Begin: begin, End: end, ContentsBegin: begin, ContentsEnd: end}
	n.Children = p.elements(begin, end, m)
	return n
}

// mode は次の要素として何を優先して認めるか(org-element の MODE)。
type mode int

const (
	modeNone mode = iota
	// modeTop はファイル先頭の section の先頭で、PROPERTIES ドロワーを
	// 認める。コメントの後も認める。
	modeTop
	// modePlanning は見出しの行の直後で、計画行と PROPERTIES ドロワー
	// を認める。
	modePlanning
	// modePropertyDrawer は計画行の直後で、PROPERTIES ドロワーを認める。
	modePropertyDrawer
)

// elements は [begin, end) の要素を並べる。
func (p *parser) elements(begin, end int, m mode) []*Node {
	var nodes []*Node
	_, pos := p.skipBlank(begin, end)
	for pos < end {
		var n *Node
		if pos > 0 && p.src[pos-1] != '\n' {
			// item の中身の先頭のように行の途中から始まる要素は、org
			// では常に段落になる
			n = p.paragraph(pos, end)
		}
		// org はコメントを計画行と PROPERTIES ドロワーより先に判定する
		if n == nil {
			n = p.lines(TypeComment, commentRe, pos, end)
		}
		if n == nil && m == modePlanning {
			n = p.singleLine(TypePlanning, planningRe, pos, end)
		}
		if n == nil && m != modeNone {
			n = p.propertyDrawer(pos, end)
		}
		if n == nil {
			n = p.element(pos, end)
		}
		nodes = append(nodes, n)
		pos = n.End
		if n.Type == TypePlanning || (m == modeTop && n.Type == TypeComment) {
			m = modePropertyDrawer
		} else {
			m = modeNone
		}
	}
	return nodes
}

// element は pos から始まる要素を 1 つ解析する。pos は空行ではない。
func (p *parser) element(pos, limit int) *Node {
	if n := p.drawer(pos, limit); n != nil {
		return n
	}
	if n := p.lines(TypeFixedWidth, fixedWidthRe, pos, limit); n != nil {
		return n
	}
	if n := p.block(pos, limit); n != nil {
		return n
	}
	if n := p.keyword(pos, limit); n != nil {
		return n
	}
	if n := p.singleLine(TypeHorizontalRule, horizontalRuleRe, pos, limit); n != nil {
		return n
	}
	if n := p.table(pos, limit); n != nil {
		return n
	}
	if n := p.plainList(pos, limit); n != nil {
		return n
	}
	return p.paragraph(pos, limit)
}

// startsElement は pos の行が段落以外の要素を始めるかを返す。段落はそ
// こで終わる(org-element-paragraph-separate に相当)。
func (p *parser) startsElement(pos, limit int) bool {
	line := p.line(pos)
	return commentRe.MatchString(line) || fixedWidthRe.MatchString(line) ||
		keywordRe.MatchString(line) || horizontalRuleRe.MatchString(line) || tableRowRe.MatchString(line) ||
		listSeparateRe.MatchString(line) ||
		p.drawer(pos, limit) != nil || p.block(pos, limit) != nil
}

// finish は中身が contentsEnd で終わる要素に、続く空行を含めた終わりを
// 設定する。
func (p *parser) finish(n *Node, pos, limit int) *Node {
	n.PostBlank, n.End = p.skipBlank(pos, limit)
	return n
}

var (
	drawerBeginRe = regexp.MustCompile(`^[ \t]*:([\p{L}\p{N}\p{M}_-]+):[ \t]*$`)
	drawerEndRe   = regexp.MustCompile(`(?i)^[ \t]*:END:[ \t]*$`)
	propertiesRe  = regexp.MustCompile(`(?i)^[ \t]*:PROPERTIES:[ \t]*$`)
	// nodePropertyRe は org-property-re。1 がキー、2 が値。
	nodePropertyRe = regexp.MustCompile(`^[ \t]*:(\S+):(?:$|[ \t]+(.*?))[ \t]*$`)
	keywordRe      = regexp.MustCompile(`^[ \t]*#\+\S+?:(?: |$)`)
	// keywordKeyRe は org-element-keyword-parser のキーの読み方。キーは
	// 最長で取るので、1 がキー、2 が値になる。
	keywordKeyRe     = regexp.MustCompile(`^[ \t]*#\+(\S*):(.*)$`)
	planningRe       = regexp.MustCompile(`(?i)^[ \t]*(?:CLOSED|DEADLINE|SCHEDULED):`)
	commentRe        = regexp.MustCompile(`^[ \t]*#(?: |$)`)
	fixedWidthRe     = regexp.MustCompile(`^[ \t]*:(?: |$)`)
	horizontalRuleRe = regexp.MustCompile(`^[ \t]*-{5,}[ \t]*$`)
	tableRowRe       = regexp.MustCompile(`^[ \t]*\|`)
	tableRuleRe      = regexp.MustCompile(`^[ \t]*\|-`)
	blockBeginRe     = regexp.MustCompile(`(?i)^[ \t]*#\+BEGIN_(\S+)`)
)

// IsDrawerBegin は line がドロワーの開始行の形かを返す。:END: で閉じて
// いなければ、org はドロワーにせず段落として読む。
func IsDrawerBegin(line string) bool {
	return drawerBeginRe.MatchString(line)
}

// drawerEnd は [pos, limit) で最初の :END: 行の位置を返す。無ければ -1。
func (p *parser) drawerEnd(pos, limit int) int {
	for ; pos < limit; pos = p.lineEnd(pos) {
		if drawerEndRe.MatchString(p.line(pos)) {
			return pos
		}
	}
	return -1
}

// drawer は :NAME: から :END: までを解析する。:END: が limit までに無け
// れば org はドロワーとみなさないので nil を返す。
func (p *parser) drawer(pos, limit int) *Node {
	if !drawerBeginRe.MatchString(p.line(pos)) {
		return nil
	}
	contents := p.lineEnd(pos)
	end := p.drawerEnd(contents, limit)
	if end < 0 {
		return nil
	}
	n := &Node{Type: TypeDrawer, Begin: pos, ContentsBegin: -1, ContentsEnd: -1}
	if contents < end {
		n.ContentsBegin, n.ContentsEnd = contents, end
		n.Children = p.elements(contents, end, modeNone)
	}
	return p.finish(n, p.lineEnd(end), limit)
}

// propertyDrawer は :PROPERTIES: から :END: までを解析する。間の行が
// すべて :KEY: の形でなければ org はプロパティドロワーとみなさない。
func (p *parser) propertyDrawer(pos, limit int) *Node {
	if !propertiesRe.MatchString(p.line(pos)) {
		return nil
	}
	contents := p.lineEnd(pos)
	var props []*Node
	line := contents
	for ; line < limit && !drawerEndRe.MatchString(p.line(line)); line = p.lineEnd(line) {
		m := nodePropertyRe.FindStringSubmatch(p.line(line))
		if m == nil {
			return nil
		}
		props = append(props, &Node{Type: TypeNodeProperty, Begin: line, End: p.lineEnd(line), ContentsBegin: -1, ContentsEnd: -1, Key: m[1], Value: m[2]})
	}
	if line >= limit {
		return nil
	}
	n := &Node{Type: TypePropertyDrawer, Begin: pos, ContentsBegin: -1, ContentsEnd: -1, Children: props}
	if contents < line {
		n.ContentsBegin, n.ContentsEnd = contents, line
	}
	return p.finish(n, p.lineEnd(line), limit)
}

// blockTypes は中身を element として解析しないブロックの種類。名前は大
// 文字で引く。ここに無い名前は special-block。
var blockTypes = map[string]Type{
	"EXAMPLE": TypeExampleBlock,
	"SRC":     TypeSrcBlock,
	"COMMENT": TypeCommentBlock,
}

// block は #+BEGIN_NAME から #+END_NAME までを解析する。#+END_NAME が
// limit までに無ければ org はブロックとみなさないので nil を返す。
func (p *parser) block(pos, limit int) *Node {
	m := blockBeginRe.FindStringSubmatch(p.line(pos))
	if m == nil {
		return nil
	}
	name := strings.ToUpper(m[1])
	contents := p.lineEnd(pos)
	end := contents
	for end < limit && !p.blockEnd(end, name) {
		end = p.lineEnd(end)
	}
	if end >= limit {
		return nil
	}
	n := &Node{Type: TypeQuoteBlock, Begin: pos, ContentsBegin: -1, ContentsEnd: -1}
	if t, ok := blockTypes[name]; ok {
		n.Type = t
		return p.finish(n, p.lineEnd(end), limit)
	}
	if name != "QUOTE" {
		n.Type = TypeSpecialBlock
	}
	if contents < end {
		n.ContentsBegin, n.ContentsEnd = contents, end
		n.Children = p.elements(contents, end, modeNone)
	}
	return p.finish(n, p.lineEnd(end), limit)
}

// blockEnd は pos の行が名前 name のブロックを閉じる #+END_NAME かを返
// す。
func (p *parser) blockEnd(pos int, name string) bool {
	line := strings.TrimLeft(p.line(pos), " \t")
	const prefix = "#+END_"
	if len(line) < len(prefix) || !strings.EqualFold(line[:len(prefix)], prefix) {
		return false
	}
	return strings.EqualFold(strings.TrimRight(line[len(prefix):], " \t"), name)
}

// lines は re に合う連続した行を、コメントや固定幅行のような 1 つの要
// 素 t にする。
func (p *parser) lines(t Type, re *regexp.Regexp, pos, limit int) *Node {
	end := pos
	for end < limit && re.MatchString(p.line(end)) {
		end = p.lineEnd(end)
	}
	if end == pos {
		return nil
	}
	n := &Node{Type: t, Begin: pos, ContentsBegin: -1, ContentsEnd: -1}
	return p.finish(n, end, limit)
}

// singleLine は re に合う 1 行を、キーワードや計画行のような要素 t にす
// る。
func (p *parser) singleLine(t Type, re *regexp.Regexp, pos, limit int) *Node {
	if !re.MatchString(p.line(pos)) {
		return nil
	}
	n := &Node{Type: t, Begin: pos, ContentsBegin: -1, ContentsEnd: -1}
	return p.finish(n, p.lineEnd(pos), limit)
}

// keyword は #+KEY: VALUE の行を解析する。
func (p *parser) keyword(pos, limit int) *Node {
	n := p.singleLine(TypeKeyword, keywordRe, pos, limit)
	if n == nil {
		return nil
	}
	m := keywordKeyRe.FindStringSubmatch(p.line(pos))
	n.Key, n.Value = strings.ToUpper(m[1]), orgTrim(m[2])
	return n
}

// orgTrim は org-trim と同じく、前後の ASCII の空白だけを除く。
func orgTrim(s string) string {
	return strings.Trim(s, " \f\t\n\r\v")
}

// table は | で始まる連続した行を表にする。table.el の表は扱わない。
func (p *parser) table(pos, limit int) *Node {
	var rows []*Node
	end := pos
	for ; end < limit && tableRowRe.MatchString(p.line(end)); end = p.lineEnd(end) {
		rows = append(rows, p.tableRow(end))
	}
	if end == pos {
		return nil
	}
	n := &Node{Type: TypeTable, Begin: pos, ContentsBegin: pos, ContentsEnd: end, Children: rows}
	return p.finish(n, end, limit)
}

// tableRow は 1 行を表の行にする。中身は最初の | の後から、行末の空白
// の前まで。罫線の行は中身を持たない。
func (p *parser) tableRow(pos int) *Node {
	n := &Node{Type: TypeTableRow, Begin: pos, End: p.lineEnd(pos), ContentsBegin: -1, ContentsEnd: -1}
	line := p.line(pos)
	if !tableRuleRe.MatchString(line) {
		n.ContentsBegin = pos + strings.IndexByte(line, '|') + 1
		n.ContentsEnd = pos + len(strings.TrimRight(line, " \t"))
		n.Children = p.objects(n.ContentsBegin, n.ContentsEnd, []Type{TypeTableCell})
	}
	return n
}

func (p *parser) paragraph(begin, limit int) *Node {
	pos := p.lineEnd(begin)
	for pos < limit && !p.blank(pos) && !p.startsElement(pos, limit) {
		pos = p.lineEnd(pos)
	}
	n := &Node{Type: TypeParagraph, Begin: begin, ContentsBegin: begin, ContentsEnd: pos}
	n.Children = p.objects(begin, pos, standardSet)
	return p.finish(n, pos, limit)
}
