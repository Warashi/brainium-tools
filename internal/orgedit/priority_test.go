package orgedit

import (
	"strings"
	"testing"
)

func TestSetPriorityMatchesEmacs(t *testing.T) {
	withB := strings.Replace(tasksOrg, "** TODO scheduled task", "** TODO [#B] scheduled task", 1)
	priority := func(name, src, id, priority string) emacsCase {
		arg := "?" + priority
		if priority == "" {
			arg = "'remove"
		}
		return emacsCase{
			name:  name,
			files: map[string]string{"tasks.org": src},
			run: func(root string, _ Env) error {
				_, err := SetPriority(root, id, priority)
				return err
			},
			elisp: `(edit-goto-id "tasks.org" "` + id + `") (org-priority ` + arg + `)`,
		}
	}
	runEmacsCases(t, []emacsCase{
		priority("TODO の後に足す", tasksOrg, "t-sched", "A"),
		priority("タグのある和文の見出しに足す", tasksOrg, "t-tag", "A"),
		priority("揃ったタグのある見出しに足す", tasksOrg, "t-logged", "C"),
		priority("TODO の無い見出しに足す", tasksOrg, "sec-next", "A"),
		priority("替える", withB, "t-sched", "A"),
		priority("外す", withB, "t-sched", ""),
	})
}

func TestSetPriorityRejectsOutOfRange(t *testing.T) {
	for _, p := range []string{"D", "a", "AB", "1"} {
		if _, err := SetPriority(t.TempDir(), "t-sched", p); err == nil {
			t.Errorf("SetPriority(%q) が通った", p)
		}
	}
}
