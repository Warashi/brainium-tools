package notesearch

import "github.com/Warashi/brainium-tools/internal/notesearch/bm25"

// field は Params に応じた統計の見方。リンク文フィールドが無効なら本文
// だけの統計を、有効ならリンク文を畳み込んだ統計を返す。
type field struct {
	scorer  bm25.Scorer
	docFreq map[string]uint64
	w       float64
}

func (ix *Index) field(p Params) field {
	if p.AnchorW == 0 {
		return field{scorer: ix.scorer, docFreq: ix.docFreq}
	}
	return field{scorer: ix.scorer, docFreq: ix.docFreqAll, w: p.AnchorW}
}

// cell は 1 語に対する 1 文書のスコア。
//
// 文書長にリンク文の語数を足さないのは、BM25F でリンク文側の b を 0 に
// 置いたことに当たる。リンク文は本数がノートの内容ではなく被リンク数で
// 決まるので、長さで割ると多く参照されているノートほど減点される。
func (ix *Index) cell(f field, term string, d document) float64 {
	tf := float64(d.freq[term]) + f.w*float64(d.anchorFreq[term])
	if tf == 0 {
		return 0
	}
	return f.scorer.Score(bm25.Stat{TermFreq: tf, DocFreq: f.docFreq[term]}, d.len)
}

// has は文書がその語を持つかどうか。スコアが 0 の文書を返さない判定に使う。
func (f field) has(term string, d document) bool {
	return d.freq[term] > 0 || (f.w > 0 && d.anchorFreq[term] > 0)
}
