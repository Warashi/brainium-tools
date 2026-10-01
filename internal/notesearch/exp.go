// この file の LSI / Dice / PRF は凍結した評価セットの hold-out で BM25 に
// 勝てず、既定の検索には入れていない。消さないのは評価クエリ集に添えて
// 記録した数値を notesearch-eval で再現できるようにするため。
package notesearch

import (
	"cmp"
	"math"
	"slices"
	"unicode"

	"github.com/Warashi/brainium-tools/internal/notesearch/tokenize"
)

// Params は実験用の調整つまみ。0 値はその段をオフにする。
type Params struct {
	LSIRank   int     // 低ランク近似の次元。0 で LSI を使わない
	LSILambda float64 // 順位融合での LSI 側の係数
	DiceK     int     // クエリ語 1 つあたりの Dice 拡張語数。0 で使わない
	DiceMin   float64 // 採用する Dice の下限
	DiceW     float64 // 拡張語の重み
	PRFDocs   int     // 擬似適合性フィードバックで見る上位件数。0 で使わない
	PRFTerms  int     // 拡張語数
	PRFAlpha  float64 // 元クエリ側の重み
	AnchorW   float64 // リンク文フィールドの重み。0 で本文だけを見る

	TranslitDist  int     // 音写の骨格を照合するときの編集距離の許容
	TranslitLoose int     // 許容を効かせる骨格の最短の長さ。0 なら長さによらず許容する
	TranslitW     float64 // 音写で足した語の重み。0 で音写照合を使わない
}

type wterm struct {
	term string
	w    float64
}

func (ix *Index) idf(term string) float64 {
	df := float64(ix.docFreq[term])
	return math.Log(1 + (float64(ix.scorer.NumDocs)-df+0.5)/(df+0.5))
}

func (ix *Index) scoreWeighted(terms []wterm, f field) []Result {
	var out []Result
	for _, d := range ix.docs {
		var sum float64
		var hit bool
		for _, t := range terms {
			if !f.has(t.term, d) {
				continue
			}
			hit = true
			sum += t.w * ix.cell(f, t.term, d)
		}
		if !hit {
			continue
		}
		out = append(out, Result{Path: d.path, Score: sum})
	}
	sortResults(out)
	return out
}

func sortResults(rs []Result) {
	slices.SortFunc(rs, func(a, b Result) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.Path, b.Path)
	})
}

// --- LSI ---

type lsiModel struct {
	sigma []float64   // 特異値(降順)
	v     [][]float64 // 右特異ベクトル。v[doc][dim]
}

var lsiCache map[*Index]*lsiModel

func (ix *Index) lsi() *lsiModel {
	if lsiCache == nil {
		lsiCache = map[*Index]*lsiModel{}
	}
	if m, ok := lsiCache[ix]; ok {
		return m
	}

	n := len(ix.docs)
	g := make([][]float64, n)
	for i := range g {
		g[i] = make([]float64, n)
	}
	postings := map[string][]int{}
	for i, d := range ix.docs {
		for t := range d.freq {
			postings[t] = append(postings[t], i)
		}
	}
	for t, ds := range postings {
		w := make([]float64, len(ds))
		for k, i := range ds {
			w[k] = ix.cell(ix.field(Params{}), t, ix.docs[i])
		}
		for a := range ds {
			for b := range ds {
				g[ds[a]][ds[b]] += w[a] * w[b]
			}
		}
	}

	lam, vec := jacobi(g)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int { return cmp.Compare(lam[b], lam[a]) })

	m := &lsiModel{sigma: make([]float64, n), v: make([][]float64, n)}
	for j := 0; j < n; j++ {
		l := lam[order[j]]
		if l < 1e-12 {
			l = 0
		}
		m.sigma[j] = math.Sqrt(l)
	}
	for i := 0; i < n; i++ {
		m.v[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			m.v[i][j] = vec[i][order[j]]
		}
	}
	lsiCache[ix] = m
	return m
}

