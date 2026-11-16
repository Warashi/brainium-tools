// Package orgtask は brainium の tasks.org から、first-step が次に一手を
// 切り出す task を決める。並べ方と候補の条件は brainium の
// .claude/skills/first-step/rules.md に従う。
package orgtask

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

// TasksFile は task を置くファイル。
const TasksFile = "tasks.org"

// 候補を取る節の見出し。
const (
	NextSection    = "Next"
	SomedaySection = "Someday"
)

// Status は Pick の結果の種類。
type Status string

const (
	// Candidate は一手を切り出す task が決まったこと。
	Candidate Status = "candidate"
	// NeedRepo は決まった task に :REPO: が無いこと。付けてから選び直す。
	NeedRepo Status = "need-repo"
	// StaleWorking は今日より前の :WORKING: があること。まだ進めているか
	// を本人に聞いてから選び直す。
	StaleWorking Status = "stale-working"
	// None は候補が無いこと。
	None Status = "none"
)

// Task は tasks.org の見出し 1 つ。
type Task struct {
	orgedit.ListEntry
}

// Tasks は tasks.org の :ID: を持つ見出しを文書の順に持つ。
type Tasks struct {
	All []*Task
}

// Load は root の tasks.org を読む。
func Load(root string) (*Tasks, error) {
	entries, err := orgedit.List(root, []string{TasksFile})
	if err != nil {
		return nil, err
	}
	ts := &Tasks{}
	for _, e := range entries {
		ts.All = append(ts.All, &Task{e})
	}
	return ts, nil
}

