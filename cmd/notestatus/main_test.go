package main

import (
	"bytes"
	"testing"
	"testing/fstest"
	"time"
)

func noteFile(props, title string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(":PROPERTIES:\n:ID: x\n" + props + ":END:\n#+title: " + title + "\n")}
}

func TestList(t *testing.T) {
	fsys := fstest.MapFS{
		"notes/b-draft.org":      noteFile(":STATUS: draft\n", "下書き"),
		"notes/a-stale.org":      noteFile(":STATUS: stable\n:VERIFIED: [2026-09-26 Sat]\n:STALE_AFTER: [2026-11-26 Thu]\n", "期限切れ"),
		"notes/c-fresh.org":      noteFile(":STATUS: stable\n:VERIFIED: [2026-09-26 Sat]\n", "検証済み"),
		"notes/d-deprecated.org": noteFile(":STATUS: deprecated\n", "廃止"),
		"notes/attach/e.org":     noteFile(":STATUS: draft\n", "サブディレクトリ"),
		"journal/2026/x.org":     noteFile(":STATUS: draft\n", "journal"),
	}
	var out bytes.Buffer
	if err := list(&out, fsys, time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	want := "notes/a-stale.org\tstale_after [2026-11-26 Thu]\t期限切れ\n" +
		"notes/b-draft.org\tdraft\t下書き\n"
	if out.String() != want {
		t.Errorf("got\n%s\nwant\n%s", out.String(), want)
	}
}
