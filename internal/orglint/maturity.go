package orglint

import (
	"strings"

	"github.com/Warashi/brainium-tools/internal/maturity"
)

// checkMaturity は notes/ の恒久ノートの成熟度プロパティを検査する。
// journal や tasks.org は成熟度を持たないので対象にしない。
func checkMaturity(f orgFile) []Problem {
	if !strings.HasPrefix(f.path, "notes/") {
		return nil
	}
	var problems []Problem
	for _, p := range maturity.Check(maturity.Parse(strings.Join(f.lines, "\n"))) {
		problems = append(problems, Problem{Path: f.path, Line: p.Line, Message: p.Message})
	}
	return problems
}
