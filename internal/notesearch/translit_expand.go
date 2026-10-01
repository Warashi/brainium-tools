package notesearch

import (
	"maps"
	"slices"

	"github.com/Warashi/brainium-tools/internal/notesearch/translit"
)

// minSkeleton は照合に使う骨格の最短の長さ。子音 1 つの骨格は編集距離 1
// で何にでも当たるので落とす。
const minSkeleton = 2

// translitExpand はクエリのカタカナ語と骨格が近い索引のラテン文字語を、
// クエリ語として足す。
//
// 文書側を書き換えず、クエリ側だけを広げる。索引の構造も 1-gram から
// 5-gram への展開も触らずに済み、6 文字のカタカナ語が n-gram の上限を
// 超える問題もここでは関係なくなる。
func (ix *Index) translitExpand(query string, base []wterm, p Params) []wterm {
	if p.TranslitW == 0 {
		return base
	}

	in := make(map[string]bool, len(base))
	for _, t := range base {
		in[t.term] = true
	}

	out := slices.Clone(base)
	added := make(map[string]bool)
	for _, run := range translit.KatakanaRuns(query) {
		k := translit.KatakanaSkeleton(run)
		if len(k) < minSkeleton {
			continue
		}
		for term, s := range ix.latinSkeletons() {
			if in[term] || added[term] || len(s) < minSkeleton {
				continue
			}
			if translit.Distance(k, s) <= allowance(k, p) {
				added[term] = true
			}
		}
	}

	// 追加の順を固定して、実行ごとに同点の並びが揺れないようにする
	for _, term := range slices.Sorted(maps.Keys(added)) {
		out = append(out, wterm{term, p.TranslitW})
	}
	return out
}

var latinSkeletonCache map[*Index]map[string]string

func (ix *Index) latinSkeletons() map[string]string {
	if latinSkeletonCache == nil {
		latinSkeletonCache = map[*Index]map[string]string{}
	}
	if s, ok := latinSkeletonCache[ix]; ok {
		return s
	}
	s := map[string]string{}
	for _, t := range ix.scriptTerms()[1] {
		if !isSpelledWord(t) {
			continue
		}
		s[t] = translit.LatinSkeleton(t)
	}
	latinSkeletonCache[ix] = s
	return s
}

// allowance は骨格 k を照合するときの編集距離の許容。
//
// 短い骨格ほど許容を絞る。子音 2 つ 3 つの骨格は距離 1 で語彙のほとんど
// に当たるので、一律の許容では長いカタカナ語で得た精度が短い語の洪水に
// 打ち消される。
func allowance(k string, p Params) int {
	if p.TranslitLoose > 0 && len(k) < p.TranslitLoose {
		return 0
	}
	return p.TranslitDist
}

// isSpelledWord は英単語として音写の相手にしてよい語かどうかを返す。
//
// 数字を含む語を外すのは、索引に :ID: の UUID の断片が大量に入っている
// ため。16 進の断片は綴りとして読めば子音の骨格を持ってしまい、どの
// カタカナ語にも当たる相手が湧く。
func isSpelledWord(term string) bool {
	for _, r := range term {
		if r >= '0' && r <= '9' {
			return false
		}
	}
	return true
}
