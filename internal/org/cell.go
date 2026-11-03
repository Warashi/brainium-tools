package org

import "regexp"

const TypeTableCell Type = "table-cell"

// tableCellSet は表のセルの中に置ける object の種類。
var tableCellSet = append([]Type{TypeLink, TypeTimestamp}, minimalSet...)

var tableCellRe = regexp.MustCompile(`(?m)^[ \t]*(.*?)[ \t]*(?:\||$)`)

// tableCell は pos から次の | までを表のセルにする。中身は前後の空白を
// 除いた部分。
func (s objectSpan) tableCell(pos int) *Node {
	m := tableCellRe.FindStringSubmatchIndex(s.p.src[pos:s.end])
	return &Node{Type: TypeTableCell, Begin: pos, End: pos + m[1], ContentsBegin: pos + m[2], ContentsEnd: pos + m[3]}
}
