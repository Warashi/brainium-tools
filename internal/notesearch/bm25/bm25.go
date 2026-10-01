// Package bm25 は BM25 のスコア計算を提供する。
//
// スコア式は Robertson and Zaragoza, "The Probabilistic Relevance Framework:
// BM25 and Beyond", Foundations and Trends in Information Retrieval, 3(4),
// 2009 の式 (3.2)(3.3)(3.12)-(3.15) に基づく。
package bm25

import (
	"math"
	"slices"
)

// Scorer はコーパス全体の統計と自由パラメータを固定したスコア計算機。
// 同一コーパスへの問い合わせでは使い回す。
type Scorer struct {
	NumDocs   uint64  // コーパス全体の文書数 N
	AvgDocLen float64 // 平均文書長 avdl
	K1        float64 // 出現回数の飽和を決める自由パラメータ k1
	B         float64 // 文書長正規化の強さを決める自由パラメータ b
}

// Stat は 1 語について、対象文書とコーパスから得た出現統計。
type Stat struct {
	// TermFreq が整数でないのは BM25F のため。複数フィールドの tf は飽和の
	// 前に重み付けして足すので、重みが整数でなければ tf も整数にならない。
	TermFreq float64 // 対象文書内の出現回数 tf
	DocFreq  uint64  // その語を含む文書数 ni
}

// Score は 1 語に対する文書のスコアを返す。docLen は対象文書の長さ dl。
func (s Scorer) Score(stat Stat, docLen uint64) float64 {
	// 原論文の式 (3.3) は ni が N に近いと負値を返す。40 文書程度のコーパスでは
	// 日本語 1-gram の「の」「は」がほぼ全文書に出るため、その語を含むだけで
	// 全文書が減点され順位が壊れる。Lucene の BM25Similarity と同じく 1 を
	// 足した形にして常に正に保つ。対数の底も同じ理由で自然対数に合わせている。
	idf := math.Log(1 + (float64(s.NumDocs)-float64(stat.DocFreq)+0.5)/(float64(stat.DocFreq)+0.5))
	norm := 1.0 - s.B + s.B*(float64(docLen)/s.AvgDocLen)
	return idf * stat.TermFreq / (stat.TermFreq + s.K1*norm)
}

// ScoreTerms はクエリを構成する複数語に対する文書のスコアを返す。
func (s Scorer) ScoreTerms(stats []Stat, docLen uint64) float64 {
	scores := make([]float64, len(stats))
	for i, stat := range stats {
		scores[i] = s.Score(stat, docLen)
	}

	// 小さい順に足すことで浮動小数点値の丸め誤差を減らす
	slices.Sort(scores)

	var sum float64
	for _, score := range scores {
		sum += score
	}

	return sum
}
