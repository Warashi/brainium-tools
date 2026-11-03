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
