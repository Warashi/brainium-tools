package notesearch_test

import (
	"slices"
	"testing"
	"testing/fstest"

	"github.com/Warashi/brainium-tools/internal/notesearch"
)

func corpus() fstest.MapFS {
	return fstest.MapFS{
		"maturity.org": {Data: []byte("#+title: ノートの成熟度\ndraft のノートを検証して stable に上げる。\n")},
		"width.org":    {Data: []byte("#+title: 行の折り返し\n本文は 70 桁で折り返す。\n")},
		"search.org":   {Data: []byte("#+title: 全文検索\nBM25 で検索する。検索は grep より速い。\n")},
		"README.md":    {Data: []byte("prometheus は org ではないファイルにしか出ない\n")},
	}
}

func paths(results []notesearch.Result) []string {
	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.Path
	}
	return out
}

func TestSearch(t *testing.T) {
	t.Parallel()

	index, err := notesearch.Build(corpus(), nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	cases := []struct {
		name  string
		query string
		top   string
	}{
		{
			name:  "クエリ語を最も多く含む文書が 1 位になる",
			query: "検索",
			top:   "search.org",
		},
		{
			name:  "表現がずれていても部分的に重なる文書が 1 位になる",
			query: "ノートを draft から stable に上げる条件",
			top:   "maturity.org",
		},
		{
			name:  "英語のクエリでも一致する",
			query: "BM25",
			top:   "search.org",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := index.Search(c.query)
			if len(got) == 0 {
				t.Fatalf("expected at least one result, got none")
			}
			if got[0].Path != c.top {
				t.Errorf("expected %q at the top, got %q", c.top, paths(got))
			}
		})
	}
}

func TestSearchRanksRelevantDocumentAboveIncidentalMatches(t *testing.T) {
	t.Parallel()

	index, err := notesearch.Build(corpus(), nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 日本語は 1-gram まで展開するので、無関係な文書にも「し」のような 1 文字が
	// 当たって低スコアで並ぶ。BM25 が返すのは真偽ではなく順位なので、狙いの
	// 文書が上位に来ていれば足りる
	got := index.Search("折り返し")
	if len(got) == 0 {
		t.Fatalf("expected at least one result, got none")
	}
	if got[0].Path != "width.org" {
		t.Errorf("expected width.org at the top, got %q", paths(got))
	}
	for _, r := range got {
		if r.Score <= 0 {
			t.Errorf("expected positive score, got %v for %q", r.Score, r.Path)
		}
	}
}

func TestSearchReturnsNothingWhenNoTermMatches(t *testing.T) {
	t.Parallel()

	index, err := notesearch.Build(corpus(), nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if got := index.Search("kubernetes"); len(got) != 0 {
		t.Errorf("expected no results, got %q", paths(got))
	}
}

func TestBuildRejectsEmptyCorpus(t *testing.T) {
	t.Parallel()

	if _, err := notesearch.Build(fstest.MapFS{}, nil); err == nil {
		t.Errorf("expected an error for an empty corpus, got nil")
	}

}

func TestSourcePreservesTheIndexedSnapshot(t *testing.T) {
	fsys := corpus()
	want := string(fsys["search.org"].Data)
	index, err := notesearch.Build(fsys, nil)
	if err != nil {
		t.Fatal(err)
	}
	fsys["search.org"].Data = []byte("Changed after indexing.")
	got, ok := index.Source("search.org")
	if !ok || got != want {
		t.Fatalf("source must match the indexed snapshot: %q, %v", got, ok)
	}
	if _, ok := index.Source("missing.org"); ok {
		t.Fatal("missing source reported as present")
	}
}

func TestSearchIndexesOnlyOrgFiles(t *testing.T) {
	t.Parallel()

	index, err := notesearch.Build(corpus(), nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// prometheus は README.md にしか出ない語
	if got := index.Search("prometheus"); len(got) != 0 {
		t.Errorf("expected non-org files to be excluded from the index, got %q", paths(got))
	}
}

func TestSearchMatchesTermsFoundOnlyInTheFileName(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"terminal-idle-watch.org": {Data: []byte("#+title: 待ち時間の検知\n端末の出力が止まったことを見る。\n")},
		"other.org":               {Data: []byte("#+title: 別のノート\n無関係な内容。\n")},
	}

	index, err := notesearch.Build(fsys, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// idle は本文に無く、ファイル名のスラグにしか出ない語
	got := index.Search("idle")
	if len(got) == 0 {
		t.Fatalf("expected the file name to be indexed, got no results")
	}
	if got[0].Path != "terminal-idle-watch.org" {
		t.Errorf("expected terminal-idle-watch.org at the top, got %q", paths(got))
	}
}

func TestAnchorField(t *testing.T) {
	t.Parallel()

	anchors := map[string][]string{"search.org": {"インデックスの張り方"}}
	index, err := notesearch.Build(corpus(), anchors)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	t.Run("重み 0 ならリンク文だけの語では引けない", func(t *testing.T) {
		t.Parallel()
		if got := index.SearchWith("インデックス", notesearch.Params{}); len(got) != 0 {
			t.Errorf("結果 = %v, want 空", paths(got))
		}
	})

	t.Run("重みを与えるとリンク文だけの語で引ける", func(t *testing.T) {
		t.Parallel()
		got := index.SearchWith("インデックス", notesearch.Params{AnchorW: 3})
		if len(got) == 0 || got[0].Path != "search.org" {
			t.Errorf("1 位 = %v, want search.org", paths(got))
		}
	})

	t.Run("重み 0 の順位は基準の BM25 と一致する", func(t *testing.T) {
		t.Parallel()
		want := paths(index.Search("検索"))
		got := paths(index.SearchWith("検索", notesearch.Params{}))
		if !slices.Equal(got, want) {
			t.Errorf("順位 = %v, want %v", got, want)
		}
	})
}

func TestTranslitExpansion(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"build-agent.org": {Data: []byte("#+title: 手元の手伝い\nagent を手元の機械で動かす。\n")},
		"other.org":       {Data: []byte("#+title: 別の記事\n無関係な内容。\n")},
	}
	index, err := notesearch.Build(fsys, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// エージェント は本文にもスラグにも無く、agent としか書かれていない
	t.Run("重み 0 ならカタカナのクエリで引けない", func(t *testing.T) {
		t.Parallel()
		if got := index.SearchWith("エージェント", notesearch.Params{}); len(got) != 0 {
			t.Errorf("結果 = %v, want 空", paths(got))
		}
	})

	t.Run("音写が当たればカタカナのクエリで引ける", func(t *testing.T) {
		t.Parallel()
		p := notesearch.Params{TranslitDist: 1, TranslitW: 1}
		got := index.SearchWith("エージェント", p)
		if len(got) == 0 || got[0].Path != "build-agent.org" {
			t.Errorf("1 位 = %v, want build-agent.org", paths(got))
		}
	})
}
