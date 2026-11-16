// Package orgedit は brainium の org ファイルを、Emacs の org の操作と同
// じ結果になるように書き換える。
package orgedit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxAttempts は、他の書き手と競合したときに操作をかけ直す回数の上限。
const maxAttempts = 5

// errConflict は、読んだ後に他の書き手がファイルを変えたことを表す。
var errConflict = errors.New("読んだ後にファイルが変わった")

// Files は 1 回の操作で読み書きするファイルを、読んだ時点の内容と組で
// 持つ。パスはリポジトリルートからの相対パス。
type Files struct {
	root string
	// read は読んだ時点の内容。ファイルが無かったものは nil。
	read  map[string]*string
	order []string
	next  map[string]string
}

// Read は rel の内容を返す。同じ操作の中で書いたものがあればそれを返す。
// ファイルが無ければ ok が false。
func (f *Files) Read(rel string) (content string, ok bool, err error) {
	if s, written := f.next[rel]; written {
		return s, true, nil
	}
	if s, seen := f.read[rel]; seen {
		if s == nil {
			return "", false, nil
		}
		return *s, true, nil
	}
	data, err := os.ReadFile(filepath.Join(f.root, rel))
	if errors.Is(err, os.ErrNotExist) {
		f.read[rel] = nil
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	s := string(data)
	f.read[rel] = &s
	return s, true, nil
}

// Write は rel の新しい内容を予約する。書き込みは操作の最後に、予約した
// 順に行う。Emacs が org ファイルを保存するときと同じく
// (require-final-newline)、変えたファイルの末尾に改行が無ければ足す。
func (f *Files) Write(rel, content string) {
	unchanged := f.read[rel] != nil && *f.read[rel] == content
	if !unchanged && content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if _, ok := f.next[rel]; !ok {
		f.order = append(f.order, rel)
	}
	f.next[rel] = content
}

// Update は fn で計算した書き換えを root 配下のファイルに書く。書く直前
// に、fn が読んだファイルがすべて読んだ時点のままかを確かめ、変わってい
// れば読み直して fn をかけ直す。書いたファイルの相対パスを、書いた順に
// 返す。
//
// 移動のように 2 つのファイルを書く操作は、移動先を先に Write する。途
// 中で失敗しても subtree は失われず、ID の重複として orglint が拾える。
func Update(root string, fn func(*Files) error) ([]string, error) {
	for range maxAttempts {
		f := &Files{root: root, read: map[string]*string{}, next: map[string]string{}}
		if err := fn(f); err != nil {
			return nil, err
		}
		err := f.commit()
		if errors.Is(err, errConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return f.changed(), nil
	}
	return nil, fmt.Errorf("%d 回かけ直しても他の書き手と競合した", maxAttempts)
}

// changed は内容が変わったファイルを書いた順に返す。
func (f *Files) changed() []string {
	var paths []string
	for _, rel := range f.order {
		if old := f.read[rel]; old == nil || *old != f.next[rel] {
			paths = append(paths, rel)
		}
	}
	return paths
}

func (f *Files) commit() error {
	for rel, old := range f.read {
		data, err := os.ReadFile(filepath.Join(f.root, rel))
		switch {
		case errors.Is(err, os.ErrNotExist):
			if old != nil {
				return errConflict
			}
		case err != nil:
			return err
		case old == nil || string(data) != *old:
			return errConflict
		}
	}
	for _, rel := range f.changed() {
		if err := writeFile(filepath.Join(f.root, rel), f.next[rel]); err != nil {
			return err
		}
	}
	return nil
}

// writeFile は同じディレクトリの一時ファイルに書いてから置き換える。読
// み手が書きかけの内容を見ないようにするため。
func writeFile(path, content string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
