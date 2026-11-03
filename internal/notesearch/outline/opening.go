package outline

import (
	"strings"
	"unicode/utf8"

	"github.com/Warashi/brainium-tools/internal/org"
)

// DefaultOpeningBytes は冒頭の文の既定の上限。長い文は手掛かりとして読
// み切れない
const DefaultOpeningBytes = 240

// Opening は最初の見出しより前にある最初の段落から、最初の文を返す。
// 段落は org の parser に従い、ドロワーやリストの中の段落は使わない。
//
// PageIndex は短い節には要約の代わりに本文そのものを載せる。ここでは LLM
// で要約を作らないので、ノートが冒頭に書く目的の 1 文を手掛かりとして
// 原文から返す。文が maxBytes を超えたら文字の境界で切り、… を足す。
// … は maxBytes に数えない。
func Opening(source string, maxBytes int) string {
	root := org.Parse(source)
	if len(root.Children) == 0 || root.Children[0].Type != org.TypeSection {
		return ""
	}
	for _, n := range root.Children[0].Children {
		if n.Type != org.TypeParagraph {
			continue
		}
		var lines []string
		for _, line := range strings.Split(linkTexts(source, n), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				lines = append(lines, line)
			}
		}
		return firstSentence(lines, maxBytes)
	}
	return ""
}

// linkTexts は段落の中身を、括弧の link を Emacs での見え方と同じく説明
// 文(無ければ行き先)に置き換えて返す。記法を残すと、手掛かりにならな
// い ID で上限を使い切る。
func linkTexts(source string, paragraph *org.Node) string {
	var b strings.Builder
	pos := paragraph.ContentsBegin
	for _, l := range org.Links(paragraph) {
		link := strings.TrimRight(source[l.Begin:l.End], " \t")
		if !strings.HasPrefix(link, "[[") {
			continue
		}
		b.WriteString(source[pos:l.Begin])
		if l.ContentsBegin >= 0 {
			b.WriteString(source[l.ContentsBegin:l.ContentsEnd])
		} else {
			b.WriteString(link[2 : len(link)-2])
		}
		pos = l.Begin + len(link)
	}
	b.WriteString(source[pos:paragraph.ContentsEnd])
	return b.String()
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
