package main

import (
	"flag"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

var journalCommand = command{
	usage: "journal ID  capture を :CAPTURED: の日付のデイリーノートへ、:CAPTURED: の順を保って移す。無ければ作る",
	run: func(c *context, usage string, args []string) error {
		rest, err := parse(flag.NewFlagSet("journal", flag.ContinueOnError), usage, args, 1)
		if err != nil {
			return err
		}
		paths, err := orgedit.Journal(c.root, c.env, rest[0])
		return printChanged(c.out, paths, err)
	},
}
