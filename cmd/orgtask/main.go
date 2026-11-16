// Command orgtask は brainium の first-step が tasks.org に対して行う決
// 定的な手順を引き受ける。次に一手を切り出す task を選び、:CURRENT_STEP:
// と :WORKING: と :REPO: を書き、完了の一手を capture し、触ったファイル
// だけをコミットする。一手の文を作るのは呼び出す agent。
//
//	orgtask <subcommand> [args]
//
// 書く前に、書くファイルに他の書き手のまだコミットされていない変更があ
// れば、何も書かずに終了コード 3 で止まる。
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/Warashi/brainium-tools/internal/orgedit"
	"github.com/Warashi/brainium-tools/internal/orgtask"
	"github.com/Warashi/brainium-tools/internal/repo"
)

// context はサブコマンドが共有する実行時の値。
type context struct {
	root string
	env  orgedit.Env
	git  git
	out  io.Writer
}

type command struct {
	usage string
	// run は usage を受け取り、引数の誤りを報告するときに使う。
	run func(c *context, usage string, args []string) error
}

var commands = map[string]command{
	"next":      nextCommand,
	"current":   currentCommand,
	"step":      stepCommand,
	"drop-step": dropStepCommand,
	"done":      doneCommand,
	"claude":    claudeCommand,
	"repo":      repoCommand,
	"unwork":    unworkCommand,
}

// order は help に並べる順。
var order = []string{"next", "current", "step", "drop-step", "done", "claude", "repo", "unwork"}

func main() {
	err := run(os.Args[1:])
	switch {
	case errors.Is(err, errDirty):
		fmt.Fprintln(os.Stderr, "orgtask:", err)
		os.Exit(3)
	case err != nil:
		fmt.Fprintln(os.Stderr, "orgtask:", err)
		os.Exit(2)
	}
}

func run(args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := repo.Root(dir)
	if err != nil {
		return err
	}
	return execute(&context{root: root, env: orgedit.DefaultEnv(), git: git{root}, out: os.Stdout}, args)
}

func execute(c *context, args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "help" {
		fmt.Fprintln(c.out, "使い方: orgtask <subcommand> [args]")
		for _, name := range order {
			fmt.Fprintf(c.out, "  %s\n", commands[name].usage)
		}
		return nil
	}
	cmd, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("サブコマンド %q は無い。orgtask help で一覧を出す", args[0])
	}
	return cmd.run(c, cmd.usage, args[1:])
}

func usageError(usage string) error {
	return fmt.Errorf("引数が違う。使い方: %s", usage)
}

var nextCommand = command{
	usage: "next [-keep-working ID]...  一手を切り出す task を選んで出す。Next に候補が無ければ Someday の 1 件を Next の末尾に移してコミットする",
	run: func(c *context, usage string, args []string) error {
		var keep []string
		for len(args) > 0 {
			if len(args) < 2 || args[0] != "-keep-working" {
				return usageError(usage)
			}
			keep, args = append(keep, args[1]), args[2:]
		}
		ts, err := orgtask.Load(c.root)
		if err != nil {
			return err
		}
		p := ts.Pick(c.env.Now, keep)
		if p.Status == orgtask.None {
			_, err := fmt.Fprintln(c.out, "status: none")
			return err
		}
		if p.FromSomeday {
			next := ts.Section(orgtask.NextSection)
			if next == nil {
				return fmt.Errorf("%s に %s 節が無い", orgtask.TasksFile, orgtask.NextSection)
			}
			err := c.write(fmt.Sprintf("task: 「%s」を Someday から Next に上げる", p.Task.Heading()), func() ([]string, error) {
				return orgedit.Move(c.root, p.Task.ID, orgedit.Destination{ParentID: next.ID})
			}, orgtask.TasksFile)
			if err != nil {
				return err
			}
		}
		return printTask(c.out, string(p.Status), p.Task, p.Status != orgtask.StaleWorking)
	},
}

