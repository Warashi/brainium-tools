package orgedit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Warashi/brainium-tools/internal/org"
)

// ListEntry は一覧の 1 行。
type ListEntry struct {
	ID   string
	Path string
	// Outline は祖先の見出しの title を / でつないだもの。
	Outline string
	Todo    string
	Title   string
	Tags    string
	// Level は見出しの深さ。
	Level int
	// Priority は優先度 cookie の値。cookie が無ければ空。
	Priority string
	// Props は PROPERTIES ドロワーの値。key は書かれたとおり。
	Props map[string]string
	// Text は見出しの行から子孫を含む subtree の終わりまで。
	Text string
}

// List は files(リポジトリルートからの相対パス)の、:ID: を持つ見出し
// を文書の順に返す。
func List(root string, files []string) ([]ListEntry, error) {
	var entries []ListEntry
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return nil, err
		}
		src := string(data)
		doc := org.Parse(src)
		c, err := readTodoConfig(doc)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		var walk func(n *org.Node, outline []string)
		walk = func(n *org.Node, outline []string) {
			for _, h := range n.Children {
				if h.Type != org.TypeHeadline {
					continue
				}
				start, end := lineBounds(src, h.Begin)
				hd := parseHeading(src[start:end], c.keywords)
				if id, ok := property(h, "ID"); ok {
					entries = append(entries, ListEntry{
						ID: id, Path: rel, Outline: strings.Join(outline, "/"),
						Todo: hd.todo, Title: hd.rest, Tags: hd.tags,
						Level: h.Level, Priority: priorityOf(hd.rest),
						Props: properties(h), Text: src[h.Begin:h.End],
					})
				}
				walk(h, append(outline[:len(outline):len(outline)], hd.rest))
			}
		}
		walk(doc, nil)
	}
	return entries, nil
}

// WriteList は一覧を ID・パス・祖先・TODO・title・タグのタブ区切りで書
// く。
func WriteList(w io.Writer, entries []ListEntry) error {
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", e.ID, e.Path, e.Outline, e.Todo, e.Title, e.Tags); err != nil {
			return err
		}
	}
	return nil
}

// priorityOf は見出しの残りの先頭にある優先度 cookie の値を返す。
func priorityOf(rest string) string {
	m := priorityRe.FindString(rest)
	if m == "" {
		return ""
	}
	return strings.TrimSpace(m)[2 : len(strings.TrimSpace(m))-1]
}

// properties は見出しの PROPERTIES ドロワーの key と値を返す。
func properties(h *org.Node) map[string]string {
	props := map[string]string{}
	if drawer := org.HeadlineProperties(h); drawer != nil {
		for _, p := range drawer.Children {
			props[p.Key] = p.Value
		}
	}
	return props
}
