package repo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Warashi/brainium-tools/internal/repo"
)

func TestRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "notes", "deep")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(nested, "a.org")
	if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		start string
	}{
		{name: "ルート自身からはルートを返す", start: root},
		{name: "サブディレクトリからは親を辿ってルートを返す", start: nested},
		{name: "ファイルからはその親を辿ってルートを返す", start: file},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := repo.Root(c.start)
			if err != nil {
				t.Fatalf("Root failed: %v", err)
			}
			want, err := filepath.EvalSymlinks(root)
			if err != nil {
				t.Fatal(err)
			}
			gotEval, err := filepath.EvalSymlinks(got)
			if err != nil {
				t.Fatal(err)
			}
			if gotEval != want {
				t.Errorf("expected %q, got %q", want, gotEval)
			}
		})
	}
}

func TestRootFailsOutsideRepository(t *testing.T) {
	t.Parallel()

	if _, err := repo.Root(t.TempDir()); err == nil {
		t.Errorf("expected an error when no .git is found, got nil")
	}
}
