package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCLITreeMarksSectionsHoldingMatches(t *testing.T) {
	root := repository(t)
	var out bytes.Buffer
	if err := execute([]string{"-format", "tree", "BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`Path: "notes/search.org"`,
		"Matched lines: 10-11\n",
		"Outline:\n  > 9-11 * Findings\n",
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
	if _, ok := report.Results[0]["outline"]; ok {
		t.Errorf("json output gained an outline field: %s", out.String())
	}
}
