package org

import (
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	TypeBold     Type = "bold"
	TypeVerbatim Type = "verbatim"
	TypeCode     Type = "code"
)

// minimalSet と standardSet は org-element-object-restrictions の同名の
// 集合のうち、解析する種類だけを持つ。
var (
	minimalSet  = []Type{TypeBold, TypeCode, TypeLatexFragment, TypeSubscript, TypeSuperscript, TypeVerbatim}
	standardSet = append([]Type{TypeLink, TypeTimestamp}, minimalSet...)
)

// objectRestrictions は中身を持つ object の中に置ける object の種類。
var objectRestrictions = map[Type][]Type{
	TypeBold:        standardSet,
	TypeLink:        minimalSet,
	TypeTableCell:   tableCellSet,
	TypeSubscript:   standardSet,
	TypeSuperscript: standardSet,
}

// objectSpan は object の解析器が狭めて見る範囲 [begin, end)。Emacs が
// 中身の範囲に narrow して解析するのに合わせ、begin を行頭として扱い、
// end より先は見ない。
type objectSpan struct {
	p          *parser
	begin, end int
}

// objects は [begin, end) の object を並べる。restriction は置ける
// object の種類。object の間の文字列は節点にしない。
func (p *parser) objects(begin, end int, restriction []Type) []*Node {
	s := objectSpan{p: p, begin: begin, end: end}
	var nodes []*Node
	for pos := begin; pos < end; {
		n := s.object(pos, restriction)
		if n == nil {
			_, size := utf8.DecodeRuneInString(p.src[pos:end])
			pos += size
			continue
		}
		if n.ContentsBegin >= 0 {
			n.Children = p.objects(n.ContentsBegin, n.ContentsEnd, objectRestrictions[n.Type])
		}
		nodes = append(nodes, n)
		pos = n.End
	}
	return nodes
}

// object は pos から始まる object を 1 つ解析する。種類は
// org-element--object-lex と同じく pos の文字で選ぶ。
func (s objectSpan) object(pos int, restriction []Type) *Node {
	allowed := func(t Type) bool { return slices.Contains(restriction, t) }
	if allowed(TypeTableCell) {
		return s.tableCell(pos)
	}
	switch s.p.src[pos] {
	case '*':
		if allowed(TypeBold) {
			return s.emphasis(pos, '*', TypeBold)
		}
	case '$':
		if allowed(TypeLatexFragment) {
			return s.latexFragment(pos)
		}
	case '\\':
		// org は \\ の後に改行(line-break)しか試さない
		if pos+1 < s.end && s.p.src[pos+1] != '\\' && allowed(TypeLatexFragment) {
			return s.latexFragment(pos)
		}
	case '_':
		if allowed(TypeSubscript) {
			return s.script(pos, TypeSubscript)
		}
	case '^':
		if allowed(TypeSuperscript) && s.startsSuperscript(pos) {
			return s.script(pos, TypeSuperscript)
		}
	case '~':
		if allowed(TypeCode) {
			return s.emphasis(pos, '~', TypeCode)
		}
	case '=':
		if allowed(TypeVerbatim) {
			return s.emphasis(pos, '=', TypeVerbatim)
		}
	case '[':
		if pos+1 < s.end && s.p.src[pos+1] == '[' {
			if allowed(TypeLink) {
				return s.link(pos)
			}
		} else if allowed(TypeTimestamp) {
			return s.timestamp(pos)
		}
	case '<':
		if allowed(TypeTimestamp) {
			if n := s.timestamp(pos); n != nil {
				return n
			}
		}
		if allowed(TypeLink) {
			return s.link(pos)
		}
	default:
		if c := s.p.src[pos] | 0x20; 'a' <= c && c <= 'z' && allowed(TypeLink) {
			return s.link(pos)
		}
	}
	return nil
}

// emphasis は pos の mark から始まる強調を解析する
// (org-element--parse-generic-emphasis)。
func (s objectSpan) emphasis(pos int, mark byte, t Type) *Node {
	if !s.opensEmphasis(pos, mark) {
		return nil
	}
	closing := s.closeEmphasis(pos+1, mark)
	if closing < 0 {
		return nil
	}
	n := &Node{Type: t, Begin: pos, ContentsBegin: -1, ContentsEnd: -1}
	if t != TypeCode && t != TypeVerbatim {
		n.ContentsBegin, n.ContentsEnd = pos+1, closing-1
	}
	n.PostBlank, n.End = s.skipSpaces(closing)
	return n
}

// opensEmphasis は pos の mark が強調を開けるかを返す。org は 1 文字戻っ
// て (行頭|前の文字) mark 非空白 を見るので、行頭の mark の直後の mark
// も開ける。
func (s objectSpan) opensEmphasis(pos int, mark byte) bool {
	if s.bol(pos) {
		r, ok := s.runeAt(pos + 1)
		return ok && !isSpace(r)
	}
	prev, size := utf8.DecodeLastRuneInString(s.p.src[s.begin:pos])
	q := pos - size
	if s.bol(q) && prev == rune(mark) {
		return true
	}
	if !isSpace(prev) && !strings.ContainsRune(`-('"{`, prev) {
		return false
	}
	r, ok := s.runeAt(pos + 1)
	return ok && !isSpace(r)
}

// closeEmphasis は from から、非空白・mark・区切りと続く最初の mark を探
// し、その直後の位置を返す。無ければ -1。
func (s objectSpan) closeEmphasis(from int, mark byte) int {
	for i := from; i < s.end; {
		r, size := utf8.DecodeRuneInString(s.p.src[i:s.end])
		at := i + size
		if !isSpace(r) && at < s.end && s.p.src[at] == mark {
			next, ok := s.runeAt(at + 1)
			if !ok || next == '\n' || isSpace(next) || strings.ContainsRune(`-.,;:!?'")}\[`, next) {
				return at + 1
			}
		}
		i = at
	}
	return -1
}

// skipSpaces は pos から続く空白とタブを飛ばし、その数と次の位置を返す。
func (s objectSpan) skipSpaces(pos int) (int, int) {
	end := pos
	for end < s.end && (s.p.src[end] == ' ' || s.p.src[end] == '\t') {
		end++
	}
	return end - pos, end
}

func (s objectSpan) bol(pos int) bool {
	return pos == s.begin || s.p.src[pos-1] == '\n'
}

// runeAt は pos の文字を返す。pos が範囲の外なら ok は false。
func (s objectSpan) runeAt(pos int) (rune, bool) {
	if pos >= s.end {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(s.p.src[pos:s.end])
	return r, true
}
