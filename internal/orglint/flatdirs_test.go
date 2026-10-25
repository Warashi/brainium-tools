package orglint

import "testing"

func TestRunFlatDirs(t *testing.T) {
	t.Run("org を含まないサブディレクトリも報告する", func(t *testing.T) {
		got := run(t, map[string]string{
			"notes/nix-flake.org":   validNote,
			"notes/spike/init.el":   "(provide 'init)\n",
			"work/ledger/data.json": "{}\n",
		})
		assertProblem(t, got, "notes/spike", "サブディレクトリ")
		assertProblem(t, got, "work/ledger", "サブディレクトリ")
	})

	t.Run("notes/ や work/ が無くても失敗しない", func(t *testing.T) {
		if got := run(t, map[string]string{"README.org": "#+title: r\n"}); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("対象を絞ればそのディレクトリを含む場合だけ報告する", func(t *testing.T) {
		files := map[string]string{
			"notes/nix-flake.org": validNote,
			"notes/spike/init.el": "(provide 'init)\n",
		}
		if got := runTargets(t, files, "notes/nix-flake.org"); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
		assertProblem(t, runTargets(t, files, "notes"), "notes/spike", "サブディレクトリ")
	})
}
