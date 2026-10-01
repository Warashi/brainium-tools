package anchor_test

import (
	"maps"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/Warashi/brainium-tools/internal/notesearch/anchor"
)

func repo() fstest.MapFS {
	return fstest.MapFS{
		"notes/search.org": {Data: []byte(
			":PROPERTIES:\n:ID:       aaaaaaaa-1111-1111-1111-111111111111\n:END:\n" +
				"#+title: 全文検索\nBM25 で検索する。\n")},
		"notes/maturity.org": {Data: []byte(
			":PROPERTIES:\n:ID:       22222222-2222-2222-2222-222222222222\n:END:\n" +
				"#+title: ノートの成熟度\n" +
				"[[id:aaaaaaaa-1111-1111-1111-111111111111][索引の張り方]]を参照。\n")},
		"tasks.org": {Data: []byte(
			"* TODO [[id:aaaaaaaa-1111-1111-1111-111111111111][検索が\n外れる件]]を直す\n" +
				"[[id:22222222-2222-2222-2222-222222222222][note maturity\nの設計]]\n" +
				"[[id:33333333-3333-3333-3333-333333333333][行き先の無いリンク]]\n" +
				"[[id:aaaaaaaa-1111-1111-1111-111111111111]]\n")},
		"journal/2026/2026-09-05.org": {Data: []byte(
			"[[id:AAAAAAAA-1111-1111-1111-111111111111][大文字の UUID]]\n")},
		".git/hooks/sample.org": {Data: []byte(
			"[[id:aaaaaaaa-1111-1111-1111-111111111111][隠しディレクトリ]]\n")},
		"README.md": {Data: []byte("[[id:aaaaaaaa-1111-1111-1111-111111111111][org ではない]]\n")},
	}
}

func TestCollect(t *testing.T) {
	t.Parallel()

	got, err := anchor.Collect(repo())
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	want := map[string][]string{
		"search.org":   {"索引の張り方", "検索が外れる件", "大文字の UUID"},
		"maturity.org": {"note maturity の設計"},
	}

	if keys := slices.Sorted(maps.Keys(got)); !slices.Equal(keys, slices.Sorted(maps.Keys(want))) {
		t.Fatalf("引き当てたノート = %v, want %v", keys, slices.Sorted(maps.Keys(want)))
	}
	for path, w := range want {
		slices.Sort(w)
		g := slices.Clone(got[path])
		slices.Sort(g)
		if !slices.Equal(g, w) {
			t.Errorf("%s のリンク文 = %q, want %q", path, g, w)
		}
	}
}
