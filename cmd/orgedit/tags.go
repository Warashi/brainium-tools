package main

import (
	"flag"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

var setTagsCommand = command{
	usage: "set-tags ID [TAG...]  見出しのタグを TAG にして揃える(org-set-tags)。TAG が無ければ外す",
	run: func(c *context, usage string, args []string) error {
		if len(args) == 0 {
			_, err := parse(flag.NewFlagSet("set-tags", flag.ContinueOnError), usage, args, 1)
			return err
		}
		paths, err := orgedit.SetTags(c.root, args[0], args[1:])
		return printChanged(c.out, paths, err)
	},
}
