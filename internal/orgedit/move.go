package orgedit

import (
	"fmt"
	"regexp"
	"strings"
)

// Destination は subtree の移動先。ParentID・BeforeID・File のどれか 1
// つを指定する。
type Destination struct {
	// ParentID は移動先の親の見出しの :ID:。子の末尾(First なら先頭)
	// に移す。org-refile と同じ。
	ParentID string
	// BeforeID は移動先の兄弟の見出しの :ID:。その手前に、同じ深さで移
	// す。
	BeforeID string
	// File は移動先のファイル(リポジトリルートからの相対パス)。深さ 1
	// で末尾(First なら最初の見出しの手前)に移す。org-refile でファイ
	// ルそのものを選んだときと同じ。
	File string
	// First は ParentID と File で、末尾ではなく先頭に移すかどうか
	// (org-reverse-note-order)。
	First bool
}

// Move は :ID: が id の見出しの subtree を dst に移す。org-refile と同じ
// く、subtree の後ろの空行ごと切り取り、深さを直して貼り、先頭の見出し
// のタグを揃え直す。移動先が別のファイルなら、移動先を先に書く。
func Move(root, id string, dst Destination) ([]string, error) {
	return Update(root, func(f *Files) error {
		return move(f, id, dst)
	})
}

func move(f *Files, id string, dst Destination) error {
	e, err := Find(f, id)
	if err != nil {
		return err
	}
	file, pos, level, err := locate(f, e, dst)
	if err != nil {
		return err
	}
	h := e.Headline
	if file == e.Path && pos > h.Begin && pos < h.End {
		return fmt.Errorf("移動先が移す subtree %s の中にある", id)
	}
	text := alignFirstHeading(shiftLevels(e.Src[h.Begin:h.End], level-h.Level))
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if file == e.Path {
		src := e.Src[:h.Begin] + e.Src[h.End:]
		if pos >= h.End {
			pos -= h.End - h.Begin
		}
		f.Write(file, insertSubtree(src, pos, text))
		return nil
	}
	dstSrc, _, err := f.Read(file)
	if err != nil {
		return err
	}
	f.Write(file, insertSubtree(dstSrc, pos, text))
	f.Write(e.Path, e.Src[:h.Begin]+e.Src[h.End:])
	return nil
}

// locate は移動先のファイル・位置・深さを返す。
func locate(f *Files, e *Entry, dst Destination) (string, int, int, error) {
	n := 0
	for _, s := range []string{dst.ParentID, dst.BeforeID, dst.File} {
		if s != "" {
			n++
		}
	}
	if n != 1 {
		return "", 0, 0, fmt.Errorf("移動先は親・兄弟・ファイルのどれか 1 つで指す")
	}
	switch {
	case dst.ParentID != "":
		p, err := Find(f, dst.ParentID)
		if err != nil {
			return "", 0, 0, err
		}
		src, h := p.Src, p.Headline
		if dst.First {
			return p.Path, nextHeading(src, lineEnd(src, h.Begin)), h.Level + 1, nil
		}
		return p.Path, h.End, h.Level + 1, nil
	case dst.BeforeID != "":
		s, err := Find(f, dst.BeforeID)
		if err != nil {
			return "", 0, 0, err
		}
		if dst.First {
			return "", 0, 0, fmt.Errorf("兄弟の手前に移すときは先頭を指定できない")
		}
		return s.Path, s.Headline.Begin, s.Headline.Level, nil
	}
	src, ok, err := f.Read(dst.File)
	if err != nil {
		return "", 0, 0, err
	}
	if !ok {
		return "", 0, 0, fmt.Errorf("%s が無い", dst.File)
	}
	if dst.First {
		return dst.File, nextHeading(src, 0), 1, nil
	}
	return dst.File, len(src), 1, nil
}

// nextHeading は pos 以降で最初の見出しの位置を返す(outline-next-heading)。
// 無ければ src の末尾。
func nextHeading(src string, pos int) int {
	for pos < len(src) {
		if starsRe.MatchString(src[pos:lineEnd(src, pos)]) {
			return pos
		}
		pos = lineEnd(src, pos)
	}
	return len(src)
}

// insertSubtree は org-paste-subtree と同じく、pos に text を貼る。pos が
// 改行で終わらない最終行の末尾なら、先に改行を足す。
func insertSubtree(src string, pos int, text string) string {
	if pos == len(src) && src != "" && !strings.HasSuffix(src, "\n") {
		return src + "\n" + text
	}
	return src[:pos] + text + src[pos:]
}

// alignFirstHeading は org-refile が貼った後にするのと同じく、先頭の見
// 出しのタグを揃え直す。
func alignFirstHeading(text string) string {
	end := strings.IndexByte(text, '\n')
	if end < 0 {
		end = len(text)
	}
	return realignTags(text[:end]) + text[end:]
}

var headlineStarsRe = regexp.MustCompile(`(?m)^\*+[ \t]`)

// shiftLevels は text の見出しの深さを delta だけ変える。org-demote・
// org-promote と同じく、変えた見出しのタグを揃え直す。
func shiftLevels(text string, delta int) string {
	if delta == 0 {
		return text
	}
	var b strings.Builder
	for _, line := range strings.SplitAfter(text, "\n") {
		if !headlineStarsRe.MatchString(line) {
			b.WriteString(line)
			continue
		}
		stars := len(line) - len(strings.TrimLeft(line, "*"))
		body, nl := strings.TrimSuffix(line, "\n"), ""
		if strings.HasSuffix(line, "\n") {
			nl = "\n"
		}
		b.WriteString(realignTags(strings.Repeat("*", stars+delta)+body[stars:]) + nl)
	}
	return b.String()
}
