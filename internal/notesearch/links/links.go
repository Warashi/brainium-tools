// Package links はノート間の id リンクを集め、ノートごとにリンク先とリ
// ンク元を返す。
//
// 検索語が本文と重ならないノートでも、ヒットしたノートからリンクで 1 手
// で届くことが多い。順位を変えずに辿る先を見せるために使う。
package links

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Warashi/brainium-tools/internal/notesearch/evidence"
)

// idLink は説明文の有無を問わない id リンク。
var idLink = regexp.MustCompile(`\[\[id:([0-9A-Fa-f-]+)\](?:\[|\])`)

// Note はリンクの片端。Path は fsys 内のファイル名。
type Note struct {
	Path  string
	Title string
}

// Graph はノート間のリンク。同じ組のリンクは 1 本に数え、自分自身への
// リンクと fsys の外を指すリンクは持たない。
type Graph struct {
	to, from map[string][]Note
}

// To は path のノートがリンクしているノートをパス順に返す。
func (g Graph) To(path string) []Note { return g.to[path] }

// From は path のノートへリンクしているノートをパス順に返す。
func (g Graph) From(path string) []Note { return g.from[path] }

// Collect は fsys 直下の org ファイルからリンクを集める。notes/ はフラッ
// ト配置の規約なので再帰しない。
func Collect(fsys fs.FS) (Graph, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return Graph{}, err
	}
	sources := make(map[string]string)
	byID := make(map[string]Note)
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".org" {
			continue
		}
		content, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return Graph{}, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		sources[e.Name()] = string(content)
		meta := evidence.ParseMetadata(string(content))
		if meta.ID != "" {
			byID[strings.ToLower(meta.ID)] = Note{Path: e.Name(), Title: meta.Title}
		}
	}
	titles := make(map[string]string, len(byID))
	for _, n := range byID {
		titles[n.Path] = n.Title
	}

	g := Graph{to: make(map[string][]Note), from: make(map[string][]Note)}
	for name, content := range sources {
		seen := make(map[string]bool)
		for _, m := range idLink.FindAllStringSubmatch(content, -1) {
			target, ok := byID[strings.ToLower(m[1])]
			if !ok || target.Path == name || seen[target.Path] {
				continue
			}
			seen[target.Path] = true
			g.to[name] = append(g.to[name], target)
			g.from[target.Path] = append(g.from[target.Path], Note{Path: name, Title: titles[name]})
		}
	}
	byPath := func(a, b Note) int { return cmp.Compare(a.Path, b.Path) }
	for _, m := range []map[string][]Note{g.to, g.from} {
		for _, notes := range m {
			slices.SortFunc(notes, byPath)
		}
	}
	return g, nil
}
