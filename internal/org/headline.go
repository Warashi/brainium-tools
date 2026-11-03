package org

import (
	"regexp"
	"strings"
)

// HeadlineProperties は見出しの PROPERTIES ドロワーを返す。無ければ nil。
// org は見出しの直後か計画行の直後にあるドロワーだけを PROPERTIES ドロ
// ワーとして解析するので、section の要素を順に見ればよい。
func HeadlineProperties(headline *Node) *Node {
	if len(headline.Children) == 0 || headline.Children[0].Type != TypeSection {
		return nil
	}
	for _, n := range headline.Children[0].Children {
		if n.Type == TypePropertyDrawer {
			return n
		}
	}
	return nil
}

var (
	// priorityRe は org-priority-regexp。org は見出しの TODO キーワードの
	// 後から looking-at で探すが、先頭の .*? のために行のどこにあっても
	// 当たり、title はその後ろから始まる。
	priorityRe = regexp.MustCompile(`\[#(?:[A-Z]|[0-9]|[1-5][0-9]|6[0-4])\] ?`)
	// tagGroupRe は org-tag-group-re。
	tagGroupRe = regexp.MustCompile(`[ \t]+:[` + alnumClass() + `_@#%:]+:[ \t]*$`)
	// todoNameRe は TODO キーワードの定義の語から高速選択キーと記録の
	// 指定を除く(org-set-regexps-and-options)。
	todoNameRe = regexp.MustCompile(`^(.*?)(?:\(([^!@/])?.*?\))?$`)
)

// headlineTitles は木の全ての headline の title の object を解析する。
// TODO キーワードはファイルのどこにある #+TODO でも決まるので、木を作っ
// た後に解析する。
func (p *parser) headlineTitles(root *Node) {
	todo := todoKeywords(root)
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Type == TypeHeadline {
			begin, end := p.titleRange(n, todo)
			if begin < end {
				n.Secondary = p.objects(begin, end, standardSet)
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
}

// todoKeywords は #+TODO・#+SEQ_TODO・#+TYP_TODO で定義された TODO キー
// ワードを返す。定義が無ければ org-todo-keywords の既定値を使う。
func todoKeywords(root *Node) []string {
	var names []string
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Type == TypeKeyword && (n.Key == "TODO" || n.Key == "SEQ_TODO" || n.Key == "TYP_TODO") {
			for _, k := range strings.Fields(n.Value) {
				if k == "|" {
					continue
				}
				if name := todoNameRe.FindStringSubmatch(k)[1]; name != "" {
					names = append(names, name)
				}
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	if names == nil {
		return []string{"TODO", "DONE"}
	}
	return names
}

// titleRange は headline の title の object を解析する範囲を返す
// (org-element--headline-parse-title)。
func (p *parser) titleRange(n *Node, todo []string) (int, int) {
	lineEnd := n.Begin + len(p.line(n.Begin))
	skip := func(pos int) int {
		for pos < lineEnd && (p.src[pos] == ' ' || p.src[pos] == '\t') {
			pos++
		}
		return pos
	}
	// keyword は pos から word が続き、その後が空白 1 個か行末かを返す。
	keyword := func(pos int, word string) bool {
		if !strings.HasPrefix(p.src[pos:lineEnd], word) {
			return false
		}
		next := pos + len(word)
		return next == lineEnd || p.src[next] == ' '
	}
	pos := skip(n.Begin + n.Level)
	for _, k := range todo {
		if keyword(pos, k) {
			pos = skip(pos + len(k))
			break
		}
	}
	if m := priorityRe.FindStringIndex(p.src[pos:lineEnd]); m != nil {
		pos += m[1]
	}
	if keyword(pos, "COMMENT") {
		pos = skip(pos + len("COMMENT"))
	}
	titleStart := pos
	for pos > n.Begin && (p.src[pos-1] == ' ' || p.src[pos-1] == '\t') {
		pos--
	}
	titleEnd := lineEnd
	if m := tagGroupRe.FindStringIndex(p.src[pos:lineEnd]); m != nil {
		titleEnd = pos + m[0]
	}
	for titleEnd > n.Begin && (p.src[titleEnd-1] == ' ' || p.src[titleEnd-1] == '\t') {
		titleEnd--
	}
	return skip(titleStart), titleEnd
}
