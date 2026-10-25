// Command orglint は brainium リポジトリの org ノートを検査する。
//
//	orglint [-fix | -fix-only] [path...]
//
// 違反があれば 1 行 1 件で出力し、終了コード 1 を返す。-fix を付けると
// 本文の段落を整形(折り返しと、和文に続くコード表記の空白)してから検
// 査する。-fix-only は整形するだけで検査せず、直せない違反が残っていて
// も正常終了する。treefmt のよう
// に formatter の非ゼロ終了を失敗として扱う入口から呼ぶためのもの。
// path にはファイルとディレクトリのどちらも渡せる。省略するとカレント
// ディレクトリを渡したのと同じで、リポジトリ全体が対象になる。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Warashi/brainium-tools/internal/orglint"
	"github.com/Warashi/brainium-tools/internal/repo"
)

func main() {
	fix := flag.Bool("fix", false, "本文の段落を整形して書き戻してから検査する")
	fixOnly := flag.Bool("fix-only", false, "本文の段落を整形して書き戻すだけで検査しない")
	flag.Parse()

	if err := run(mode{fix: *fix || *fixOnly, check: !*fixOnly}, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "orglint:", err)
		os.Exit(2)
	}
}

// mode は実行する工程を表す。
type mode struct {
	fix   bool // 整形して書き戻す
	check bool // 検査して違反を報告する
}

func run(m mode, args []string) error {
	root, targets, err := resolve(args)
	if err != nil {
		return err
	}

	fsys := os.DirFS(root)
	paths, err := orglint.OrgFiles(fsys)
	if err != nil {
		return err
	}
	if missing := targets.Unmatched(paths); len(missing) > 0 {
		return fmt.Errorf("検査対象の org ファイルが見つからない: %s", strings.Join(missing, " "))
	}

	if m.fix {
		if err := fixAll(root, targets.Filter(paths)); err != nil {
			return err
		}
	}
	if !m.check {
		return nil
	}

	problems, err := orglint.Run(fsys, targets...)
	if err != nil {
		return err
	}
	for _, p := range problems {
		fmt.Println(p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
	return nil
}

// resolve は引数からリポジトリルートと、そこからの相対パスで表した検査
// 対象を求める。ルートは引数の位置から上に .git を辿って決める。
// .git が無い場所でもディレクトリを 1 つ渡せばそこをルートとして扱う。
// nix のビルドサンドボックスには .git が無く、flake の check が store
// パスを渡して呼ぶため。
func resolve(args []string) (string, orglint.Targets, error) {
	if len(args) == 0 {
		args = []string{"."}
	}

	abs := make([]string, len(args))
	for i, a := range args {
		p, err := filepath.Abs(a)
		if err != nil {
			return "", nil, err
		}
		if _, err := os.Stat(p); err != nil {
			return "", nil, err
		}
		abs[i] = p
	}

	root, err := repo.Root(abs[0])
	if err != nil {
		if len(args) == 1 && repo.IsDir(abs[0]) {
			return abs[0], nil, nil
		}
		return "", nil, err
	}

	var targets orglint.Targets
	for _, p := range abs {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return "", nil, err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".." || strings.HasPrefix(rel, "../") {
			return "", nil, fmt.Errorf("%s はリポジトリ %s の外にある", p, root)
		}
		targets = append(targets, rel)
	}
	return root, targets, nil
}

// fixAll は paths の org ファイルを整形し、変わったものだけ書き戻す。
func fixAll(root string, paths []string) error {
	for _, p := range paths {
		name := filepath.Join(root, filepath.FromSlash(p))
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		formatted := orglint.Format(data)
		if string(formatted) == string(data) {
			continue
		}
		info, err := os.Stat(name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(name, formatted, info.Mode().Perm()); err != nil {
			return err
		}
		fmt.Println(p + ": 整形した")
	}
	return nil
}
