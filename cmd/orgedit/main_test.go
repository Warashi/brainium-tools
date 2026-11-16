package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

const inbox = ":PROPERTIES:\n:ID:       file-inbox\n:END:\n#+title: Inbox\n"

const tasks = `#+todo: TODO(t!) | DONE(d!)
#+startup: logdone logdrawer

* Next
:PROPERTIES:
:ID:       sec-next
:END:
`

func newContext(t *testing.T, in string) (*context, *strings.Builder) {
	t.Helper()
	root := t.TempDir()
	for name, content := range map[string]string{"inbox.org": inbox, "tasks.org": tasks} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := &strings.Builder{}
	env := orgedit.Env{
		Now:   time.Date(2026, 11, 16, 21, 8, 0, 0, time.UTC),
		NewID: func() string { return "new-id" },
	}
	return &context{root: root, env: env, in: strings.NewReader(in), out: out}, out
}

func TestCaptureThenRefileAndClose(t *testing.T) {
	c, out := newContext(t, "本文。\n")
	steps := [][]string{
		{"capture", "-title", "作業"},
		{"move", "new-id", "-parent", "sec-next"},
		{"todo", "new-id", "TODO"},
		{"set-property", "new-id", "WORKING", "-now"},
		{"delete-property", "new-id", "WORKING"},
		{"todo", "new-id", "DONE"},
		{"set-tags", "new-id", "handoff"},
	}
	for _, args := range steps {
		if err := execute(c, args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if want := "new-id\ninbox.org\ntasks.org\ninbox.org\ntasks.org\ntasks.org\ntasks.org\ntasks.org\ntasks.org\n"; out.String() != want {
		t.Errorf("出力 = %q, want %q", out.String(), want)
	}
	data, _ := os.ReadFile(filepath.Join(c.root, "tasks.org"))
	// 表示幅 12 の見出しの後ろに、:handoff: の右端が 77 桁になるだけ空ける
	want := tasks + "** DONE 作業" + strings.Repeat(" ", 77-12-len(":handoff:")) + `:handoff:
CLOSED: [2026-11-16 Mon 21:08]
:PROPERTIES:
:ID:       new-id
:CAPTURED: [2026-11-16 Mon 21:08]
:END:
:LOGBOOK:
- State "DONE"       from "TODO"       [2026-11-16 Mon 21:08]
- State "TODO"       from              [2026-11-16 Mon 21:08]
:END:

本文。

`
	if string(data) != want {
		t.Errorf("tasks.org:\n%s", data)
	}
}

func TestFlagsMayFollowTheID(t *testing.T) {
	c, _ := newContext(t, "本文。\n")
	for _, args := range [][]string{
		{"capture", "-title", "作業"},
		{"set-property", "new-id", "WORKING", "-now"},
		{"todo", "new-id", "-none"},
	} {
		if err := execute(c, args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	data, _ := os.ReadFile(filepath.Join(c.root, "inbox.org"))
	if !strings.Contains(string(data), ":WORKING:  [2026-11-16 Mon 21:08]") {
		t.Errorf("inbox.org:\n%s", data)
	}
}

func TestPrioritySetsAndRemovesTheCookie(t *testing.T) {
	c, _ := newContext(t, "本文。\n")
	for _, step := range []struct {
		args []string
		want string
	}{
		{[]string{"capture", "-title", "作業"}, "* 作業\n"},
		{[]string{"priority", "new-id", "A"}, "* [#A] 作業\n"},
		{[]string{"priority", "new-id", "-none"}, "* 作業\n"},
	} {
		if err := execute(c, step.args); err != nil {
			t.Fatalf("%v: %v", step.args, err)
		}
		data, _ := os.ReadFile(filepath.Join(c.root, "inbox.org"))
		if !strings.Contains(string(data), step.want) {
			t.Errorf("%v の後の inbox.org:\n%s", step.args, data)
		}
	}
}

func TestArgumentErrors(t *testing.T) {
	for _, args := range [][]string{
		{"nope"},
		{"todo", "only-id"},
		{"move", "id"},
		{"move", "id", "-parent", "a", "-before", "b"},
		{"set-property", "id", "KEY"},
		{"capture"},
		{"priority", "id"},
	} {
		c, _ := newContext(t, "")
		if err := execute(c, args); err == nil {
			t.Errorf("%v: エラーにならない", args)
		}
	}
}

func TestHelpListsEveryCommand(t *testing.T) {
	c, out := newContext(t, "")
	if err := execute(c, []string{"help"}); err != nil {
		t.Fatal(err)
	}
	for name := range commands {
		if !strings.Contains(out.String(), "  "+name) {
			t.Errorf("help に %s が無い", name)
		}
	}
}

func TestRunInAnotherRepository(t *testing.T) {
	c, _ := newContext(t, "")
	if err := os.MkdirAll(filepath.Join(c.root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-C", c.root, "set-property", "sec-next", "WORKING", "x"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(c.root, "tasks.org"))
	if !strings.Contains(string(data), ":WORKING:  x") {
		t.Errorf("tasks.org:\n%s", data)
	}
}
