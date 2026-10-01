package presentation

import (
	"strings"
	"testing"
)

func treeReport() Report {
	r := report()
	r.Results[0].Snippets[0].Kind = "match"
	r.Results[0].Snippets = append(r.Results[0].Snippets, Snippet{StartLine: 1, EndLine: 1, Kind: "context", Text: "#+title"})
	r.Results[0].Outline = []Section{
		{Level: 1, Title: "生き物", StartLine: 2, EndLine: 9},
		{Level: 2, Title: "猫\t偽", StartLine: 3, EndLine: 5, Matched: true},
		{Level: 2, Title: "犬", StartLine: 6, EndLine: 9},
	}
	return r
}

func TestRenderTreeShowsOutlineAndMatchedSections(t *testing.T) {
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
		"Matched lines: 3-4\n",
		"    2-9 * 生き物\n  > 3-5 ** 猫\\t偽\n    6-9 ** 犬\n",
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

func TestRenderTreeWithoutHeadingsSaysSo(t *testing.T) {
	got, err := Render(treeReport(), FormatTree, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Path: \"notes/other.org\"\nTitle: \"別のノート\"\nMatched lines: (no matched excerpt)\nOutline: (no headings)\n") {
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
