package bm25_test

import (
	"testing"

	"github.com/Warashi/brainium-tools/internal/notesearch/bm25"
)

const epsilon = 0.00001

func TestScore(t *testing.T) {
	t.Parallel()

	scorer := bm25.Scorer{NumDocs: 1000, AvgDocLen: 50, K1: 1.2, B: 0.75}

	cases := []struct {
		name     string
		scorer   bm25.Scorer
		stat     bm25.Stat
		docLen   uint64
		expected float64
	}{
		{
			name:     "語がコーパスの一部にだけ出る文書はスコアを持つ",
			scorer:   scorer,
			stat:     bm25.Stat{TermFreq: 10, DocFreq: 100},
			docLen:   100,
			expected: 1.89967,
		},
		{
			name:     "語が出現しない文書のスコアは 0 になる",
			scorer:   scorer,
			stat:     bm25.Stat{TermFreq: 0, DocFreq: 100},
			docLen:   100,
			expected: 0,
		},
		{
			name:     "b=1 では文書長で完全に正規化する",
			scorer:   bm25.Scorer{NumDocs: 1000, AvgDocLen: 50, K1: 1.2, B: 1.0},
			stat:     bm25.Stat{TermFreq: 10, DocFreq: 100},
			docLen:   100,
			expected: 1.85371,
		},
		{
			name:     "b=0 では文書長を無視する",
			scorer:   bm25.Scorer{NumDocs: 1000, AvgDocLen: 50, K1: 1.2, B: 0},
			stat:     bm25.Stat{TermFreq: 10, DocFreq: 100},
			docLen:   100,
			expected: 2.05232,
		},
		{
			name:     "全文書に出現する語でもスコアは負にならない",
			scorer:   bm25.Scorer{NumDocs: 40, AvgDocLen: 50, K1: 1.2, B: 0.75},
			stat:     bm25.Stat{TermFreq: 3, DocFreq: 40},
			docLen:   100,
			expected: 0.00722,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := c.scorer.Score(c.stat, c.docLen)
			if got < c.expected-epsilon || c.expected+epsilon < got {
				t.Errorf("expected %v, got %v", c.expected, got)
			}
		})
	}
}

func TestScoreIgnoresDocLenWhenBIsZero(t *testing.T) {
	t.Parallel()

	scorer := bm25.Scorer{NumDocs: 1000, AvgDocLen: 50, K1: 1.2, B: 0}
	stat := bm25.Stat{TermFreq: 10, DocFreq: 100}

	short := scorer.Score(stat, 100)
	long := scorer.Score(stat, 200)
	if short < long-epsilon || long+epsilon < short {
		t.Errorf("expected same score regardless of docLen, got %v and %v", short, long)
	}
}

func TestScoreTerms(t *testing.T) {
	t.Parallel()

	scorer := bm25.Scorer{NumDocs: 1000, AvgDocLen: 50, K1: 1.2, B: 0.75}

	cases := []struct {
		name     string
		stats    []bm25.Stat
		docLen   uint64
		expected float64
	}{
		{
			name:     "複数語のスコアは各語のスコアの合計になる",
			stats:    []bm25.Stat{{TermFreq: 10, DocFreq: 100}, {TermFreq: 5, DocFreq: 200}},
			docLen:   100,
			expected: 3.03202,
		},
		{
			name:     "語が 1 つもなければスコアは 0 になる",
			stats:    nil,
			docLen:   100,
			expected: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := scorer.ScoreTerms(c.stats, c.docLen)
			if got < c.expected-epsilon || c.expected+epsilon < got {
				t.Errorf("expected %v, got %v", c.expected, got)
			}
		})
	}
}