var currentCommand = command{
	usage: "current  :CURRENT_STEP: のある task を出す。無ければ status: none",
	run: func(c *context, usage string, args []string) error {
		if len(args) != 0 {
			return usageError(usage)
		}
		t, err := current(c)
		if err != nil || t == nil {
			if err == nil {
				_, err = fmt.Fprintln(c.out, "status: none")
			}
			return err
		}
		return printTask(c.out, "current", t, true)
	},
}

var stepCommand = command{
	usage: "step ID STEP  ID の task に :CURRENT_STEP: を STEP で書き、他の task の :CURRENT_STEP: を消してコミットする",
	run: func(c *context, usage string, args []string) error {
		if len(args) != 2 || strings.TrimSpace(args[1]) == "" || strings.Contains(args[1], "\n") {
			return usageError(usage)
		}
		ts, err := orgtask.Load(c.root)
		if err != nil {
			return err
		}
		t := ts.ByID(args[0])
		if t == nil {
			return fmt.Errorf("%s に ID %s の見出しが無い", orgtask.TasksFile, args[0])
		}
		return c.write(fmt.Sprintf("task: 「%s」の一手を渡す", t.Heading()), func() ([]string, error) {
			var changed []string
			for _, other := range ts.All {
				if other.ID != t.ID && other.Props["CURRENT_STEP"] != "" {
					paths, err := orgedit.SetProperty(c.root, other.ID, "CURRENT_STEP", "")
					if err != nil {
						return nil, err
					}
					changed = append(changed, paths...)
				}
			}
			paths, err := orgedit.SetProperty(c.root, t.ID, "CURRENT_STEP", args[1])
			return append(changed, paths...), err
		}, orgtask.TasksFile)
	},
}

var dropStepCommand = command{
	usage: "drop-step  :CURRENT_STEP: を消してコミットする",
	run: func(c *context, usage string, args []string) error {
		if len(args) != 0 {
			return usageError(usage)
		}
		t, err := requireCurrent(c)
		if err != nil {
			return err
		}
		return c.write(fmt.Sprintf("task: 「%s」の一手を取り下げる", t.Heading()), func() ([]string, error) {
			return orgedit.SetProperty(c.root, t.ID, "CURRENT_STEP", "")
		}, orgtask.TasksFile)
	},
}

var doneCommand = command{
	usage: "done  :CURRENT_STEP: を消し、終えた一手を inbox.org に capture してコミットし、その task を出す。repeater 付きの task は capture しない",
	run: func(c *context, usage string, args []string) error {
		if len(args) != 0 {
			return usageError(usage)
		}
		t, err := requireCurrent(c)
		if err != nil {
			return err
		}
		step := t.Props["CURRENT_STEP"]
		err = c.write(fmt.Sprintf("task: 「%s」の一手を終える", t.Heading()), func() ([]string, error) {
			paths, err := orgedit.SetProperty(c.root, t.ID, "CURRENT_STEP", "")
			if err != nil || t.Repeating() {
				return paths, err
			}
			_, captured, err := orgedit.Capture(c.root, c.env, step, "[["+"id:"+t.ID+"]["+linkDescription(t.Heading())+"]] の一手。\n")
			return append(paths, captured...), err
		}, orgtask.TasksFile, orgedit.InboxFile)
		if err != nil {
			return err
		}
		ts, err := orgtask.Load(c.root)
		if err != nil {
			return err
		}
		return printTask(c.out, "done", ts.ByID(t.ID), true)
	},
}

var claudeCommand = command{
	usage: "claude  :CURRENT_STEP: の task を Claude が進めているとして、:CURRENT_STEP: を消し :WORKING: に今の時刻を書いてコミットする。:REPO: が要る",
	run: func(c *context, usage string, args []string) error {
		if len(args) != 0 {
			return usageError(usage)
		}
		t, err := requireCurrent(c)
		if err != nil {
			return err
		}
		if len(t.Repos()) == 0 {
			return fmt.Errorf("「%s」の :REPO: にリポジトリが無い。Claude を起動するリポジトリを orgtask repo %s <リポジトリ>... で付けてから呼び直す", t.Heading(), t.ID)
		}
		return c.write(fmt.Sprintf("task: 「%s」を Claude に渡し、着手中の印を付ける", t.Heading()), func() ([]string, error) {
			paths, err := orgedit.SetProperty(c.root, t.ID, "CURRENT_STEP", "")
			if err != nil {
				return nil, err
			}
			more, err := orgedit.SetProperty(c.root, t.ID, "WORKING", c.env.Now.Format("[2006-01-02 Mon 15:04]"))
			return append(paths, more...), err
		}, orgtask.TasksFile)
	},
}

