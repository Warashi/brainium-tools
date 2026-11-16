package orgedit

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testNow は差分テストの操作の時刻。testdata/edit.el の edit-time と揃
// える。
var testNow = time.Date(2026, 11, 16, 21, 8, 0, 0, time.UTC)

// testIDs は差分テストで新しく振る ID の列。
var testIDs = []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"}

func testEnv(home string) Env {
	ids := append([]string(nil), testIDs...)
	return Env{
		Now:   testNow,
		NewID: func() string { id := ids[0]; ids = ids[1:]; return id },
		Home:  home,
	}
}

// emacsCase は 1 つの操作を Go と Emacs の org で行い、結果のファイル群
// が一致するかを比べる。
type emacsCase struct {
	name  string
	files map[string]string
	// run は Go の操作。root はリポジトリルート。
	run func(root string, env Env) error
	// elisp は Emacs の操作。リポジトリルートを default-directory にして
	// 評価する。
	elisp string
	// unchanged は、Emacs の操作がファイルを変えないケースかどうか。両方
	// が何もせずに一致するのを防ぐため、それ以外は変わったことを確かめる。
	unchanged bool
}

// lookEmacs は emacs のパスを返す。無ければテストを飛ばすが、
// ORG_DIFF_REQUIRE=1 なら失敗にして、正解を出せないまま通るのを防ぐ。
func lookEmacs(t *testing.T) string {
	t.Helper()
	emacs, err := exec.LookPath("emacs")
	if err != nil {
		if os.Getenv("ORG_DIFF_REQUIRE") == "1" {
			t.Fatal("ORG_DIFF_REQUIRE=1 だが emacs が PATH に無い")
		}
		t.Skip("emacs が PATH に無い")
	}
	return emacs
}

func runEmacsCases(t *testing.T, cases []emacsCase) {
	t.Helper()
	emacs := lookEmacs(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := runEmacsCase(t, emacs, c)
			if same := equalFiles(want, c.files); same != c.unchanged {
				t.Fatalf("Emacs の操作でファイルが変わったか = %v, want %v", !same, !c.unchanged)
			}
			got := runGoCase(t, c)
			compareFiles(t, got, want)
		})
	}
}

// writeRepo は home の下にリポジトリを作り、そのルートを返す。ARCHIVE_FILE
// が home を ~ と略すのを確かめるため、リポジトリは home の下に置く。
func writeRepo(t *testing.T, home string, files map[string]string) string {
	t.Helper()
	root := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runEmacsCase(t *testing.T, emacs string, c emacsCase) map[string]string {
	t.Helper()
	home := t.TempDir()
	root := writeRepo(t, home, c.files)
	el, err := filepath.Abs("testdata/edit.el")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, id := range testIDs {
		ids = append(ids, strconv.Quote(id))
	}
	form := "(progn (setq edit-ids (list " + strings.Join(ids, " ") + ")) " + c.elisp + " (edit-finish))"
	cmd := exec.Command(emacs, "-Q", "--batch", "-l", el, "--eval", form)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "HOME="+home, "TZ=UTC")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("emacs: %v\n%s", err, out)
	}
	return readRepo(t, root)
}

func runGoCase(t *testing.T, c emacsCase) map[string]string {
	t.Helper()
	home := t.TempDir()
	root := writeRepo(t, home, c.files)
	if err := c.run(root, testEnv(home)); err != nil {
		t.Fatal(err)
	}
	return readRepo(t, root)
}

func readRepo(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		files[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// lineDiff は最初に食い違う行の前後を、Go と Emacs の両方について %q
// で並べる。
func lineDiff(got, want string) string {
	g, w := strings.SplitAfter(got, "\n"), strings.SplitAfter(want, "\n")
	i := 0
	for i < len(g) && i < len(w) && g[i] == w[i] {
		i++
	}
	window := func(lines []string) string {
		var b strings.Builder
		for j := max(0, i-3); j < min(len(lines), i+4); j++ {
			mark := "  "
			if j == i {
				mark = "> "
			}
			b.WriteString(mark + strconv.Itoa(j+1) + " " + strconv.Quote(lines[j]) + "\n")
		}
		return b.String()
	}
	return "--- Go\n" + window(g) + "--- Emacs\n" + window(w)
}

func equalFiles(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for rel, s := range a {
		if b[rel] != s {
			return false
		}
	}
	return true
}

func compareFiles(t *testing.T, got, want map[string]string) {
	t.Helper()
	for rel, w := range want {
		g, ok := got[rel]
		if !ok {
			t.Errorf("%s: Go は作らなかったが Emacs は作った:\n%s", rel, w)
			continue
		}
		if g != w {
			t.Errorf("%s が Emacs と違う:\n%s", rel, lineDiff(g, w))
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			t.Errorf("%s: Emacs は作らなかったが Go は作った", rel)
		}
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
