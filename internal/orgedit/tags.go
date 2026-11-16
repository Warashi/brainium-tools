package orgedit

import (
	"fmt"
	"regexp"
	"strings"
)

var tagNameRe = regexp.MustCompile(`^[\p{L}\p{N}_@#%]+$`)

// SetTags は :ID: が id の見出しのタグを tags にする。org-set-tags と同
// じく、タグを org-tags-column に揃える。tags が空ならタグを外す。
func SetTags(root, id string, tags []string) ([]string, error) {
	for _, t := range tags {
		if !tagNameRe.MatchString(t) {
			return nil, fmt.Errorf("タグ %q に使えない文字がある", t)
		}
	}
	return Update(root, func(f *Files) error {
		e, err := Find(f, id)
		if err != nil {
			return err
		}
		start, end := lineBounds(e.Src, e.Headline.Begin)
		f.Write(e.Path, e.Src[:start]+withTags(e.Src[start:end], tags)+e.Src[end:])
		return nil
	})
}

// withTags は見出しの行(改行なし)のタグを tags に替える。
func withTags(line string, tags []string) string {
	head := line
	if m := starsRe.FindStringIndex(line); m != nil {
		if t := tagsRe.FindStringSubmatchIndex(line[m[1]:]); t != nil {
			head = line[:m[1]+t[0]]
		}
	}
	head = strings.TrimRight(head, " \t")
	if len(tags) == 0 {
		return head
	}
	return alignTags(head, ":"+strings.Join(tags, ":")+":")
}
