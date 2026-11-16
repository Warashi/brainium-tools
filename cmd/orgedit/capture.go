package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

var captureCommand = command{
	usage: "capture -title TITLE < BODY  inbox.org の先頭に entry を足す。1 行目に新しい ID、続けて書き換えたファイルを出す",
	run: func(c *context, usage string, args []string) error {
		flags := flag.NewFlagSet("capture", flag.ContinueOnError)
		title := flags.String("title", "", "見出し")
		if _, err := parse(flags, usage, args, 0); err != nil {
			return err
		}
		body, err := io.ReadAll(c.in)
		if err != nil {
			return err
		}
		id, paths, err := orgedit.Capture(c.root, c.env, *title, string(body))
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(c.out, id); err != nil {
			return err
		}
		return printChanged(c.out, paths, nil)
	},
}
