package presentation

import (
	"fmt"
	"strings"
)

// Section is one heading of a pruned note outline. Bytes covers the section
// and its descendants, KeyItems names folded descendants, and Matched marks
// the innermost shown section holding a matched snippet.
type Section struct {
	Level     int      `json:"level"`
	Title     string   `json:"title"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Bytes     int      `json:"bytes"`
	KeyItems  []string `json:"key_items,omitempty"`
	Matched   bool     `json:"matched"`
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
	fmt.Fprintln(&b, "Outline lines are start-end source lines with sizes; > marks sections holding matched lines.")
	fmt.Fprintln(&b, "tree: worth navigating the outline; read whole: reading the note costs less, so its headings are listed as Key.")
	fmt.Fprintln(&b, linksLegend)
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
		writeLinks(&b, report, sel, item)
		navigation := "read whole"
		if item.Tree {
			navigation = "tree"
		}
		fmt.Fprintf(&b, "Size: %s; %s\n", kb(item.Bytes), navigation)
		if item.Opening != "" {
			fmt.Fprintf(&b, "Opening: %s\n", quote(item.Opening))
		}
		fmt.Fprintf(&b, "Matched lines: %s\n", matchedLines(item.Snippets))
		if len(item.KeyItems) > 0 {
			fmt.Fprintf(&b, "Key: %s\n", keyList(item.KeyItems))
		}
		if len(item.Outline) == 0 {
			if len(item.KeyItems) == 0 {
				fmt.Fprintln(&b, "Outline: (no headings)")
			}
			continue
		}
		fmt.Fprintln(&b, "Outline:")
		for _, s := range item.Outline {
			mark := " "
			if s.Matched {
				mark = ">"
			}
			fmt.Fprintf(&b, "  %s %d-%d %s %s %s", mark, s.StartLine, s.EndLine, kb(s.Bytes), strings.Repeat("*", s.Level), escapeControls(s.Title))
			if len(s.KeyItems) > 0 {
				fmt.Fprintf(&b, "; key: %s", keyList(s.KeyItems))
			}
			b.WriteByte('\n')
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

func kb(n int) string { return fmt.Sprintf("%.1fKB", float64(n)/1024) }

func keyList(items []string) string {
	escaped := make([]string, len(items))
	for i, item := range items {
		escaped[i] = escapeControls(item)
	}
	return strings.Join(escaped, " / ")
}