func jacobi(a [][]float64) ([]float64, [][]float64) {
	n := len(a)
	m := make([][]float64, n)
	for i := range m {
		m[i] = slices.Clone(a[i])
	}
	v := make([][]float64, n)
	for i := range v {
		v[i] = make([]float64, n)
		v[i][i] = 1
	}
	for sweep := 0; sweep < 100; sweep++ {
		off := 0.0
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				off += m[i][j] * m[i][j]
			}
		}
		if off < 1e-18 {
			break
		}
		for p := 0; p < n; p++ {
			for q := p + 1; q < n; q++ {
				if math.Abs(m[p][q]) < 1e-15 {
					continue
				}
				theta := (m[q][q] - m[p][p]) / (2 * m[p][q])
				t := math.Copysign(1, theta) / (math.Abs(theta) + math.Sqrt(theta*theta+1))
				c := 1 / math.Sqrt(t*t+1)
				s := t * c
				for k := 0; k < n; k++ {
					x, y := m[k][p], m[k][q]
					m[k][p], m[k][q] = c*x-s*y, s*x+c*y
				}
				for k := 0; k < n; k++ {
					x, y := m[p][k], m[q][k]
					m[p][k], m[q][k] = c*x-s*y, s*x+c*y
				}
				for k := 0; k < n; k++ {
					x, y := v[k][p], v[k][q]
					v[k][p], v[k][q] = c*x-s*y, s*x+c*y
				}
			}
		}
	}
	lam := make([]float64, n)
	for i := range lam {
		lam[i] = m[i][i]
	}
	return lam, v
}

func (ix *Index) searchLSI(terms []wterm, k int) []Result {
	m := ix.lsi()
	n := len(ix.docs)
	if k > n {
		k = n
	}

	a := make([]float64, n)
	for _, t := range terms {
		if ix.docFreq[t.term] == 0 {
			continue
		}
		for d := range ix.docs {
			a[d] += t.w * ix.cell(ix.field(Params{}), t.term, ix.docs[d])
		}
	}

	qhat := make([]float64, k)
	for j := 0; j < k; j++ {
		if m.sigma[j] == 0 {
			continue
		}
		var dot float64
		for d := 0; d < n; d++ {
			dot += a[d] * m.v[d][j]
		}
		qhat[j] = dot / (m.sigma[j] * m.sigma[j])
	}
	var qn float64
	for _, x := range qhat {
		qn += x * x
	}
	qn = math.Sqrt(qn)
	if qn == 0 {
		return nil
	}

	var out []Result
	for i := 0; i < n; i++ {
		var dn, dot float64
		for j := 0; j < k; j++ {
			dn += m.v[i][j] * m.v[i][j]
			dot += m.v[i][j] * qhat[j]
		}
		dn = math.Sqrt(dn)
		if dn == 0 {
			continue
		}
		out = append(out, Result{Path: ix.docs[i].path, Score: dot / (qn * dn)})
	}
	sortResults(out)
	return out
}

func fuse(a, b []Result, lambda float64) []Result {
	const rrfK = 60
	score := map[string]float64{}
	for i, r := range a {
		score[r.Path] += 1 / (rrfK + float64(i+1))
	}
	for i, r := range b {
		score[r.Path] += lambda / (rrfK + float64(i+1))
	}
	var out []Result
	for p, s := range score {
		out = append(out, Result{Path: p, Score: s})
	}
	sortResults(out)
	return out
}

// --- Dice によるクエリ拡張 ---

