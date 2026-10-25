package orglint

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
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

// joinLine は折り返された 2 行をつなぐ。欧文どうしは空白で区切り、
// 和文の途中はそのままつなぐ。和文と欧文の境界には空白を補う。元の
// 行が空白の位置で折り返されていた場合にその空白を復元するためで、
// 「全角と半角英数の間に空白を置く」このリポジトリの書き方に合わせる。
// 空白のない境界にも空白が入るので幅が 1 桁増え、Emacs の折り返し位置
// とはずれうる。空白を補わない側に倒すと、折り返しで空白が消えた境界を
// 復元できずに書き方の側が崩れるので、そちらは採らない。
func joinLine(a, b string) string {
	if a == "" || b == "" {
		return a + b
	}
	if needsSpace(markupEdge(a, true), markupEdge(b, false)) {
		return a + " " + b
	}
	return a + b
}

// markupEdge は s の末尾(fromEnd)または先頭にある、表示される文字を
// 返す。Emacs が隠すリンクの記法だけを読み飛ばすので、[[id:x][gopher]]
// では説明文の末尾が、=foo= では区切りの = 自体が端になる。前者は記法が
// 見えないので説明文で空白の要否が決まり、後者は = が見えるぶん和文と
// の間に空白が要る。隠れる文字しかなければ端の文字をそのまま返す。
func markupEdge(s string, fromEnd bool) rune {
	rs := []rune(s)
	ws := runeWidths(s, false)
	if fromEnd {
		for i := range slices.Backward(rs) {
			if ws[i] > 0 {
				return rs[i]
			}
		}
		return rs[len(rs)-1]
	}
	for i := range rs {
		if ws[i] > 0 {
			return rs[i]
		}
	}
	return rs[0]
}

// markupDelims は org のマークアップの区切り記号。記号ではあるが、
// =orglint= のようなコード表記を和文の間に置くときは空白で区切る書き方を
// しているので、空白を要する側として扱う。
const markupDelims = "=~*/_"

// trailingPunct は直後に空白を置く半角の区切り。「基盤になる: 内容」の
// ように、和文が続く場合も空白を空ける書き方をしている。手前に和文が
// あっても空白は入れないので、spacedLatin とは違って向きがある。
const trailingPunct = ":;,.!?"

func needsSpace(a, b rune) bool {
	if runeWidth(a) == 1 && runeWidth(b) == 1 {
		return true
	}
	if containsRune(trailingPunct, a) && isWideLetter(b) {
		return true
	}
	// 全角の句読点や括弧とコード表記の区切りの間にも空白が要る。org は
	// 前後に半角しか許さないので、ここを詰めると組まれない。折り返しで
	// 空白が消えた境界を復元できないと、checkEmphasis の違反を -fix が
	// 作り直すことになる。
	if isWidePunct(a) && containsRune(verbatimDelims, b) {
		return true
	}
	if containsRune(verbatimDelims, a) && isWidePunct(b) {
		return true
	}
	return (isWideLetter(a) && spacedLatin(b)) || (spacedLatin(a) && isWideLetter(b))
}

// verbatimDelims はコード表記の区切り記号。強調の区切りは他にもあるが、
// 前後の文字を検査しているのはこの 2 つだけなので、空白の復元もここに
// 揃える。
const verbatimDelims = "=~"

// isWidePunct は全角の句読点・括弧・記号かを判定する。
func isWidePunct(r rune) bool {
	return runeWidth(r) == 2 && (unicode.IsPunct(r) || unicode.IsSymbol(r))
}

// spacedLatin は和文との間に空白を置く半角文字かを判定する。「実施した
// (2026-07-22)」のように括弧や句読点は和文に直接続けて書くので、和文の
// 側だけを見て空白を補うと元になかった空白が入る。
func spacedLatin(r rune) bool {
	if runeWidth(r) != 1 {
		return false
	}
	if containsRune(markupDelims, r) {
		return true
	}
	return !unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r)
}

// isWideLetter は全角の文字のうち句読点・記号ではないものを判定する。
// 「。」や「」の直後には空白を補わない。
func isWideLetter(r rune) bool {
	return runeWidth(r) == 2 && !unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r)
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
