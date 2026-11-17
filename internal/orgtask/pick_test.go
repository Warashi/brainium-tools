package orgtask

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// wednesday は平日の操作の時刻。
var wednesday = time.Date(2026, 11, 18, 9, 0, 0, 0, time.Local)

const header = `#+todo: TODO(t!) WIP(w!) | DONE(d!) CANCELLED(c!)

* Next
:PROPERTIES:
:ID:       sec-next
:END:
`

const someday = `
* Someday
:PROPERTIES:
:ID:       sec-someday
:END:
** TODO someday first
:PROPERTIES:
:ID:       s-first
:END:
** TODO someday second
:PROPERTIES:
:ID:       s-second
:END:
`

// task は Next か Someday の task を 1 つ書く。lines は見出しの直後に置
// く計画行とプロパティ(:ID: を除く)。
func task(head, id string, lines ...string) string {
	var planning, props []string
	for _, l := range lines {
		if strings.HasPrefix(l, ":") {
			props = append(props, l)
		} else {
			planning = append(planning, l)
		}
	}
	s := "** " + head + "\n"
	for _, l := range planning {
		s += l + "\n"
	}
	s += ":PROPERTIES:\n:ID:       " + id + "\n"
	for _, l := range props {
		s += l + "\n"
	}
	return s + ":END:\n本文 " + id + "\n"
}

func load(t *testing.T, src string) *Tasks {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, TasksFile), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	ts, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func TestPickOrder(t *testing.T) {
	cases := []struct {
		name  string
		tasks []string
		want  string
	}{
		{"ファイル順", []string{task("TODO a", "a"), task("TODO b", "b")}, "a"},
		{"優先度 A が期日より先", []string{
			task("TODO due", "due", "DEADLINE: <2026-11-18 Wed>"),
			task("TODO [#A] prio", "prio"),
		}, "prio"},
		{"優先度の無い task は B", []string{
			task("TODO [#C] low", "low"),
			task("TODO plain", "plain"),
		}, "plain"},
		{"期日の近いものが先", []string{
			task("TODO later", "later", "DEADLINE: <2026-11-30 Mon>"),
			task("TODO sooner", "sooner", "DEADLINE: <2026-11-20 Fri>"),
		}, "sooner"},
		{"DEADLINE が無ければ SCHEDULED を期日とする", []string{
			task("TODO plain", "plain"),
			task("TODO sched", "sched", "SCHEDULED: <2026-11-17 Tue>"),
		}, "sched"},
		{"DEADLINE を SCHEDULED より優先する", []string{
			task("TODO a", "a", "DEADLINE: <2026-11-25 Wed> SCHEDULED: <2026-11-10 Tue>"),
			task("TODO b", "b", "DEADLINE: <2026-11-20 Fri>"),
		}, "b"},
		{"WIP が TODO より先", []string{task("TODO a", "a"), task("WIP b", "b")}, "b"},
		{"WIP の次に本人の手が要るもの", []string{
			task("TODO a", "a"),
			task("TODO b :handoff:", "b"),
		}, "b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := load(t, header+strings.Join(c.tasks, "")).Pick(wednesday, nil)
			if got.Task == nil || got.Task.ID != c.want {
				t.Errorf("Pick = %s %v, want %s", got.Status, got.Task, c.want)
			}
		})
	}
}

func TestPickSkips(t *testing.T) {
	cases := []struct {
		name  string
		tasks []string
		now   time.Time
	}{
		{"WORKING 付き", []string{task("TODO w", "w", ":WORKING: [2026-11-18 Wed 08:00]", ":REPO: x")}, wednesday},
		{"SCHEDULED が未来", []string{task("TODO s", "s", "SCHEDULED: <2026-11-19 Thu>")}, wednesday},
		{"SCHEDULED が今日の後の時刻", []string{task("TODO s", "s", "SCHEDULED: <2026-11-18 Wed 14:00>")}, wednesday},
		{"SCHEDULED の時刻の範囲が後", []string{task("TODO s", "s", "SCHEDULED: <2026-11-18 Wed 9:30-10:00 .+1w>")}, wednesday},
		{"SCHEDULED の時刻の範囲が今日終わった", []string{task("TODO s", "s", "SCHEDULED: <2026-11-18 Wed 7:00-8:59>")}, wednesday},
		{"SCHEDULED の時刻の範囲が前の日に終わった", []string{task("TODO s", "s", "SCHEDULED: <2026-11-17 Tue 09:00-10:00 .+1d>")}, wednesday},
		{"平日の weekend", []string{task("TODO we :weekend:", "we")}, wednesday},
		{"DONE", []string{task("DONE d", "d")}, wednesday},
		{"WORKING の task と REPO が重なる", []string{
			task("TODO w", "w", ":WORKING: [2026-11-18 Wed 08:00]", ":REPO: github.com/x/a"),
			task("TODO same", "same", ":REPO: github.com/x/b github.com/x/a"),
		}, wednesday},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := load(t, header+strings.Join(c.tasks, "")+task("TODO rest", "rest", ":REPO: r")).Pick(c.now, nil)
			if got.Status != Candidate || got.Task.ID != "rest" {
				t.Errorf("Pick = %s %v, want candidate rest", got.Status, got.Task)
			}
		})
	}
}

