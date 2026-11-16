package orgedit

import (
	"strings"
	"testing"
)

func TestListShowsHeadlinesWithIDs(t *testing.T) {
	root := writeRepo(t, t.TempDir(), map[string]string{"tasks.org": tasksOrg, "inbox.org": inboxOrg})
	entries, err := List(root, []string{"tasks.org", "inbox.org"})
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := WriteList(&b, entries); err != nil {
		t.Fatal(err)
	}
	var want strings.Builder
	for _, row := range [][]string{
		{"sec-next", "tasks.org", "", "", "Next", ""},
		{"t-tag", "tasks.org", "Next", "TODO", "和文の task", ":handoff:"},
		{"t-sched", "tasks.org", "Next", "TODO", "scheduled task", ""},
		{"t-logged", "tasks.org", "Next", "WIP", "logged task", ":weekend:"},
		{"t-child", "tasks.org", "Next/logged task", "", "child of logged", ""},
		{"t-done", "tasks.org", "Next", "DONE", "closed task", ""},
		{"sec-someday", "tasks.org", "", "", "Someday", ""},
		{"t-last", "tasks.org", "Someday", "TODO", "last task", ""},
		{"c-new", "inbox.org", "", "", "新しい capture", ""},
		{"c-old", "inbox.org", "", "", "古い capture", ""},
	} {
		want.WriteString(strings.Join(row, "\t") + "\n")
	}
	if b.String() != want.String() {
		t.Errorf("got:\n%s\nwant:\n%s", b.String(), want.String())
	}
}

func TestListCarriesLevelPriorityPropertiesAndText(t *testing.T) {
	src := strings.Replace(tasksOrg, "** TODO scheduled task", "** TODO [#A] scheduled task", 1)
	root := writeRepo(t, t.TempDir(), map[string]string{"tasks.org": src})
	entries, err := List(root, []string{"tasks.org"})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]ListEntry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	if e := byID["t-sched"]; e.Level != 2 || e.Priority != "A" {
		t.Errorf("t-sched: Level %d Priority %q", e.Level, e.Priority)
	}
	if e := byID["t-tag"]; e.Priority != "" {
		t.Errorf("優先度の無い見出しの Priority = %q", e.Priority)
	}
	logged := byID["t-logged"]
	if logged.Props["CURRENT_STEP"] != "何かをする" {
		t.Errorf("t-logged の Props = %v", logged.Props)
	}
	if !strings.HasPrefix(logged.Text, "** WIP logged task") || !strings.Contains(logged.Text, "*** child of logged") || strings.Contains(logged.Text, "closed task") {
		t.Errorf("t-logged の Text が subtree ではない:\n%s", logged.Text)
	}
}
