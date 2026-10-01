package main

import (
	"github.com/Warashi/brainium-tools/internal/notesearch/outline"
	"github.com/Warashi/brainium-tools/internal/notesearch/presentation"
)

// withOutline は item に、notetree <path> と同じ基準で剪定した目次と冒頭
// の文を付ける。一致した抜粋は、それを含む最も深い表示中の節に印を付け
// る。畳んだ節の範囲は配下を含むので、配下で一致すれば畳んだ節に付く。
// 文書ごと畳んだときは印を付ける節が無く、行番号だけが手掛かりになる。
func withOutline(item *presentation.Item, source string) {
	p := outline.Prune(source, outline.DefaultRouteBytes)
	item.Bytes, item.Tree = p.Bytes, !p.Collapsed
	item.Opening = outline.Opening(source, outline.DefaultOpeningBytes)
	item.KeyItems = p.KeyItems
	shown := make([]outline.Section, len(p.Sections))
	for i, node := range p.Sections {
		shown[i] = node.Section
		item.Outline = append(item.Outline, presentation.Section{
			Level: node.Level, Title: node.Title, StartLine: node.StartLine, EndLine: node.EndLine,
			Bytes: node.Bytes, KeyItems: node.KeyItems,
		})
	}
	for _, snippet := range item.Snippets {
		if snippet.Kind != "match" {
			continue
		}
		if i := outline.Innermost(shown, snippet.StartLine); i >= 0 {
			item.Outline[i].Matched = true
		}
	}
}
