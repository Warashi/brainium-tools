// Command notesearch は brainium の恒久ノートを BM25 で全文検索する。
//
//	notesearch <query>
//
// 原文の抜粋・行番号・成熟度と、検索対象・打ち切りの有無を返す。
// -format paths で従来のパスだけの出力を使える。
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Warashi/brainium-tools/internal/notesearch"
	"github.com/Warashi/brainium-tools/internal/notesearch/evidence"
	"github.com/Warashi/brainium-tools/internal/notesearch/presentation"
	"github.com/Warashi/brainium-tools/internal/repo"
)

// notesDir はノートを置くディレクトリ。tasks.org や journal/ は対象にしない。
const notesDir = "notes"

const (
	maxResults      = 10
	defaultMaxBytes = 24000
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "notesearch:", err)
		os.Exit(2)
	}
}

func run(args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return execute(args, cwd, os.Stdout)
}

func execute(args []string, cwd string, out io.Writer) error {
	flags := flag.NewFlagSet("notesearch", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	format := flags.String("format", "text", "出力形式: text / json / paths")
	limit := flags.Int("limit", maxResults, "上位ノートの最大件数 (正の整数)")
	maxBytes := flags.Int("max-bytes", defaultMaxBytes, "出力全体の最大 UTF-8 バイト数 (正の整数)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			var help strings.Builder
			fmt.Fprintln(&help, "使い方: notesearch [flags] <query>")
			fmt.Fprintln(&help, "フラグはクエリの前に指定。- で始まるクエリは -- の後に指定。")
			flags.SetOutput(&help)
			flags.PrintDefaults()
			_, err = io.WriteString(out, help.String())
			return err
		}
		return err
	}
	query := strings.Join(flags.Args(), " ")
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("使い方: notesearch [flags] <query>")
	}
	if !utf8.ValidString(query) {
		return fmt.Errorf("query is not valid UTF-8")
	}
	if *limit <= 0 || *maxBytes <= 0 {
		return fmt.Errorf("-limit と -max-bytes は正の整数が必要")
	}
	outputFormat := presentation.Format(*format)
	switch outputFormat {
	case presentation.FormatText, presentation.FormatJSON, presentation.FormatPaths:
	default:
		return fmt.Errorf("unknown output format %q", *format)
	}
	root, err := repo.Root(cwd)
	if err != nil {
		return err
	}
	report, err := search(os.DirFS(filepath.Join(root, notesDir)), query, *limit, outputFormat != presentation.FormatPaths)
	if err != nil {
		return fmt.Errorf("%s: %w", notesDir, err)
	}
	data, err := presentation.Render(report, outputFormat, *maxBytes)
	if err != nil {
		return err
	}
	n, err := out.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func search(fsys fs.FS, query string, limit int, withEvidence bool) (presentation.Report, error) {
	index, err := notesearch.Build(fsys, nil)
	if err != nil {
		return presentation.Report{}, err
	}
	results := index.Search(query)
	report := presentation.Report{
		Query: query, Scope: notesDir + "/", TotalMatches: len(results),
		Truncated: len(results) > limit,
		Results:   make([]presentation.Item, 0, min(len(results), limit)),
	}
	if len(results) > limit {
		results = results[:limit]
	}
	for _, r := range results {
		item := presentation.Item{Path: path.Join(notesDir, r.Path)}
		if withEvidence {
			source, ok := index.Source(r.Path)
			if !ok {
				return presentation.Report{}, fmt.Errorf("indexed source missing for %q", r.Path)
			}
			if !utf8.ValidString(source) || !utf8.ValidString(item.Path) {
				return presentation.Report{}, fmt.Errorf("source or path is not valid UTF-8: %q", r.Path)
			}
			doc := evidence.Extract(source, query)
			item.ID, item.Title, item.Status = doc.Metadata.ID, doc.Metadata.Title, doc.Metadata.Status
			item.Verified, item.StaleAfter = doc.Metadata.Verified, doc.Metadata.StaleAfter
			for _, snippet := range doc.Snippets {
				item.Snippets = append(item.Snippets, presentation.Snippet{
					StartLine: snippet.StartLine, EndLine: snippet.EndLine,
					Headings: snippet.Headings, Text: snippet.Text, Kind: snippet.Kind,
				})
			}
		}
		report.Results = append(report.Results, item)
	}
	return report, nil
}
