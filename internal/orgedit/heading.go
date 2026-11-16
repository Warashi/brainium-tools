package orgedit

import (
	"regexp"
	"strings"

	"github.com/Warashi/brainium-tools/internal/textwidth"
)

// tagsColumn は org-tags-column の既定値。負なので、タグの右端をこの桁
// の絶対値に揃える。
const tagsColumn = -77

var (
	starsRe = regexp.MustCompile(`^(\*+)[ \t]+`)
	// tagsRe は org-tag-line-re の、タグとその前の空白の部分。
	tagsRe = regexp.MustCompile(`(?:^|[ \t]+)(:[\p{L}\p{N}_@#%:]+:)[ \t]*$`)
)

// heading は見出しの 1 行を、星・TODO キーワード・残り・タグに分けたもの。
type heading struct {
	stars string
	todo  string
	// rest は TODO キーワードとタグを除いた残り。優先度や COMMENT を含む。
	rest string
	tags string
}

// parseHeading は見出しの行を分ける。keywords はそのファイルの TODO キ
// ーワード。
func parseHeading(line string, keywords []string) heading {
	line = strings.TrimSuffix(line, "\n")
	m := starsRe.FindStringSubmatchIndex(line)
	if m == nil {
		// 星だけの見出し("*" や "** ")
		return heading{stars: strings.TrimRight(line, " \t")}
	}
	h := heading{stars: line[m[2]:m[3]]}
	body := line[m[1]:]
	if t := tagsRe.FindStringSubmatchIndex(body); t != nil {
		h.tags = body[t[2]:t[3]]
		body = body[:t[0]]
	}
	body = strings.TrimRight(body, " \t")
	for _, k := range keywords {
		if body == k || strings.HasPrefix(body, k+" ") {
			h.todo = k
			body = strings.TrimLeft(body[len(k):], " \t")
			break
		}
	}
	h.rest = body
	return h
}

// String は見出しの行を改行なしで組み立てる。タグがあれば org-align-tags
// と同じ桁に揃える。
func (h heading) String() string {
	parts := []string{h.stars}
	if h.todo != "" {
		parts = append(parts, h.todo)
	}
	if h.rest != "" {
		parts = append(parts, h.rest)
	}
	s := strings.Join(parts, " ")
	if h.tags == "" {
		return s
	}
	return alignTags(s, h.tags)
}

// alignTags は org--align-tags-here と同じく、タグの右端を tagsColumn に
// 揃え、見出しとの間を空白 1 個以上空ける。
func alignTags(head, tags string) string {
	col := textwidth.String(head)
	to := max(-tagsColumn-textwidth.String(tags), col+1)
	return head + strings.Repeat(" ", to-col) + tags
}

// realignTags は見出しの行(改行なし)のタグの前の空白だけを揃え直す。
// タグが無ければそのまま返す。揃え直すのは、org がタグを揃え直す操作
// (TODO の変更・深さの変更)のときだけ。
func realignTags(line string) string {
	m := starsRe.FindStringIndex(line)
	if m == nil {
		return line
	}
	t := tagsRe.FindStringSubmatchIndex(line[m[1]:])
	if t == nil {
		return line
	}
	head := strings.TrimRight(line[:m[1]+t[2]], " \t")
	return alignTags(head, line[m[1]+t[2]:])
}

// replaceTodo は見出しの行(改行なし)の TODO キーワードを next に替え
// る。next が空ならキーワードを外す。
func replaceTodo(line string, keywords []string, next string) string {
	m := starsRe.FindStringIndex(line)
	if m == nil {
		// 星だけの見出し
		if next == "" {
			return line
		}
		return strings.TrimRight(line, " \t") + " " + next
	}
	body := line[m[1]:]
	h := parseHeading(line, keywords)
	switch {
	case h.todo == "" && next == "":
		return line
	case h.todo == "":
		if strings.TrimSpace(body) == "" {
			return line[:m[1]] + next
		}
		return line[:m[1]] + next + " " + body
	case next == "":
		return line[:m[1]] + strings.TrimLeft(body[len(h.todo):], " \t")
	}
	return line[:m[1]] + next + body[len(h.todo):]
}
