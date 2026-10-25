package orglint

import (
	"regexp"
	"unicode"
)

// noWrapRe は途中で折り返せない範囲。URL だけを 1 かたまりとして扱う。
// org リンクやコード表記は含めない。Emacs の fill も [[id:...][説明]]
// の説明部分や =nix flake update= の空白は普通に折り返すため。
var noWrapRe = regexp.MustCompile(`\bhttps?://[^\s\]]+`)

// headBanned は行頭に置けない文字。Emacs の kinsoku-bol 相当。
const headBanned = "、。，．,.!?！？)]}）］｝」』】〉》〕〟’”ー〜～々ゝゞヽヾ・:;：；゛゜…‥%％‰℃" +
	"ぁぃぅぇぉっゃゅょゎァィゥェォッャュョヮヵヶ"

// tailBanned は行末に置けない文字。Emacs の kinsoku-eol 相当。
const tailBanned = "([{（［｛「『【〈《〔〝‘“＄＃£¥#"

// kinsokuLimit は禁則処理で行末を伸ばせる文字数。Emacs の
// kinsoku-limit に合わせる。
const kinsokuLimit = 4

// fillBreak は Emacs の fill が line を最初に折り返す位置を rune 単位で
// 返す。折り返さない場合は -1 を返す。offset は行頭に付く継続行の
// インデント幅。hideIndent は line の行頭の字下げが不可視属性を継承して
// いるかで、直前の折り返しが空白を飲み込まなかったときだけ真になる。
//
// Emacs の挙動を実測して次の順に決めている。
//
//  1. limit 桁に届くまで進む。0 桁の文字は桁が足りていても読み飛ばし、
//     全角文字が limit をまたぐ場合はその文字ごと通り越す。行末に
//     着いたら折り返さない。桁ちょうどで止まったときはさらに 1 文字
//     進めるので、行末の 1 文字が limit + 2 桁目まではみ出しうる。
//  2. そこまでにある最後の折り返し点まで戻る。禁則に触れる場合は
//     行末を伸ばして飲み込む。
//  3. 折り返し点がなければ(長い URL など)、そこから後ろにある最初の
//     折り返し点まで進む。
func fillBreak(line string, limit int, offset int, hideIndent bool) int {
	rs := []rune(line)
	ws := runeWidths(line, hideIndent)
	spans := noWrapSpans(rs)

	// 1. move-to-column 相当。0 桁の文字は桁が足りていても読み飛ばす。
	reach := len(rs)
	width := offset
	for i := range rs {
		if width >= limit && ws[i] > 0 {
			reach = i
			break
		}
		width += ws[i]
	}
	if reach >= len(rs) {
		return -1
	}
	// ちょうど limit 桁で止まったときは 1 文字進む。全角文字が limit を
	// またいで通り越したときは進まない。
	if width <= limit {
		reach++
	}

	// 2. そこまでにある最後の折り返し点へ戻る。
	at := -1
	for i := 1; i < reach; i++ {
		if hasContent(rs[:i]) && isBreakPoint(rs, i, spans) {
			at = i
		}
	}
	// 3. 見つからなければ後ろ向きに探す。
	if at < 0 {
		for i := reach; i < len(rs); i++ {
			if isBreakPoint(rs, i, spans) {
				at = i
				break
			}
		}
		if at < 0 {
			return -1
		}
	}

	// 空白で折り返すときは空白自体を行末に残さない。
	for at > 0 && unicode.IsSpace(rs[at-1]) {
		at--
	}
	at = kinsoku(rs, ws, at, limit, offset)
	if at < 0 || !hasContent(rs[:at]) || !hasContent(rs[at:]) {
		return -1
	}
	return at
}

// kinsoku は禁則処理で折り返し位置をずらす。Emacs の kinsoku 関数と
// 同じく、行頭に置けない文字はまず行末を伸ばして飲み込み(追い出し)、
// 伸ばしすぎになるなら手前の折り返し点まで戻す(追い込み)。Emacs の fill
// は折り返し位置の前後がどちらも ASCII なら kinsoku を呼ばないので、
// 「あ ( foo」の ( は空白の位置で折り返して行末に残る。
func kinsoku(rs []rune, ws []int, at, limit, offset int) int {
	if at > 0 && at < len(rs) && rs[at-1] <= unicode.MaxASCII && rs[at] <= unicode.MaxASCII {
		return at
	}
	if at < len(rs) && containsRune(headBanned, rs[at]) {
		if longer := kinsokuLonger(rs, ws, at, limit, offset); longer >= 0 {
			return longer
		}
		return kinsokuShorter(rs, at)
	}
	if at > 0 && containsRune(tailBanned, rs[at-1]) {
		return kinsokuShorter(rs, at)
	}
	return at
}

// kinsokuLonger は行末を伸ばせる位置を返す。limit + kinsokuLimit 桁に
// 収まらなければ -1 を返す。
func kinsokuLonger(rs []rune, ws []int, at, limit, offset int) int {
	i := at + 1
	for i < len(rs) && (containsRune(headBanned, rs[i]) || !breakable(rs, i)) {
		i++
	}
	width := offset
	for _, w := range ws[:i] {
		width += w
	}
	if width < limit+kinsokuLimit {
		return i
	}
	return -1
}

// kinsokuShorter は手前にある折り返し可能な位置を返す。行頭まで戻り
// きった場合は元の位置をそのまま返す。
func kinsokuShorter(rs []rune, at int) int {
	i := at - 1
	for i > 0 && (containsRune(tailBanned, rs[i-1]) ||
		(i < len(rs) && containsRune(headBanned, rs[i])) ||
		!breakable(rs, i)) {
		i--
	}
	if i > 0 {
		return i
	}
	return at
}

// breakable は禁則処理の走査で折り返してよい位置かを返す。空白の位置
// と全角文字の直後が候補になる。
func breakable(rs []rune, i int) bool {
	return i > 0 && (unicode.IsSpace(rs[i-1]) || runeWidth(rs[i-1]) == 2)
}

// isBreakPoint は rs[i] の直前で折り返せるかを返す。禁則は考慮せず、
// 空白と全角文字に接する境界だけを候補にする。
func isBreakPoint(rs []rune, i int, spans [][2]int) bool {
	for _, s := range spans {
		if s[0] < i && i < s[1] {
			return false
		}
	}
	if unicode.IsSpace(rs[i]) {
		return true
	}
	// 欧文の単語の途中では折り返さない。
	return runeWidth(rs[i-1]) == 2 || runeWidth(rs[i]) == 2
}

func containsRune(set string, r rune) bool {
	for _, c := range set {
		if c == r {
			return true
		}
	}
	return false
}

func hasContent(rs []rune) bool {
	for _, r := range rs {
		if !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

// noWrapSpans は折り返せない範囲を rune 単位の [start, end) で返す。
func noWrapSpans(rs []rune) [][2]int {
	line := string(rs)
	matches := noWrapRe.FindAllStringIndex(line, -1)
	if matches == nil {
		return nil
	}
	index := runeIndex(line)
	spans := make([][2]int, 0, len(matches))
	for _, m := range matches {
		spans = append(spans, [2]int{index[m[0]], index[m[1]]})
	}
	return spans
}
