package orglint

import (
	"strings"
	"testing"
)

const tasksHeader = `:PROPERTIES:
:ID:       aaaaaaaa-0000-0000-0000-000000000010
:END:
#+title: tasks
* Next
`

// taskOf は見出し行を含めて n 行になる task を返す。
func taskOf(heading string, n int) string {
	return heading + "\n" + strings.Repeat("- 本文\n", n-1)
}

func TestTaskLength(t *testing.T) {
	t.Run("上限ちょうどの task は通る", func(t *testing.T) {
		body := tasksHeader + taskOf("** TODO a", maxTaskLines) + "* Someday\n"
		if got := run(t, map[string]string{"tasks.org": body}); len(got) != 0 {
			t.Errorf("unexpected problems: %v", got)
		}
	})
	t.Run("上限を 1 行超える task は落ちる", func(t *testing.T) {
		body := tasksHeader + "** TODO a\n" + taskOf("** TODO b", maxTaskLines+1)
		got := run(t, map[string]string{"tasks.org": body})
		if len(got) != 1 || got[0].Line != 7 {
			t.Fatalf("want 1 problem at line 7, got %v", got)
		}
		assertProblem(t, got, "tasks.org", "行")
	})
	t.Run("下位見出しも task の行数に含める", func(t *testing.T) {
		half := maxTaskLines / 2
		body := tasksHeader + taskOf("** TODO a", half) + taskOf("*** 詳細", maxTaskLines-half+1)
		assertProblem(t, run(t, map[string]string{"tasks.org": body}), "tasks.org", "行")
	})
	t.Run("tasks.org 以外は対象にしない", func(t *testing.T) {
		body := validNote + taskOf("** 長い節", maxTaskLines+1)
		for _, p := range run(t, map[string]string{"notes/foo.org": body}) {
			if strings.Contains(p.Message, "上限") {
				t.Errorf("unexpected problem: %v", p)
			}
		}
	})
}
