package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
