// Package notesearch は org ノートの集合に BM25 の索引を張り、
// クエリに対して関連度の高い順にファイルを返す。
package notesearch

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/Warashi/brainium-tools/internal/notesearch/bm25"
	"github.com/Warashi/brainium-tools/internal/notesearch/tokenize"
)

// BM25 の自由パラメータ。Lucene の BM25Similarity の既定値に合わせている。
const (
	defaultK1 = 1.2
	defaultB  = 0.75
)

// Result は 1 件の検索結果。
type Result struct {
	Path  string
	Score float64
}

// Index はノート集合の転置索引。Build で作り、Search で引く。
type Index struct {
	scorer bm25.Scorer
	docs   []document
	// docFreq は語ごとの、その語を含む文書数 ni
	docFreq map[string]uint64
	// docFreqAll は本文とリンク文のどちらかに語を含む文書数。リンク文
	// フィールドを有効にしたときの ni
	docFreqAll map[string]uint64

	totalLen       uint64
	totalAnchorLen uint64
}

type document struct {
	path   string
	source string
	len    uint64            // 語数 dl
	freq   map[string]uint64 // 語ごとの文書内出現回数 tf

	// リンク文フィールド。本文とは別に数え、Params.AnchorW で重み付ける
	anchorLen  uint64
	anchorFreq map[string]uint64
}

// Build は fsys 直下の org ファイルから索引を作る。anchors は notes/ の
// ファイル名ごとのリンク文で、nil なら本文だけの索引になる。
//
// notes/ はフラット配置でサブディレクトリを作らない規約なので再帰しない。
// 索引は永続化せず毎回作り直す。40 ファイル規模では都度構築で足りる。
func Build(fsys fs.FS, anchors map[string][]string) (*Index, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read corpus directory: %w", err)
	}

	index := &Index{docFreq: make(map[string]uint64), docFreqAll: make(map[string]uint64)}

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".org" {
			continue
		}

		content, err := fs.ReadFile(fsys, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		// 索引はファイル全文を対象にする。#+title: はそのノートを最もよく表す
		// 語が並ぶ行なので落とさない。:ID: の UUID は文書ごとに固有でクエリに
		// 現れないため、混ざっていても検索結果に影響しない
		//
		// ファイル名も本文と同じ重みで索引に入れる。notes/ のファイル名は内容を
		// 表す英語スラグという規約があり、本文が日本語で書かれていても英語の
		// 語彙はここにしか無いことがある
		slug := strings.TrimSuffix(entry.Name(), ".org")
		tokens := tokenize.Tokens(slug + "\n" + string(content))

		doc := document{path: entry.Name(), source: string(content), len: uint64(len(tokens)), freq: make(map[string]uint64)}
		for _, token := range tokens {
			doc.freq[token]++
		}
		for token := range doc.freq {
			index.docFreq[token]++
		}

		anchorTokens := tokenize.Tokens(strings.Join(anchors[entry.Name()], "\n"))
		doc.anchorLen = uint64(len(anchorTokens))
		doc.anchorFreq = make(map[string]uint64)
		for _, token := range anchorTokens {
			doc.anchorFreq[token]++
		}
		for token := range doc.anchorFreq {
			if doc.freq[token] == 0 {
				index.docFreqAll[token]++
			}
		}
		for token := range doc.freq {
			index.docFreqAll[token]++
		}

		index.docs = append(index.docs, doc)
		index.totalLen += doc.len
		index.totalAnchorLen += doc.anchorLen
	}

	if len(index.docs) == 0 {
		return nil, errors.New("no org files to index")
	}

	index.scorer = bm25.Scorer{
		NumDocs:   uint64(len(index.docs)),
		AvgDocLen: float64(index.totalLen) / float64(len(index.docs)),
		K1:        defaultK1,
		B:         defaultB,
	}

	return index, nil
}

// Source は索引を構築した時点の原文を返す。抜粋と順位を同じ版に揃えるため、
// 検索結果の表示時にはファイルを読み直さない。
func (ix *Index) Source(name string) (string, bool) {
	for _, doc := range ix.docs {
		if doc.path == name {
			return doc.source, true
		}
	}
	return "", false
}

// Search は query に関連する文書をスコアの降順で返す。
// スコアが 0 の文書、つまりクエリの語を 1 つも含まない文書は返らない。
func (ix *Index) Search(query string) []Result {
	// クエリ側は語の重複を数えない。日本語の n-gram 展開では同じ語が何度も
	// 出るが、それは切り出しの副産物であって重みの意図ではない
	terms := slices.Compact(slices.Sorted(slices.Values(tokenize.Tokens(query))))

	var results []Result
	for _, doc := range ix.docs {
		stats := make([]bm25.Stat, 0, len(terms))
		for _, term := range terms {
			if tf := doc.freq[term]; tf > 0 {
				stats = append(stats, bm25.Stat{TermFreq: float64(tf), DocFreq: ix.docFreq[term]})
			}
		}
		if len(stats) == 0 {
			continue
		}
		results = append(results, Result{Path: doc.path, Score: ix.scorer.ScoreTerms(stats, doc.len)})
	}

	// 同点のときの並びをパス順で固定し、実行ごとに結果が揺れないようにする
	slices.SortFunc(results, func(a, b Result) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.Path, b.Path)
	})

	return results
}
