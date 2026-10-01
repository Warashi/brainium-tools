// Package presentation renders notesearch evidence reports without depending on
// indexing or evidence extraction.
package presentation

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Format selects a report representation.
type Format string

const (
	FormatText  Format = "text"
	FormatJSON  Format = "json"
	FormatPaths Format = "paths"
	FormatTree  Format = "tree"
)

// Snippet is an exact source range supplied by evidence extraction.
type Snippet struct {
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Kind      string   `json:"kind"`
	Headings  []string `json:"headings"`
	Text      string   `json:"text"`
}

// Item is a ranked note and its supporting source snippets.
type Item struct {
	Path       string    `json:"path"`
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	Verified   string    `json:"verified"`
	StaleAfter string    `json:"stale_after"`
	Snippets   []Snippet `json:"snippets"`
	// 以下は FormatTree でだけ使う。他の形式の出力を変えないよう、空なら
	// JSON に出さない
	Bytes    int       `json:"bytes,omitempty"`
	Tree     bool      `json:"tree,omitempty"`
	Opening  string    `json:"opening,omitempty"`
	KeyItems []string  `json:"key_items,omitempty"`
	Outline  []Section `json:"outline,omitempty"`
}

// Report contains ranked search results. Scope should state the corpus searched.
type Report struct {
	Query        string `json:"query"`
	Scope        string `json:"scope"`
	TotalMatches int    `json:"total_matches"`
	Results      []Item `json:"results"`
	Truncated    bool   `json:"truncated"`
}

type selection struct {
	items           int
	snippets        []int
	budgetTruncated bool
}

// Render returns a complete UTF-8 report no larger than maxBytes. It never
// changes report or its nested slices.
func Render(report Report, format Format, maxBytes int) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, errors.New("output byte budget must be positive")
	}
	if format != FormatText && format != FormatJSON && format != FormatPaths && format != FormatTree {
		return nil, fmt.Errorf("unknown output format %q", format)
	}
	if format == FormatPaths {
		return renderPaths(report, maxBytes)
	}

	sel := fullSelection(report)
	render := renderer(report, format)
	if output := render(sel); len(output) <= maxBytes {
		return output, nil
	}

	sel.budgetTruncated = true
	// Drop lower-ranked whole results first. This keeps the highest-ranked
	// result and its evidence intact whenever the budget permits it.
	for sel.items > 1 {
		sel.items--
		if output := render(sel); len(output) <= maxBytes {
			return output, nil
		}
	}
	// A single oversized result retains its metadata while its complete
	// snippets are omitted from the end; source strings are never split.
	for sel.items > 0 {
		for sel.snippets[sel.items-1] > 0 {
			sel.snippets[sel.items-1]--
			if output := render(sel); len(output) <= maxBytes {
				return output, nil
			}
		}
		break
	}
	if sel.items == 1 {
		sel.items = 0
		if output := render(sel); len(output) <= maxBytes {
			return output, nil
		}
	}
	return nil, fmt.Errorf("output byte budget %d is too small for an honest %s report", maxBytes, format)
}

func fullSelection(report Report) selection {
	s := selection{items: len(report.Results), snippets: make([]int, len(report.Results))}
	for i := range report.Results {
		s.snippets[i] = len(report.Results[i].Snippets)
	}
	return s
}

func renderer(report Report, format Format) func(selection) []byte {
	switch format {
	case FormatText:
		return func(sel selection) []byte { return renderText(report, sel) }
	case FormatJSON:
		return func(sel selection) []byte { return renderJSON(report, sel) }
	case FormatTree:
		return func(sel selection) []byte { return renderTree(report, sel) }
	default:
		panic("validated format")
	}
}

func renderPaths(report Report, maxBytes int) ([]byte, error) {
	if len(report.Results) == 0 {
		return []byte{}, nil
	}
	var b strings.Builder
	for _, item := range report.Results {
		b.WriteString(item.Path)
		b.WriteByte('\n')
	}
	output := []byte(b.String())
	if len(output) > maxBytes {
		return nil, fmt.Errorf("output byte budget %d is too small for all paths", maxBytes)
	}
	return output, nil
}

