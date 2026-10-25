package orglint

import "testing"

// 期待値は Emacs 31.1 を fill-separate-heterogeneous-words-with-space t
// にして org-fill-paragraph で 2 行をつないだ結果。
func TestFormatJoinsLinesLikeEmacsHeterogeneousSpace(t *testing.T) {
	for _, tc := range []struct {
		name, first, second, want string
	}{
		{"和文どうしは詰める", "設定を", "変える。", "設定を変える。"},
		{"和文と全角の句読点は詰める", "変える。", "次に", "変える。次に"},
		{"全角の英数は和文として詰める", "ＡＢＣ", "を使う。", "ＡＢＣを使う。"},
		{"和文の後の英字は空ける", "検査に", "orglint を使う。", "検査に orglint を使う。"},
		{"英字の後の和文は空ける", "orglint", "の検査。", "orglint の検査。"},
		{"全角の句読点の後の英字も空ける", "変える。", "orglint を使う。", "変える。 orglint を使う。"},
		{"和文の後の半角括弧も空ける", "実施した", "(2026-07-22)。", "実施した (2026-07-22)。"},
		{"和文の後の二重引用符も空ける", "設定では", "\"-x\" を渡す。", "設定では \"-x\" を渡す。"},
		{"コード表記の区切りと和文は空ける", "設定は", "=foo= で変える。", "設定は =foo= で変える。"},
		{"リンクの閉じ括弧は説明文が和文でも空ける", "[[id:x][行幅]]", "の検査。", "[[id:x][行幅]] の検査。"},
		{"曖昧幅の記号は半角として空ける", "A →", "あ", "A → あ"},
		{"欧文どうしは空ける", "fill", "column", "fill column"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := format(t, tc.first+"\n"+tc.second+"\n"); got != tc.want+"\n" {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want+"\n")
			}
		})
	}
}
