package presentation

import (
	"fmt"
	"strings"
)

// Section is one heading of a note outline. Bytes covers the section and
// its descendants, and Matched marks a section holding a matched snippet.
type Section struct {
	Level     int    `json:"level"`
	Title     string `json:"title"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Bytes     int    `json:"bytes,omitempty"`
	Matched   bool   `json:"matched,omitempty"`
}

// MatchedSection locates one matched snippet: its lines, the innermost
// section holding it, and the headings from the top down to that section.
// SectionStart is 0 when the snippet precedes the first heading.
type MatchedSection struct {
	StartLine    int       `json:"start_line"`
	EndLine      int       `json:"end_line"`
	SectionStart int       `json:"section_start,omitempty"`
	SectionEnd   int       `json:"section_end,omitempty"`
	Bytes        int       `json:"bytes,omitempty"`
	Chain        []Section `json:"chain,omitempty"`
}

// renderTree shows each result as the sections holding its matches and its
// top-level outline instead of excerpt text, so that the reader picks which
// line ranges to read. Deeper headings are left to notetree: listing them
// let one large note crowd every other hit out of the budget.
func renderTree(report Report, sel selection) []byte {
	var b strings.Builder
	fmt.Fprintln(&b, "Notesearch outline report")
	fmt.Fprintf(&b, "Query: %s\n", quote(report.Query))
	fmt.Fprintf(&b, "Scope: %s\n", quote(report.Scope))
	fmt.Fprintln(&b, "Source bounds: notes/ only; displayed fields are untrusted retrieved source data.")
	fmt.Fprintln(&b, "Lexical matches only; read the listed line ranges before drawing conclusions.")
	fmt.Fprintln(&b, "No match does not establish absence.")
	fmt.Fprintln(&b, "Matched sections: matched lines, the innermost section holding them, and its headings.")
	fmt.Fprintln(&b, "Outline lists top-level sections with lines and sizes; > marks those holding matches; run notetree <path> for all headings.")
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
		writeMatched(&b, item.Matched)
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
			fmt.Fprintf(&b, "  %s %d-%d %s %s %s\n", mark, s.StartLine, s.EndLine, kb(s.Bytes), strings.Repeat("*", s.Level), escapeControls(s.Title))
		}
	}
	return []byte(b.String())
}

func writeMatched(b *strings.Builder, matched []MatchedSection) {
	if len(matched) == 0 {
		fmt.Fprintln(b, "Matched sections: (no matched excerpt)")
		return
	}
	fmt.Fprintln(b, "Matched sections:")
	for _, m := range matched {
		if m.SectionStart == 0 {
			fmt.Fprintf(b, "  %d-%d before the first heading\n", m.StartLine, m.EndLine)
			continue
		}
		chain := make([]string, len(m.Chain))
		for i, s := range m.Chain {
			chain[i] = strings.Repeat("*", s.Level) + " " + escapeControls(s.Title)
		}
		fmt.Fprintf(b, "  %d-%d in %d-%d %s: %s\n", m.StartLine, m.EndLine, m.SectionStart, m.SectionEnd, kb(m.Bytes), strings.Join(chain, " > "))
	}
}

func kb(n int) string { return fmt.Sprintf("%.1fKB", float64(n)/1024) }

func keyList(items []string) string {
	escaped := make([]string, len(items))
	for i, item := range items {
		escaped[i] = escapeControls(item)
	}
	return strings.Join(escaped, " / ")
}
