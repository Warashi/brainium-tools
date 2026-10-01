// Package eval は凍結した評価クエリで notesearch のランキングを測る。
//
// クエリ集はノートの内容から作るのでツールの外に置き、作り方と
// train/test の扱いはクエリ集に添える。test は 1 回引いたら使い切りなので、
// このパッケージは split を選ぶ手段は持つが、既定は train にする。
package eval

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/Warashi/brainium-tools/internal/notesearch"
)

// Query は評価クエリ 1 件。Want は notes/ からの相対パス。
// Bilingual は日英併記版の問いで、候補 4 の実験でだけ使う。
type Query struct {
	Text      string
	Want      string
	Split     string
	Gap       bool
	Bilingual string
}

// Load は TSV(query / expected / split / gap / bilingual)を読む。
// 1 行目は見出し。
func Load(r io.Reader) ([]Query, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var out []Query
	line := 0
	for sc.Scan() {
		line++
		if line == 1 || strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		c := strings.Split(sc.Text(), "\t")
		if len(c) != 5 {
			return nil, fmt.Errorf("%d 行目: 列が 5 つでない", line)
		}
		out = append(out, Query{Text: c[0], Want: c[1], Split: c[2], Gap: c[3] == "1", Bilingual: c[4]})
	}
	return out, sc.Err()
}

// Select は split が一致するクエリだけ返す。"all" なら全件。
func Select(qs []Query, split string) []Query {
	var out []Query
	for _, q := range qs {
		if split == "all" || q.Split == split {
			out = append(out, q)
		}
	}
	return out
}

// Metrics は正解ノートの順位から集計した指標。MRR は平均済み。
type Metrics struct {
	N, Hit1, Hit10 int
	MRR            float64
}

func (m Metrics) String() string {
	return fmt.Sprintf("n=%2d hit@1=%2d hit@10=%2d MRR=%.3f", m.N, m.Hit1, m.Hit10, m.MRR)
}

func (m *Metrics) add(rank int) {
	m.N++
	if rank == 1 {
		m.Hit1++
	}
	if rank >= 1 && rank <= 10 {
		m.Hit10++
	}
	if rank > 0 {
		m.MRR += 1 / float64(rank)
	}
}

func (m *Metrics) finish() {
	if m.N > 0 {
		m.MRR /= float64(m.N)
	}
}

// Rank は正解ノートの順位(1 始まり)。上位に無ければ 0。
func Rank(ix *notesearch.Index, q Query, p notesearch.Params) int {
	return rank(ix, q.Text, q, p)
}

// RankBilingual は日英併記版の問いで引いたときの順位。
func RankBilingual(ix *notesearch.Index, q Query, p notesearch.Params) int {
	return rank(ix, q.Bilingual, q, p)
}

func rank(ix *notesearch.Index, text string, q Query, p notesearch.Params) int {
	for i, r := range ix.SearchWith(text, p) {
		if r.Path == q.Want {
			return i + 1
		}
	}
	return 0
}

// Evaluate は全体と gap 部分集合の指標を返す。rank は順位の取り方で、
// 併記版で測るときは RankBilingual を渡す。
func Evaluate(ix *notesearch.Index, qs []Query, p notesearch.Params, rank func(*notesearch.Index, Query, notesearch.Params) int) (all, gap Metrics) {
	for _, q := range qs {
		r := rank(ix, q, p)
		all.add(r)
		if q.Gap {
			gap.add(r)
		}
	}
	all.finish()
	gap.finish()
	return all, gap
}
