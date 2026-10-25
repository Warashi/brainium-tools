package orglint

import (
	"strings"
	"testing"
)

func format(t *testing.T, body string) string {
	t.Helper()
	return string(Format([]byte(body)))
}

func TestFormat(t *testing.T) {
	t.Run("70 桁を超える段落を折り返す", func(t *testing.T) {
		got := format(t, strings.Repeat("あ", 50)+"\n")
		want := strings.Repeat("あ", 35) + "\n" + strings.Repeat("あ", 15) + "\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("1 行で 70 桁に収まる段落には触れない", func(t *testing.T) {
		body := "短い行。\n\n- 箇条書き\n- 次の項目\n"
		if got := format(t, body); got != body {
			t.Errorf("got:\n%s\nwant:\n%s", got, body)
		}
	})

	t.Run("70 桁に収まる段落でも折り返しが早ければつなぎ直す", func(t *testing.T) {
		body := "短い行。\nもう 1 行。\n"
		want := "短い行。もう 1 行。\n"
		if got := format(t, body); got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("段落内の短い行は 1 行につないでから折り返す", func(t *testing.T) {
		body := "あいうえお\n" + strings.Repeat("か", 80) + "\n"
		got := format(t, body)
		if strings.Count(got, "\n") != 3 {
			t.Errorf("want 2 lines, got:\n%s", got)
		}
		if !strings.HasPrefix(got, "あいうえお"+strings.Repeat("か", 30)+"\n") {
			t.Errorf("つなぎ直しの結果が違う:\n%s", got)
		}
	})

	t.Run("固定幅行には触れない", func(t *testing.T) {
		body := ": " + strings.Repeat("あ", 50) + "\n: 次の行\n:\n"
		if got := format(t, body); got != body {
			t.Errorf("got:\n%s\nwant:\n%s", got, body)
		}
	})

	t.Run("リスト項目の継続行はマーカーの幅だけ字下げする", func(t *testing.T) {
		got := format(t, "- "+strings.Repeat("あ", 50)+"\n")
		want := "- " + strings.Repeat("あ", 34) + "\n  " + strings.Repeat("あ", 16) + "\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("番号付きリストの継続行も content の桁に揃える", func(t *testing.T) {
		got := format(t, "1. "+strings.Repeat("あ", 50)+"\n")
		if !strings.Contains(got, "\n   ") {
			t.Errorf("継続行が 3 桁字下げされていない:\n%s", got)
		}
	})

	t.Run("次のリスト項目は別の段落として扱う", func(t *testing.T) {
		body := "- " + strings.Repeat("あ", 50) + "\n- 次の項目\n"
		got := format(t, body)
		if !strings.HasSuffix(got, "\n- 次の項目\n") {
			t.Errorf("項目が混ざっている:\n%s", got)
		}
	})

	t.Run("見出し・表・ブロック・ドロワーには触れない", func(t *testing.T) {
		long := strings.Repeat("あ", 50)
		body := "* " + long + "\n" +
			"#+caption: " + long + "\n" +
			"| " + long + " |\n" +
			":PROPERTIES:\n:NOTE: " + long + "\n:END:\n" +
			"#+begin_src go\n// " + long + "\n#+end_src\n"
		if got := format(t, body); got != body {
			t.Errorf("got:\n%s\nwant:\n%s", got, body)
		}
	})

	t.Run("リンクの記法は幅に数えずに折り返す", func(t *testing.T) {
		// 説明文 10 桁 + 本文 70 桁。記法を数えると説明文の途中で切れる。
		link := "[[id:0f1e2d3c-4b5a-4968-8778-a6b5c4d3e2f1][org の行幅]]"
		got := format(t, link+strings.Repeat("あ", 35)+"\n")
		want := link + strings.Repeat("あ", 30) + "\n" + strings.Repeat("あ", 5) + "\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("表示幅が 70 桁に収まるリンクだけの行には触れない", func(t *testing.T) {
		body := "対象: [[id:C0FFEE00-1234-4ABC-9DEF-0123456789AB][screen の escape と vim の Ctrl-c が衝突する]]\n"
		if got := format(t, body); got != body {
			t.Errorf("got:\n%s\nwant:\n%s", got, body)
		}
	})

	t.Run("リンクの直前の空白で折り返した継続行は字下げを幅に数える", func(t *testing.T) {
		// リンクの直前で空白なしに折り返すと字下げがリンクの不可視属性を
		// 継承して幅に数えられないが、つなぎ直すと ] の前に空白が入り、
		// その空白で折り返した字下げは見えるので 2 行目が 72 桁になる。
		// Emacs も 1 回目の fill では前者の形を作り、2 回目で変える。
		// orglint は 2 回目以降に変わらない形を採る。
		link := "[[id:0f1e2d3c-4b5a-4968-8778-a6b5c4d3e2f1][行幅]]"
		got := format(t, "- "+strings.Repeat("あ", 34)+link+strings.Repeat("あ", 33)+"\n")
		want := "- " + strings.Repeat("あ", 34) + "\n  " + link + strings.Repeat("あ", 32) + "\n  あ\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("整形した結果は違反を残さない", func(t *testing.T) {
		body := strings.Repeat("あ", 300) + "\n- " + strings.Repeat("か", 200) + "\n"
		got := format(t, body)
		if problems := checkFormat(parseOrg("notes/a.org", []byte(got))); len(problems) != 0 {
			t.Errorf("整形後にも違反が残る %v:\n%s", problems, got)
		}
	})

	t.Run("整形は冪等である", func(t *testing.T) {
		body := strings.Repeat("あ", 300) + "\n\n- " + strings.Repeat("か", 200) + "\n"
		once := format(t, body)
		if twice := format(t, once); twice != once {
			t.Errorf("2 回目で変わった:\n%s\n---\n%s", once, twice)
		}
	})

	t.Run("つなぎ目の空白で幅が増えても冪等である", func(t *testing.T) {
		// 和文と欧文の境界に空白が入ると 1 桁増えて折り返し位置が動く。
		body := strings.Repeat("あ", 34) + "orglint " + strings.Repeat("い", 40) + "\n"
		once := format(t, body)
		if twice := format(t, once); twice != once {
			t.Errorf("2 回目で変わった:\n%s\n---\n%s", once, twice)
		}
		if problems := checkFormat(parseOrg("notes/a.org", []byte(once))); len(problems) != 0 {
			t.Errorf("整形後にも違反が残る %v:\n%s", problems, once)
		}
	})
}
