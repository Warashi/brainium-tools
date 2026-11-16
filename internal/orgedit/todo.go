package orgedit

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Warashi/brainium-tools/internal/org"
)

// SetTodo は :ID: が id の見出しの TODO キーワードを state にする。state
// が空ならキーワードを外す。org-todo と同じく、完了の状態に入れば
// CLOSED を足し、未完了の状態に戻れば CLOSED を消し、! の付いた状態に入
// れば LOGBOOK に記録を足し、タグを揃え直す。
func SetTodo(root string, env Env, id, state string) ([]string, error) {
	return Update(root, func(f *Files) error {
		e, err := Find(f, id)
		if err != nil {
			return err
		}
		src, err := setTodo(e.Src, e.Root, id, state, env)
		if err != nil {
			return err
		}
		f.Write(e.Path, src)
		return nil
	})
}

func setTodo(src string, root *org.Node, id, state string, env Env) (string, error) {
	c, err := readTodoConfig(root)
	if err != nil {
		return "", err
	}
	if state != "" && !c.isTodo(state) {
		return "", fmt.Errorf("%s はこのファイルの TODO キーワードに無い(%s)", state, strings.Join(c.keywords, " "))
	}
	h, err := FindIn(root, id)
	if err != nil {
		return "", err
	}
	start, end := lineBounds(src, h.Begin)
	old := parseHeading(src[start:end], c.keywords).todo
	src = src[:start] + realignTags(replaceTodo(src[start:end], c.keywords, state)) + src[end:]

	logging := len(c.logState) > 0 || c.logDone
	if !logging {
		return src, nil
	}
	nowDone := c.done[state] && !c.done[old]
	becameTodo := state == "" || (!c.done[state] && (old == "" || c.done[old]))
	if becameTodo {
		if _, h, err = reparse(src, id); err != nil {
			return "", err
		}
		src = removeClosed(src, h)
	}
	if nowDone && c.logDone {
		if _, h, err = reparse(src, id); err != nil {
			return "", err
		}
		src = addClosed(src, h, env)
	}
	if state != "" && c.logState[state] {
		if !c.logDrawer {
			return "", fmt.Errorf("#+STARTUP: logdrawer の無いファイルの記録は扱えない")
		}
		if _, h, err = reparse(src, id); err != nil {
			return "", err
		}
		src = addLogNote(src, h, stateNote(state, old, env))
	}
	return src, nil
}

// stateNote は org-log-note-headings の state の既定値
// "State %-12s from %-12s %t" で記録の 1 行を作る。
func stateNote(state, old string, env Env) string {
	quote := func(s string) string {
		if s == "" {
			return ""
		}
		return `"` + s + `"`
	}
	return fmt.Sprintf("- State %-12s from %-12s %s", quote(state), quote(old), inactiveTimestamp(env.Now))
}

var (
	closedRe = regexp.MustCompile(`CLOSED: *\[[^]\n]*\]`)
	// planningKeywordRe は org-keyword-time-not-clock-regexp の、キーワー
	// ドの部分。
	planningKeywordRe = regexp.MustCompile(`(?:CLOSED|DEADLINE|SCHEDULED):`)
)

// addClosed は org-add-planning-info と同じく、計画行の先頭に CLOSED を
// 入れる。計画行が無ければ見出しの直後に作る。
func addClosed(src string, h *org.Node, env Env) string {
	closed := "CLOSED: " + inactiveTimestamp(env.Now)
	p := planning(h)
	if p == nil {
		return insertLines(src, lineEnd(src, h.Begin), closed+"\n")
	}
	start, end := lineBounds(src, p.Begin)
	line := withoutClosed(src[start:end])
	body := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(body)]
	if body == "" {
		line = indent + closed
	} else {
		line = indent + closed + " " + body
	}
	return src[:start] + line + src[end:]
}

// removeClosed は計画行の CLOSED を消す。計画行が空になれば行ごと消す。
func removeClosed(src string, h *org.Node) string {
	p := planning(h)
	if p == nil {
		return src
	}
	start, end := lineBounds(src, p.Begin)
	line := withoutClosed(src[start:end])
	if strings.TrimSpace(line) == "" {
		// 前の行の行末から消す(delete-region (line-end-position 0) ...)
		return src[:start-1] + src[end:]
	}
	return src[:start] + line + src[end:]
}

// withoutClosed は計画行から CLOSED を次のキーワードの手前まで消し、行
// 末の空白を除く。
func withoutClosed(line string) string {
	m := closedRe.FindStringIndex(line)
	if m == nil {
		return line
	}
	cut := len(line)
	if k := planningKeywordRe.FindStringIndex(line[m[1]:]); k != nil {
		cut = m[1] + k[0]
	}
	return strings.TrimRight(line[:m[0]]+line[cut:], " \t")
}

// lineBounds は pos を含む行の先頭と、改行を除いた終わりを返す。
func lineBounds(src string, pos int) (int, int) {
	start := strings.LastIndexByte(src[:pos], '\n') + 1
	end := lineEnd(src, pos)
	if end > start && src[end-1] == '\n' {
		end--
	}
	return start, end
}

// addLogNote は org-store-log-note と同じく、LOGBOOK ドロワーの先頭に
// note を足す。ドロワーが無ければ見出しのメタデータの直後に作る。
func addLogNote(src string, h *org.Node, note string) string {
	meta := endOfMetaData(src, h)
	limit := h.End
	if h.ContentsBegin >= 0 {
		for _, c := range h.Children {
			if c.Type == org.TypeHeadline {
				limit = c.Begin
				break
			}
		}
	}
	for pos := meta; pos < limit; pos = lineEnd(src, pos) {
		if strings.EqualFold(strings.TrimSpace(src[pos:lineEnd(src, pos)]), ":LOGBOOK:") {
			at := lineEnd(src, pos)
			return insertLines(src, at, note+"\n")
		}
	}
	return insertLines(src, meta, ":LOGBOOK:\n"+note+"\n:END:\n")
}