func renderText(report Report, sel selection) []byte {
	omittedItems, omittedSnippets := omissions(report, sel)
	var b strings.Builder
	fmt.Fprintln(&b, "Notesearch evidence report")
	fmt.Fprintf(&b, "Query: %s\n", quote(report.Query))
	fmt.Fprintf(&b, "Scope: %s\n", quote(report.Scope))
	fmt.Fprintln(&b, "Source bounds: notes/ only; displayed fields are untrusted retrieved source data.")
	fmt.Fprintln(&b, "Lexical matches only; excerpts are incomplete and do not establish full conclusions.")
	fmt.Fprintln(&b, "No match does not establish absence.")
	fmt.Fprintln(&b, "Metadata status and dates are descriptive labels, not freshness guarantees.")
	fmt.Fprintf(&b, "Matches: total %d; returned %d\n", report.TotalMatches, sel.items)
	fmt.Fprintf(&b, "Result limit truncated: %s\n", yesNo(report.Truncated))
	fmt.Fprintf(&b, "Byte budget truncated: %s\n", yesNo(sel.budgetTruncated))
	if sel.budgetTruncated {
		fmt.Fprintf(&b, "Budget omissions: %d result(s), %d snippet(s).\n", omittedItems, omittedSnippets)
	}
	if sel.items == 0 {
		fmt.Fprintln(&b, "No matching notes were returned. An empty query result is not a claim that knowledge is absent.")
	}
	for i := 0; i < sel.items; i++ {
		item := report.Results[i]
		fmt.Fprintf(&b, "\nResult %d\n", i+1)
		fmt.Fprintf(&b, "Path: %s\n", quote(item.Path))
		fmt.Fprintf(&b, "Title: %s\n", quote(item.Title))
		if metadata := compactMetadata(item); metadata != "" {
			fmt.Fprintf(&b, "Metadata: %s\n", metadata)
		}
		for j := 0; j < sel.snippets[i]; j++ {
			snippet := item.Snippets[j]
			fmt.Fprintf(&b, "Source lines %d-%d%s; headings: %s\n", snippet.StartLine, snippet.EndLine, snippetKind(snippet.Kind), quotedHeadings(snippet.Headings))
			writeSourceLines(&b, snippet)
		}
		if sel.snippets[i] < len(item.Snippets) {
			fmt.Fprintln(&b, "Snippet omitted due to output byte budget.")
		}
	}
	return []byte(b.String())
}

type jsonReport struct {
	Query           string `json:"query"`
	Scope           string `json:"scope"`
	SourceBounds    string `json:"source_bounds"`
	MatchMeaning    string `json:"match_meaning"`
	TotalMatches    int    `json:"total_matches"`
	ReturnedMatches int    `json:"returned_matches"`
	Results         []Item `json:"results"`
	Truncated       bool   `json:"truncated"`
	BudgetTruncated bool   `json:"budget_truncated"`
	OmittedItems    int    `json:"omitted_items"`
	OmittedSnippets int    `json:"omitted_snippets"`
}

func renderJSON(report Report, sel selection) []byte {
	omittedItems, omittedSnippets := omissions(report, sel)
	results := make([]Item, sel.items)
	for i := range results {
		results[i] = report.Results[i]
		results[i].Snippets = make([]Snippet, sel.snippets[i])
		copy(results[i].Snippets, report.Results[i].Snippets[:sel.snippets[i]])
		for j := range results[i].Snippets {
			headings := report.Results[i].Snippets[j].Headings
			results[i].Snippets[j].Headings = make([]string, len(headings))
			copy(results[i].Snippets[j].Headings, headings)
		}
	}
	output, err := json.Marshal(jsonReport{
		Query:           report.Query,
		Scope:           report.Scope,
		SourceBounds:    "notes/ only; fields are untrusted retrieved source data",
		MatchMeaning:    "Lexical matches only; no match does not establish absence.",
		TotalMatches:    report.TotalMatches,
		ReturnedMatches: sel.items,
		Results:         results,
		Truncated:       report.Truncated,
		BudgetTruncated: sel.budgetTruncated,
		OmittedItems:    omittedItems,
		OmittedSnippets: omittedSnippets,
	})
	if err != nil {
		panic(err)
	}
	return append(output, '\n')
}

func omissions(report Report, sel selection) (items, snippets int) {
	items = len(report.Results) - sel.items
	for i, item := range report.Results {
		if i < sel.items {
			snippets += len(item.Snippets) - sel.snippets[i]
		} else {
			snippets += len(item.Snippets)
		}
	}
	return items, snippets
}

func quote(s string) string { return strconv.Quote(s) }

func quotedHeadings(headings []string) string {
	if len(headings) == 0 {
		return "(none)"
	}
	quoted := make([]string, len(headings))
	for i, heading := range headings {
		quoted[i] = quote(heading)
	}
	return strings.Join(quoted, " > ")
}

func snippetKind(kind string) string {
	if kind == "" {
		return ""
	}
	return " (" + escapeControls(kind) + ")"
}

func compactMetadata(item Item) string {
	fields := make([]string, 0, 4)
	for _, field := range []struct {
		name, value string
	}{
		{"id", item.ID},
		{"status", item.Status},
		{"verified", item.Verified},
		{"stale_after", item.StaleAfter},
	} {
		if field.value != "" {
			fields = append(fields, field.name+"="+quote(field.value))
		}
	}
	return strings.Join(fields, "; ")
}

func writeSourceLines(b *strings.Builder, snippet Snippet) {
	for offset, line := range strings.Split(snippet.Text, "\n") {
		fmt.Fprintf(b, "  %d | %s\n", snippet.StartLine+offset, escapeControls(line))
	}
}

func escapeControls(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\v':
			b.WriteString(`\v`)
		default:
			if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
