// Command orgedit は brainium の org ファイルを、Emacs の org の操作と同
// じ結果になるように書き換える。agent が見出しやドロワーを手で書く代わ
// りに使う。
//
//	orgedit <subcommand> [flags] [args]
//
// 見出しは :ID: で指す。書き換えたファイルを、リポジトリルートからの相
// 対パスで 1 行ずつ出す。git には触らない。読んでから書くまでの間に他の
// 書き手がファイルを変えたら、読み直してかけ直す。
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/Warashi/brainium-tools/internal/orgedit"
	"github.com/Warashi/brainium-tools/internal/repo"
)

// command は 1 つのサブコマンド。
type command struct {
	usage string
	// run は usage を受け取り、引数の誤りを報告するときに使う。
	run func(c *context, usage string, args []string) error
}

// context はサブコマンドが共有する実行時の値。
type context struct {
	root string
	env  orgedit.Env
	in   io.Reader
	out  io.Writer
}

var commands = map[string]command{
	"set-property":    setPropertyCommand,
	"delete-property": deletePropertyCommand,
	"todo":            todoCommand,
	"list":            listCommand,
	"move":            moveCommand,
}

func main() {
	wd, err := os.Getwd()
	if err == nil {
		var root string
		root, err = repo.Root(wd)
		if err == nil {
			c := &context{root: root, env: orgedit.DefaultEnv(), in: os.Stdin, out: os.Stdout}
			err = execute(c, os.Args[1:])
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "orgedit:", err)
		os.Exit(2)
	}
}

func execute(c *context, args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "help" {
		usage(c.out)
		return nil
	}
	cmd, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("サブコマンド %q は無い。orgedit help で一覧を出す", args[0])
	}
	return cmd.run(c, cmd.usage, args[1:])
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "使い方: orgedit <subcommand> [flags] [args]")
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(w, "  %s\n", commands[name].usage)
	}
}

// parse は flags で args を解析し、残りの引数が n 個であることを確かめる。
func parse(flags *flag.FlagSet, usage string, args []string, n int) ([]string, error) {
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, fmt.Errorf("使い方: %s", usage)
		}
		return nil, err
	}
	if flags.NArg() != n {
		return nil, fmt.Errorf("引数の数が違う。使い方: %s", usage)
	}
	return flags.Args(), nil
}

// printChanged は書き換えたファイルを 1 行ずつ出す。
func printChanged(w io.Writer, paths []string, err error) error {
	if err != nil {
		return err
	}
	if len(paths) > 0 {
		_, err = fmt.Fprintln(w, strings.Join(paths, "\n"))
	}
	return err
}
