package outline

import (
	"strings"
	"unicode/utf8"
)

// Opening は最初の見出しより前にある最初の段落から、最初の文を返す。
//
// PageIndex は短い節には要約の代わりに本文そのものを載せる。ここでは LLM
// で要約を作らないので、ノートが冒頭に書く目的の 1 文を手掛かりとして
// 原文のまま返す。文が maxBytes を超えたら文字の境界で切り、… を足す。
// … は maxBytes に数えない。
func Opening(source string, maxBytes int) string {
	var paragraph []string
	inDrawer, inBlock := false, false
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if _, _, ok := heading(line); ok {
			break
		}
		lower := strings.ToLower(trimmed)
		switch {
		case inBlock:
			inBlock = !strings.HasPrefix(lower, "#+end_")
			continue
		case strings.HasPrefix(lower, "#+begin_"):
			inBlock = true
			continue
		case inDrawer:
			inDrawer = !strings.EqualFold(trimmed, ":END:")
			continue
		case len(trimmed) > 2 && trimmed[0] == ':' && trimmed[len(trimmed)-1] == ':':
			inDrawer = true
			continue
		case strings.HasPrefix(trimmed, "#"):
			continue
		case trimmed == "":
			if len(paragraph) > 0 {
				return firstSentence(paragraph, maxBytes)
			}
			continue
		}
		paragraph = append(paragraph, trimmed)
	}
	return firstSentence(paragraph, maxBytes)
}

func firstSentence(lines []string, maxBytes int) string {
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			prev, _ := utf8.DecodeLastRuneInString(b.String())
			next, _ := utf8.DecodeRuneInString(line)
			// 和文の折り返しは空白なしでつなぐ。欧文どうしだけ空白を戻す
			if prev < utf8.RuneSelf && next < utf8.RuneSelf {
				b.WriteByte(' ')
			}
		}
		b.WriteString(line)
	}
	text := b.String()
	if i := strings.Index(text, "。"); i >= 0 {
		text = text[:i+len("。")]
	} else if i := strings.Index(text, ". "); i >= 0 {
		text = text[:i+1]
	}
	if len(text) <= maxBytes {
		return text
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + "…"
}
