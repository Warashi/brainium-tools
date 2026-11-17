package orgedit

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Warashi/brainium-tools/internal/org"
)

// todoConfig はファイルの #+TODO と #+STARTUP から決まる TODO の設定。
type todoConfig struct {
	keywords []string
	done     map[string]bool
	// head はキーワードから、その系列の先頭のキーワードを引く。
	head map[string]string
	// logState は org-todo-log-states。その状態に入るときに記録を残す
	// キーワード。
	logState map[string]bool
	// logDone は org-log-done が time かどうか。
	logDone bool
	// logDrawer は org-log-into-drawer が LOGBOOK かどうか。
	logDrawer bool
	// logReschedule は org-log-reschedule が time かどうか。
	logReschedule bool
}

// todoDefRe は TODO キーワードの定義の語を、名前と括弧の中に分ける。
var todoDefRe = regexp.MustCompile(`^(.*?)(?:\((.*)\))?$`)

// readTodoConfig は root の keyword から TODO の設定を読む。org の既定
// (TODO | DONE、記録なし)から始め、#+STARTUP の logdone・logdrawer な
// どで上書きする。
func readTodoConfig(root *org.Node) (todoConfig, error) {
	c := todoConfig{done: map[string]bool{}, logState: map[string]bool{}, head: map[string]string{}}
	var sequences [][]string
	var startup []string
	walkKeywords(root, func(k *org.Node) {
		switch k.Key {
		case "TODO", "SEQ_TODO", "TYP_TODO":
			sequences = append(sequences, strings.Fields(k.Value))
		case "STARTUP":
			startup = append(startup, strings.Fields(k.Value)...)
		}
	})
	if sequences == nil {
		sequences = [][]string{{"TODO", "DONE"}}
	}
	for _, words := range sequences {
		bar := -1
		for i, w := range words {
			if w == "|" {
				bar = i
			}
		}
		if bar < 0 {
			// | が無ければ最後の語が完了の状態
			bar = len(words) - 2
		}
		first := ""
		for i, w := range words {
			if w == "|" {
				continue
			}
			m := todoDefRe.FindStringSubmatch(w)
			name, spec := m[1], m[2]
			if first == "" {
				first = name
			}
			c.head[name] = first
			c.keywords = append(c.keywords, name)
			if i > bar {
				c.done[name] = true
			}
			if err := c.readLogSpec(name, spec); err != nil {
				return c, err
			}
		}
	}
	for _, s := range startup {
		switch strings.ToLower(s) {
		case "logdone":
			c.logDone = true
		case "nologdone":
			c.logDone = false
		case "logdrawer":
			c.logDrawer = true
		case "nologdrawer":
			c.logDrawer = false
		case "logreschedule":
			c.logReschedule = true
		case "nologreschedule":
			c.logReschedule = false
		case "lognotedone", "lognoteclock-out", "lognotereschedule", "lognoteredeadline", "lognoterepeat":
			return c, fmt.Errorf("#+STARTUP: %s はメモの入力が要るので扱えない", s)
		}
	}
	return c, nil
}

// readLogSpec はキーワードの括弧の中の記録の指定を読む。"t!" のような
// 入るときの時刻の記録だけを扱う。
func (c *todoConfig) readLogSpec(name, spec string) error {
	if spec == "" {
		return nil
	}
	// 先頭の 1 文字は高速選択のキー
	rest := spec
	if r := []rune(spec); r[0] != '!' && r[0] != '@' && r[0] != '/' {
		rest = string(r[1:])
	}
	switch rest {
	case "":
	case "!":
		c.logState[name] = true
	default:
		return fmt.Errorf("TODO キーワード %s(%s) の記録の指定は扱えない。扱えるのは ! だけ", name, spec)
	}
	return nil
}

func (c todoConfig) isTodo(k string) bool {
	for _, w := range c.keywords {
		if w == k {
			return true
		}
	}
	return false
}

// walkKeywords は root 配下の keyword を文書の順にたどる。
func walkKeywords(n *org.Node, fn func(*org.Node)) {
	if n.Type == org.TypeKeyword {
		fn(n)
	}
	for _, c := range n.Children {
		walkKeywords(c, fn)
	}
}
