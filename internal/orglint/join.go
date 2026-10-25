package orglint

import "unicode"

// joinLine は折り返された 2 行をつなぐ。Emacs で
// fill-separate-heterogeneous-words-with-space を t にした fill と同じく、
// つなぎ目の両側がどちらも nospaceRanges の文字なら詰め、それ以外は空白
// 1 個で区切る。判定は表示ではなく生の文字で行うので、リンクの ]] や
// コード表記の = も半角の文字として扱う。
func joinLine(a, b string) string {
	if a == "" || b == "" {
		return a + b
	}
	if unicode.Is(nospaceRanges, lastRune(a)) && unicode.Is(nospaceRanges, firstRune(b)) {
		return a + b
	}
	return a + " " + b
}

// nospaceRanges は Emacs 31.1 の fill-nospace-between-words-table で値を
// 持つ範囲を、map-char-table でそのまま書き出したもの。
var nospaceRanges = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x02EA, 0x02EB, 1},
		{0x2E80, 0x2FDF, 1},
		{0x3000, 0x312F, 1},
		{0x31A0, 0x9FFF, 1},
		{0xF900, 0xFAFF, 1},
		{0xFE30, 0xFE4F, 1},
		{0xFF00, 0xFFEF, 1},
	},
	R32: []unicode.Range32{
		{0x1AFF0, 0x1B16F, 1},
		{0x1F200, 0x1F200, 1},
		{0x1F202, 0x1F219, 1},
		{0x1F21B, 0x1F22E, 1},
		{0x1F230, 0x1F231, 1},
		{0x1F237, 0x1F237, 1},
		{0x1F23B, 0x1F24F, 1},
		{0x1F252, 0x1F2FF, 1},
		{0x20000, 0x2A6DF, 1},
		{0x2A700, 0x2EE5F, 1},
		{0x2F800, 0x2FA1F, 1},
		{0x30000, 0x3347F, 1},
	},
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func lastRune(s string) rune {
	var last rune
	for _, r := range s {
		last = r
	}
	return last
}
