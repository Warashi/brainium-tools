package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLITreeFoldsShortNotes(t *testing.T) {
	root := repository(t)
	var out bytes.Buffer
	if err := execute([]string{"-format", "tree", "BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	want := "Path: \"notes/search.org\"\n" +
		"Title: \"Search decisions\"\n" +
		"Metadata: id=\"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa\"; status=\"draft\"; verified=\"[2026-09-01 Tue]\"; stale_after=\"[2026-10-01 Thu]\"\n" +
		"Size: 0.2KB; read whole\n" +
		"Matched sections:\n  10-11 in 9-11 0.1KB: * Findings\n" +
		"Key: Findings\n"
	if !strings.Contains(out.String(), want) {
		t.Errorf("tree output does not contain %q:\n%s", want, out.String())
	}
}

func TestCLITreeLocatesDeepMatchUnderTopLevelOutline(t *testing.T) {
	root := repository(t)
	long := "#+title: Long decisions\nBM25 の経緯をまとめる。\n" + // 1-2
		"* History\n" + strings.Repeat("filler line\n", 600) + // 3-603
		"* Ranking\n" + strings.Repeat("filler line\n", 600) + // 604-1204
		"** Baseline\nBM25 stays the baseline.\n" // 1205-1206
	if err := os.WriteFile(filepath.Join(root, "notes", "long.org"), []byte(long), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := execute([]string{"-format", "tree", "baseline"}, root, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Size: 14.2KB; tree\n",
		"Opening: \"BM25 の経緯をまとめる。\"\n",
		"Matched sections:\n  1206-1206 in 1205-1206 0.0KB: * Ranking > ** Baseline\n",
		"Outline:\n    3-603 7.0KB * History\n  > 604-1206 7.1KB * Ranking\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("tree output does not contain %q:\n%s", want, out.String())
		}
	}
}

func TestCLIOtherFormatsDoNotCarryOutline(t *testing.T) {
	root := repository(t)
	var out bytes.Buffer
	if err := execute([]string{"-format", "json", "BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Results []map[string]json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"outline", "opening", "tree", "key_items", "bytes", "matched_sections"} {
		if _, ok := report.Results[0][field]; ok {
			t.Errorf("json output gained a %s field: %s", field, out.String())
		}
	}
}
