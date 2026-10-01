package main

import (
	"bytes"
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
		"notes/plain.org":  "#+title: 見出しの無いノート\n本文\n",
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

func TestCorpusOutline(t *testing.T) {
	root := repository(t)

	t.Run("引数が無ければ notes/ の全ノートを第 1 階層まで出す", func(t *testing.T) {
		got := run(t, root)
		for _, want := range []string{
			"notes/design.org (11 lines) [draft] 検索の設計\n  6-9 * 目的\n  10-11 * 結論\n",
			"notes/plain.org (2 lines) 見出しの無いノート\n",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("出力に %q が無い:\n%s", want, got)
			}
		}
		for _, unwanted := range []string{"** 詳細", "readme.md", "journal", "tasks.org"} {
			if strings.Contains(got, unwanted) {
				t.Errorf("出力に %q が入っている:\n%s", unwanted, got)
			}
		}
	})

	t.Run("-depth で深さを変えられる", func(t *testing.T) {
		got := run(t, root, "-depth", "2")
		if !strings.Contains(got, "  8-9 ** 詳細\n") {
			t.Errorf("第 2 階層が無い:\n%s", got)
		}
		got = run(t, root, "-depth", "0")
		if strings.Contains(got, "* 目的") {
			t.Errorf("-depth 0 で見出しが出ている:\n%s", got)
		}
	})

	t.Run("タイトルと見出しの制御文字はエスケープする", func(t *testing.T) {
		got := run(t, root)
		if !strings.Contains(got, `制御文字\u001b`) || !strings.Contains(got, `見出し\t1`) {
			t.Errorf("制御文字がエスケープされていない:\n%q", got)
		}
	})
}

func TestFileOutline(t *testing.T) {
	root := repository(t)

	t.Run("パスを渡すとそのファイルを全階層で出す", func(t *testing.T) {
		got := run(t, filepath.Join(root, "notes"), "design.org")
		want := "notes/design.org (11 lines) [draft] 検索の設計\n  6-9 * 目的\n  8-9 ** 詳細\n  10-11 * 結論\n"
		if !strings.HasSuffix(got, want) {
			t.Errorf("got:\n%s\nwant suffix:\n%s", got, want)
		}
	})

	t.Run("notes/ の外の org ファイルも渡せる", func(t *testing.T) {
		got := run(t, root, "tasks.org")
		if !strings.Contains(got, "tasks.org (2 lines) Tasks\n  2-2 * Next\n") {
			t.Errorf("got:\n%s", got)
		}
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

func TestInvalidDepth(t *testing.T) {
	if err := execute([]string{"-depth", "-2"}, repository(t), new(bytes.Buffer)); err == nil {
		t.Error("want error")
	}
}
