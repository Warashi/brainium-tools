// Package anchor はリポジトリ全体の org から id リンクの説明文を集め、
// リンク先の notes/ ファイルごとにまとめる。
//
// リンク文は、別の文脈で書かれたそのノートの短い言い換えなので、本文と
// 語彙がずれているほど検索の橋渡しに効く。web 検索がアンカーテキストを
// 索引に入れたのと同じ狙い。
package anchor

import (
	"fmt"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// notesDir は索引の対象。リンク元は全ディレクトリから集めるが、引き当て
// 先はここに限る。
const notesDir = "notes"

// fileID はファイル先頭の PROPERTIES ドロワーの :ID: を拾う。notes/ は
// 1 ファイル 1 ノードなので、最初に現れた :ID: がファイルレベルのもの。
var fileID = regexp.MustCompile(`(?m)^:ID:\s+([0-9A-Fa-f-]+)\s*$`)

// idLink は説明文つきの id リンク。org は fill で説明文の途中に改行を
// 入れるので、複数行にまたがるものも拾う。
var idLink = regexp.MustCompile(`(?s)\[\[id:([0-9A-Fa-f-]+)\]\[(.*?)\]\]`)

// Collect は fsys 内の org を走査し、notes/ のファイル名をキーにリンク文
// を返す。引き当てられない UUID と、説明文の無いリンクは落とす。
func Collect(fsys fs.FS) (map[string][]string, error) {
	files, err := orgFiles(fsys)
	if err != nil {
		return nil, err
	}

	target := make(map[string]string) // UUID(小文字) -> notes/ のファイル名
	for name, content := range files {
		if path.Dir(name) != notesDir {
			continue
		}
		if m := fileID.FindStringSubmatch(content); m != nil {
			target[strings.ToLower(m[1])] = path.Base(name)
		}
	}

	out := make(map[string][]string)
	// 走査順を固定して、同じ corpus なら同じ順でリンク文が並ぶようにする
	for _, name := range slices.Sorted(maps.Keys(files)) {
		for _, m := range idLink.FindAllStringSubmatch(files[name], -1) {
			p, ok := target[strings.ToLower(m[1])]
			if !ok {
				continue
			}
			if text := unfill(m[2]); text != "" {
				out[p] = append(out[p], text)
			}
		}
	}
	return out, nil
}

func orgFiles(fsys fs.FS) (map[string]string, error) {
	out := make(map[string]string)
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// .git や .agent-shell を舐めない。org は隠しディレクトリに置かない
			if name != "." && strings.HasPrefix(path.Base(name), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if path.Ext(name) != ".org" {
			return nil
		}
		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		out[name] = string(content)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// unfill は fill で入った改行を畳む。
//
// 一律に空白へ置き換えないのは、日本語の途中で折り返された行がそこで語
// として切れてしまうため。Emacs の表示と同じく、改行をまたぐ両側が非
// ASCII なら何も挟まない。
func unfill(s string) string {
	var b strings.Builder
	for i, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i > 0 && b.Len() > 0 && needsSpace(lastRune(b.String()), firstRune(line)) {
			b.WriteByte(' ')
		}
		b.WriteString(line)
	}
	return b.String()
}

func needsSpace(a, b rune) bool {
	return a < unicode.MaxASCII || b < unicode.MaxASCII
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func lastRune(s string) rune {
	var last rune
	for _, r := range s {
		last = r
	}
	return last
}
