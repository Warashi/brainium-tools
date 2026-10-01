package orglint

import (
	"fmt"
	"regexp"
)

// maxTaskLines は tasks.org の 1 task(見出し行と下位見出しを含む)に
// 許す行数。task 本文は現状と次の一手だけを書く規約で、超えた分は
// note や journal へ移す合図とする。個別に上限を上書きする手段を設け
// ないのは、それが追記を続ける抜け道になるため。
const maxTaskLines = 40

var taskHeadingRe = regexp.MustCompile(`^\*\* `)

// checkTaskLength は tasks.org の ** 見出しごとに、次の * / ** 見出し
// の直前までの行数が maxTaskLines を超えるものを報告する。
func checkTaskLength(f orgFile) []Problem {
	if f.path != "tasks.org" {
		return nil
	}
	var problems []Problem
	start := -1
	flush := func(end int) {
		if start >= 0 && end-start > maxTaskLines {
			problems = append(problems, Problem{
				Path:    f.path,
				Line:    start + 1,
				Message: fmt.Sprintf("task が %d 行あり、上限の %d 行を超える。経緯は journal、知識は note へ移す", end-start, maxTaskLines),
			})
		}
		start = -1
	}
	for i, line := range f.lines {
		if f.kinds[i] == kindBlock {
			continue
		}
		switch {
		case topHeadingRe.MatchString(line):
			flush(i)
		case taskHeadingRe.MatchString(line):
			flush(i)
			start = i
		}
	}
	end := len(f.lines)
	if end > 0 && f.lines[end-1] == "" {
		// 末尾の改行で strings.Split が作る空要素は行ではない。
		end--
	}
	flush(end)
	return problems
}
