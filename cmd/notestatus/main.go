// Command notestatus は精錬待ちの恒久ノートを列挙する。
//
//	notestatus [-today YYYY-MM-DD]
//
// notes/ 直下のノートのうち、:STATUS: が draft のものと :STALE_AFTER: の
// 日付を過ぎたものを、パス・理由・タイトルのタブ区切りで 1 行ずつ出力す
// る。deprecated は出さない。-today を省略すると実行日で判定する。
// リポジトリ内のどのディレクトリから呼んでもよい。
package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Warashi/brainium-tools/internal/maturity"
	"github.com/Warashi/brainium-tools/internal/repo"
)

const notesDir = "notes"

var titleRe = regexp.MustCompile(`(?im)^#\+title:\s*(.*?)\s*$`)

func main() {
	todayFlag := flag.String("today", "", "判定に使う日付 (YYYY-MM-DD)。省略時は実行日")
	flag.Parse()

	if err := run(*todayFlag); err != nil {
		fmt.Fprintln(os.Stderr, "notestatus:", err)
		os.Exit(2)
	}
}

func run(todayFlag string) error {
	today := time.Now()
	if todayFlag != "" {
		t, err := time.Parse("2006-01-02", todayFlag)
		if err != nil {
			return fmt.Errorf("-today: %w", err)
		}
		today = t
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := repo.Root(wd)
	if err != nil {
		return err
	}
	return list(os.Stdout, os.DirFS(root), today)
}

// list は fsys をリポジトリルートとして精錬待ちのノートを w に書く。
// notes/ はフラット配置なので直下だけを見る。
func list(w io.Writer, fsys fs.FS, today time.Time) error {
	entries, err := fs.ReadDir(fsys, notesDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".org" {
			continue
		}
		p := path.Join(notesDir, e.Name())
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		content := string(data)
		reasons := maturity.Pending(maturity.Parse(content), today)
		if len(reasons) == 0 {
			continue
		}
		var title string
		if m := titleRe.FindStringSubmatch(content); m != nil {
			title = m[1]
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", p, strings.Join(reasons, ", "), title); err != nil {
			return err
		}
	}
	return nil
}
