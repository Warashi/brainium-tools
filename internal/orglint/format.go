package orglint

import (
	"slices"
	"strings"

	"github.com/Warashi/brainium-tools/internal/org"
)

// Format は本文の段落を fillColumn で折り返し直した内容を返す。和文に
// 直接続いて組まれないコード表記には、先に空白を補う。見出し・表・ブ
// ロック・ドロワーには手を入れない。
func Format(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	root := org.Parse(string(data))
	l := lineLayout(string(data), root, lines)
	skips := verbatimSkips(root, lines)
	for i := range lines {
		if l.kinds[i] == kindBody {
			lines[i] = spaceVerbatim(lines[i], skips[i])
		}
	}
	out, _ := formatLines(lines, l)
	return []byte(strings.Join(out, "\n"))
}

// formatLines は整形した行と、整形で内容が変わった段落の先頭の行番号
// (1 始まり)を返す。整形結果を書き戻す -fix と、一致しない段落を報告
// する検査の両方がこれを使う。
func formatLines(lines []string, l layout) ([]string, []int) {
	out := make([]string, 0, len(lines))
	var changed []int
	for i := 0; i < len(lines); {
		para := l.paras[i]
		if para < 0 {
			out = append(out, lines[i])
			i++
			continue
		}
		end := i + 1
		for end < len(lines) && l.paras[end] == para {
			end++
		}
		filled := fillParagraph(lines[i:end], l.indents[para])
		if !slices.Equal(lines[i:end], filled) {
			changed = append(changed, i+1)
		}
		out = append(out, filled...)
		i = end
	}
	return out, changed
}

// fillPasses は fillParagraph が折り返しを繰り返す上限。
const fillPasses = 4

// fillParagraph は段落を折り返し直した結果が変わらなくなるまで
// fillOnce を繰り返す。つなぎ目に空白を補うと段落の幅が変わって
// 折り返し位置も動くため、1 回では正規形に届かないことがある。
func fillParagraph(para []string, indent string) []string {
	for range fillPasses {
		next := fillOnce(para, indent)
		if slices.Equal(next, para) {
			break
		}
		para = next
	}
	return para
}

// fillOnce は段落を 1 行につないでから fillColumn で折り返し直し、継続
// 行に indent を付ける。コメントの # のように空白以外を含む indent は、
// Emacs の fill-prefix と同じく継続行から剥がしてつなぎ、その中と直後で
// は折り返さない。
func fillOnce(para []string, indent string) []string {
	prefix := strings.TrimRight(indent, " \t")
	head := 0
	if strings.TrimSpace(indent) != "" {
		head = len([]rune(indent))
	}
	text := strings.TrimRight(para[0], " \t")
	for _, line := range para[1:] {
		line = strings.TrimPrefix(strings.TrimLeft(line, " \t"), strings.TrimLeft(prefix, " \t"))
		text = joinLine(text, strings.TrimSpace(line))
	}

	// 表示幅は 1 行につないだ段落から求め、折り返しで切り出す rune に
	// 付けて引き継ぐ。切り出した後の断片を解析し直すと、途中で割れた
	// リンクを parser が見つけられない。
	rs := []rune(text)
	ws := runeWidths(text, head)
	var out []string
	for {
		at := fillBreak(rs, ws, fillColumn, 0, head)
		if at < 0 {
			break
		}
		out = append(out, strings.TrimRight(string(rs[:at]), " \t"))
		for at < len(rs) && (rs[at] == ' ' || rs[at] == '\t') {
			at++
		}
		ind := []rune(indent)
		indWidths := make([]int, len(ind))
		for i, r := range ind {
			indWidths[i] = runeWidth(r)
		}
		rs = append(ind, rs[at:]...)
		ws = append(indWidths, ws[at:]...)
	}
	return append(out, string(rs))
}

func leadingSpace(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}
