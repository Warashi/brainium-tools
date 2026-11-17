package orgedit

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Warashi/brainium-tools/internal/org"
)

var (
	// whenRe は Schedule が受ける日時。日付と、時刻かその範囲。
	whenRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})(?: (\d{1,2}:\d{2})(?:-(\d{1,2}:\d{2}))?)?$`)
	// scheduledRe は計画行の SCHEDULED を、次のキーワードの手前まで。
	scheduledRe = regexp.MustCompile(`SCHEDULED: *<[^>\n]*>`)
)

// Schedule は :ID: が id の見出しの SCHEDULED を when にする
// (org-schedule)。when は YYYY-MM-DD に、空白を挟んで H:MM か
// H:MM-H:MM を続けてよい。元の SCHEDULED の repeater は保つ。when が空
// なら SCHEDULED を外す。
func Schedule(root, id, when string) ([]string, error) {
	var ts string
	if when != "" {
		var err error
		if ts, err = scheduleStamp(when); err != nil {
			return nil, err
		}
	}
	return Update(root, func(f *Files) error {
		e, err := Find(f, id)
		if err != nil {
			return err
		}
		c, err := readTodoConfig(e.Root)
		if err != nil {
			return err
		}
		if c.logReschedule {
			return fmt.Errorf("#+STARTUP: logreschedule の記録は扱えない")
		}
		f.Write(e.Path, withSchedule(e.Src, e.Headline, ts))
		return nil
	})
}

// scheduleStamp は when を、括弧と repeater を除いたタイムスタンプの中身
// にする。
func scheduleStamp(when string) (string, error) {
	m := whenRe.FindStringSubmatch(when)
	if m == nil {
		return "", fmt.Errorf("日時 %q は YYYY-MM-DD[ H:MM[-H:MM]] で書く", when)
	}
	day, err := time.Parse("2006-01-02", m[1])
	if err != nil {
		return "", fmt.Errorf("日付 %q を読めない: %w", m[1], err)
	}
	ts := day.Format("2006-01-02 Mon")
	for i, c := range m[2:] {
		if c == "" {
			continue
		}
		t, err := time.Parse("15:04", c)
		if err != nil {
			return "", fmt.Errorf("時刻 %q を読めない: %w", c, err)
		}
		sep := " "
		if i > 0 {
			sep = "-"
		}
		ts += sep + t.Format("15:04")
	}
	return ts, nil
}

// withSchedule は org-add-planning-info と同じく、見出し h の計画
// 行から SCHEDULED を消し、ts が空でなければ CLOSED も消して行の先頭に
// 入れ直す。計画行が
// 空になれば行ごと消し、無ければ見出しの直後に作る。
func withSchedule(src string, h *org.Node, ts string) string {
	p := planning(h)
	if p == nil {
		if ts == "" {
			return src
		}
		return insertLines(src, lineEnd(src, h.Begin), "SCHEDULED: <"+ts+">\n")
	}
	start, end := lineBounds(src, p.Begin)
	line := src[start:end]
	if m := scheduledRe.FindStringIndex(line); m != nil {
		old := activeTimestampRe.FindStringSubmatch(line[m[0]:m[1]])
		if ts != "" && old != nil && repeaterRe.MatchString(old[3]) {
			ts += old[3]
		}
		cut := len(line)
		if k := planningKeywordRe.FindStringIndex(line[m[1]:]); k != nil {
			cut = m[1] + k[0]
		}
		line = strings.TrimRight(line[:m[0]]+line[cut:], " \t")
	}
	if ts != "" {
		line = withoutClosed(line)
	}
	body := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(body)]
	switch {
	case ts == "" && body == "":
		return src[:start-1] + src[end:]
	case ts == "":
		return src[:start] + line + src[end:]
	case body == "":
		return src[:start] + indent + "SCHEDULED: <" + ts + ">" + src[end:]
	}
	return src[:start] + indent + "SCHEDULED: <" + ts + "> " + body + src[end:]
}