// ByID は :ID: が id の見出しを返す。無ければ nil。
func (ts *Tasks) ByID(id string) *Task {
	for _, t := range ts.All {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// Section は深さ 1 の節 name を返す。無ければ nil。
func (ts *Tasks) Section(name string) *Task {
	for _, t := range ts.All {
		if t.Level == 1 && t.Title == name {
			return t
		}
	}
	return nil
}

// CurrentStep は :CURRENT_STEP: を持つ最初の task を返す。無ければ nil。
func (ts *Tasks) CurrentStep() *Task {
	for _, t := range ts.All {
		if t.Props["CURRENT_STEP"] != "" {
			return t
		}
	}
	return nil
}

// Pick は Pick の結果。
type Pick struct {
	Status Status
	// Task は None のとき nil。
	Task *Task
	// FromSomeday は、Next に候補が無く Someday から取ったこと。
	FromSomeday bool
}

// Pick は now の時点で一手を切り出す task を決める。keep は、今日より前
// の :WORKING: でも本人がまだ進めていると答えた task の ID。
func (ts *Tasks) Pick(now time.Time, keep []string) Pick {
	today := now.Format("2006-01-02")
	busy := map[string]bool{}
	for _, t := range ts.All {
		w := t.Props["WORKING"]
		if w == "" {
			continue
		}
		if date(w) < today && !slices.Contains(keep, t.ID) {
			return Pick{Status: StaleWorking, Task: t}
		}
		for _, r := range t.Repos() {
			busy[r] = true
		}
	}
	eligible := func(t *Task) bool {
		if t.Todo != "TODO" && t.Todo != "WIP" || t.Props["WORKING"] != "" {
			return false
		}
		if s := t.planning("SCHEDULED"); s != "" && s > today {
			return false
		}
		if t.HasTag("weekend") && now.Weekday() != time.Saturday && now.Weekday() != time.Sunday {
			return false
		}
		return !slices.ContainsFunc(t.Repos(), func(r string) bool { return busy[r] })
	}
	next := slices.DeleteFunc(ts.children(NextSection), func(t *Task) bool { return !eligible(t) })
	slices.SortStableFunc(next, compare)
	if len(next) > 0 {
		return found(next[0], false)
	}
	for _, t := range ts.children(SomedaySection) {
		if eligible(t) {
			return found(t, true)
		}
	}
	return Pick{Status: None}
}

func found(t *Task, fromSomeday bool) Pick {
	if len(t.Repos()) == 0 {
		return Pick{Status: NeedRepo, Task: t, FromSomeday: fromSomeday}
	}
	return Pick{Status: Candidate, Task: t, FromSomeday: fromSomeday}
}

// children は節 section の直下の task を文書の順に返す。
func (ts *Tasks) children(section string) []*Task {
	var tasks []*Task
	for _, t := range ts.All {
		if t.Level == 2 && t.Outline == section {
			tasks = append(tasks, t)
		}
	}
	return tasks
}

// compare は優先度、期日、WIP、本人の手が要るものの順に並べる。
func compare(a, b *Task) int {
	if c := strings.Compare(a.priority(), b.priority()); c != 0 {
		return c
	}
	da, db := a.due(), b.due()
	switch {
	case da != "" && db == "":
		return -1
	case da == "" && db != "":
		return 1
	case da != db:
		return strings.Compare(da, db)
	}
	if c := first(a.Todo == "WIP", b.Todo == "WIP"); c != 0 {
		return c
	}
	return first(a.HasTag("handoff"), b.HasTag("handoff"))
}

// first は a だけが真なら -1、b だけが真なら 1 を返す。
func first(a, b bool) int {
	switch {
	case a && !b:
		return -1
	case !a && b:
		return 1
	}
	return 0
}

// priority は org の既定(org-priority-default)と同じく、cookie が無け
// れば B とする。
func (t *Task) priority() string {
	if t.Priority == "" {
		return "B"
	}
	return t.Priority
}

// due は DEADLINE の日付を、無ければ SCHEDULED の日付を返す。
func (t *Task) due() string {
	if d := t.planning("DEADLINE"); d != "" {
		return d
	}
	return t.planning("SCHEDULED")
}

// Repos は :REPO: に空白区切りで書いたリポジトリを返す。
func (t *Task) Repos() []string {
	return strings.Fields(t.Props["REPO"])
}

// HasTag は見出しにタグ tag が付いているかを返す。
func (t *Task) HasTag(tag string) bool {
	return slices.Contains(strings.Split(strings.Trim(t.Tags, ":"), ":"), tag)
}

// Heading は優先度 cookie を除いた見出しを返す。
func (t *Task) Heading() string {
	return strings.TrimSpace(cookieRe.ReplaceAllString(t.Title, ""))
}

var (
	cookieRe   = regexp.MustCompile(`^\[#[^]]+\] ?`)
	dateRe     = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	planningRe = regexp.MustCompile(`^[ \t]*(?:(?:CLOSED|DEADLINE|SCHEDULED):[ \t]*[<\[][^]>]*[]>][ \t]*)+$`)
	// repeaterRe は timestamp の中の repeater(+1d・++1w・.+1m など)。
	repeaterRe = regexp.MustCompile(`<[^>]*[ \t](?:\.\+|\+\+|\+)[0-9]+[hdwmy][^>]*>`)
)

// date は timestamp の日付 YYYY-MM-DD を返す。
func date(ts string) string {
	return dateRe.FindString(ts)
}

// planningLine は見出しの直後の計画行を返す。無ければ空。
func (t *Task) planningLine() string {
	lines := strings.SplitN(t.Text, "\n", 3)
	if len(lines) < 2 || !planningRe.MatchString(lines[1]) {
		return ""
	}
	return lines[1]
}

// planning は計画行の keyword(DEADLINE・SCHEDULED)の日付を返す。無け
// れば空。
func (t *Task) planning(keyword string) string {
	line := t.planningLine()
	i := strings.Index(line, keyword+":")
	if i < 0 {
		return ""
	}
	return date(line[i:])
}

// Repeating は計画行に repeater があるかを返す。
func (t *Task) Repeating() bool {
	return repeaterRe.MatchString(t.planningLine())
}
