package orglint

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// displayWidth は s の表示幅を返す。全角 1 文字を 2 桁、それ以外を
// 1 桁として数える。org の記法は解釈しない。
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

var (
	// linkRe は org のブラケットリンク [[パス]] と [[パス][説明]]。
	linkRe = regexp.MustCompile(`\[\[([^\[\]]+)\](?:\[([^\[\]]+)\])?\]`)
	// openLinkRe は行末で閉じていないリンクの前半 [[パス][ 。
	openLinkRe = regexp.MustCompile(`\[\[[^\[\]]+\]\[[^\[\]]*$`)
)

// emphasisPre と emphasisPost は org-emphasis-regexp-components の
// 既定で =…= や ~…~ の前後に置ける文字。em ダッシュ以外はすべて半角
// なので、「例えば=aaa=は」のように和文へ直に続けたものは組み直されず
// ただの文字として表示される。org を使うのはこのリポジトリだけでは
// ないため、設定を広げる側ではなく本文を既定に合わせる側に倒す。
const (
	emphasisPre  = "-—\t\n\f\r ('\"{"
	emphasisPost = "-—\t\n\f\r .,:!?;'\")}["
)

// verbatimStrictSpans はコード表記として組まれる範囲をバイト単位で返す。
// 前後の文字は正規表現に含めずに 1 文字ずつ見る。=a= =b= のように候補が
// 続く行で、直前の候補が区切りの空白を食って次を取りこぼさないため。
func verbatimStrictSpans(line string) [][]int {
	typeset, _ := emphasisSpans(line)
	return typeset
}

// emphasisSpans は行中のコード表記の候補を、組まれるもの(typeset)と
// 前後の文字の条件を満たさず組まれないもの(dangling)に分けて返す。
//
// 組まれる範囲は Emacs の org-verbatim-re に合わせる。開き記号の前が
// 条件を満たし、直後が空白でなければ、そこから先で「直前が空白でなく、
// 直後が条件を満たす」最初の同じ記号を閉じ記号とする。中身に区切り記号
// が入っても構わないので、=--opt=val-<name>= は内側の = で閉じずに
// 全体が組まれる。組まれる範囲に含まれない候補を dangling とする。
func emphasisSpans(line string) (typeset, dangling [][]int) {
	for i := 0; i < len(line); {
		if end := typesetEnd(line, i); end > 0 {
			typeset = append(typeset, []int{i, end})
			i = end
			continue
		}
		i++
	}
	for _, m := range verbatimRe.FindAllStringIndex(line, -1) {
		if !inside(typeset, m[0], m[1]) {
			dangling = append(dangling, m)
		}
	}
	return typeset, dangling
}

// typesetEnd は line[start] を開き記号とするコード表記が組まれるとき、
// その終端のバイト位置を返す。組まれないときは -1 を返す。
func typesetEnd(line string, start int) int {
	marker := line[start]
	if !strings.ContainsRune(verbatimDelims, rune(marker)) || !emphasisPreOK(line, start) {
		return -1
	}
	if start+1 >= len(line) {
		return -1
	}
	if r, _ := utf8.DecodeRuneInString(line[start+1:]); unicode.IsSpace(r) {
		return -1
	}
	for j := start + 2; j < len(line); j++ {
		if line[j] != marker || spaceBefore(line, j) {
			continue
		}
		if emphasisPostOK(line, j+1) {
			return j + 1
		}
	}
	return -1
}

// spaceBefore は line[:at] の末尾の文字が空白かを返す。
func spaceBefore(line string, at int) bool {
	r, _ := utf8.DecodeLastRuneInString(line[:at])
	return unicode.IsSpace(r)
}

func emphasisPreOK(line string, start int) bool {
	if start == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(line[:start])
	return strings.ContainsRune(emphasisPre, r)
}

func emphasisPostOK(line string, end int) bool {
	if end >= len(line) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(line[end:])
	return strings.ContainsRune(emphasisPost, r)
}

// runeWidths は line の rune ごとの表示幅を返す。org-link-descriptive
// が既定で有効な Emacs はブラケットリンクの角括弧とパスを隠し、説明が
// あれば説明だけ、なければパスだけを表示する。隠れる部分は 0 桁として
// 数える。auto-fill-mode が見る桁数も同じくこの表示幅なので、行幅の
// 判定と折り返し位置の計算はどちらもこちらを使う。
func runeWidths(line string, hideIndent bool) []int {
	rs := []rune(line)
	ws := make([]int, len(rs))
	for i, r := range rs {
		ws[i] = runeWidth(r)
	}
	for _, span := range hiddenSpans(line, hideIndent) {
		for i := span[0]; i < span[1]; i++ {
			ws[i] = 0
		}
	}
	return ws
}

