package main

import (
	"flag"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

var priorityCommand = command{
	usage: "priority ID A|B|C | priority -none ID  見出しの優先度 cookie を替える(org-priority)",
	run: func(c *context, usage string, args []string) error {
		flags := flag.NewFlagSet("priority", flag.ContinueOnError)
		none := flags.Bool("none", false, "cookie を外す")
		args, withNone := takeFlag(args, "none")
		n := 2
		if withNone {
			n = 1
		}
		rest, err := parse(flags, usage, args, n)
		*none = withNone
		if err != nil {
			return err
		}
		priority := ""
		if !*none {
			priority = rest[1]
		}
		paths, err := orgedit.SetPriority(c.root, rest[0], priority)
		return printChanged(c.out, paths, err)
	},
}
