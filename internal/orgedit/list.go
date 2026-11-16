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