func isLatin(s string) bool {
	for _, r := range s {
		if !unicode.Is(unicode.Latin, r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

var postingsCache map[*Index]map[string]map[int]bool

func (ix *Index) postings() map[string]map[int]bool {
	if postingsCache == nil {
		postingsCache = map[*Index]map[string]map[int]bool{}
	}
	if p, ok := postingsCache[ix]; ok {
		return p
	}
	p := map[string]map[int]bool{}
	for i, d := range ix.docs {
		for t := range d.freq {
			if p[t] == nil {
				p[t] = map[int]bool{}
			}
			p[t][i] = true
		}
	}
	postingsCache[ix] = p
	return p
}

// diceExpand はクエリ語ごとに、字種をまたぐ共起語を Dice 係数で拾う。
// 重みは Dice に idf を掛けた値にして、遍在語が高い係数を持っても効かないようにする。
func (ix *Index) diceExpand(base []wterm, p Params) []wterm {
	if p.DiceK == 0 {
		return base
	}
	post := ix.postings()
	in := map[string]bool{}
	for _, t := range base {
		in[t.term] = true
	}

	type kv struct {
		t string
		v float64
	}
	var added []kv
	for _, q := range base {
		pq := post[q.term]
		if len(pq) < 2 {
			continue
		}
		var cands []kv
		var side []string
		if isLatin(q.term) {
			side = ix.scriptTerms()[0]
		} else {
			side = ix.scriptTerms()[1]
		}
		for _, t := range side {
			pt := post[t]
			if in[t] || len(pt) < 2 {
				continue
			}
			inter := 0
			for i := range pq {
				if pt[i] {
					inter++
				}
			}
			if inter == 0 {
				continue
			}
			dice := 2 * float64(inter) / float64(len(pq)+len(pt))
			if dice < p.DiceMin {
				continue
			}
			cands = append(cands, kv{t, dice * ix.idf(t)})
		}
		slices.SortFunc(cands, func(a, b kv) int {
			if c := cmp.Compare(b.v, a.v); c != 0 {
				return c
			}
			return cmp.Compare(a.t, b.t)
		})
		if len(cands) > p.DiceK {
			cands = cands[:p.DiceK]
		}
		added = append(added, cands...)
	}

	out := slices.Clone(base)
	seen := map[string]bool{}
	var maxv float64
	for _, a := range added {
		maxv = max(maxv, a.v)
	}
	for _, a := range added {
		if seen[a.t] || maxv == 0 {
			continue
		}
		seen[a.t] = true
		out = append(out, wterm{a.t, p.DiceW * a.v / maxv})
	}
	return out
}

// --- 擬似適合性フィードバック ---

func (ix *Index) prfExpand(base []wterm, seed []Result, p Params) []wterm {
	if p.PRFDocs == 0 || len(seed) == 0 {
		return base
	}
	if len(seed) > p.PRFDocs {
		seed = seed[:p.PRFDocs]
	}
	in := map[string]bool{}
	for _, t := range base {
		in[t.term] = true
	}

	byPath := map[string]document{}
	for _, d := range ix.docs {
		byPath[d.path] = d
	}

	cand := map[string]float64{}
	for _, r := range seed {
		d := byPath[r.Path]
		for t := range d.freq {
			if in[t] {
				continue
			}
			// 拡張語の採点も BM25 の重みで行う。生の tf/dl だと遍在語が上に来る
			cand[t] += ix.cell(ix.field(Params{}), t, d)
		}
	}

	type kv struct {
		t string
		v float64
	}
	all := make([]kv, 0, len(cand))
	for t, v := range cand {
		all = append(all, kv{t, v})
	}
	slices.SortFunc(all, func(a, b kv) int {
		if c := cmp.Compare(b.v, a.v); c != 0 {
			return c
		}
		return cmp.Compare(a.t, b.t)
	})
	if len(all) > p.PRFTerms {
		all = all[:p.PRFTerms]
	}
	var norm float64
	for _, e := range all {
		norm += e.v
	}
	if norm == 0 {
		return base
	}

	out := make([]wterm, 0, len(base)+len(all))
	for _, b := range base {
		out = append(out, wterm{b.term, p.PRFAlpha * b.w})
	}
	for _, e := range all {
		out = append(out, wterm{e.t, (1 - p.PRFAlpha) * e.v / norm * float64(len(base))})
	}
	return out
}

// SearchWith は Params に従って BM25 / Dice / LSI / PRF を組み合わせて引く。
func (ix *Index) SearchWith(query string, p Params) []Result {
	terms := slices.Compact(slices.Sorted(slices.Values(tokenize.Tokens(query))))
	base := make([]wterm, 0, len(terms))
	for _, t := range terms {
		base = append(base, wterm{t, 1})
	}

	q := ix.diceExpand(ix.translitExpand(query, base, p), p)
	f := ix.field(p)

	rank := ix.scoreWeighted(q, f)
	if p.LSIRank > 0 {
		rank = fuse(rank, ix.searchLSI(q, p.LSIRank), p.LSILambda)
	}
	if p.PRFDocs > 0 {
		q2 := ix.prfExpand(q, rank, p)
		rank = ix.scoreWeighted(q2, f)
		if p.LSIRank > 0 {
			rank = fuse(rank, ix.searchLSI(q2, p.LSIRank), p.LSILambda)
		}
	}
	return rank
}

var scriptCache map[*Index][2][]string

// scriptTerms は索引の語をラテン文字とそれ以外に分けて返す。
// 字種をまたぐ組だけを見るので、全語彙を舐めずに反対側だけを走査する。
func (ix *Index) scriptTerms() [2][]string {
	if scriptCache == nil {
		scriptCache = map[*Index][2][]string{}
	}
	if s, ok := scriptCache[ix]; ok {
		return s
	}
	var s [2][]string
	for t := range ix.postings() {
		if isLatin(t) {
			s[1] = append(s[1], t)
		} else {
			s[0] = append(s[0], t)
		}
	}
	slices.Sort(s[0])
	slices.Sort(s[1])
	scriptCache[ix] = s
	return s
}
