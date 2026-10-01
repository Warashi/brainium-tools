package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Warashi/brainium-tools/internal/notesearch"
)

const sampleNote = ":PROPERTIES:\n:ID: aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa\n:STATUS: draft\n:VERIFIED: [2026-09-01 Tue]\n:STALE_AFTER: [2026-10-01 Thu]\n:END:\n#+title: Search decisions\n\n* Findings\nBM25 keeps the baseline ranking.\nDo not interpret lexical matches as confidence.\n"

func repository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{".git", "notes", "tools"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for name, text := range map[string]string{
		"notes/search.org": sampleNote,
		"notes/other.org":  "#+title: Another note\nA baseline comparison.\n",
		"tasks.org":        "#+title: Tasks\nexcludedtaskonly\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestEvidenceCLI(t *testing.T) {
	root := repository(t)
	var out bytes.Buffer
	if err := execute([]string{"-format", "json", "BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Query        string `json:"query"`
		Scope        string `json:"scope"`
		TotalMatches int    `json:"total_matches"`
		Results      []struct {
			Path       string `json:"path"`
			ID         string `json:"id"`
			Title      string `json:"title"`
			Status     string `json:"status"`
			Verified   string `json:"verified"`
			StaleAfter string `json:"stale_after"`
			Snippets   []struct {
				StartLine int      `json:"start_line"`
				EndLine   int      `json:"end_line"`
				Headings  []string `json:"headings"`
				Text      string   `json:"text"`
			} `json:"snippets"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Query != "BM25" || report.Scope != "notes/" || report.TotalMatches != 1 || len(report.Results) != 1 {
		t.Fatalf("unexpected report: %s", out.String())
	}
	item := report.Results[0]
	if item.Path != "notes/search.org" || item.Title != "Search decisions" || item.Status != "draft" ||
		item.ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || item.Verified != "[2026-09-01 Tue]" || item.StaleAfter != "[2026-10-01 Thu]" {
		t.Fatalf("metadata not preserved: %+v", item)
	}
	if len(item.Snippets) == 0 {
		t.Fatal("no source evidence")
	}
	lines := strings.Split(sampleNote, "\n")
	found := false
	for _, snippet := range item.Snippets {
		if snippet.StartLine < 1 || snippet.EndLine > len(lines) || snippet.EndLine < snippet.StartLine {
			t.Fatalf("invalid source coordinates: %+v", snippet)
		}
		if want := strings.Join(lines[snippet.StartLine-1:snippet.EndLine], "\n"); snippet.Text != want {
			t.Fatalf("source changed: got %q, want %q", snippet.Text, want)
		}
		found = found || strings.Contains(snippet.Text, "BM25")
	}
	if !found {
		t.Fatal("matching evidence missing")
	}
}

func TestCLIFormatsAndWorkingDirectory(t *testing.T) {
	root := repository(t)
	for _, cwd := range []string{root, filepath.Join(root, "tools"), filepath.Join(root, "notes")} {
		var out bytes.Buffer
		if err := execute([]string{"-format", "paths", "BM25"}, cwd, &out); err != nil {
			t.Fatal(err)
		}
		if out.String() != "notes/search.org\n" {
			t.Fatalf("legacy paths changed: %q", out.String())
		}
	}
	var out bytes.Buffer
	if err := execute([]string{"BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"notes/search.org", "BM25 keeps", "draft", "notes/"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("default output does not contain %q: %s", want, out.String())
		}
	}
}

func TestCLINoMatchesDoesNotImplyKnowledgeAbsent(t *testing.T) {
	root := repository(t)
	var out bytes.Buffer
	if err := execute([]string{"-format", "json", "excludedtaskonly"}, root, &out); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Scope        string            `json:"scope"`
		TotalMatches int               `json:"total_matches"`
		Results      []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Scope != "notes/" || report.TotalMatches != 0 || report.Results == nil || len(report.Results) != 0 {
		t.Fatalf("incorrect empty scope report: %s", out.String())
	}
	out.Reset()
	if err := execute([]string{"-format", "paths", "excludedtaskonly"}, root, &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("legacy empty output changed: %q", out.String())
	}
}

func TestCLIRejectsInvalidArgumentsBeforeReadingRepository(t *testing.T) {
	for _, args := range [][]string{
		nil, {" "}, {"\xff"}, {"-limit", "0", "query"}, {"-limit", "-1", "query"},
		{"-max-bytes", "0", "query"}, {"-max-bytes", "-1", "query"},
		{"-format", "yaml", "query"}, {"-unknown", "query"},
	} {
		var out bytes.Buffer
		err := execute(args, t.TempDir(), &out)
		if err == nil || strings.Contains(err.Error(), ".git") {
			t.Errorf("args %q should fail argument validation, got %v", args, err)
		}
		if out.Len() != 0 {
			t.Errorf("invalid args produced success output: %q", out.String())
		}
	}
}

func TestCLILimitsAndHelp(t *testing.T) {
	root := repository(t)
	var out bytes.Buffer
	if err := execute([]string{"-format", "json", "-limit", "1", "baseline"}, root, &out); err != nil {
		t.Fatal(err)
	}
	var report struct {
		TotalMatches int               `json:"total_matches"`
		Results      []json.RawMessage `json:"results"`
		Truncated    bool              `json:"truncated"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.TotalMatches != 2 || len(report.Results) != 1 || !report.Truncated {
		t.Fatalf("incorrect limit: %s", out.String())
	}
	out.Reset()
	if err := execute([]string{"-max-bytes", "1", "BM25"}, root, &out); err == nil || out.Len() != 0 {
		t.Fatalf("tiny budget should error without partial output: %v %q", err, out.String())
	}
	out.Reset()
	if err := execute([]string{"-h"}, t.TempDir(), &out); err != nil || !strings.Contains(out.String(), "-format") {
		t.Fatalf("help unavailable outside repo: %v %q", err, out.String())
	}
}

func TestCLIExplicitFlagLikeQuery(t *testing.T) {
	root := repository(t)
	var out bytes.Buffer
	if err := execute([]string{"-format", "paths", "--", "-BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "notes/search.org\n" {
		t.Fatalf("-- did not preserve query: %q", out.String())
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCLIPropagatesWriteError(t *testing.T) {
	if err := execute([]string{"BM25"}, repository(t), failedWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write error lost: %v", err)
	}
}

func TestEvidencePreservesRankingAndDoesNotDropDeprecatedNotes(t *testing.T) {
	fsys := fstest.MapFS{
		"past.org":         {Data: []byte(":PROPERTIES:\n:STATUS: deprecated\n:END:\n#+title: Prior attempt\nPRF failed the held-out comparison.\n")},
		"now.org":          {Data: []byte(":PROPERTIES:\n:STATUS: stable\n:END:\n#+title: Current design\nWe use BM25 instead of PRF.\n")},
		"undocumented.org": {Data: []byte("#+title: Unverified\nA PRF experiment.\n")},
	}
	index, err := notesearch.Build(fsys, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := index.Search("PRF")
	got, err := search(fsys, "PRF", 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != len(want) || got.TotalMatches != len(want) {
		t.Fatalf("results lost: %+v", got)
	}
	for i, result := range got.Results {
		if result.Path != "notes/"+want[i].Path {
			t.Errorf("ranking changed at %d: %s != %s", i, result.Path, want[i].Path)
		}
		if result.Path == "notes/past.org" && result.Status != "deprecated" {
			t.Errorf("past decision lost its status: %+v", result)
		}
		if result.Path == "notes/undocumented.org" && result.Status != "" {
			t.Errorf("missing maturity was inferred: %+v", result)
		}
	}
}

func TestCLISurfacesRepositoryAndSourceErrors(t *testing.T) {
	for _, mutate := range []func(*testing.T, string){
		func(t *testing.T, root string) {
			t.Helper()
			if err := os.Rename(filepath.Join(root, "notes"), filepath.Join(root, "moved")); err != nil {
				t.Fatal(err)
			}
		},
		func(t *testing.T, root string) {
			t.Helper()
			if err := os.Symlink("missing.org", filepath.Join(root, "notes", "broken.org")); err != nil {
				t.Fatal(err)
			}
		},
	} {
		root := repository(t)
		mutate(t, root)
		var out bytes.Buffer
		if err := execute([]string{"BM25"}, root, &out); err == nil || out.Len() != 0 {
			t.Fatalf("source error not propagated: %v, %q", err, out.String())
		}
	}
}
