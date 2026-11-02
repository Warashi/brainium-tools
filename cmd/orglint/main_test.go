package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceIfUnchanged(t *testing.T) {
	t.Run("読んだ後に変わっていれば書かずにエラーにし、変更を残す", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "tasks.org")
		if err := os.WriteFile(name, []byte("他のセッションの変更\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := replaceIfUnchanged(name, []byte("読んだ内容\n"), []byte("整形した内容\n")); err == nil {
			t.Fatal("エラーにならなかった")
		}

		got, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "他のセッションの変更\n" {
			t.Errorf("上書きされた: %q", got)
		}
		assertOnlyFile(t, name)
	})

	t.Run("変わっていなければ権限を保って置き換える", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "tasks.org")
		if err := os.WriteFile(name, []byte("読んだ内容\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		if err := replaceIfUnchanged(name, []byte("読んだ内容\n"), []byte("整形した内容\n")); err != nil {
			t.Fatalf("replaceIfUnchanged: %v", err)
		}

		got, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "整形した内容\n" {
			t.Errorf("got %q", got)
		}
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("権限が %o になった", perm)
		}
		assertOnlyFile(t, name)
	})
}

// assertOnlyFile は name のディレクトリに name 以外のファイル(一時ファイ
// ルの残骸)が無いことを確かめる。
func assertOnlyFile(t *testing.T, name string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(name))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(name) {
			t.Errorf("余分なファイルが残った: %s", e.Name())
		}
	}
}

func TestRunFixOnly(t *testing.T) {
	t.Run("直せない違反が残っていても折り返しだけ直して正常終了する", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		// :ID: が無いので検査は必ず違反になるが、-fix-only では検査しない
		note := filepath.Join(root, "notes", "long.org")
		if err := os.MkdirAll(filepath.Dir(note), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "#+title: 長い\n\n" + strings.Repeat("あ", 50) + "\n"
		if err := os.WriteFile(note, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := run(mode{fix: true, check: false}, []string{note}); err != nil {
			t.Fatalf("run: %v", err)
		}

		got, err := os.ReadFile(note)
		if err != nil {
			t.Fatal(err)
		}
		want := "#+title: 長い\n\n" + strings.Repeat("あ", 35) + "\n" + strings.Repeat("あ", 15) + "\n"
		if string(got) != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})
}