func TestPickTakesTasksScheduledAtAPassedTime(t *testing.T) {
	for _, ts := range []string{"<2026-11-18 Wed 9:00>", "<2026-11-18 Wed 08:30 .+1d>", "<2026-11-17 Tue 23:00>", "<2026-11-18 Wed 08:00-09:00>"} {
		got := load(t, header+task("TODO s", "s", "SCHEDULED: "+ts)).Pick(wednesday, nil)
		if got.Task == nil || got.Task.ID != "s" {
			t.Errorf("SCHEDULED: %s の Pick = %s %v, want s", ts, got.Status, got.Task)
		}
	}
}

func TestPickTakesWeekendTasksOnWeekends(t *testing.T) {
	saturday := time.Date(2026, 11, 21, 9, 0, 0, 0, time.Local)
	got := load(t, header+task("TODO we :weekend:", "we")).Pick(saturday, nil)
	if got.Status != NeedRepo || got.Task.ID != "we" {
		t.Errorf("Pick = %s %v", got.Status, got.Task)
	}
}

func TestPickIgnoresWorkingOnClosedTasks(t *testing.T) {
	src := header +
		task("DONE old", "old", ":WORKING: [2026-11-17 Tue 23:00]", ":REPO: github.com/x/a") +
		task("CANCELLED now", "now", ":WORKING: [2026-11-18 Wed 08:00]", ":REPO: github.com/x/a") +
		task("TODO same", "same", ":REPO: github.com/x/a")
	got := load(t, src).Pick(wednesday, nil)
	if got.Status != Candidate || got.Task.ID != "same" {
		t.Errorf("Pick = %s %v, want candidate same", got.Status, got.Task)
	}
}

func TestPickAsksAboutStaleWorking(t *testing.T) {
	src := header + task("TODO a", "a", ":REPO: r") + task("TODO old", "old", ":WORKING: [2026-11-17 Tue 23:00]")
	ts := load(t, src)
	if got := ts.Pick(wednesday, nil); got.Status != StaleWorking || got.Task.ID != "old" {
		t.Errorf("Pick = %s %v, want stale-working old", got.Status, got.Task)
	}
	if got := ts.Pick(wednesday, []string{"old"}); got.Status != Candidate || got.Task.ID != "a" {
		t.Errorf("keep したときの Pick = %s %v, want candidate a", got.Status, got.Task)
	}
}

func TestPickAsksForRepo(t *testing.T) {
	got := load(t, header+task("TODO a", "a")).Pick(wednesday, nil)
	if got.Status != NeedRepo || got.Task.ID != "a" {
		t.Errorf("Pick = %s %v, want need-repo a", got.Status, got.Task)
	}
}

func TestTasksWithoutRepositoriesDoNotConflict(t *testing.T) {
	src := header +
		task("TODO w", "w", ":WORKING: [2026-11-18 Wed 08:00]", ":REPO: "+NoRepo) +
		task("TODO free", "free", ":REPO: "+NoRepo)
	got := load(t, src).Pick(wednesday, nil)
	if got.Status != Candidate || got.Task.ID != "free" {
		t.Errorf("Pick = %s %v, want candidate free", got.Status, got.Task)
	}
}

func TestLinesPointAtTheSubtree(t *testing.T) {
	ts := load(t, header+task("TODO a", "a")+task("TODO b", "b"))
	// header は 6 行、task は 5 行
	if b := ts.ByID("b"); b.Line != 12 || b.EndLine != 16 {
		t.Errorf("b の行 = %d-%d, want 12-16", b.Line, b.EndLine)
	}
}

func TestPickFallsBackToSomeday(t *testing.T) {
	src := header + task("TODO w", "w", ":WORKING: [2026-11-18 Wed 08:00]", ":REPO: r") + someday
	got := load(t, src).Pick(wednesday, nil)
	if got.Status != NeedRepo || got.Task.ID != "s-first" || !got.FromSomeday {
		t.Errorf("Pick = %+v", got)
	}
}

func TestPickReportsNone(t *testing.T) {
	if got := load(t, header).Pick(wednesday, nil); got.Status != None || got.Task != nil {
		t.Errorf("Pick = %+v", got)
	}
}

func TestCurrentStep(t *testing.T) {
	ts := load(t, header+task("TODO a", "a")+task("TODO b", "b", ":CURRENT_STEP: 開く"))
	if got := ts.CurrentStep(); got == nil || got.ID != "b" {
		t.Errorf("CurrentStep = %v", got)
	}
	if got := load(t, header+task("TODO a", "a")).CurrentStep(); got != nil {
		t.Errorf("CurrentStep = %v, want nil", got)
	}
}

func TestRepeating(t *testing.T) {
	ts := load(t, header+task("TODO r", "r", "SCHEDULED: <2026-11-18 Wed .+1d>")+task("TODO n", "n", "SCHEDULED: <2026-11-18 Wed>"))
	if !ts.ByID("r").Repeating() || ts.ByID("n").Repeating() {
		t.Error("repeater の判定が違う")
	}
}
