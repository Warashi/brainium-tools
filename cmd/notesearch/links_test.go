package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// linkedRepository は sampleNote を、検索語を含まない 2 つのノートとリン
// クでつなぐ。
func linkedRepository(t *testing.T) string {
	t.Helper()
	root := repository(t)
	for name, text := range map[string]string{
		"notes/target.org": ":PROPERTIES:\n:ID: bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb\n:END:\n#+title: Linked target\n\nunrelated words\n",
		"notes/source.org": ":PROPERTIES:\n:ID: cccccccc-cccc-cccc-cccc-cccccccccccc\n:END:\n#+title: Linking source\n\n[[id:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa][検索の判断]]\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	search := sampleNote + "See [[id:bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb][the target]].\n"
	if err := os.WriteFile(filepath.Join(root, "notes/search.org"), []byte(search), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCLIShowsLinksOfHits(t *testing.T) {
	root := linkedRepository(t)
	want := "Links:\n  <- \"notes/source.org\"\n" +
		"  -> \"notes/target.org\"\n"
	for _, format := range []string{"text", "tree"} {
		var out bytes.Buffer
		if err := execute([]string{"-format", format, "BM25"}, root, &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), want) {
			t.Errorf("%s output does not contain %q:\n%s", format, want, out.String())
		}
	}
}

func TestCLIJSONCarriesLinksOfHits(t *testing.T) {
	root := linkedRepository(t)
	var out bytes.Buffer
	if err := execute([]string{"-format", "json", "BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	type note struct {
		Path  string `json:"path"`
		Title string `json:"title"`
	}
	var report struct {
		Results []struct {
			LinksTo    []note `json:"links_to"`
			LinkedFrom []note `json:"linked_from"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	got := report.Results[0]
	if len(got.LinksTo) != 1 || got.LinksTo[0] != (note{"notes/target.org", "Linked target"}) ||
		len(got.LinkedFrom) != 1 || got.LinkedFrom[0] != (note{"notes/source.org", "Linking source"}) {
		t.Errorf("unexpected links: %s", out.String())
	}
}

func TestCLIOmitsLinksSectionWithoutLinks(t *testing.T) {
	root := repository(t)
	var out bytes.Buffer
	if err := execute([]string{"BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "\nLinks:\n") {
		t.Errorf("output shows an empty links section:\n%s", out.String())
	}
}

func TestCLIRefersToShownResultsByNumber(t *testing.T) {
	root := linkedRepository(t)
	both := ":PROPERTIES:\n:ID: bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb\n:END:\n#+title: Linked target\n\n" +
		"BM25 [[id:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa][back]]\n"
	if err := os.WriteFile(filepath.Join(root, "notes/target.org"), []byte(both), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := execute([]string{"BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Links:\n  <> \"notes/search.org\" (Result 2)\n",
		"Links:\n  <- \"notes/source.org\"\n  <> \"notes/target.org\" (Result 1)\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, out.String())
		}
	}
}

func TestCLIDoesNotReferToResultsDroppedByBudget(t *testing.T) {
	root := linkedRepository(t)
	both := ":PROPERTIES:\n:ID: bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb\n:END:\n#+title: Linked target\n\n" +
		"BM25 " + strings.Repeat("padding ", 400) + "\n"
	if err := os.WriteFile(filepath.Join(root, "notes/target.org"), []byte(both), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := execute([]string{"-max-bytes", "2000", "BM25"}, root, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Result 2") {
		t.Fatalf("budget did not drop the second result; adjust the fixture:\n%s", out.String())
	}
	if want := "  -> \"notes/target.org\"\n"; !strings.Contains(out.String(), want) {
		t.Errorf("output does not contain %q:\n%s", want, out.String())
	}
}
