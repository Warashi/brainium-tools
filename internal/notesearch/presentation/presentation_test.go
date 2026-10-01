package presentation

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func report() Report {
	return Report{
		Query:        "猫",
		Scope:        "notes/ only",
		TotalMatches: 2,
		Truncated:    true,
		Results: []Item{{
			Path:       "notes/cat.org",
			ID:         "id-1",
			Title:      "猫\n偽の見出し",
			Status:     "stable",
			Verified:   "2026-09-18",
			StaleAfter: "2027-01-01",
			Snippets: []Snippet{{
				StartLine: 3,
				EndLine:   4,
				Headings:  []string{"生き物", "猫"},
				Text:      "猫は\n\tかわいい",
			}},
		}, {
			Path:  "notes/other.org",
			Title: "別のノート",
		}},
	}
}

func TestRenderTextShowsBoundedEvidence(t *testing.T) {
	got, err := Render(report(), FormatText, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, want := range []string{
		`Scope: "notes/ only"`,
		"Matches: total 2; returned 2",
		"Result limit truncated: yes",
		"Lexical matches only; excerpts are incomplete and do not establish full conclusions.",
		`Title: "猫\n偽の見出し"`,
		`Source lines 3-4; headings: "生き物" > "猫"`,
		"  3 | 猫は",
		`  4 | \tかわいい`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("text missing %q:\n%s", want, text)
		}
	}
}

func TestRenderJSONSchemaAndSourceExactness(t *testing.T) {
	got, err := Render(report(), FormatJSON, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Results []Item `json:"results"`
	}
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Results) != 2 || decoded.Results[0].Snippets[0].Text != "猫は\n\tかわいい" {
		t.Fatalf("JSON did not preserve source: %#v", decoded.Results)
	}
	if !bytes.HasSuffix(got, []byte("\n")) {
		t.Fatalf("JSON must end with a newline: %q", got)
	}

	empty, err := Render(Report{}, FormatJSON, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(empty, []byte(`"results":[]`)) {
		t.Fatalf("results must be an array: %s", empty)
	}
	noSnippets, err := Render(Report{Results: []Item{{}}}, FormatJSON, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(noSnippets, []byte(`"snippets":[]`)) {
		t.Fatalf("snippets must be an array: %s", noSnippets)
	}
	context, err := Render(Report{Results: []Item{{Snippets: []Snippet{{Kind: "context"}}}}}, FormatJSON, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(context, []byte(`"headings":[]`)) {
		t.Fatalf("context headings must be an array: %s", context)
	}
}

func TestRenderPathsPreservesLegacyEmptyAndRejectsBudgetLoss(t *testing.T) {
	empty, err := Render(Report{}, FormatPaths, 1)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty paths = %q, %v", empty, err)
	}
	got, err := Render(report(), FormatPaths, 10_000)
	if err != nil || string(got) != "notes/cat.org\nnotes/other.org\n" {
		t.Fatalf("paths = %q, %v", got, err)
	}
	if _, err := Render(report(), FormatPaths, 1); err == nil {
		t.Fatal("expected paths budget error")
	}
}

func TestRenderExactBudgetAndMultibyte(t *testing.T) {
	r := Report{Query: "猫", Scope: "notes/ only"}
	full, err := Render(r, FormatText, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Render(r, FormatText, len(full))
	if err != nil || !bytes.Equal(got, full) {
		t.Fatalf("exact budget = %q, %v", got, err)
	}
	if _, err := Render(r, FormatText, len(full)-1); err == nil {
		t.Fatal("expected too-small minimal report error")
	}

	jsonFull, err := Render(r, FormatJSON, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	jsonExact, err := Render(r, FormatJSON, len(jsonFull))
	if err != nil || !bytes.Equal(jsonExact, jsonFull) {
		t.Fatalf("exact JSON budget = %q, %v", jsonExact, err)
	}
	if _, err := Render(r, FormatJSON, len(jsonFull)-1); err == nil {
		t.Fatal("expected JSON budget error after newline")
	}
}

func TestRenderTextKeepsSourceLinesBoundedAndReadable(t *testing.T) {
	r := Report{Query: "q", Scope: "notes/ only", TotalMatches: 1, Results: []Item{{
		Path: "notes/a\nforged.org", Title: "title",
		Snippets: []Snippet{{StartLine: 42, EndLine: 43, Text: "日本語\ttext\nsecond\rline"}},
	}}}
	got, err := Render(r, FormatText, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, want := range []string{
		`Path: "notes/a\nforged.org"`,
		"  42 | 日本語\\ttext",
		"  43 | second\\rline",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("text missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Source text:") {
		t.Fatalf("source must be rendered as lines:\n%s", text)
	}
	if strings.Count(text, "freshness guarantees") != 1 {
		t.Fatalf("freshness caveat must appear once:\n%s", text)
	}
}

func TestRenderTextLabelsSuppliedSnippetKind(t *testing.T) {
	r := Report{Results: []Item{{Snippets: []Snippet{
		{StartLine: 1, EndLine: 8, Kind: "context"},
		{StartLine: 9, EndLine: 9},
	}}}}
	got, err := Render(r, FormatText, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "Source lines 1-8 (context);") {
		t.Fatalf("missing context label:\n%s", text)
	}
	if strings.Contains(text, "Source lines 9-9 (") {
		t.Fatalf("empty kind must stay unspecified:\n%s", text)
	}
}

func TestRenderOmitsLargeSnippetHonestly(t *testing.T) {
	r := Report{Query: "q", Scope: "notes/ only", TotalMatches: 1, Results: []Item{{
		Path: "notes/a.org", Title: "A",
		Snippets: []Snippet{{StartLine: 1, EndLine: 1, Text: strings.Repeat("猫", 500)}},
	}}}
	got, err := Render(r, FormatText, 700)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte(strings.Repeat("猫", 20))) ||
		!bytes.Contains(got, []byte("Snippet omitted due to output byte budget.")) {
		t.Fatalf("large snippet was not honestly omitted:\n%s", got)
	}
}

func TestRenderDoesNotMutateAndValidatesOptions(t *testing.T) {
	r := report()
	before, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	one, err := Render(r, FormatText, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	two, err := Render(r, FormatText, 10_000)
	if err != nil || !bytes.Equal(one, two) {
		t.Fatalf("render not deterministic: %v", err)
	}
	after, _ := json.Marshal(r)
	if !bytes.Equal(before, after) {
		t.Fatal("Render mutated its input")
	}
	truncated := report()
	truncated.Results[0].Snippets[0].Text = strings.Repeat("猫", 500)
	before, err = json.Marshal(truncated)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Render(truncated, FormatJSON, 1_200); err != nil {
		t.Fatal(err)
	}
	after, _ = json.Marshal(truncated)
	if !bytes.Equal(before, after) {
		t.Fatal("budget-truncated Render mutated its input")
	}
	for _, tc := range []struct {
		format Format
		budget int
	}{{"xml", 10}, {FormatText, 0}, {FormatJSON, -1}} {
		if _, err := Render(r, tc.format, tc.budget); err == nil {
			t.Errorf("expected error for %q / %d", tc.format, tc.budget)
		}
	}
}
