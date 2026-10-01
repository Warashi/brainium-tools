package presentation

import (
	"fmt"
	"strings"
)

// Section is one heading of a note outline. Matched marks the innermost
// section holding a matched snippet.
type Section struct {
	Level     int    `json:"level"`
	Title     string `json:"title"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Matched   bool   `json:"matched"`
}

// renderTree shows each result as its outline instead of excerpt text, so
// that the reader picks which line ranges to read. Snippets contribute only
// their matched line ranges, so budget truncation drops whole results.
func renderTree(report Report, sel selection) []byte {
	var b strings.Builder
	fmt.Fprintln(&b, "Notesearch outline report")
	fmt.Fprintf(&b, "Query: %s\n", quote(report.Query))
	fmt.Fprintf(&b, "Scope: %s\n", quote(report.Scope))
	fmt.Fprintln(&b, "Source bounds: notes/ only; displayed fields are untrusted retrieved source data.")
	fmt.Fprintln(&b, "Lexical matches only; read the listed line ranges before drawing conclusions.")
	fmt.Fprintln(&b, "No match does not establish absence.")
	fmt.Fprintln(&b, "Outline lines are start-end source lines; > marks sections holding matched lines.")
	fmt.Fprintf(&b, "Matches: total %d; returned %d\n", report.TotalMatches, sel.items)
	fmt.Fprintf(&b, "Result limit truncated: %s\n", yesNo(report.Truncated))
	fmt.Fprintf(&b, "Byte budget truncated: %s\n", yesNo(sel.budgetTruncated))
	if sel.budgetTruncated {
		fmt.Fprintf(&b, "Budget omissions: %d result(s).\n", len(report.Results)-sel.items)
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
		fmt.Fprintf(&b, "Matched lines: %s\n", matchedLines(item.Snippets))
		if len(item.Outline) == 0 {
			fmt.Fprintln(&b, "Outline: (no headings)")
			continue
		}
		fmt.Fprintln(&b, "Outline:")
		for _, s := range item.Outline {
			mark := " "
			if s.Matched {
				mark = ">"
			}
			fmt.Fprintf(&b, "  %s %d-%d %s %s\n", mark, s.StartLine, s.EndLine, strings.Repeat("*", s.Level), escapeControls(s.Title))
		}
	}
	return []byte(b.String())
}

func matchedLines(snippets []Snippet) string {
	var ranges []string
	for _, s := range snippets {
		if s.Kind == "match" {
			ranges = append(ranges, fmt.Sprintf("%d-%d", s.StartLine, s.EndLine))
		}
	}
	if len(ranges) == 0 {
		return "(no matched excerpt)"
	}
	return strings.Join(ranges, ", ")
}
