package orglint

import (
	"slices"
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

	t.Run("# の直後が空白でない行は org と同じく段落の一部としてつなぐ", func(t *testing.T) {
		got := format(t, "前の行。\n#7456 の件。\n")
		want := "前の行。 #7456 の件。\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("水平線は段落につながない", func(t *testing.T) {
		body := "前の行。\n-----\n次の行。\n"
		if got := format(t, body); got != body {
			t.Errorf("got:\n%s\nwant:\n%s", got, body)
		}
	})

	t.Run("マーカーの後の空白が複数でも継続行は org と同じくマーカーの直後 + 1 桁に揃える", func(t *testing.T) {
		got := format(t, "-   "+strings.Repeat("あ", 50)+"\n")
		want := "-   " + strings.Repeat("あ", 33) + "\n  " + strings.Repeat("あ", 17) + "\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
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

	// コメントの期待値は Emacs 30 の org-fill-paragraph(fill-column 70)の
	// 実測。継続行の頭がリンクで始まる場合の不可視属性の継承は測っていない。
	t.Run("コメントは継続行の頭に # と空白を付けて折り返す", func(t *testing.T) {
		a := func(n int) string { return strings.Repeat("あ", n) }
		for _, c := range []struct{ name, body, want string }{
			{"長い行", "# " + a(50) + "\n", "# " + a(34) + "\n# " + a(16) + "\n"},
			{"和文の行は詰めてつなぐ", "# 短い\n# 行\n", "# 短い行\n"},
			{"欧文の行は空白でつなぐ", "# foo bar\n# baz\n", "# foo bar baz\n"},
			{"和文と欧文の境界は空白でつなぐ", "# あいう\n# orglint の\n", "# あいう orglint の\n"},
			{"字下げしたコメント", "  # " + a(50) + "\n", "  # " + a(33) + "\n  # " + a(17) + "\n"},
			{"# の後の空白が複数", "#  " + a(50) + "\n", "#  " + a(33) + "\n#  " + a(17) + "\n"},
			{"# の後の - も継続行の頭に含める", "# - " + a(50) + "\n", "# - " + a(33) + "\n# - " + a(17) + "\n"},
			{"継続行の頭は 1 行目に揃える", "# あい\n#  " + a(50) + "\n", "# あい" + a(32) + "\n# " + a(18) + "\n"},
			{"# だけの行で段落を分ける", "# " + a(50) + "\n#\n# あ\n# \n# い\n", "# " + a(34) + "\n# " + a(16) + "\n#\n# あ\n# \n# い\n"},
			{"リストの中のコメントも行自身の頭に揃える", "- item\n  # " + a(50) + "\n", "- item\n  # " + a(33) + "\n  # " + a(17) + "\n"},
			{"リンクの記法は幅に数えない", "# [[id:0f1e2d3c-4b5a-4968-8778-a6b5c4d3e2f1][行幅]]" + a(50) + "\n",
				"# [[id:0f1e2d3c-4b5a-4968-8778-a6b5c4d3e2f1][行幅]]" + a(32) + "\n# " + a(18) + "\n"},
		} {
			t.Run(c.name, func(t *testing.T) {
				if got := format(t, c.body); got != c.want {
					t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
				}
			})
		}
	})

	t.Run("コメントの頭の直後では折り返さない", func(t *testing.T) {
		url := "https://example.com/" + strings.Repeat("a", 52)
		for _, head := range []string{"# ", "# - "} {
			got := format(t, head+url+" あ\n")
			want := head + url + "\n" + head + "あ\n"
			if got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		}
	})

	t.Run("コメントの整形は冪等である", func(t *testing.T) {
		body := "# - " + strings.Repeat("あ", 300) + "\n#\n  # " + strings.Repeat("か", 200) + "\n"
		once := format(t, body)
		if twice := format(t, once); twice != once {
			t.Errorf("2 回目で変わった:\n%s\n---\n%s", once, twice)
		}
	})

	t.Run("comment ブロックの中は空行で区切った素の文章として折り返す", func(t *testing.T) {
		a := func(n int) string { return strings.Repeat("あ", n) }
		block := func(s string) string { return "#+begin_comment\n" + s + "#+end_comment\n" }
		link := "[[id:0f1e2d3c-4b5a-4968-8778-a6b5c4d3e2f1][行幅]]"
		for _, c := range []struct{ name, body, want string }{
			{"長い行", block(a(50) + "\n"), block(a(35) + "\n" + a(15) + "\n")},
			{"空行で段落を分ける", block("あい\nうえ\n\nかき\n"), block("あいうえ\n\nかき\n")},
			{"和文と欧文の境界は空白でつなぐ", block("あいう\norglint の\n"), block("あいう orglint の\n")},
			{"リストもキーワード行もつなぐ", block("- あい\n#+foo: bar\nうえ\n"), block("- あい #+foo: bar うえ\n")},
			{"リストの記号は継続行の頭に付けない", block("- " + a(50) + "\n"), block("- " + a(34) + "\n" + a(16) + "\n")},
			{"コメントの # も継続行の頭に付けない", block("# " + a(50) + "\n"), block("# " + a(34) + "\n" + a(16) + "\n")},
			{"1 行の段落は 1 行目の字下げに揃える", block("  " + a(50) + "\n"), block("  " + a(34) + "\n  " + a(16) + "\n")},
			{"複数行の段落は 2 行目の字下げに揃える", block("あい\n  " + a(50) + "\n"), block("あい" + a(33) + "\n  " + a(17) + "\n")},
			{"1 行目だけの字下げは継続行に付けない", block("  あい\n" + a(50) + "\n"), block("  あい" + a(32) + "\n" + a(18) + "\n")},
			{"リンクの記法は幅に数えない", block(link + a(50) + "\n"), block(link + a(33) + "\n" + a(17) + "\n")},
			{"字下げしたブロック", "  #+begin_comment\n  " + a(50) + "\n  #+end_comment\n",
				"  #+begin_comment\n  " + a(34) + "\n  " + a(16) + "\n  #+end_comment\n"},
		} {
			t.Run(c.name, func(t *testing.T) {
				if got := format(t, c.body); got != c.want {
					t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
				}
			})
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

	t.Run("見出し直後の CLOSED の行は次の段落とつながない", func(t *testing.T) {
		closed := "CLOSED: [2026-10-30 Fri 08:25]"
		got := format(t, "* DONE 見出し\n"+closed+"\n"+strings.Repeat("あ", 50)+"\n")
		want := "* DONE 見出し\n" + closed + "\n" +
			strings.Repeat("あ", 35) + "\n" + strings.Repeat("あ", 15) + "\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("見出し直後の計画行には触れない", func(t *testing.T) {
		for _, planning := range []string{
			"SCHEDULED: <2026-10-20 Tue +1w>",
			"DEADLINE: <2026-10-20 Tue>",
			"CLOSED: [2026-10-30 Fri 08:25] SCHEDULED: <2026-10-20 Tue>",
		} {
			body := "** TODO 見出し\n" + planning + "\n本文。\n"
			if got := format(t, body); got != body {
				t.Errorf("got:\n%s\nwant:\n%s", got, body)
			}
		}
	})

	t.Run("見出しの直後にない計画行の書き出しは本文として扱う", func(t *testing.T) {
		got := format(t, "SCHEDULED: と書く行。\nもう 1 行。\n")
		want := "SCHEDULED: と書く行。もう 1 行。\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
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

func TestUnclosedDrawer(t *testing.T) {
	body := ":PROPERTIES:\n:ID: aaaaaaaa-0000-0000-0000-000000000001\n:END:\n#+title: tasks\n* Next\n** TODO a\n" +
		":PROPERTIES:\n:ID:       aaaaaaaa-0000-0000-0000-000000000002\n:CAPTURED: [2026-11-02 Mon 13:51]:END:\n\n本文。\n"

	t.Run("閉じていないドロワーを報告し、折り返しの違反にはしない", func(t *testing.T) {
		got := run(t, map[string]string{"tasks.org": body})
		want := []Problem{{Path: "tasks.org", Line: 7, Message: "ドロワーが :END: で閉じていない"}}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("-fix は閉じていないドロワーの行をつながない", func(t *testing.T) {
		if got := format(t, body); got != body {
			t.Errorf("got:\n%s\nwant:\n%s", got, body)
		}
	})
}
