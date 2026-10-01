package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIWholeOutputBudget(t *testing.T) {
	root := repository(t)
	for _, format := range []string{"text", "json"} {
		var full bytes.Buffer
		if err := execute([]string{"-format", format, "baseline"}, root, &full); err != nil {
			t.Fatal(err)
		}
		for _, budget := range []struct {
			flag string
			size int
		}{{"1", 1}, {"600", 600}, {"1000", 1000}, {"2000", 2000}, {"24000", 24000}} {
			var out bytes.Buffer
			err := execute([]string{"-format", format, "-max-bytes", budget.flag, "baseline"}, root, &out)
			if err != nil {
				if out.Len() != 0 {
					t.Errorf("error produced partial %s output: %q", format, out.String())
				}
				continue
			}
			if out.Len() > budget.size {
				t.Fatalf("%s budget %d exceeded: %d", format, budget.size, out.Len())
			}
			if format != "json" {
				continue
			}
			var report struct {
				TotalMatches    int               `json:"total_matches"`
				ReturnedMatches int               `json:"returned_matches"`
				BudgetTruncated bool              `json:"budget_truncated"`
				Results         []json.RawMessage `json:"results"`
			}
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatalf("budget produced invalid JSON: %v", err)
			}
			if report.TotalMatches != 2 || report.ReturnedMatches != len(report.Results) {
				t.Errorf("budget obscures result counts: %s", out.String())
			}
			if full.Len() > budget.size && !report.BudgetTruncated {
				t.Errorf("budget omission unreported: %s", out.String())
			}
		}
	}
}

func TestJSONRejectsNonUTF8SourceInsteadOfReplacingBytes(t *testing.T) {
	root := repository(t)
	if err := os.WriteFile(filepath.Join(root, "notes", "invalid.org"), []byte("invalidencoding\n\xff\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := execute([]string{"-format", "json", "invalidencoding"}, root, &out); err == nil || out.Len() != 0 {
		t.Fatalf("invalid UTF-8 source must fail explicitly: %v, %q", err, out.String())
	}
}
