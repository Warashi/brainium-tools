package main

import (
	"flag"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

var moveCommand = command{
	usage: "move ID (-parent ID | -before ID | -file FILE) [-first]  subtree を親の子の末尾・兄弟の手前・ファイルの深さ 1 の末尾へ移す(org-refile)。-first なら末尾ではなく先頭",
	run: func(c *context, usage string, args []string) error {
		flags := flag.NewFlagSet("move", flag.ContinueOnError)
		var dst orgedit.Destination
		flags.StringVar(&dst.ParentID, "parent", "", "移動先の親の見出しの :ID:")
		flags.StringVar(&dst.BeforeID, "before", "", "移動先の兄弟の見出しの :ID:。その手前に移す")
		flags.StringVar(&dst.File, "file", "", "移動先のファイル。深さ 1 で移す")
		flags.BoolVar(&dst.First, "first", false, "末尾ではなく先頭に移す")
		// ID を flag より前に書けるよう、先頭の ID を外してから解析する
		if len(args) == 0 {
			_, err := parse(flags, usage, args, 1)
			return err
		}
		id := args[0]
		if _, err := parse(flags, usage, args[1:], 0); err != nil {
			return err
		}
		paths, err := orgedit.Move(c.root, id, dst)
		return printChanged(c.out, paths, err)
	},
}
