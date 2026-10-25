package orglint

import (
	"slices"
	"unicode"
	"unicode/utf8"
)

// spaceVerbatim は和文に直接続いて組まれないコード表記に、半角空白を補っ
// て組まれるようにする。補うのは、前が和文の文字か開き括弧以外の全角の
// 句読点・記号の場合と、後ろが和文の文字の場合で、補った結果が組まれな
// いものには触れない。後ろの全角の句読点・閉じ括弧、前の開き括弧や半角
// スラッシュは、語を足すか句読点を動かすかが文意で決まるので報告に任せ
// る。
func spaceVerbatim(line string) string {
	_, dangling := emphasisSpans(line)
	for _, s := range slices.Backward(dangling) {
		start, end := s[0], s[1]
		before, _ := utf8.DecodeLastRuneInString(line[:start])
		after, _ := utf8.DecodeRuneInString(line[end:])
		spaceBefore := start > 0 && (isWideLetter(before) || isWidePunct(before) && !unicode.Is(unicode.Ps, before))
		spaceAfter := end < len(line) && isWideLetter(after)
		if !spaceBefore && !spaceAfter {
			continue
		}
		candidate := line[:end]
		if spaceAfter {
			candidate += " "
		}
		candidate += line[end:]
		if spaceBefore {
			candidate = candidate[:start] + " " + candidate[start:]
			start++
			end++
		}
		if typesetEnd(candidate, start) == end {
			line = candidate
		}
	}
	return line
}
