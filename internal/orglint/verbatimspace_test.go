package orglint

import (
	"strings"
	"testing"
)

func TestFormatSpacesVerbatimNextToJapanese(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"前が和文なら空白を補う", "設定は=foo= で変える。\n", "設定は =foo= で変える。\n"},
		{"後ろが和文なら空白を補う", "設定は =foo=で変える。\n", "設定は =foo= で変える。\n"},
		{"両側が和文なら両側に補う", "これは=foo=です。\n", "これは =foo= です。\n"},
		{"~ も同じく補う", "これは~foo~です。\n", "これは ~foo~ です。\n"},
		{"後ろが和文の句読点なら補っても組まれないので触れない", "これは=foo=。\n", "これは=foo=。\n"},
		{"前が半角スラッシュなら触れない", "path/=foo= です。\n", "path/=foo= です。\n"},
		{"前が全角の句読点なら空白を補う", "例えば、=foo= は組まれる。\n", "例えば、 =foo= は組まれる。\n"},
		{"前が全角の記号なら空白を補う", "表・=#+= 行・〜=x= を除く。\n", "表・ =#+= 行・〜 =x= を除く。\n"},
		{"前が全角の開き括弧なら触れない", "(=foo= です。\n", "(=foo= です。\n"},
		{"等号で挟んだ数式は組まれないので触れない", "長さ=10、幅=20 とする。\n", "長さ=10、幅=20 とする。\n"},
		{"組まれているものには触れない", "これは =foo= です。\n", "これは =foo= です。\n"},
		{"固定幅行には触れない", ": これは=foo=です。\n", ": これは=foo=です。\n"},
		{"ブロックの中には触れない", "#+begin_example\nこれは=foo=です。\n#+end_example\n", "#+begin_example\nこれは=foo=です。\n#+end_example\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := format(t, tc.body); got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestFormatSpacedVerbatimSettlesInOnePass(t *testing.T) {
	// 補った空白が折り返し位置の前後どこに来ても、1 回の整形で正規形に
	// 届き、コード表記が組まれる。
	for k := 28; k <= 38; k++ {
		body := strings.Repeat("あ", k) + "は=orglint=で" + strings.Repeat("い", 40) + "\n"
		once := format(t, body)
		if twice := format(t, once); twice != once {
			t.Errorf("k=%d: 2 回目で変わった:\n%s\n---\n%s", k, once, twice)
		}
		f := parseOrg("notes/a.org", []byte(once))
		if problems := append(checkFormat(f), checkEmphasis(f)...); len(problems) != 0 {
			t.Errorf("k=%d: 整形後にも違反が残る %v:\n%s", k, problems, once)
		}
	}
}

func TestCheckFormatDoesNotReportMissingVerbatimSpace(t *testing.T) {
	// 空白の欠けはコード表記の検査が報告する。折り返しの違反として重ね
	// て出さない。
	f := parseOrg("notes/a.org", []byte("これは=foo=です。\n"))
	if problems := checkFormat(f); len(problems) != 0 {
		t.Errorf("折り返しの違反として報告した: %v", problems)
	}
}
