package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

// defaultListFiles は list にファイルを渡さないときに見るファイル。
var defaultListFiles = []string{"tasks.org", "inbox.org"}

var listCommand = command{
	usage: "list [FILE...]  :ID: を持つ見出しを「ID パス 祖先 TODO title タグ」のタブ区切りで出す。既定は tasks.org と inbox.org",
	run: func(c *context, usage string, args []string) error {
		flags := flag.NewFlagSet("list", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		if err := flags.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return fmt.Errorf("使い方: %s", usage)
			}
			return err
		}
		files := flags.Args()
		if len(files) == 0 {
			files = defaultListFiles
		}
		entries, err := orgedit.List(c.root, files)
		if err != nil {
			return err
		}
		return orgedit.WriteList(c.out, entries)
	},
}
