package orglint

import (
	"fmt"

	"github.com/Warashi/brainium-tools/internal/org"
)

// maxTaskLines は tasks.org の 1 task(見出し行と下位見出しを含む)に
// 許す行数。task 本文は現状と次の一手だけを書く規約で、超えた分は
// note や journal へ移す合図とする。個別に上限を上書きする手段を設け
// ないのは、それが追記を続ける抜け道になるため。
const maxTaskLines = 40

// checkTaskLength は tasks.org の ** 見出しごとに、下位見出しを含む
// 見出しの範囲の行数が maxTaskLines を超えるものを報告する。
func checkTaskLength(f orgFile) []Problem {
	if f.path != "tasks.org" {
		return nil
	}
	var problems []Problem
	var walk func(n *org.Node)
	walk = func(n *org.Node) {
		for _, c := range n.Children {
			if c.Type != org.TypeHeadline {
				continue
			}
			if c.Level != 2 {
				walk(c)
				continue
			}
			if lines := f.lineCount(c.Begin, c.End); lines > maxTaskLines {
				problems = append(problems, Problem{
					Path:    f.path,
					Line:    f.lineAt(c.Begin),
					Message: fmt.Sprintf("task が %d 行あり、上限の %d 行を超える。経緯は journal、知識は note へ移す", lines, maxTaskLines),
				})
			}
		}
	}
	walk(f.root)
	return problems
}
