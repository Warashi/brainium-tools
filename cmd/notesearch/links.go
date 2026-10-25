package main

import (
	"path"

	"github.com/Warashi/brainium-tools/internal/notesearch/links"
	"github.com/Warashi/brainium-tools/internal/notesearch/presentation"
)

// withLinks は item に、そのノートのリンク先とリンク元を付ける。順位は
// 変えず、検索語が重ならないが 1 手で届くノートを見せる。
func withLinks(item *presentation.Item, graph links.Graph, name string) {
	item.LinksTo = toPresentation(graph.To(name))
	item.LinkedFrom = toPresentation(graph.From(name))
}

func toPresentation(notes []links.Note) []presentation.Link {
	if len(notes) == 0 {
		return nil
	}
	out := make([]presentation.Link, len(notes))
	for i, n := range notes {
		out[i] = presentation.Link{Path: path.Join(notesDir, n.Path), Title: n.Title}
	}
	return out
}
