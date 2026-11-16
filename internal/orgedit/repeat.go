package orgedit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Warashi/brainium-tools/internal/org"
)

var (
	// repeaterRe は org-repeat-re の repeater の部分。
	repeaterRe = regexp.MustCompile(`([.+]?\+)(\d+)([hdwmy])`)
	// activeTimestampRe はアクティブなタイムスタンプを、日付・曜日と時刻・
	// 残り(repeater や警告)に分ける。
	activeTimestampRe = regexp.MustCompile(`<(\d{4}-\d{2}-\d{2})(?: [^ >0-9+.-]+)?((?: \d{1,2}:\d{2}(?:-\d{1,2}:\d{2})?)?)([^>]*)>`)
	// plannedRe は計画行の SCHEDULED と DEADLINE。
	plannedRe = regexp.MustCompile(`(SCHEDULED|DEADLINE): *(<[^>\n]*>)`)
)

// repeating は見出しの計画行に、0 でない repeater を持つタイムスタンプが
// あるかを返す(org-get-repeat)。
func repeating(src string, h *org.Node) bool {
	p := planning(h)
	if p == nil {
		return false
	}
	start, end := lineBounds(src, p.Begin)
	for _, m := range plannedRe.FindAllStringSubmatch(src[start:end], -1) {
		if r := repeaterRe.FindStringSubmatch(m[2]); r != nil {
			if n, _ := strconv.Atoi(r[2]); n != 0 {
				return true
			}
		}
	}
	return false
}

// repeat は org-auto-repeat-maybe と同じく、完了にした repeater 付きの見
// 出しを未完了に戻す。CLOSED を消し、LAST_REPEAT を足し、計画行の
// repeater 付きのタイムスタンプを進め、LOGBOOK に完了の記録を足す。
// 戻す状態は REPEAT_TO_STATE か、元の状態の系列の先頭。
func repeat(src, id string, c todoConfig, state, old string, env Env) (string, error) {
	_, h, err := reparse(src, id)
	if err != nil {
		return "", err
	}
	if err := checkRepeaters(src, h); err != nil {
		return "", err
	}
	to := c.head[old]
	if v, ok := property(h, "REPEAT_TO_STATE"); ok && c.isTodo(v) {
		to = v
	}
	start, end := lineBounds(src, h.Begin)
	src = src[:start] + realignTags(replaceTodo(src[start:end], c.keywords, to)) + src[end:]
	if _, h, err = reparse(src, id); err != nil {
		return "", err
	}
	src = removeClosed(src, h)
	if _, h, err = reparse(src, id); err != nil {
		return "", err
	}
	if src, err = PutProperty(src, h, "LAST_REPEAT", inactiveTimestamp(env.Now)); err != nil {
		return "", err
	}
	if _, h, err = reparse(src, id); err != nil {
		return "", err
	}
	if src, err = shiftPlanning(src, h, env.Now); err != nil {
		return "", err
	}
	if !c.logDrawer {
		return "", fmt.Errorf("#+STARTUP: logdrawer の無いファイルの記録は扱えない")
	}
	if _, h, err = reparse(src, id); err != nil {
		return "", err
	}
	return addLogNote(src, h, stateNote(state, old, env)), nil
}

// checkRepeaters は扱えない repeater を断る。本文のタイムスタンプの
// repeater と、時間単位の repeater。
func checkRepeaters(src string, h *org.Node) error {
	body := src[afterPlanning(src, h):entryEnd(h)]
	for _, ts := range activeTimestampRe.FindAllString(body, -1) {
		if repeaterRe.MatchString(ts) {
			return fmt.Errorf("本文のタイムスタンプの repeater %s は扱えない", ts)
		}
	}
	p := planning(h)
	start, end := lineBounds(src, p.Begin)
	for _, m := range plannedRe.FindAllStringSubmatch(src[start:end], -1) {
		if r := repeaterRe.FindStringSubmatch(m[2]); r != nil && r[3] == "h" {
			return fmt.Errorf("時間単位の repeater %s は扱えない", m[2])
		}
	}
	return nil
}

// entryEnd は見出しの最初の子の見出しの手前(無ければ subtree の終わり)
// を返す(org-entry-end-position)。
func entryEnd(h *org.Node) int {
	for _, c := range h.Children {
		if c.Type == org.TypeHeadline {
			return c.Begin
		}
	}
	return h.End
}

// shiftPlanning は計画行の repeater 付きのタイムスタンプを進める。
// repeater の無い SCHEDULED は、後ろの空白ごと消す。
func shiftPlanning(src string, h *org.Node, now time.Time) (string, error) {
	p := planning(h)
	start, end := lineBounds(src, p.Begin)
	line := src[start:end]
	var b strings.Builder
	last := 0
	for _, m := range plannedRe.FindAllStringSubmatchIndex(line, -1) {
		b.WriteString(line[last:m[0]])
		last = m[1]
		keyword, ts := line[m[2]:m[3]], line[m[4]:m[5]]
		if !repeaterRe.MatchString(ts) {
			if keyword == "SCHEDULED" {
				for last < len(line) && (line[last] == ' ' || line[last] == '\t') {
					last++
				}
				continue
			}
			b.WriteString(line[m[0]:m[1]])
			continue
		}
		shifted, err := shiftTimestamp(ts, now)
		if err != nil {
			return "", err
		}
		b.WriteString(line[m[0]:m[4]] + shifted)
	}
	b.WriteString(line[last:])
	return src[:start] + strings.TrimRight(b.String(), " \t") + src[end:], nil
}

// shiftTimestamp は repeater に従ってタイムスタンプの日付を進める。
// .+ は今日から、++ は今日より後になるまで、+ は元の日付から n 回分進め
// る。
func shiftTimestamp(ts string, now time.Time) (string, error) {
	m := activeTimestampRe.FindStringSubmatch(ts)
	r := repeaterRe.FindStringSubmatch(ts)
	if m == nil || r == nil {
		return "", fmt.Errorf("タイムスタンプ %s を読めない", ts)
	}
	date, err := time.ParseInLocation("2006-01-02", m[1], now.Location())
	if err != nil {
		return "", err
	}
	n, _ := strconv.Atoi(r[2])
	add := func(t time.Time) time.Time {
		switch r[3] {
		case "d":
			return t.AddDate(0, 0, n)
		case "w":
			return t.AddDate(0, 0, 7*n)
		case "m":
			return t.AddDate(0, n, 0)
		}
		return t.AddDate(n, 0, 0)
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch r[1] {
	case ".+":
		date = add(today)
	case "++":
		date = add(date)
		for !date.After(today) {
			date = add(date)
		}
	default:
		date = add(date)
	}
	return "<" + date.Format("2006-01-02 Mon") + m[2] + m[3] + ">", nil
}
