package org

import (
	"regexp"
	"strings"
)

const (
	TypePlainList Type = "plain-list"
	TypeItem      Type = "item"
)

var (
	itemRe     = regexp.MustCompile(`^(?:[ \t]*(?:[-+]|[0-9]+[.)])|[ \t]+\*)(?:[ \t]+|$)`)
	fullItemRe = regexp.MustCompile(`(?i)^[ \t]*((?:[-+*]|(?:[0-9]+|[A-Za-z])[.)])(?:[ \t]+|$))` +
		`(?:\[@(?:start:)?(?:[0-9]+|[A-Za-z])\][ \t]*)?` +
		`(?:\[[ X-]\](?:[ \t]+|$))?` +
		`(?:(.*)[ \t]+::(?:[ \t]+|$))?`)
	// listSeparateRe は段落を終わらせる箇条書きの行
	// (org-element-paragraph-separate のリストの部分)。
	listSeparateRe = regexp.MustCompile(`^[ \t]*(?:[-+*]|[0-9]+[.)])(?:[ \t]|$)`)
)

// listItem は org-element--list-struct の 1 項目。
type listItem struct {
	indent int
	end    int
}

// plainList は pos から始まるリストを解析する。item の終わりは、最初の
// リストで計算した構造を入れ子のリストでも使う。入れ子の内側だけで計算
// し直すと、子の item の後の空行の扱いが org とずれる。
func (p *parser) plainList(pos, limit int) *Node {
	if !itemRe.MatchString(p.line(pos)) {
		return nil
	}
	if _, ok := p.listItems[pos]; !ok {
		p.listStruct(pos, limit)
	}
	first := p.listItems[pos]
	contentsEnd := first.end
	for {
		next, ok := p.listItems[contentsEnd]
		if !ok || next.indent != first.indent {
			break
		}
		contentsEnd = next.end
	}
	contentsEnd = p.afterLastNonBlank(pos, contentsEnd)
	n := &Node{Type: TypePlainList, Begin: pos, ContentsBegin: pos, ContentsEnd: contentsEnd}
	for item := pos; item < contentsEnd; item = n.Children[len(n.Children)-1].End {
		n.Children = append(n.Children, p.item(item, limit))
	}
	n.PostBlank, n.End = p.skipBlank(contentsEnd, limit)
	return n
}

// listStruct は pos から始まるリストの各 item の深さと終わりを
// p.listItems に記録する(org-element--list-struct)。
func (p *parser) listStruct(pos, limit int) {
	type open struct {
		begin int
		listItem
	}
	var items []open
	closeAll := func(end int) {
		for _, it := range items {
			p.listItems[it.begin] = listItem{indent: it.indent, end: end}
		}
	}
	for {
		switch {
		case pos >= limit:
			closeAll(p.afterLastNonBlank(0, limit))
			return
		case p.listEnd(pos):
			closeAll(pos)
			return
		case itemRe.MatchString(p.line(pos)):
			indent := p.indent(pos)
			for len(items) > 0 && indent <= items[len(items)-1].indent {
				it := items[len(items)-1]
				items = items[:len(items)-1]
				p.listItems[it.begin] = listItem{indent: it.indent, end: pos}
			}
			items = append(items, open{begin: pos, listItem: listItem{indent: indent}})
		case p.blank(pos):
		default:
			indent := p.indent(pos)
			end := p.afterLastNonBlank(0, pos)
			for indent <= items[len(items)-1].indent {
				it := items[len(items)-1]
				items = items[:len(items)-1]
				p.listItems[it.begin] = listItem{indent: it.indent, end: end}
				if len(items) == 0 {
					return
				}
			}
			pos = p.skipBlockOrDrawer(pos, limit)
		}
		pos = p.lineEnd(pos)
	}
}

// listEnd は pos の行とその次の行が続けて空行かを返す(org-list-end-re)。
func (p *parser) listEnd(pos int) bool {
	next := p.lineEnd(pos)
	return p.blank(pos) && strings.HasSuffix(p.src[pos:next], "\n") &&
		next < len(p.src) && p.blank(next) && strings.HasSuffix(p.src[next:p.lineEnd(next)], "\n")
}

// skipBlockOrDrawer は pos がブロックかドロワーの開始行で、閉じる行が
// limit までにあればその行の位置を、でなければ pos を返す。
func (p *parser) skipBlockOrDrawer(pos, limit int) int {
	line := p.line(pos)
	if m := blockBeginRe.FindStringSubmatch(line); m != nil {
		for end := p.lineEnd(pos); end < limit; end = p.lineEnd(end) {
			if p.blockEnd(end, m[1]) {
				return end
			}
		}
	} else if drawerBeginRe.MatchString(line) {
		// org は :END: をドロワーの開始行から探すので、:END: の行はそ
		// の行自身で閉じる
		if end := p.drawerEnd(pos, limit); end >= 0 {
			return end
		}
	}
	return pos
}

// item は pos から始まる item を解析する。順序付きのリストでは
// 「語 :: 説明」の語も中身に含める。
func (p *parser) item(pos, limit int) *Node {
	m := fullItemRe.FindStringSubmatchIndex(p.line(pos))
	end := min(limit, p.listItems[pos].end)
	n := &Node{Type: TypeItem, Begin: pos, End: end, ContentsBegin: -1, ContentsEnd: -1}
	contents := pos + m[1]
	if m[4] >= 0 && strings.ContainsAny(p.src[pos+m[2]:pos+m[3]], ".)") {
		contents = pos + m[4]
	}
	contents += len(p.src[contents:end]) - len(strings.TrimLeft(p.src[contents:end], " \r\t\n"))
	if contents == end {
		n.PostBlank = p.countLines(pos, end)
		return n
	}
	if contents >= p.lineEnd(pos) {
		contents = p.lineStart(contents)
	}
	n.ContentsBegin, n.ContentsEnd = contents, p.afterLastNonBlank(contents, end)
	n.PostBlank = p.countLines(n.ContentsEnd, end)
	n.Children = p.elements(contents, n.ContentsEnd, modeNone)
	return n
}

// indent は pos の行の字下げの桁数を、タブを 8 桁ごとの位置まで進めて数
// える。
func (p *parser) indent(pos int) int {
	col := 0
	for _, c := range p.line(pos) {
		switch c {
		case ' ':
			col++
		case '\t':
			col += 8 - col%8
		default:
			return col
		}
	}
	return col
}

// afterLastNonBlank は [floor, pos) の末尾の空白と改行を飛ばし、最後に
// 残った行の次の行頭を返す。
func (p *parser) afterLastNonBlank(floor, pos int) int {
	trimmed := floor + len(strings.TrimRight(p.src[floor:pos], " \r\t\n"))
	if trimmed == floor {
		return floor
	}
	return p.lineEnd(trimmed - 1)
}

func (p *parser) lineStart(pos int) int {
	return strings.LastIndexByte(p.src[:pos], '\n') + 1
}

// countLines は Emacs の count-lines と同じく、[begin, end) にかかる行数
// を返す。
func (p *parser) countLines(begin, end int) int {
	s := p.src[begin:end]
	n := strings.Count(s, "\n")
	if s != "" && !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}
