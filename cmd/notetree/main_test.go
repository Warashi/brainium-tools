package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const designNote = ":PROPERTIES:\n:ID: x\n:STATUS: draft\n:END:\n#+title: 検索の設計\n" + // 1-5
	"* 目的\n" + // 6
	"本文\n" + // 7
	"** 詳細\n" + // 8
	"本文\n" + // 9
	"* 結論\n" + // 10
	"本文\n" // 11

// longNote は 2 つの大きな節を持ち、剪定しても木が残るノート。
var longNote = "#+title: 長いノート\n#+filetags: :search:go:\n" + // 1-2
	"冒頭の 1 文。続き。\n" + // 3
	"* 前半\n" + // 4
	strings.Repeat("あ\n", 3000) + // 5-3004
	"* 後半\n" + // 3005
	strings.Repeat("い\n", 3000) + // 3006-6005
	"** 小節\n" + // 6006
	"本文\n" // 6007

func repository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{".git", "notes", "journal"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for name, text := range map[string]string{
		"notes/design.org": designNote,
		"notes/long.org":   longNote,
		"notes/plain.org":  "#+title: 見出しの無いノート\n本文。\n",
		"notes/readme.md":  "* org ではない\n",
		"journal/2026.org": "#+title: 日記\n* 出来事\n",
		"tasks.org":        "#+title: Tasks\n* Next\n",
		"notes/tab\tx.org": "#+title: 制御文字\x1b\n* 見出し\t1\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func run(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	if err := execute(args, cwd, &out); err != nil {
		t.Fatalf("execute(%q) failed: %v", args, err)
	}
	return out.String()
}

func assertContains(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("出力に %q が無い:\n%s", want, got)
		}
	}
}

func assertNotContains(t *testing.T, got string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(got, u) {
			t.Errorf("出力に %q が入っている:\n%s", u, got)
		}
	}
}

func TestCorpusOutline(t *testing.T) {
	root := repository(t)
	got := run(t, root)

	t.Run("ノートごとに冒頭の文と第 1 階層の見出しだけを出す", func(t *testing.T) {
		assertContains(t, got,
			"notes/design.org 0.1KB [draft] 検索の設計\n  key: 目的 / 結論\n",
			"notes/long.org 23.5KB tree 長いノート\n  tags: search go\n  > 冒頭の 1 文。\n  key: 前半 / 後半\n",
			"notes/plain.org 0.0KB 見出しの無いノート\n  > 本文。\n",
		)
		assertNotContains(t, got, "4-3004", "小節", "詳細")
	})

	t.Run("notes/ 直下の org ファイルだけを対象にする", func(t *testing.T) {
		assertNotContains(t, got, "readme.md", "journal", "tasks.org")
	})

	t.Run("タイトルと見出しの制御文字はエスケープする", func(t *testing.T) {
		assertContains(t, got, `制御文字\u001b`, `見出し\t1`)
	})

	t.Run("予算に収まれば打ち切らない", func(t *testing.T) {
		assertContains(t, got, "Byte budget truncated: no\n")
	})
}

// stages は予算を 1 バイトずつ減らしたときの打ち切りの段階を順に返す。
// 出力が予算を超えたら失敗にする。
func stages(t *testing.T, root string, args ...string) []string {
	t.Helper()
	full := run(t, root, args...)
	var out []string
	for budget := len(full); budget > 0; budget-- {
		var buf bytes.Buffer
		if err := execute(append([]string{"-max-bytes", fmt.Sprint(budget)}, args...), root, &buf); err != nil {
			return append(out, "error")
		}
		if buf.Len() > budget {
			t.Fatalf("budget %d: output is %d bytes", budget, buf.Len())
		}
		line := strings.Split(buf.String(), "\n")[3]
		if i := strings.Index(line, "; omitted notes:"); i >= 0 {
			line = line[:i] + "; omitted notes"
		}
		if len(out) == 0 || out[len(out)-1] != line {
			out = append(out, line)
		}
	}
	return out
}

