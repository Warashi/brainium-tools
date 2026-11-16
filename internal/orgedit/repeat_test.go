package orgedit

import "testing"

// repeatOrg は repeater 付きの task を並べた差分テストの入力。今日は
// testNow の 2026-11-16 Mon。
const repeatOrg = `#+startup: logdone logdrawer
#+todo: TODO(t!) WIP(w!) | DONE(d!) CANCELLED(c!)

* Next
** TODO 毎日の task :weekend:
SCHEDULED: <2026-11-14 Sat .+1d>
:PROPERTIES:
:ID:       r-daily
:LAST_REPEAT: [2026-11-13 Fri 06:13]
:END:
:LOGBOOK:
- State "DONE"       from "TODO"       [2026-11-13 Fri 06:13]
:END:

毎日。

** WIP 毎週の task
SCHEDULED: <2026-11-07 Sat ++1w>
:PROPERTIES:
:ID:       r-weekly
:END:

毎週。

** TODO 毎月の task
DEADLINE: <2026-10-31 Sat 09:00 +1m>
:PROPERTIES:
:ID:       r-monthly
:END:

** TODO 予定日を消す task
SCHEDULED: <2026-11-10 Tue> DEADLINE: <2026-11-15 Sun .+2d>
:PROPERTIES:
:ID:       r-drop
:END:

** TODO repeater が 0 の task
SCHEDULED: <2026-11-14 Sat .+0d>
:PROPERTIES:
:ID:       r-zero
:END:
`

func TestRepeatMatchesEmacs(t *testing.T) {
	done := func(name, id, state string) emacsCase {
		return emacsCase{
			name:  name,
			files: map[string]string{"tasks.org": repeatOrg},
			run: func(root string, env Env) error {
				_, err := SetTodo(root, env, id, state)
				return err
			},
			elisp: `(edit-goto-id "tasks.org" "` + id + `") (edit-todo "` + state + `")`,
		}
	}
	runEmacsCases(t, []emacsCase{
		done("今日から進める", "r-daily", "DONE"),
		done("今日より後になるまで進める", "r-weekly", "DONE"),
		done("元の日付から進める", "r-monthly", "DONE"),
		done("repeater の無い SCHEDULED を消す", "r-drop", "DONE"),
		done("中止でも進める", "r-daily", "CANCELLED"),
		done("repeater が 0 なら完了にする", "r-zero", "DONE"),
	})
}

func TestRepeatRejectsUnsupportedRepeaters(t *testing.T) {
	for name, src := range map[string]string{
		"本文":   "#+startup: logdrawer\n* TODO a\nSCHEDULED: <2026-11-14 Sat .+1d>\n:PROPERTIES:\n:ID:       x\n:END:\n<2026-11-20 Fri +1w>\n",
		"時間単位": "#+startup: logdrawer\n* TODO a\nSCHEDULED: <2026-11-14 Sat 10:00 +1h>\n:PROPERTIES:\n:ID:       x\n:END:\n",
	} {
		root := writeRepo(t, t.TempDir(), map[string]string{"tasks.org": src})
		if _, err := SetTodo(root, testEnv(""), "x", "DONE"); err == nil {
			t.Errorf("%s の repeater を受け付けた", name)
		}
	}
}
