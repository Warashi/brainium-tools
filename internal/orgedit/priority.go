package orgedit

import (
	"fmt"
	"regexp"
	"strings"
)

// priorityValueRe は org-priority-highest から org-priority-lowest の既
// 定の範囲。
var priorityValueRe = regexp.MustCompile(`^[ABC]$`)

// SetPriority は :ID: が id の見出しの優先度を priority(A・B・C)にする。
// org-priority と同じく、cookie を TODO キーワードの直後に置く。priority
// が空なら cookie を外す。
func SetPriority(root, id, priority string) ([]string, error) {
	if priority != "" && !priorityValueRe.MatchString(priority) {
		return nil, fmt.Errorf("優先度 %q は A・B・C のどれかにする", priority)
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
		start, end := lineBounds(e.Src, e.Headline.Begin)
		f.Write(e.Path, e.Src[:start]+withPriority(e.Src[start:end], c.keywords, priority)+e.Src[end:])
		return nil
	})
}

// withPriority は見出しの行(改行なし)の優先度 cookie を priority に替え
// る。org-priority と同じくタグを揃え直す。
func withPriority(line string, keywords []string, priority string) string {
	h := parseHeading(line, keywords)
	h.rest = priorityRe.ReplaceAllString(h.rest, "")
	if priority != "" {
		h.rest = strings.TrimRight("[#"+priority+"] "+h.rest, " ")
	}
	return h.String()
}
