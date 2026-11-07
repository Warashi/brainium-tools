package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Warashi/brainium-tools/internal/org"
)

// objectTypes は object の種類。object の End は後ろの空白(PostBlank 個)
// を含むので、範囲と抜粋からは外す。element の PostBlank は空行の数。
var objectTypes = []org.Type{
	org.TypeBold, org.TypeCode, org.TypeLatexFragment, org.TypeLink,
	org.TypeSubscript, org.TypeSuperscript, org.TypeTableCell,
	org.TypeTimestamp, org.TypeVerbatim,
}

// textContainers は中身が object と素の文字列からなる種類。
var textContainers = []org.Type{
	org.TypeBold, org.TypeLink, org.TypeParagraph, org.TypeSubscript,
	org.TypeSuperscript, org.TypeTableCell,
}

type renderer struct {
	src      string
	starts   []int
	from, to int
	b        strings.Builder
}

// render は root の子孫のうち from 行から to 行と重なる節点を木にする。
func render(src string, root *org.Node, from, to int) string {
	r := &renderer{src: src, from: from, to: to, starts: []int{0}}
	for i := range len(src) {
		if src[i] == '\n' {
			r.starts = append(r.starts, i+1)
		}
	}
	for _, c := range root.Children {
		r.node(c, 0)
	}
	return r.b.String()
}

func (r *renderer) node(n *org.Node, depth int) {
	end := n.End
	isObject := slices.Contains(objectTypes, n.Type)
	if isObject {
		end -= n.PostBlank
	}
	if !r.overlaps(n.Begin, end) {
		return
	}
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(&r.b, "%s%s %s", indent, n.Type, r.span(n.Begin, end))
	switch {
	case n.Type == org.TypeHeadline:
		fmt.Fprintf(&r.b, " level=%d", n.Level)
	case n.Type == org.TypeLink:
		fmt.Fprintf(&r.b, " type=%s path=%q", n.LinkType, n.Path)
	case n.Key != "":
		fmt.Fprintf(&r.b, " key=%s value=%q", n.Key, n.Value)
	}
	if len(n.Secondary) > 0 {
		// title や tag の範囲は木に残らないので、行そのものを添える
		line, _, _ := strings.Cut(r.src[n.Begin:], "\n")
		fmt.Fprintf(&r.b, " raw=%q", line)
	}
	if isObject && !slices.Contains(textContainers, n.Type) || n.Type == org.TypeLink && n.ContentsBegin < 0 {
		fmt.Fprintf(&r.b, " %q", r.src[n.Begin:end])
	}
	r.b.WriteString("\n")

	if len(n.Secondary) > 0 {
		label := "title"
		if n.Type == org.TypeItem {
			label = "tag"
		}
		fmt.Fprintf(&r.b, "%s  %s\n", indent, label)
		for _, c := range n.Secondary {
			r.node(c, depth+2)
		}
	}
	if !slices.Contains(textContainers, n.Type) || n.ContentsBegin < 0 {
		for _, c := range n.Children {
			r.node(c, depth+1)
		}
		return
	}
	at := n.ContentsBegin
	for _, c := range n.Children {
		r.text(at, c.Begin, depth+1)
		r.node(c, depth+1)
		at = c.End
		if slices.Contains(objectTypes, c.Type) {
			at -= c.PostBlank
		}
	}
	r.text(at, n.ContentsEnd, depth+1)
}

func (r *renderer) text(begin, end, depth int) {
	if begin >= end || !r.overlaps(begin, end) {
		return
	}
	fmt.Fprintf(&r.b, "%stext %s %q\n", strings.Repeat("  ", depth), r.span(begin, end), r.src[begin:end])
}

// overlaps は [begin, end) が出す行の範囲と重なるかを返す。
func (r *renderer) overlaps(begin, end int) bool {
	last := max(begin, end-1)
	return r.line(begin) <= r.to && r.line(last) >= r.from
}

// line はバイト位置 off の行番号(1 始まり)を返す。
func (r *renderer) line(off int) int {
	return sort.Search(len(r.starts), func(i int) bool { return r.starts[i] > off })
}

func (r *renderer) span(begin, end int) string {
	return r.pos(begin) + "-" + r.pos(end)
}

func (r *renderer) pos(off int) string {
	l := r.line(off)
	return fmt.Sprintf("%d:%d", l, utf8.RuneCountInString(r.src[r.starts[l-1]:off])+1)
}
