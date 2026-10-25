package orglint

import (
	"errors"
	"io/fs"
	"path"
)

// flatDirs はサブディレクトリを置かない規約のディレクトリ。
var flatDirs = []string{"notes", "work"}

// checkFlatDirs は flatDirs の直下にあるディレクトリを報告する。org を
// 含まないディレクトリは org ファイルの配置の検査に現れないので、ファイ
// ルの一覧ではなくディレクトリそのものを見る。
func checkFlatDirs(fsys fs.FS) ([]Problem, error) {
	var problems []Problem
	for _, dir := range flatDirs {
		entries, err := fs.ReadDir(fsys, dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				problems = append(problems, Problem{
					Path:    path.Join(dir, e.Name()),
					Message: dir + "/ にサブディレクトリは作らない",
				})
			}
		}
	}
	return problems, nil
}
