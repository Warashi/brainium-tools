package orgedit

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Warashi/brainium-tools/internal/org"
)

// Entry は :ID: で引いた見出し 1 つと、それを含むファイル。
type Entry struct {
	// Path はリポジトリルートからの相対パス。
	Path     string
	Src      string
	Root     *org.Node
	Headline *org.Node
}

// Find はリポジトリの org ファイルから、:ID: が id の見出しを探す。見
// 出しが無いか、2 つ以上あればエラーにする。
func Find(f *Files, id string) (*Entry, error) {
	if id == "" {
		return nil, fmt.Errorf("ID が空")
	}
	var found []*Entry
	err := filepath.WalkDir(f.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != f.root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".org") {
			return nil
		}
		// 全ファイルを解析すると遅いので、ID の文字列を含むものだけを読む
		data, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(data), id) {
			return err
		}
		rel, err := filepath.Rel(f.root, p)
		if err != nil {
			return err
		}
		src, _, err := f.Read(rel)
		if err != nil {
			return err
		}
		root := org.Parse(src)
		for _, h := range headlinesWithID(root, id) {
			found = append(found, &Entry{Path: rel, Src: src, Root: root, Headline: h})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	switch len(found) {
	case 0:
		return nil, fmt.Errorf("ID %s の見出しが無い", id)
	case 1:
		return found[0], nil
	}
	var where []string
	for _, e := range found {
		where = append(where, e.Path)
	}
	return nil, fmt.Errorf("ID %s の見出しが複数ある: %s", id, strings.Join(where, ", "))
}

// FindIn は src の中から :ID: が id の見出しを 1 つ探す。
func FindIn(root *org.Node, id string) (*org.Node, error) {
	hs := headlinesWithID(root, id)
	if len(hs) != 1 {
		return nil, fmt.Errorf("ID %s の見出しが %d 個ある", id, len(hs))
	}
	return hs[0], nil
}

func headlinesWithID(root *org.Node, id string) []*org.Node {
	var hs []*org.Node
	walkHeadlines(root, func(h *org.Node) {
		if v, ok := property(h, "ID"); ok && v == id {
			hs = append(hs, h)
		}
	})
	return hs
}

// walkHeadlines は n 配下の見出しを文書の順にたどる。
func walkHeadlines(n *org.Node, fn func(*org.Node)) {
	for _, c := range n.Children {
		if c.Type == org.TypeHeadline {
			fn(c)
			walkHeadlines(c, fn)
		}
	}
}

// property は見出しの PROPERTIES ドロワーにある key の値を返す。org と
// 同じく key の大文字・小文字は区別しない。
func property(h *org.Node, key string) (string, bool) {
	drawer := org.HeadlineProperties(h)
	if drawer == nil {
		return "", false
	}
	for _, p := range drawer.Children {
		if strings.EqualFold(p.Key, key) {
			return p.Value, true
		}
	}
	return "", false
}
