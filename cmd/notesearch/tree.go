package main

import (
	"github.com/Warashi/brainium-tools/internal/notesearch/outline"
	"github.com/Warashi/brainium-tools/internal/notesearch/presentation"
)

// markedOutline は source の見出しの木に、一致した抜粋を含む最も深い節
// の印を付ける。抜粋が最初の見出しより前なら印は付かない。
func markedOutline(source string, snippets []presentation.Snippet) []presentation.Section {
	sections := outline.Parse(source)
	out := make([]presentation.Section, len(sections))
	for i, s := range sections {
		out[i] = presentation.Section{Level: s.Level, Title: s.Title, StartLine: s.StartLine, EndLine: s.EndLine}
	}
	for _, snippet := range snippets {
		if snippet.Kind != "match" {
			continue
		}
		if i := outline.Innermost(sections, snippet.StartLine); i >= 0 {
			out[i].Matched = true
		}
	}
	return out
}
