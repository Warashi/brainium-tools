package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeOrg(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "a.org")
	if err := os.WriteFile(p, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExecute(t *testing.T) {
	t.Run("節点を字下げした木で出し、object の間の文字列を text として出す", func(t *testing.T) {
		p := writeOrg(t, "* 見出し =x=\nこれは=foo [[https://x.example][bar]]= です\n")
		var out bytes.Buffer
		if err := execute([]string{p}, &out); err != nil {
			t.Fatal(err)
		}
		want := `headline 1:1-3:1 level=1 raw="* 見出し =x="
  title
    verbatim 1:7-1:10 "=x="
  section 2:1-3:1
    paragraph 2:1-3:1
      text 2:1-2:9 "これは=foo "
      link 2:9-2:35 type=https path="//x.example"
        text 2:30-2:33 "bar"
      text 2:35-3:1 "= です\n"
`
		if got := out.String(); got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("-lines と重なる節点だけを祖先と一緒に出す", func(t *testing.T) {
		p := writeOrg(t, "a\n\nb =c=\n\nd\n")
		var out bytes.Buffer
		if err := execute([]string{"-lines", "3", p}, &out); err != nil {
			t.Fatal(err)
		}
		want := `section 1:1-6:1
  paragraph 3:1-5:1
    text 3:1-3:3 "b "
    verbatim 3:3-3:6 "=c="
    text 3:6-4:1 "\n"
`
		if got := out.String(); got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("-lines は範囲でも渡せる", func(t *testing.T) {
		p := writeOrg(t, "a\n\nb\n\nd\n")
		var out bytes.Buffer
		if err := execute([]string{"-lines", "3-5", p}, &out); err != nil {
			t.Fatal(err)
		}
		want := `section 1:1-6:1
  paragraph 3:1-5:1
    text 3:1-4:1 "b\n"
  paragraph 5:1-6:1
    text 5:1-6:1 "d\n"
`
		if got := out.String(); got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	for _, tc := range []struct{ name string }{{"x"}, {"5-3"}, {"0"}, {"1-"}} {
		t.Run("不正な -lines "+tc.name+" はエラーにする", func(t *testing.T) {
			p := writeOrg(t, "a\n")
			if err := execute([]string{"-lines", tc.name, p}, &bytes.Buffer{}); err == nil {
				t.Error("want error")
			}
		})
	}

	t.Run("ファイルは 1 つだけ渡す", func(t *testing.T) {
		if err := execute(nil, &bytes.Buffer{}); err == nil {
			t.Error("want error")
		}
	})
}
