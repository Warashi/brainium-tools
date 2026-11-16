package main

import (
	"flag"
	"os"

	"github.com/Warashi/brainium-tools/internal/orgedit"
)

// archiveFileRootEnv は ARCHIVE_FILE に書くリポジトリルートを渡す環境変
// 数。本人の Emacs と別のパスで開くコンテナなどで、本人の環境のパスを書
// くために使う。
const archiveFileRootEnv = "ORGEDIT_ARCHIVE_FILE_ROOT"

var archiveCommand = command{
	usage: "archive ID  subtree を <ファイル>_archive の末尾へ移す(org-archive-subtree)。ARCHIVE_FILE のルートは $" + archiveFileRootEnv + " で替えられる",
	run: func(c *context, usage string, args []string) error {
		rest, err := parse(flag.NewFlagSet("archive", flag.ContinueOnError), usage, args, 1)
		if err != nil {
			return err
		}
		env := c.env
		env.ArchiveFileRoot = os.Getenv(archiveFileRootEnv)
		paths, err := orgedit.Archive(c.root, env, rest[0])
		return printChanged(c.out, paths, err)
	},
}
