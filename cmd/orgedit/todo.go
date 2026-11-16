package main

import (
	"flag"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

var todoCommand = command{
	usage: "todo ID STATE | todo -none ID  TODO キーワードを替える。CLOSED と LOGBOOK はファイルの #+TODO・#+STARTUP に従う",
	run: func(c *context, usage string, args []string) error {
		flags := flag.NewFlagSet("todo", flag.ContinueOnError)
		none := flags.Bool("none", false, "キーワードを外す")
		n := 2
		if len(args) > 0 && (args[0] == "-none" || args[0] == "--none") {
			n = 1
		}
		rest, err := parse(flags, usage, args, n)
		if err != nil {
			return err
		}
		state := ""
		if !*none {
			state = rest[1]
		}
		paths, err := orgedit.SetTodo(c.root, c.env, rest[0], state)
		return printChanged(c.out, paths, err)
	},
}