// hiddenSpans は Emacs が隠すリンク記法の範囲を rune 単位の [start, end)
// で返す。コード表記の中のリンクは org が組み立て直さないので、記法が
// そのまま表示される。
func hiddenSpans(line string, hideIndent bool) [][2]int {
	verbatim := verbatimStrictSpans(line)
	index := runeIndex(line)
	var spans [][2]int
	add := func(start, end int) {
		spans = append(spans, [2]int{index[start], index[end]})
	}

	head, tail := len(line), 0
	for _, m := range linkRe.FindAllStringSubmatchIndex(line, -1) {
		if inside(verbatim, m[0], m[1]) {
			continue
		}
		// 説明があればパスまでの前半と末尾の ]] が隠れる。なければ
		// 両端の [[ と ]] だけが隠れてパスが見える。
		open := m[0] + 2
		if m[4] >= 0 {
			open = m[4]
		}
		add(m[0], open)
		add(m[1]-2, m[1])
		head, tail = min(head, m[0]), m[1]
	}

	// 説明の途中で折り返されたリンクの断片。Emacs は段落を 1 行に
	// つないだ状態で不可視属性を付けてから折り返すので、折り返しで
	// リンクが 2 行に割れても両端の記法は隠れたままになる。
	if at := unclosedTail(line[:head], verbatim); at >= 0 {
		add(at, at+2)
	}
	if m := openLinkRe.FindStringIndex(line[tail:]); m != nil && !inside(verbatim, tail+m[0], tail+m[1]) {
		open := strings.LastIndex(line[tail+m[0]:tail+m[1]], "][") + 2
		add(tail+m[0], tail+m[0]+open)
	}

	// 行頭の字下げの直後にリンクが始まる行では字下げも隠れる。Emacs の
	// fill は折り返しで入れる改行に「改行の次の文字」のテキスト属性を
	// 与え、続けて挿入する字下げがそれを継承するため。空白の位置で
	// 折り返したときは改行が空白の属性を受け取るので継承は起きない。
	indent := len([]rune(leadingSpace(line)))
	if hideIndent && indent > 0 {
		for _, s := range spans {
			if s[0] == indent {
				spans = append(spans, [2]int{0, indent})
				break
			}
		}
	}
	return spans
}

// unclosedTail は開き記法のないまま行中に現れる ]] の位置を返す。
func unclosedTail(prefix string, verbatim [][]int) int {
	at := strings.Index(prefix, "]]")
	if at < 0 || strings.Contains(prefix[:at], "[[") || inside(verbatim, at, at+2) {
		return -1
	}
	return at
}

// inside は [start, end) がいずれかの span に収まっているかを返す。
func inside(spans [][]int, start, end int) bool {
	for _, s := range spans {
		if s[0] <= start && end <= s[1] {
			return true
		}
	}
	return false
}

// runeIndex は s のバイト位置から rune 位置への対応を返す。
func runeIndex(s string) map[int]int {
	index := make(map[int]int, len(s)+1)
	n := 0
	for b := range s {
		index[b] = n
		n++
	}
	index[len(s)] = n
	return index
}

func runeWidth(r rune) int {
	i := sort.Search(len(wideRanges), func(i int) bool { return wideRanges[i][1] >= r })
	if i < len(wideRanges) && wideRanges[i][0] <= r {
		return 2
	}
	return 1
}

