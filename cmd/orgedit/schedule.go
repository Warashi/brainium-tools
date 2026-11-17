package main

import (
	"flag"
	"strings"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

var scheduleCommand = command{
	usage: "schedule ID YYYY-MM-DD [H:MM[-H:MM]] | schedule -none ID  SCHEDULED を替える(org-schedule)。repeater は保つ",
	run: func(c *context, usage string, args []string) error {
		args, none := takeFlag(args, "none")
		n := 2
		switch {
		case none:
			n = 1
		case len(args) == 3:
			n = 3
		}
		rest, err := parse(flag.NewFlagSet("schedule", flag.ContinueOnError), usage, args, n)
		if err != nil {
			return err
		}
		paths, err := orgedit.Schedule(c.root, rest[0], strings.Join(rest[1:], " "))
		return printChanged(c.out, paths, err)
	},
}
