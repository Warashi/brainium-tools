package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

const inbox = ":PROPERTIES:\n:ID:       file-inbox\n:END:\n#+title: Inbox\n"

const tasks = `#+todo: TODO(t!) WIP(w!) | DONE(d!)

* Next
:PROPERTIES:
:ID:       sec-next
:END:
** TODO [#A] 記事を書く
:PROPERTIES:
:ID:       t-article
:REPO:     github.com/x/blog
:END:
次の一手: 下書きを開く

** TODO 毎日の掃除
SCHEDULED: <2026-11-18 Wed .+1d>
:PROPERTIES:
:ID:       t-daily
:END:

* Someday
:PROPERTIES:
:ID:       sec-someday
:END:
** TODO いつか
:PROPERTIES:
:ID:       t-someday
:END:
`

// newRepo は tasks.org と inbox.org をコミットした git リポジトリを作る。
func newRepo(t *testing.T, tasksSrc string) (*context, *strings.Builder) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git が PATH に無い")
	}
	root := t.TempDir()
	for name, content := range map[string]string{"inbox.org": inbox, "tasks.org": tasksSrc} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g := git{root}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.name", "test"},
		{"config", "user.email", "test@example.com"},
		{"config", "commit.gpgsign", "false"},
		{"add", "."},
		{"commit", "-q", "-m", "init"},
	} {
		if _, err := g.run(args...); err != nil {
			t.Fatal(err)
		}
	}
	out := &strings.Builder{}
	env := orgedit.Env{
		Now:   time.Date(2026, 11, 18, 9, 0, 0, 0, time.Local),
		NewID: func() string { return "new-id" },
	}
	return &context{root: root, env: env, git: g, out: out}, out
}

func (c *context) mustRun(t *testing.T, args ...string) {
	t.Helper()
	if err := execute(c, args); err != nil {
		t.Fatalf("%v: %v", args, err)
	}
}

func (c *context) lastCommit(t *testing.T) string {
	t.Helper()
	out, err := c.git.run("log", "-1", "--format=%s", "--name-only")
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out)
}

