package orglint

import (
	"strings"
	"testing"
)

// 期待値は Emacs 31.1 の org-fill-paragraph の結果。
func TestFormatAppliesKinsokuOnlyNextToNonASCII(t *testing.T) {
	head := strings.Repeat("あ", 33)
	for _, tc := range []struct {
		name, body, want string
	}{
		{"空白で折り返すなら半角の開き括弧を行末に残す",
			head + " ( foo bar baz", head + " (\nfoo bar baz"},
		{"和文に接していれば半角の開き括弧を次の行へ送る",
			head + "い(いいいい", head + "い\n(いいいい"},
		{"全角の開き括弧は空白で折り返しても次の行へ送る",
			head + " （ foo bar baz", head + "\n（ foo bar baz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := format(t, tc.body+"\n"); got != tc.want+"\n" {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want+"\n")
			}
		})
	}
}
