// Package maturity は恒久ノートの成熟度(:STATUS: / :VERIFIED: /
// :STALE_AFTER:)を読み、規約に沿っているかの検査と、精錬待ちかの判定を
// 行う。
package maturity

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Property はファイルレベルの PROPERTIES ドロワーにある値 1 つ。Line は
// 1 始まりで、値が無いときは 0。
type Property struct {
	Value string
	Line  int
}

// Props はノート 1 つの成熟度のプロパティ。
type Props struct {
	Status     Property
	Verified   Property
	StaleAfter Property
}

var (
	propRe = regexp.MustCompile(`^\s*:([A-Za-z_]+):\s*(.*?)\s*$`)
	// dateRe は org のタイムスタンプ。曜日と時刻は省略を許す。
	dateRe = regexp.MustCompile(`^[\[<](\d{4}-\d{2}-\d{2})(?: [^\]>\d][^\]>]*?)?(?: \d{1,2}:\d{2})?[\]>]$`)
)

// Parse はファイル先頭の PROPERTIES ドロワーだけを読む。見出しに付いた
// ドロワーはノート全体の成熟度ではないので見ない。
func Parse(content string) Props {
	var p Props
	lines := strings.Split(content, "\n")
	first := 0
	for first < len(lines) && strings.TrimSpace(lines[first]) == "" {
		first++
	}
	if first >= len(lines) || !strings.EqualFold(strings.TrimSpace(lines[first]), ":PROPERTIES:") {
		return p
	}
	for i := first + 1; i < len(lines); i++ {
		m := propRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		prop := Property{Value: m[2], Line: i + 1}
		switch strings.ToUpper(m[1]) {
		case "END":
			return p
		case "STATUS":
			p.Status = prop
		case "VERIFIED":
			p.Verified = prop
		case "STALE_AFTER":
			p.StaleAfter = prop
		}
	}
	return p
}

// Problem は成熟度の規約違反 1 件。Line が 0 の違反は行を特定しない。
type Problem struct {
	Line    int
	Message string
}

// Check は成熟度のプロパティが規約に沿っているかを検査する。日付の値が
// 読めないと Pending が期限を判定できないので、書式まで見る。今日の日付
// との比較はしない。時間が経つだけで lint が落ちるようになるため。
func Check(p Props) []Problem {
	var problems []Problem
	switch p.Status.Value {
	case "":
		problems = append(problems, Problem{Message: ":STATUS: を必須とする"})
	case "draft", "deprecated":
	case "stable":
		if p.Verified.Value == "" {
			problems = append(problems, Problem{Line: p.Status.Line, Message: "stable には :VERIFIED: を必須とする"})
		}
	default:
		problems = append(problems, Problem{Line: p.Status.Line, Message: fmt.Sprintf(":STATUS: %s は draft / stable / deprecated のいずれかにする", p.Status.Value)})
	}
	for _, d := range []struct {
		name string
		prop Property
	}{{"VERIFIED", p.Verified}, {"STALE_AFTER", p.StaleAfter}} {
		if d.prop.Value == "" {
			continue
		}
		if _, ok := parseDate(d.prop.Value); !ok {
			problems = append(problems, Problem{Line: d.prop.Line, Message: fmt.Sprintf(":%s: が org の日付 [YYYY-MM-DD Day] として読めない", d.name)})
		}
	}
	return problems
}

// Pending はノートが精錬待ちである理由を返す。精錬待ちでなければ空。
// draft は昇格待ち、:STALE_AFTER: の日付を過ぎたものは再検証待ち。
// deprecated は履歴として残すものなので、どちらも対象にしない。
func Pending(p Props, today time.Time) []string {
	if p.Status.Value == "deprecated" {
		return nil
	}
	var reasons []string
	if p.Status.Value == "draft" {
		reasons = append(reasons, "draft")
	}
	if d, ok := parseDate(p.StaleAfter.Value); ok {
		y, m, dd := today.Date()
		if d.Before(time.Date(y, m, dd, 0, 0, 0, 0, time.UTC)) {
			reasons = append(reasons, "stale_after "+p.StaleAfter.Value)
		}
	}
	return reasons
}

func parseDate(v string) (time.Time, bool) {
	m := dateRe.FindStringSubmatch(v)
	if m == nil {
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", m[1])
	return d, err == nil
}
