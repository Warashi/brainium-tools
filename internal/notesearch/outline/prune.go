package outline

// Pruned は畳んだ後の文書の目次。
type Pruned struct {
	Bytes     int
	Collapsed bool     // 文書ごと丸ごと読むほうが安い
	KeyItems  []string // Collapsed のとき、畳んだ見出し
	Sections  []Node   // Collapsed でないとき、残した節
}

// Node は残した節。KeyItems は畳んだ配下の見出しで、Bytes は配下を含む
// 原文のバイト数。
type Node struct {
	Section
	Bytes    int
	KeyItems []string
}

// Sized は全節を、配下を含むバイト数付きで返す。
func Sized(source string) []Node {
	sections := Parse(source)
	offsets := lineOffsets(source)
	nodes := make([]Node, len(sections))
	for i, s := range sections {
		nodes[i] = Node{Section: s, Bytes: byteAt(source, offsets, s.EndLine+1) - byteAt(source, offsets, s.StartLine)}
	}
	return nodes
}

// Prune は、目次をたどるより丸ごと読むほうが安い部分木を畳む。
//
// PageIndex の tree_optimize の merge を、ページ数の代わりに原文のバイト
// 数で数えたもの。節 v を丸ごと読む量は S(v)、たどる場合の最悪の量は
// R + max(v の子に入らない部分, 子の最悪の量) で、routeCost が R に当た
// る。下から順に S(v) がたどる量以下なら畳み、消した見出しは KeyItems に
// 残す。消すと、その節に何が書いてあるかの手掛かりまで失うため。文書全
// 体も、最初の見出しより前を子に入らない部分とする根として同じ判定を
// 受ける。
func Prune(source string, routeCost int) Pruned {
	nodes := Sized(source)
	offsets := lineOffsets(source)
	at := func(line int) int { return byteAt(source, offsets, line) }

	n := len(nodes)
	sections := make([]Section, n)
	size := make([]int, n)
	own := make([]int, n)
	parent := make([]int, n)
	var stack []int
	for i, node := range nodes {
		s := node.Section
		sections[i] = s
		size[i] = node.Bytes
		own[i] = size[i]
		for len(stack) > 0 && sections[stack[len(stack)-1]].Level >= s.Level {
			stack = stack[:len(stack)-1]
		}
		parent[i] = -1
		if len(stack) > 0 {
			parent[i] = stack[len(stack)-1]
		}
		stack = append(stack, i)
	}
	// 子に入らない部分は、見出しから最初の子の直前まで。子は出現順に並ぶ
	// ので、後ろから上書きすれば最初の子の位置が残る
	rootOwn := len(source)
	for i := n - 1; i >= 0; i-- {
		if p := parent[i]; p >= 0 {
			own[p] = at(sections[i].StartLine) - at(sections[p].StartLine)
		} else {
			rootOwn = at(sections[i].StartLine)
		}
	}

	// worst は子の最悪の量の最大値。親は子より前に並ぶので、後ろから見れ
	// ば子は親より先に決まる
	cost := make([]int, n)
	worst := make([]int, n)
	collapsed := make([]bool, n)
	hasChild := make([]bool, n)
	rootWorst, rootHasChild := 0, false
	for i := n - 1; i >= 0; i-- {
		cost[i] = size[i]
		if hasChild[i] {
			if tree := routeCost + max(own[i], worst[i]); size[i] > tree {
				cost[i] = tree
			} else {
				collapsed[i] = true
			}
		}
		if p := parent[i]; p >= 0 {
			hasChild[p] = true
			worst[p] = max(worst[p], cost[i])
		} else {
			rootHasChild = true
			rootWorst = max(rootWorst, cost[i])
		}
	}

	out := Pruned{Bytes: len(source)}
	if !rootHasChild || len(source) <= routeCost+max(rootOwn, rootWorst) {
		out.Collapsed = true
		for _, s := range sections {
			out.KeyItems = append(out.KeyItems, s.Title)
		}
		return out
	}
	// 畳んだ節の配下は表に出さず、見出しを畳んだ節の KeyItems に載せる。
	// owner は見出しを載せる先の out.Sections の添字
	shown := make([]bool, n)
	owner := make([]int, n)
	for i := range sections {
		p := parent[i]
		if p >= 0 && (!shown[p] || collapsed[p]) {
			owner[i] = owner[p]
			node := &out.Sections[owner[i]]
			node.KeyItems = append(node.KeyItems, sections[i].Title)
			continue
		}
		shown[i] = true
		owner[i] = len(out.Sections)
		out.Sections = append(out.Sections, Node{Section: sections[i], Bytes: size[i]})
	}
	return out
}

// byteAt は 1 始まりの line の先頭のバイト位置。最終行より後なら末尾。
func byteAt(source string, offsets []int, line int) int {
	if line-1 < len(offsets) {
		return offsets[line-1]
	}
	return len(source)
}

// lineOffsets は各行の先頭のバイト位置を返す。行の数え方は Parse と同じ。
func lineOffsets(source string) []int {
	offsets := []int{0}
	for i := 0; i < len(source)-1; i++ {
		if source[i] == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}
