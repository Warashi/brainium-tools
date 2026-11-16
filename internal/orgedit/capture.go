package orgedit

import (
	"fmt"
	"strings"

	"github.com/Warashi/brainium-tools/internal/orglint"
)

// InboxFile は capture の投入先。
const InboxFile = "inbox.org"

// Capture は inbox.org の先頭に、title の見出しと body の本文の entry を
// 足す。見出しの直後の PROPERTIES ドロワーに新しい :ID: と :CAPTURED:
// を入れ、本文は orglint と同じ幅で折り返す。足した entry の ID を返す。
//
// org-capture で次の template を :prepend t :empty-lines 1 で使ったとき
// と同じ形になる。
//
//   - TITLE
//     :PROPERTIES:
//     :ID:       %(org-id-new)
//     :CAPTURED: %U
//     :END:
//
//     BODY
func Capture(root string, env Env, title, body string) (string, []string, error) {
	title = strings.TrimSpace(title)
	if title == "" || strings.Contains(title, "\n") {
		return "", nil, fmt.Errorf("見出しは空でない 1 行にする")
	}
	body = strings.Trim(body, "\n")
	for _, line := range strings.Split(body, "\n") {
		if starsRe.MatchString(line) {
			return "", nil, fmt.Errorf("本文に見出しの行がある: %q", line)
		}
	}
	id := env.NewID()
	entry := "* " + title + "\n" + ":PROPERTIES:\n" + propertyLine("ID", id) + propertyLine("CAPTURED", inactiveTimestamp(env.Now)) + ":END:\n"
	if body != "" {
		entry += "\n" + body + "\n"
	}
	entry = string(orglint.Format([]byte(entry)))
	paths, err := Update(root, func(f *Files) error {
		src, ok, err := f.Read(InboxFile)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s が無い", InboxFile)
		}
		f.Write(InboxFile, placeEntry(src, nextHeading(src, 0), entry))
		return nil
	})
	return id, paths, err
}

// placeEntry は org-capture-place-entry と同じく、pos に entry を置き、
// 前後の空行を 1 行ずつにする(:empty-lines 1)。
func placeEntry(src string, pos int, entry string) string {
	before := src[:pos]
	if before != "" && !strings.HasSuffix(before, "\n") {
		before += "\n"
	}
	after := src[pos:]
	// 後ろに続く空白だけの行を消す
	for {
		i := strings.IndexByte(after, '\n')
		if i < 0 || strings.TrimLeft(after[:i], " \t") != "" {
			break
		}
		after = after[i+1:]
	}
	return backOverEmptyLines(before) + "\n" + backOverEmptyLines(entry) + "\n" + after
}

// backOverEmptyLines は org-back-over-empty-lines と同じく、末尾の空行を
// 除く。最後の空でない行の改行は残す。
func backOverEmptyLines(s string) string {
	p := len(strings.TrimRight(s, " \t\n\r"))
	if i := strings.IndexByte(s[p:], '\n'); i >= 0 {
		return s[:p+i+1]
	}
	return s
}
