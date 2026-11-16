package main

import (
	"flag"
	"fmt"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

var setPropertyCommand = command{
	usage: "set-property [-now] ID KEY [VALUE]  見出しのプロパティを VALUE にする。-now なら VALUE の代わりに今の時刻",
	run: func(c *context, usage string, args []string) error {
		flags := flag.NewFlagSet("set-property", flag.ContinueOnError)
		now := flags.Bool("now", false, "VALUE の代わりに今の時刻 [YYYY-MM-DD Day HH:MM] を入れる")
		n := 3
		if len(args) > 0 && (args[0] == "-now" || args[0] == "--now") {
			n = 2
		}
		rest, err := parse(flags, usage, args, n)
		if err != nil {
			return err
		}
		value := c.env.Now.Format("[2006-01-02 Mon 15:04]")
		if !*now {
			value = rest[2]
		}
		if value == "" {
			return fmt.Errorf("VALUE が空。消すときは delete-property を使う")
		}
		paths, err := orgedit.SetProperty(c.root, rest[0], rest[1], value)
		return printChanged(c.out, paths, err)
	},
}

var deletePropertyCommand = command{
	usage: "delete-property ID KEY  見出しのプロパティを消す。空になったドロワーも消す",
	run: func(c *context, usage string, args []string) error {
		rest, err := parse(flag.NewFlagSet("delete-property", flag.ContinueOnError), usage, args, 2)
		if err != nil {
			return err
		}
		paths, err := orgedit.SetProperty(c.root, rest[0], rest[1], "")
		return printChanged(c.out, paths, err)
	},
}
