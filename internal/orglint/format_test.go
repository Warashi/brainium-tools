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

	t.Run("和文と欧文の境界でつなぐときは空白を補う", func(t *testing.T) {
		body := strings.Repeat("あ", 34) + " orglint\nの検査を足す。" + strings.Repeat("い", 40) + "\n"
		got := format(t, body)
		if !strings.Contains(got, "orglint の検査") {
			t.Errorf("空白が失われている:\n%s", got)
		}
	})

	t.Run("和文の途中と句読点の後には空白を補わない", func(t *testing.T) {
		body := strings.Repeat("あ", 34) + "です。\norglint を足す。" + strings.Repeat("い", 40) + "\n"
		got := format(t, body)
		if strings.Contains(got, "。 orglint") {
			t.Errorf("句読点の後に空白が入っている:\n%s", got)
		}
	})

	t.Run("和文と括弧の境界には空白を補わない", func(t *testing.T) {
		body := strings.Repeat("あ", 34) + "た\n(2026-07-22)。" + strings.Repeat("い", 40) + "\n"
		got := format(t, body)
		if strings.Contains(got, "た (2026") {
			t.Errorf("括弧の前に空白が入っている:\n%s", got)
		}
	})

	t.Run("コロンの直後には和文でも空白を補う", func(t *testing.T) {
		body := "設定の置き場になる:\n[[id:x][共有]]の話。\n"
		got := format(t, body)
		if !strings.Contains(got, "なる: [[id:x]") {
			t.Errorf("コロンの後の空白が失われている:\n%s", got)
		}
	})

	t.Run("コード表記の区切りと和文の境界には空白を補う", func(t *testing.T) {
		body := strings.Repeat("あ", 30) + "は =orglint=\nの検査。" + strings.Repeat("い", 40) + "\n"
		got := format(t, body)
		if !strings.Contains(got, "=orglint= の検査") {
			t.Errorf("区切りの後の空白が失われている:\n%s", got)
		}
	})

	t.Run("和文の句読点とコード表記の区切りの間には空白を補う", func(t *testing.T) {
		body := "あああ。\n=orglint= の検査。\n"
		got := format(t, body)
		if !strings.Contains(got, "。 =orglint=") {
			t.Errorf("句読点の後の空白が失われている:\n%s", got)
		}
	})

	t.Run("リンクの直後は説明文の末尾で空白の要否を決める", func(t *testing.T) {
		tail := strings.Repeat("い", 40)
		latin := format(t, "[[id:x][gopher]]\nのノート。"+tail+"\n")
		if !strings.Contains(latin, "[[id:x][gopher]] のノート") {
			t.Errorf("欧文で終わるリンクの後に空白がない:\n%s", latin)
		}
		wide := format(t, "[[id:x][行幅]]\nの検査。"+tail+"\n")
		if !strings.Contains(wide, "[[id:x][行幅]]の検査") {
			t.Errorf("和文で終わるリンクの後に空白が入っている:\n%s", wide)
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

	t.Run("リンクの直前で折り返した継続行は字下げを幅に数えない", func(t *testing.T) {
		// 2 行目は素の幅では 72 桁だが、字下げがリンクの不可視属性を
		// 継承するので Emacs には 70 桁に見える。
		link := "[[id:0f1e2d3c-4b5a-4968-8778-a6b5c4d3e2f1][行幅]]"
		got := format(t, "- "+strings.Repeat("あ", 34)+link+strings.Repeat("あ", 33)+"\n")
		want := "- " + strings.Repeat("あ", 34) + "\n  " + link + strings.Repeat("あ", 33) + "\n"
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
