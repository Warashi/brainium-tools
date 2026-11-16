package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// errDirty は、書く予定のファイルに他の書き手のまだコミットされていない
// 変更があること。混ぜてコミットしないよう、書く前に止める。
var errDirty = errors.New("まだコミットされていない変更がある")

// git は root のリポジトリで git を呼ぶ。
type git struct{ root string }

func (g git) run(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", g.root}, args...)...)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out.String(), nil
}

// ensureClean は files に、作業ツリーか index のまだコミットされていない
// 変更があれば errDirty を返す。
func (g git) ensureClean(files ...string) error {
	out, err := g.run(append([]string{"status", "--porcelain", "--"}, files...)...)
	if err != nil {
		return err
	}
	if out = strings.TrimSpace(out); out != "" {
		return fmt.Errorf("%w。他のセッションのコミットを待つか、まとめてコミットしてから呼び直す:\n%s", errDirty, out)
	}
	return nil
}

// commit は files を 1 つずつ add し、files だけを message でコミットす
// る。files が空なら何もしない。
func (g git) commit(message string, files ...string) error {
	if len(files) == 0 {
		return nil
	}
	for _, f := range files {
		if _, err := g.run("add", "--", f); err != nil {
			return err
		}
	}
	_, err := g.run(append([]string{"commit", "-q", "-m", message, "--"}, files...)...)
	return err
}
