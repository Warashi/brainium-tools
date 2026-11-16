package orgedit

import "testing"

const journal1116 = `:PROPERTIES:
:ID:       file-1116
:CREATED:  [2026-11-16 Mon 09:00]
:END:
#+title: 2026-11-16

* 朝の作業
:PROPERTIES:
:ID:       j-early
:CAPTURED: [2026-11-16 Mon 09:00]
:END:

朝。

* 夜の作業
:PROPERTIES:
:ID:       j-late
:CAPTURED: [2026-11-16 Mon 22:00]
:END:

夜。
`

func TestJournalMatchesEmacs(t *testing.T) {
	files := map[string]string{"inbox.org": inboxOrg, "journal/2026/2026-11-16.org": journal1116}
	earlyOnly := map[string]string{"inbox.org": inboxOrg, "journal/2026/2026-11-16.org": journal1116[:len(journal1116)-len("* 夜の作業\n:PROPERTIES:\n:ID:       j-late\n:CAPTURED: [2026-11-16 Mon 22:00]\n:END:\n\n夜。\n")]}
	runEmacsCases(t, []emacsCase{
		{
			name:  "後の capture の手前に移す",
			files: files,
			run: func(root string, env Env) error {
				_, err := Journal(root, env, "c-new")
				return err
			},
			elisp: `(edit-move-before "inbox.org" "c-new" "journal/2026/2026-11-16.org" "j-late")`,
		},
		{
			name:  "後の capture が無ければ末尾に移す",
			files: earlyOnly,
			run: func(root string, env Env) error {
				_, err := Journal(root, env, "c-new")
				return err
			},
			elisp: `(edit-refile "inbox.org" "c-new" "journal/2026/2026-11-16.org")`,
		},
	})
}

func TestJournalCreatesTheDailyNote(t *testing.T) {
	home := t.TempDir()
	root := writeRepo(t, home, map[string]string{"inbox.org": inboxOrg})
	changed, err := Journal(root, testEnv(home), "c-old")
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 2 || changed[0] != "journal/2026/2026-11-13.org" || changed[1] != "inbox.org" {
		t.Errorf("changed = %v", changed)
	}
	got := readRepo(t, root)
	want := ":PROPERTIES:\n:ID:       " + testIDs[0] + "\n:CREATED:  [2026-11-13 Fri 06:22]\n:END:\n#+title: 2026-11-13\n\n" +
		"* 古い capture\n:PROPERTIES:\n:ID:       c-old\n:CAPTURED: [2026-11-13 Fri 06:22]\n:END:\n\n本文 2\n"
	if got["journal/2026/2026-11-13.org"] != want {
		t.Errorf("journal:\n%s", lineDiff(got["journal/2026/2026-11-13.org"], want))
	}
	if contains(got["inbox.org"], "c-old") {
		t.Error("inbox.org に capture が残っている")
	}
}
