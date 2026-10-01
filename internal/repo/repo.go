// Package repo は brainium リポジトリの位置を解決する。
package repo

import (
	"fmt"
	"os"
	"path/filepath"
)

// Root は p から親を辿り、.git を持つディレクトリを返す。
// これにより CLI はリポジトリ内のどこからでも呼べる。
func Root(p string) (string, error) {
	dir := p
	if !IsDir(dir) {
		dir = filepath.Dir(dir)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%s の親に .git が見つからない", p)
		}
		dir = parent
	}
}

// IsDir は p がディレクトリかどうかを返す。
func IsDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
