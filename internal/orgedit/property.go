package orgedit

import (
	"fmt"
	"strings"

	"github.com/Warashi/brainium-tools/internal/org"
)

// propertyLine は org-property-format の既定値 "%-10s %s" で 1 行を作る。
func propertyLine(key, value string) string {
	return fmt.Sprintf("%-10s %s\n", ":"+key+":", value)
}

// PutProperty は org-entry-put と同じく、見出し h の key を value にした
// src を返す。key が既にあればその行を置き換え、無ければドロワーの末尾
// に足す。ドロワーが無ければ見出しと計画行の直後に作る。
func PutProperty(src string, h *org.Node, key, value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s の値が空", key)
	}
	line := propertyLine(key, value)
	drawer := org.HeadlineProperties(h)
	if drawer == nil {
		pos := afterPlanning(src, h)
		return insertLines(src, pos, ":PROPERTIES:\n"+line+":END:\n"), nil
	}
	for _, p := range drawer.Children {
		if strings.EqualFold(p.Key, key) {
			return src[:p.Begin] + line + src[p.End:], nil
		}
	}
	return src[:drawerEndLine(src, drawer)] + line + src[drawerEndLine(src, drawer):], nil
}

// DeleteProperty は org-entry-delete と同じく、見出し h の key の行を消
// した src を返す。ドロワーが空になればドロワーごと消す。key が無ければ
// src をそのまま返す。
func DeleteProperty(src string, h *org.Node, key string) string {
	drawer := org.HeadlineProperties(h)
	if drawer == nil {
		return src
	}
	for _, p := range drawer.Children {
		if !strings.EqualFold(p.Key, key) {
			continue
		}
		if len(drawer.Children) == 1 {
			return src[:drawer.Begin] + src[lineEnd(src, drawerEndLine(src, drawer)):]
		}
		return src[:p.Begin] + src[p.End:]
	}
	return src
}

// drawerEndLine はドロワーの :END: 行の先頭の位置を返す。
func drawerEndLine(src string, drawer *org.Node) int {
	if drawer.ContentsBegin >= 0 {
		return drawer.ContentsEnd
	}
	return lineEnd(src, drawer.Begin)
}

// afterPlanning は見出しの行と、その直後の計画行の後ろの位置を返す。
func afterPlanning(src string, h *org.Node) int {
	pos := lineEnd(src, h.Begin)
	if p := planning(h); p != nil {
		pos = lineEnd(src, p.Begin)
	}
	return pos
}

// planning は見出しの計画行を返す。無ければ nil。
func planning(h *org.Node) *org.Node {
	if len(h.Children) == 0 || h.Children[0].Type != org.TypeSection {
		return nil
	}
	if first := h.Children[0].Children; len(first) > 0 && first[0].Type == org.TypePlanning {
		return first[0]
	}
	return nil
}

// endOfMetaData は org-end-of-meta-data と同じく、見出しの行・計画行・
// PROPERTIES ドロワーの後ろの位置を返す。
func endOfMetaData(src string, h *org.Node) int {
	pos := afterPlanning(src, h)
	if d := org.HeadlineProperties(h); d != nil {
		pos = lineEnd(src, drawerEndLine(src, d))
	}
	return pos
}

// insertLines は pos に改行で終わる lines を差し込む。pos が改行で終わ
// らない最終行の末尾なら、先に改行を足す。
func insertLines(src string, pos int, lines string) string {
	if pos == len(src) && src != "" && !strings.HasSuffix(src, "\n") {
		lines = "\n" + lines
	}
	return src[:pos] + lines + src[pos:]
}

// lineEnd は pos を含む行の、改行を含めた終わりを返す。
func lineEnd(src string, pos int) int {
	if i := strings.IndexByte(src[pos:], '\n'); i >= 0 {
		return pos + i + 1
	}
	return len(src)
}
