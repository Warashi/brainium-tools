package presentation

import (
	"strings"
	"testing"
)

func treeReport() Report {
	r := report()
	r.Results[0].Snippets[0].Kind = "match"
	r.Results[0].Snippets = append(r.Results[0].Snippets, Snippet{StartLine: 1, EndLine: 1, Kind: "context", Text: "#+title"})
	r.Results[0].Bytes = 48640
	r.Results[0].Tree = true
	r.Results[0].Opening = "猫の話。"
	r.Results[0].Outline = []Section{
		{Level: 1, Title: "生き物", StartLine: 2, EndLine: 9, Bytes: 1024, Matched: true},
		{Level: 1, Title: "植物", StartLine: 10, EndLine: 12, Bytes: 300},
	}
	r.Results[0].Matched = []MatchedSection{{
		StartLine: 3, EndLine: 4, SectionStart: 3, SectionEnd: 5, Bytes: 512,
		Chain: []Section{{Level: 1, Title: "生き物"}, {Level: 2, Title: "猫\t偽"}},
	}}
	r.Results[1].Bytes = 2458
	r.Results[1].Matched = []MatchedSection{{StartLine: 1, EndLine: 2}}
	r.Results[1].KeyItems = []string{"概要", "詳細"}
	return r
}

func TestRenderTreeShowsTopLevelOutlineAndMatchedSections(t *testing.T) {
	got, err := Render(treeReport(), FormatTree, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, want := range []string{
		"Matches: total 2; returned 2",
		"Result limit truncated: yes",
		"No match does not establish absence.",
		`Title: "猫\n偽の見出し"`,
		"Size: 47.5KB; tree\n",
		"Opening: \"猫の話。\"\n",
		"Matched sections:\n  3-4 in 3-5 0.5KB: * 生き物 > ** 猫\\t偽\n",
		"Outline:\n  > 2-9 1.0KB * 生き物\n    10-12 0.3KB * 植物\n",
		"notetree <path>",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("tree missing %q:\n%s", want, text)
		}
	}
	// 抜粋の本文は出さない。読む範囲は行番号で選ばせる
	if strings.Contains(text, "かわいい") {
		t.Errorf("tree must not include snippet text:\n%s", text)
	}
}

func TestRenderTreeFoldedNoteListsKeyItems(t *testing.T) {
	got, err := Render(treeReport(), FormatTree, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	want := "Path: \"notes/other.org\"\nTitle: \"別のノート\"\nSize: 2.4KB; read whole\n" +
		"Matched sections:\n  1-2 before the first heading\nKey: 概要 / 詳細\n"
	if !strings.Contains(string(got), want) {
		t.Errorf("folded note not reported as %q:\n%s", want, got)
	}
}

func TestRenderTreeWithoutHeadingsSaysSo(t *testing.T) {
	r := treeReport()
	r.Results[1].KeyItems = nil
	r.Results[1].Matched = nil
	got, err := Render(r, FormatTree, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Matched sections: (no matched excerpt)\nOutline: (no headings)\n") {
		t.Errorf("note without headings not reported:\n%s", got)
	}
}

func TestRenderTreeDropsLowerResultsForBudget(t *testing.T) {
	full, err := Render(treeReport(), FormatTree, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Render(treeReport(), FormatTree, len(full)-1)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "Byte budget truncated: yes") ||
		!strings.Contains(text, "Budget omissions: 1 result(s).") ||
		strings.Contains(text, "notes/other.org") {
		t.Errorf("lower result not dropped honestly:\n%s", text)
	}
	if _, err := Render(treeReport(), FormatTree, 10); err == nil {
		t.Error("want error for a budget too small for an honest report")
	}
}
