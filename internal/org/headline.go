package org

// HeadlineProperties は見出しの PROPERTIES ドロワーを返す。無ければ nil。
// org は見出しの直後か計画行の直後にあるドロワーだけを PROPERTIES ドロ
// ワーとして解析するので、section の要素を順に見ればよい。
func HeadlineProperties(headline *Node) *Node {
	if len(headline.Children) == 0 || headline.Children[0].Type != TypeSection {
		return nil
	}
	for _, n := range headline.Children[0].Children {
		if n.Type == TypePropertyDrawer {
			return n
		}
	}
	return nil
}
