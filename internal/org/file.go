package org

import "strings"

// FileProperties は最初の見出しより前の section にある PROPERTIES ドロ
// ワーを返す。無ければ nil。
func FileProperties(root *Node) *Node {
	if len(root.Children) == 0 || root.Children[0].Type != TypeSection {
		return nil
	}
	for _, n := range root.Children[0].Children {
		if n.Type == TypePropertyDrawer {
			return n
		}
	}
	return nil
}

// FileTitle は最初の見出しより前の section にある最初の TITLE keyword の
// 値を返す。無ければ空文字列。ドロワーの中の keyword は section の直接の
// 子ではないので拾わない。
func FileTitle(root *Node) string {
	if len(root.Children) == 0 || root.Children[0].Type != TypeSection {
		return ""
	}
	for _, n := range root.Children[0].Children {
		if n.Type == TypeKeyword && n.Key == "TITLE" {
			return n.Value
		}
	}
	return ""
}

// FileTags は #+FILETAGS の tag を文書の順に返す。org はファイルのどこ
// にある FILETAGS の keyword も集める(org-collect-keywords)。
func FileTags(root *Node) []string {
	var tags []string
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Type == TypeKeyword && n.Key == "FILETAGS" {
			for _, word := range strings.Fields(n.Value) {
				for _, tag := range strings.Split(word, ":") {
					if tag != "" {
						tags = append(tags, tag)
					}
				}
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return tags
}