var repoCommand = command{
	usage: "repo ID REPO...  task の :REPO: を、一手を実行するリポジトリ(github.com/<owner>/<repo>)にしてコミットする。リポジトリが要らなければ REPO を - 1 つにする",
	run: func(c *context, usage string, args []string) error {
		invalid := func(r string) bool { return strings.TrimSpace(r) == "" || strings.ContainsAny(r, " \t\n") }
		if len(args) < 2 || slices.ContainsFunc(args[1:], invalid) || len(args) > 2 && slices.Contains(args[1:], orgtask.NoRepo) {
			return usageError(usage)
		}
		t, err := find(c, args[0])
		if err != nil {
			return err
		}
		return c.write(fmt.Sprintf("task: 「%s」に一手を実行するリポジトリを記す", t.Heading()), func() ([]string, error) {
			return orgedit.SetProperty(c.root, t.ID, "REPO", strings.Join(args[1:], " "))
		}, orgtask.TasksFile)
	},
}

var unworkCommand = command{
	usage: "unwork ID  task の :WORKING: を消してコミットする。今日より前の :WORKING: を本人がもう進めていないと答えたときに使う",
	run: func(c *context, usage string, args []string) error {
		if len(args) != 1 {
			return usageError(usage)
		}
		t, err := find(c, args[0])
		if err != nil {
			return err
		}
		return c.write(fmt.Sprintf("task: 「%s」の着手中の印を外す", t.Heading()), func() ([]string, error) {
			return orgedit.SetProperty(c.root, t.ID, "WORKING", "")
		}, orgtask.TasksFile)
	},
}

// write は files にまだコミットされていない変更が無いことを確かめてから
// edit で書き、書き換えたファイルを message でコミットする。
func (c *context) write(message string, edit func() ([]string, error), files ...string) error {
	if err := c.git.ensureClean(files...); err != nil {
		return err
	}
	paths, err := edit()
	if err != nil {
		return err
	}
	var changed []string
	for _, p := range paths {
		if !slices.Contains(changed, p) {
			changed = append(changed, p)
		}
	}
	return c.git.commit(message, changed...)
}

func current(c *context) (*orgtask.Task, error) {
	ts, err := orgtask.Load(c.root)
	if err != nil {
		return nil, err
	}
	return ts.CurrentStep(), nil
}

func requireCurrent(c *context) (*orgtask.Task, error) {
	t, err := current(c)
	if err == nil && t == nil {
		err = errors.New(":CURRENT_STEP: のある task が無い")
	}
	return t, err
}

func find(c *context, id string) (*orgtask.Task, error) {
	ts, err := orgtask.Load(c.root)
	if err != nil {
		return nil, err
	}
	t := ts.ByID(id)
	if t == nil {
		return nil, fmt.Errorf("%s に ID %s の見出しが無い", orgtask.TasksFile, id)
	}
	return t, nil
}

// linkDescription はリンクの説明に入れられない ] を除く。
func linkDescription(s string) string {
	return strings.NewReplacer("[", "", "]", "").Replace(s)
}

// printTask は task を「key: 値」の行と、空行に続く subtree の本文で出
// す。
func printTask(w io.Writer, status string, t *orgtask.Task, withText bool) error {
	var b strings.Builder
	fmt.Fprintf(&b, "status: %s\nid: %s\ntitle: %s\nlines: %s:%d-%d\n", status, t.ID, t.Heading(), orgtask.TasksFile, t.Line, t.EndLine)
	for _, key := range []string{"REPO", "CURRENT_STEP", "WORKING"} {
		if v := t.Props[key]; v != "" {
			fmt.Fprintf(&b, "%s: %s\n", strings.ToLower(key), v)
		}
	}
	if withText {
		b.WriteString("\n" + strings.TrimRight(t.Text, "\n") + "\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}
