package orglint

import (
	"strings"
	"testing"
)

func TestCheckMaturity(t *testing.T) {
	noStatus := strings.Replace(validNote, ":STATUS:   draft\n", "", 1)
	t.Run("notes/ のノートは :STATUS: を必須とする", func(t *testing.T) {
		assertProblem(t, run(t, map[string]string{"notes/foo.org": noStatus}), "notes/foo.org", ":STATUS:")
	})
	t.Run("stable の検証日の欠落を行付きで報告する", func(t *testing.T) {
		body := strings.Replace(validNote, "draft", "stable", 1)
		got := run(t, map[string]string{"notes/foo.org": body})
		if len(got) != 1 || got[0].Line != 3 {
			t.Fatalf("want 1 problem at line 3, got %v", got)
		}
		assertProblem(t, got, "notes/foo.org", ":VERIFIED:")
	})
	t.Run("notes/ 以外は対象にしない", func(t *testing.T) {
		if got := run(t, map[string]string{"journal/2026/2026-07-19.org": noStatus}); len(got) != 0 {
			t.Errorf("unexpected problems: %v", got)
		}
	})
}
