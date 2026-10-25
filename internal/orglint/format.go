package orglint

import (
	"regexp"
	"slices"
	"strings"
)

// listItemRe はリスト項目の行頭マーカー。継続行はマーカーの幅だけ
// 字下げする。
var listItemRe = regexp.MustCompile(`^\s*(?:[-+]|[0-9]+[.)])\s+`)

// Format は本文の段落を fillColumn で折り返し直した内容を返す。和文に
// 直接続いて組まれないコード表記には、先に空白を補う。見出し・表・ブ
// ロック・ドロワーには手を入れない。
func Format(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	kinds := scanLines(lines)
	for i := range lines {
		if kinds[i] == kindBody {
			lines[i] = spaceVerbatim(lines[i])
		}
	}
	out, _ := formatLines(lines, kinds)
	return []byte(strings.Join(out, "\n"))
}

// formatLines は整形した行と、整形で内容が変わった段落の先頭の行番号
// (1 始まり)を返す。整形結果を書き戻す -fix と、一致しない段落を報告
// する検査の両方がこれを使う。
func formatLines(lines []string, kinds []lineKind) ([]string, []int) {
	out := make([]string, 0, len(lines))
	var changed []int
	for i := 0; i < len(lines); {
		if kinds[i] != kindBody {
			out = append(out, lines[i])
			i++
			continue
		}
		end := paragraphEnd(lines, kinds, i)
		para := lines[i:end]
		filled := fillParagraph(para)
		if !slices.Equal(para, filled) {
			changed = append(changed, i+1)
		}
		out = append(out, filled...)
		i = end
	}
	return out, changed
}

// paragraphEnd は lines[start] から続く 1 段落の終端(半開区間)を返す。
// 新しいリスト項目の開始と、継続行として辻褄の合わない字下げの行で
// 段落を切る。
func paragraphEnd(lines []string, kinds []lineKind, start int) int {
	indent := continuationIndent(lines[start])
	end := start + 1
	for ; end < len(lines); end++ {
		if kinds[end] != kindBody || listItemRe.MatchString(lines[end]) {
			break
		}
		if leadingSpace(lines[end]) != indent {
			break
		}
	}
	return end
}

// fillPasses は fillParagraph が折り返しを繰り返す上限。
const fillPasses = 4

// fillParagraph は段落を折り返し直した結果が変わらなくなるまで
// fillOnce を繰り返す。つなぎ目に空白を補うと段落の幅が変わって
// 折り返し位置も動くため、1 回では正規形に届かないことがある。
func fillParagraph(para []string) []string {
	for range fillPasses {
		next := fillOnce(para)
		if slices.Equal(next, para) {
			break
		}
		para = next
	}
	return para
}

// fillOnce は段落を 1 行につないでから fillColumn で折り返し直す。
func fillOnce(para []string) []string {
	indent := continuationIndent(para[0])
	text := strings.TrimRight(para[0], " \t")
	for _, line := range para[1:] {
		text = joinLine(text, strings.TrimSpace(line))
	}

	var out []string
	hideIndent := false
	for {
		at := fillBreak(text, fillColumn, 0, hideIndent)
		if at < 0 {
			break
		}
		rs := []rune(text)
		rest := string(rs[at:])
		trimmed := strings.TrimLeft(rest, " \t")
		hideIndent = trimmed == rest
		out = append(out, strings.TrimRight(string(rs[:at]), " \t"))
		text = indent + trimmed
	}
	return append(out, text)
}

// continuationIndent は段落の 2 行目以降に付ける字下げを返す。
func continuationIndent(line string) string {
	if m := listItemRe.FindString(line); m != "" {
		return strings.Repeat(" ", len([]rune(m)))
	}
	return leadingSpace(line)
}

func leadingSpace(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}
