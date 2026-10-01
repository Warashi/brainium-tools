package translit_test

import (
	"slices"
	"testing"

	"github.com/Warashi/brainium-tools/internal/notesearch/translit"
)

func TestSkeletonMatchesAcrossScripts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		katakana string
		english  string
	}{
		{"エージェント", "agent"},
		{"ノート", "note"},
		{"タスク", "task"},
		{"リンク", "link"},
		{"インデックス", "index"},
		{"セッション", "session"},
		{"ファイル", "file"},
		{"チーム", "team"},
	}

	for _, c := range cases {
		t.Run(c.katakana, func(t *testing.T) {
			t.Parallel()
			k, e := translit.KatakanaSkeleton(c.katakana), translit.LatinSkeleton(c.english)
			if translit.Distance(k, e) > 1 {
				t.Errorf("%s = %q と %s = %q の距離が 1 を超える", c.katakana, k, c.english, e)
			}
		})
	}
}

func TestSkeletonSeparatesUnrelatedWords(t *testing.T) {
	t.Parallel()

	k := translit.KatakanaSkeleton("エージェント")
	for _, w := range []string{"emacs", "search", "org", "commit"} {
		if d := translit.Distance(k, translit.LatinSkeleton(w)); d <= 1 {
			t.Errorf("エージェント と %s の距離が %d で、無関係な語が当たる", w, d)
		}
	}
}

func TestKatakanaRunsIgnoresOtherScripts(t *testing.T) {
	t.Parallel()

	got := translit.KatakanaRuns("エージェントのセッションを記録する")
	want := []string{"エージェント", "セッション"}
	if !slices.Equal(got, want) {
		t.Errorf("カタカナ区間 = %q, want %q", got, want)
	}
}