func (c *context) read(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(c.root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestHandOutAndFinishAStep(t *testing.T) {
	c, out := newRepo(t, tasks)

	c.mustRun(t, "next")
	if want := "status: candidate\nid: t-article\ntitle: 記事を書く\nlines: tasks.org:7-12\nrepo: github.com/x/blog\n\n** TODO [#A] 記事を書く\n"; !strings.HasPrefix(out.String(), want) {
		t.Errorf("next:\n%s", out)
	}

	c.mustRun(t, "step", "t-article", "下書きを開く")
	if got := c.lastCommit(t); got != "task: 「記事を書く」の一手を渡す\n\ntasks.org" {
		t.Errorf("step のコミット:\n%s", got)
	}
	if !strings.Contains(c.read(t, "tasks.org"), ":CURRENT_STEP: 下書きを開く\n") {
		t.Errorf("tasks.org:\n%s", c.read(t, "tasks.org"))
	}

	out.Reset()
	c.mustRun(t, "done")
	if got := c.lastCommit(t); got != "task: 「記事を書く」の一手を終える\n\ninbox.org\ntasks.org" {
		t.Errorf("done のコミット:\n%s", got)
	}
	if strings.Contains(c.read(t, "tasks.org"), "CURRENT_STEP") {
		t.Error(":CURRENT_STEP: が残っている")
	}
	if !strings.Contains(c.read(t, "inbox.org"), "* 下書きを開く\n:PROPERTIES:\n:ID:       new-id\n:CAPTURED: [2026-11-18 Wed 09:00]\n:END:\n\n[[id:t-article][記事を書く]] の一手。\n") {
		t.Errorf("inbox.org:\n%s", c.read(t, "inbox.org"))
	}
	if !strings.HasPrefix(out.String(), "status: done\nid: t-article\n") {
		t.Errorf("done:\n%s", out)
	}
}

func TestDoneDoesNotCaptureRepeatingTasks(t *testing.T) {
	c, _ := newRepo(t, tasks)
	c.mustRun(t, "step", "t-daily", "掃除機をかける")
	c.mustRun(t, "done")
	if got := c.lastCommit(t); got != "task: 「毎日の掃除」の一手を終える\n\ntasks.org" {
		t.Errorf("done のコミット:\n%s", got)
	}
	if c.read(t, "inbox.org") != inbox {
		t.Errorf("inbox.org:\n%s", c.read(t, "inbox.org"))
	}
}

func TestStepMovesTheOnlyCurrentStep(t *testing.T) {
	c, _ := newRepo(t, tasks)
	c.mustRun(t, "step", "t-article", "開く")
	c.mustRun(t, "step", "t-daily", "眺める")
	if n := strings.Count(c.read(t, "tasks.org"), ":CURRENT_STEP:"); n != 1 {
		t.Errorf(":CURRENT_STEP: が %d 個ある", n)
	}
	c.mustRun(t, "drop-step")
	if strings.Contains(c.read(t, "tasks.org"), ":CURRENT_STEP:") {
		t.Error("drop-step の後に :CURRENT_STEP: が残っている")
	}
}

func TestHandToClaudeThenPickAnother(t *testing.T) {
	c, out := newRepo(t, tasks)
	c.mustRun(t, "step", "t-article", "開く")
	c.mustRun(t, "claude")
	src := c.read(t, "tasks.org")
	if strings.Contains(src, ":CURRENT_STEP:") || !strings.Contains(src, ":WORKING:  [2026-11-18 Wed 09:00]") {
		t.Errorf("tasks.org:\n%s", src)
	}
	out.Reset()
	c.mustRun(t, "next")
	if !strings.HasPrefix(out.String(), "status: need-repo\nid: t-daily\n") {
		t.Errorf("next:\n%s", out)
	}
	c.mustRun(t, "repo", "t-daily", "github.com/x/home")
	out.Reset()
	c.mustRun(t, "next")
	if !strings.HasPrefix(out.String(), "status: candidate\nid: t-daily\ntitle: 毎日の掃除\nlines: tasks.org:15-20\nrepo: github.com/x/home\n") {
		t.Errorf("next:\n%s", out)
	}
}

func TestClaudeNeedsRepo(t *testing.T) {
	c, _ := newRepo(t, tasks)
	c.mustRun(t, "step", "t-daily", "開く")
	if err := execute(c, []string{"claude"}); err == nil || !strings.Contains(err.Error(), "orgtask repo t-daily") {
		t.Errorf("claude: %v", err)
	}
}

func TestNextPromotesFromSomeday(t *testing.T) {
	src := strings.Replace(tasks, "** TODO [#A] 記事を書く", "** DONE [#A] 記事を書く", 1)
	src = strings.Replace(src, "** TODO 毎日の掃除", "** DONE 毎日の掃除", 1)
	c, out := newRepo(t, src)
	c.mustRun(t, "next")
	if !strings.HasPrefix(out.String(), "status: need-repo\nid: t-someday\n") {
		t.Errorf("next:\n%s", out)
	}
	if got := c.lastCommit(t); got != "task: 「いつか」を Someday から Next に上げる\n\ntasks.org" {
		t.Errorf("コミット:\n%s", got)
	}
	got := c.read(t, "tasks.org")
	if strings.Index(got, "** TODO いつか") > strings.Index(got, "* Someday") {
		t.Errorf("Next に移っていない:\n%s", got)
	}
}

func TestStaleWorkingIsAskedThenKeptOrDropped(t *testing.T) {
	src := strings.Replace(tasks, ":REPO:     github.com/x/blog", ":REPO:     github.com/x/blog\n:WORKING:  [2026-11-17 Tue 20:00]", 1)
	c, out := newRepo(t, src)
	c.mustRun(t, "next")
	if out.String() != "status: stale-working\nid: t-article\ntitle: 記事を書く\nlines: tasks.org:7-13\nrepo: github.com/x/blog\nworking: [2026-11-17 Tue 20:00]\n" {
		t.Errorf("next:\n%s", out)
	}
	out.Reset()
	c.mustRun(t, "next", "-keep-working", "t-article")
	if !strings.HasPrefix(out.String(), "status: need-repo\nid: t-daily\n") {
		t.Errorf("keep したときの next:\n%s", out)
	}
	c.mustRun(t, "unwork", "t-article")
	out.Reset()
	c.mustRun(t, "next")
	if !strings.HasPrefix(out.String(), "status: candidate\nid: t-article\n") {
		t.Errorf("unwork の後の next:\n%s", out)
	}
}

func TestRefusesToMixUncommittedChanges(t *testing.T) {
	c, _ := newRepo(t, tasks)
	if err := os.WriteFile(filepath.Join(c.root, "inbox.org"), []byte(inbox+"\n* 他のセッション\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := c.read(t, "tasks.org")
	c.mustRun(t, "step", "t-article", "開く")
	err := execute(c, []string{"done"})
	if !errors.Is(err, errDirty) {
		t.Fatalf("done: %v, want errDirty", err)
	}
	if after := c.read(t, "tasks.org"); after == before || !strings.Contains(after, ":CURRENT_STEP:") {
		t.Error("止まったのに tasks.org を書き換えた")
	}
}

func TestArgumentErrors(t *testing.T) {
	for _, args := range [][]string{
		{"nope"},
		{"next", "-keep-working"},
		{"step", "t-article"},
		{"step", "t-article", "two\nlines"},
		{"repo", "t-article"},
		{"repo", "t-article", "-", "github.com/x/a"},
		{"unwork"},
		{"current", "x"},
	} {
		c, _ := newRepo(t, tasks)
		if err := execute(c, args); err == nil {
			t.Errorf("%v: エラーにならない", args)
		}
	}
}
