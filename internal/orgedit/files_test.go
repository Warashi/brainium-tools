package orgedit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateRetriesWhenAnotherWriterChangesTheFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.org")
	if err := os.WriteFile(path, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	attempts := 0
	changed, err := Update(root, func(f *Files) error {
		attempts++
		src, _, err := f.Read("a.org")
		if err != nil {
			return err
		}
		if attempts == 1 {
			// 読んだ後に別の書き手が足した行を消さないことを確かめる
			if err := os.WriteFile(path, []byte("one\nother\n"), 0o644); err != nil {
				return err
			}
		}
		f.Write("a.org", src+"mine\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "one\nother\nmine\n" {
		t.Errorf("file = %q", data)
	}
	if strings.Join(changed, ",") != "a.org" {
		t.Errorf("changed = %v", changed)
	}
}

func TestUpdateGivesUpWhenTheFileKeepsChanging(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.org")
	n := 0
	_, err := Update(root, func(f *Files) error {
		if _, _, err := f.Read("a.org"); err != nil {
			return err
		}
		n++
		f.Write("a.org", "mine\n")
		return os.WriteFile(path, []byte(strings.Repeat("x", n)), 0o644)
	})
	if err == nil {
		t.Fatal("競合し続けてもエラーにならない")
	}
}

func TestUpdateKeepsTheFileMode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.org")
	if err := os.WriteFile(path, []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(root, func(f *Files) error {
		f.Write("a.org", "two\n")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v", info.Mode().Perm())
	}
}
