package org

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