func TestOutlineBudget(t *testing.T) {
	root := repository(t)

	t.Run("全体では key、冒頭の文の順に削り、最後にノートを省く", func(t *testing.T) {
		want := []string{
			"Byte budget truncated: no",
			"Byte budget truncated: yes; dropped key items",
			"Byte budget truncated: yes; dropped key items, opening sentences",
			"Byte budget truncated: yes; dropped key items, opening sentences; omitted notes",
			"error",
		}
		if got := stages(t, root); strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("stages:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	})

	t.Run("パスを渡したときは見出しも削れる", func(t *testing.T) {
		want := []string{
			"Byte budget truncated: no",
			"Byte budget truncated: yes; dropped key items",
			"Byte budget truncated: yes; dropped key items, opening sentences",
			"Byte budget truncated: yes; dropped key items, opening sentences, headings",
			"Byte budget truncated: yes; dropped key items, opening sentences, headings; omitted notes",
			"error",
		}
		got := stages(t, root, "notes/long.org", "notes/design.org")
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("stages:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	})

	t.Run("削った要素は出さない", func(t *testing.T) {
		full := run(t, root)
		got := run(t, root, "-max-bytes", fmt.Sprint(len(full)-1))
		assertNotContains(t, got, "  key:")
		assertContains(t, got, "  > 冒頭の 1 文。")
	})
}

func TestFixedDepth(t *testing.T) {
	root := repository(t)

	t.Run("-depth を渡すと剪定せずにその深さまで出す", func(t *testing.T) {
		got := run(t, root, "-depth", "1")
		assertContains(t, got, "  6-9 0.0KB * 目的\n  10-11 0.0KB * 結論\n")
		assertNotContains(t, got, "** 詳細", "  key:", "; key:")
	})

	t.Run("-depth 0 は見出しを出さない", func(t *testing.T) {
		assertNotContains(t, run(t, root, "-depth", "0"), "* 目的")
	})
}

func TestFileOutline(t *testing.T) {
	root := repository(t)

	t.Run("パスを渡すと剪定した木を行範囲と大きさ付きで出す", func(t *testing.T) {
		got := run(t, filepath.Join(root, "notes"), "long.org")
		assertContains(t, got,
			"notes/long.org 23.5KB tree 長いノート\n",
			"  4-3004 11.7KB * 前半\n",
			"  3005-6007 11.7KB * 後半; key: 小節\n",
		)
		assertNotContains(t, got, "design.org")
	})

	t.Run("丸ごと読むほうが安いノートは見出しを key に畳む", func(t *testing.T) {
		got := run(t, root, "notes/design.org")
		assertContains(t, got, "notes/design.org 0.1KB [draft] 検索の設計\n  key: 目的 / 詳細 / 結論\n")
		assertNotContains(t, got, "6-9")
	})

	t.Run("notes/ の外の org ファイルも渡せる", func(t *testing.T) {
		assertContains(t, run(t, root, "tasks.org"), "tasks.org 0.0KB Tasks\n  key: Next\n")
	})

	t.Run("リポジトリの外のパスはエラーにする", func(t *testing.T) {
		outside := filepath.Join(t.TempDir(), "x.org")
		if err := os.WriteFile(outside, []byte("* x\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := execute([]string{outside}, root, new(bytes.Buffer)); err == nil {
			t.Error("want error")
		}
	})

	t.Run("存在しないパスはエラーにする", func(t *testing.T) {
		if err := execute([]string{"notes/missing.org"}, root, new(bytes.Buffer)); err == nil {
			t.Error("want error")
		}
	})
}

func TestInvalidFlags(t *testing.T) {
	root := repository(t)
	for _, args := range [][]string{{"-depth", "-1"}, {"-max-bytes", "0"}, {"-route-bytes", "-1"}} {
		if err := execute(args, root, new(bytes.Buffer)); err == nil {
			t.Errorf("execute(%q): want error", args)
		}
	}
}