// wideRanges は 2 桁として数える文字の範囲を昇順に並べたもの。Emacs 30.2
// の char-width-table から幅 2 の範囲をそのまま書き出したもので、East
// Asian Width の Wide・Fullwidth に一致する。
//
// East Asian Ambiguous(→ ⇒ ※ ① … ℃ ± × ─ ■ ‐ — “ ” Ⅰ など)は 1 桁に
// なる。Emacs は既定の char-width-table でこれらを 1 桁として扱い、
// auto-fill-mode の折り返し位置もそれに従うため、行幅の判定も Emacs に
// 合わせる。
//
// 書き出しに使ったコード:
//
//	emacs -Q --batch --eval \
//	  '(dotimes (c #x110000) (princ (format "%X %d\n" c (char-width c))))'
//
// 制御文字(U+0000〜U+001F・U+007F)は Emacs では ^X の 2 桁表示になるが、
// org の本文には現れないので除いた。
var wideRanges = [][2]rune{
	{0x1100, 0x115F}, // ハングル字母
	// 記号・絵文字のうち Wide のもの
	{0x231A, 0x231B},
	{0x2329, 0x232A},
	{0x23E9, 0x23EC},
	{0x23F0, 0x23F0},
	{0x23F3, 0x23F3},
	{0x25FD, 0x25FE},
	{0x2614, 0x2615},
	{0x2648, 0x2653},
	{0x267F, 0x267F},
	{0x2693, 0x2693},
	{0x26A1, 0x26A1},
	{0x26AA, 0x26AB},
	{0x26BD, 0x26BE},
	{0x26C4, 0x26C5},
	{0x26CE, 0x26CE},
	{0x26D4, 0x26D4},
	{0x26EA, 0x26EA},
	{0x26F2, 0x26F3},
	{0x26F5, 0x26F5},
	{0x26FA, 0x26FA},
	{0x26FD, 0x26FD},
	{0x2705, 0x2705},
	{0x270A, 0x270B},
	{0x2728, 0x2728},
	{0x274C, 0x274C},
	{0x274E, 0x274E},
	{0x2753, 0x2755},
	{0x2757, 0x2757},
	{0x2795, 0x2797},
	{0x27B0, 0x27B0},
	{0x27BF, 0x27BF},
	{0x2B1B, 0x2B1C},
	{0x2B50, 0x2B50},
	{0x2B55, 0x2B55},
	// CJK・かな・ハングル
	{0x2E80, 0x2E99},
	{0x2E9B, 0x2EF3},
	{0x2F00, 0x2FD5},
	{0x2FF0, 0x303E},
	{0x3041, 0x3096},
	{0x3099, 0x30FF},
	{0x3105, 0x312F},
	{0x3131, 0x31E3},
	{0x31EF, 0x3247},
	{0x3250, 0x4DBF},
	{0x4E00, 0xA48C},
	{0xA490, 0xA4C6},
	{0xA960, 0xA97C},
	{0xAC00, 0xD7A3},
	{0xF900, 0xFAFF},
	{0xFE10, 0xFE19},
	{0xFE30, 0xFE6B},
	{0xFF01, 0xFF60},
	{0xFFE0, 0xFFE6},
	// 表意文字記号・西夏文字・仮名補助
	{0x16FE0, 0x16FE4},
	{0x16FF0, 0x16FF1},
	{0x17000, 0x187F7},
	{0x18800, 0x18CD5},
	{0x18D00, 0x18D08},
	{0x1AFF0, 0x1AFF3},
	{0x1AFF5, 0x1AFFB},
	{0x1AFFD, 0x1AFFE},
	{0x1B000, 0x1B122},
	{0x1B132, 0x1B132},
	{0x1B150, 0x1B152},
	{0x1B155, 0x1B155},
	{0x1B164, 0x1B167},
	{0x1B170, 0x1B2FB},
	// 囲み CJK 補助・絵文字・レガシー計算記号
	{0x1F004, 0x1F004},
	{0x1F0CF, 0x1F0CF},
	{0x1F18E, 0x1F18E},
	{0x1F191, 0x1F19A},
	{0x1F1AD, 0x1F1AD},
	{0x1F200, 0x1F202},
	{0x1F210, 0x1F23B},
	{0x1F240, 0x1F248},
	{0x1F250, 0x1F251},
	{0x1F260, 0x1F265},
	{0x1F300, 0x1F320},
	{0x1F32D, 0x1F335},
	{0x1F337, 0x1F37C},
	{0x1F37E, 0x1F393},
	{0x1F3A0, 0x1F3CA},
	{0x1F3CF, 0x1F3D3},
	{0x1F3E0, 0x1F3F0},
	{0x1F3F4, 0x1F3F4},
	{0x1F3F8, 0x1F43E},
	{0x1F440, 0x1F440},
	{0x1F442, 0x1F4FC},
	{0x1F4FF, 0x1F53D},
	{0x1F54B, 0x1F54E},
	{0x1F550, 0x1F567},
	{0x1F57A, 0x1F57A},
	{0x1F595, 0x1F596},
	{0x1F5A4, 0x1F5A4},
	{0x1F5FB, 0x1F64F},
	{0x1F680, 0x1F6C5},
	{0x1F6CC, 0x1F6CC},
	{0x1F6D0, 0x1F6D2},
	{0x1F6D5, 0x1F6D7},
	{0x1F6DC, 0x1F6DF},
	{0x1F6EB, 0x1F6EC},
	{0x1F6F4, 0x1F6FC},
	{0x1F7E0, 0x1F7EB},
	{0x1F7F0, 0x1F7F0},
	{0x1F90C, 0x1F93A},
	{0x1F93C, 0x1F945},
	{0x1F947, 0x1FA53},
	{0x1FA60, 0x1FA6D},
	{0x1FA70, 0x1FA74},
	{0x1FA78, 0x1FA7C},
	{0x1FA80, 0x1FA88},
	{0x1FA90, 0x1FABD},
	{0x1FABF, 0x1FAC5},
	{0x1FACE, 0x1FADB},
	{0x1FAE0, 0x1FAE8},
	{0x1FAF0, 0x1FAF8},
	{0x1FB00, 0x1FB92},
	{0x20000, 0x3FFFF}, // CJK 統合漢字拡張 B 以降
}

// verbatimDelims はコード表記の区切り記号。強調の区切りは他にもあるが、
// 前後の文字を検査しているのはこの 2 つだけ。
const verbatimDelims = "=~"
