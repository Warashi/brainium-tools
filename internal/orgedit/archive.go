package orgedit

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Warashi/brainium-tools/internal/org"
)

// Archive は :ID: が id の見出しの subtree を、org-archive-subtree の既
// 定(org-archive-location が "%s_archive::")と同じく <ファイル>_archive
// の末尾に深さ 1 で移し、ARCHIVE_TIME・ARCHIVE_FILE・ARCHIVE_OLPATH・
// ARCHIVE_CATEGORY・ARCHIVE_TODO・ARCHIVE_ITAGS を足す。archive のファ
// イルを先に書く。
func Archive(root string, env Env, id string) ([]string, error) {
	return Update(root, func(f *Files) error {
		e, err := Find(f, id)
		if err != nil {
			return err
		}
		if keywordValue(e.Root, "ARCHIVE") != "" {
			return fmt.Errorf("%s の #+ARCHIVE は扱えない", e.Path)
		}
		path := e.Path + "_archive"
		ctx, err := archiveContext(root, env, e)
		if err != nil {
			return err
		}
		h := e.Headline
		text := shiftLevels(e.Src[h.Begin:h.End], 1-h.Level)
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		dst, ok, err := f.Read(path)
		if err != nil {
			return err
		}
		if !ok {
			dst = "#    -*- mode: org -*-\n\n" + fmt.Sprintf("\nArchived entries from file %s\n\n", filepath.Join(root, e.Path))
		}
		dst += "\n"
		begin := len(dst)
		dst += text
		for _, p := range ctx {
			if p[1] == "" {
				continue
			}
			var archived *org.Node
			for _, c := range org.Parse(dst).Children {
				if c.Type == org.TypeHeadline && c.Begin == begin {
					archived = c
				}
			}
			if archived == nil {
				return fmt.Errorf("archive した見出しが %s に見つからない", path)
			}
			if dst, err = PutProperty(dst, archived, "ARCHIVE_"+p[0], p[1]); err != nil {
				return err
			}
		}
		f.Write(path, dst)
		f.Write(e.Path, e.Src[:h.Begin]+e.Src[h.End:])
		return nil
	})
}

// archiveContext は org-archive-save-context-info の既定値
// (time file olpath category todo itags)の順に、名前と値の組を返す。
func archiveContext(root string, env Env, e *Entry) ([][2]string, error) {
	c, err := readTodoConfig(e.Root)
	if err != nil {
		return nil, err
	}
	ancestors := ancestorsOf(e.Root, e.Headline)
	var olpath, itags []string
	seen := map[string]bool{}
	addTags := func(tags []string) {
		for _, t := range tags {
			if !seen[t] {
				seen[t] = true
				itags = append(itags, t)
			}
		}
	}
	addTags(org.FileTags(e.Root))
	for _, a := range ancestors {
		start, end := lineBounds(e.Src, a.Begin)
		hd := parseHeading(e.Src[start:end], c.keywords)
		olpath = append(olpath, outlineTitle(hd.rest))
		addTags(splitTags(hd.tags))
	}
	start, end := lineBounds(e.Src, e.Headline.Begin)
	todo := parseHeading(e.Src[start:end], c.keywords).todo
	return [][2]string{
		{"TIME", env.Now.Format("2006-01-02 Mon 15:04")},
		{"FILE", archiveFileName(root, env, e.Path)},
		{"OLPATH", strings.Join(olpath, "/")},
		{"CATEGORY", category(e, ancestors)},
		{"TODO", todo},
		{"ITAGS", strings.Join(itags, " ")},
	}, nil
}

// archiveFileName は ARCHIVE_FILE の値を返す。Emacs は開いたファイルの
// パスを、home を ~ と略して書く(abbreviate-file-name)。
// Env.ArchiveFileRoot があれば、リポジトリルートの代わりにそれを使う。
func archiveFileName(root string, env Env, rel string) string {
	if env.ArchiveFileRoot != "" {
		return strings.TrimSuffix(env.ArchiveFileRoot, "/") + "/" + filepath.ToSlash(rel)
	}
	p := filepath.Join(root, rel)
	if env.Home != "" {
		if r, err := filepath.Rel(env.Home, p); err == nil && !strings.HasPrefix(r, "..") {
			return "~/" + filepath.ToSlash(r)
		}
	}
	return p
}

// category は org-get-category と同じく、CATEGORY を見出し・祖先・ファ
// イルの順に引き継いで決める。どこにも無ければファイル名から拡張子を除
// いたもの。
func category(e *Entry, ancestors []*org.Node) string {
	if v, ok := property(e.Headline, "CATEGORY"); ok {
		return v
	}
	for i := len(ancestors) - 1; i >= 0; i-- {
		if v, ok := property(ancestors[i], "CATEGORY"); ok {
			return v
		}
	}
	if d := org.FileProperties(e.Root); d != nil {
		for _, p := range d.Children {
			if strings.EqualFold(p.Key, "CATEGORY") {
				return p.Value
			}
		}
	}
	if v := keywordValue(e.Root, "CATEGORY"); v != "" {
		return v
	}
	base := filepath.Base(e.Path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// ancestorsOf は h の祖先の見出しを外側から返す。
func ancestorsOf(root, h *org.Node) []*org.Node {
	var path []*org.Node
	var walk func(n *org.Node) bool
	walk = func(n *org.Node) bool {
		for _, c := range n.Children {
			if c.Type != org.TypeHeadline {
				continue
			}
			if c == h {
				return true
			}
			if c.Begin < h.Begin && h.Begin < c.End {
				path = append(path, c)
				return walk(c)
			}
		}
		return false
	}
	walk(root)
	return path
}

var (
	priorityRe    = regexp.MustCompile(`^\[#[^]]+\] ?`)
	bracketLinkRe = regexp.MustCompile(`\[\[([^]]*)\](?:\[([^]]*)\])?\]`)
)

// outlineTitle は org-get-outline-path と同じく、見出しの残りから優先度
// と COMMENT を除き、リンクを説明(無ければ行き先)に置き換える。
func outlineTitle(rest string) string {
	rest = priorityRe.ReplaceAllString(rest, "")
	rest = strings.TrimPrefix(rest, "COMMENT ")
	return bracketLinkRe.ReplaceAllStringFunc(rest, func(s string) string {
		m := bracketLinkRe.FindStringSubmatch(s)
		if m[2] != "" {
			return m[2]
		}
		return m[1]
	})
}

func splitTags(tags string) []string {
	var out []string
	for _, t := range strings.Split(tags, ":") {
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// keywordValue は最後の #+KEY の値を返す。無ければ空文字列。
func keywordValue(root *org.Node, key string) string {
	v := ""
	walkKeywords(root, func(k *org.Node) {
		if k.Key == key {
			v = k.Value
		}
	})
	return v
}
