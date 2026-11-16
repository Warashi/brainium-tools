package orgedit

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Warashi/brainium-tools/internal/org"
)

// capturedRe は :CAPTURED: の値から日付と時刻を取り出す。
var capturedRe = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2}) [^ \]]+(?: (\d{2}:\d{2}))?\]$`)

// Journal は :ID: が id の capture を、:CAPTURED: の日付のデイリーノート
// journal/<YYYY>/<YYYY-MM-DD>.org へ移す。ID と :CAPTURED: は引き継ぐ。
// デイリーノートが無ければ、新しい :ID: と、:CREATED: にその capture の
// :CAPTURED: を入れて作る。
//
// デイリーノートの中は :CAPTURED: の順に保つ。後の capture があればその
// 手前に移し(Destination.BeforeID)、無ければ末尾に移す
// (Destination.File)。
func Journal(root string, env Env, id string) ([]string, error) {
	newID := env.NewID()
	return Update(root, func(f *Files) error {
		e, err := Find(f, id)
		if err != nil {
			return err
		}
		if e.Headline.Level != 1 {
			return fmt.Errorf("ID %s は深さ 1 の見出しではない", id)
		}
		captured, ok := property(e.Headline, "CAPTURED")
		if !ok {
			return fmt.Errorf("ID %s に :CAPTURED: が無い", id)
		}
		date, err := capturedTime(captured)
		if err != nil {
			return err
		}
		file := journalPath(date)
		if file == e.Path {
			return fmt.Errorf("ID %s は既に %s にある", id, file)
		}
		src, ok, err := f.Read(file)
		if err != nil {
			return err
		}
		if !ok {
			src = journalHeader(newID, captured, date)
			f.Write(file, src)
		}
		dst := Destination{File: file}
		if later := laterEntry(org.Parse(src), date); later != "" {
			dst = Destination{BeforeID: later}
		}
		return move(f, id, dst)
	})
}

func journalPath(date time.Time) string {
	return path.Join("journal", date.Format("2006"), date.Format("2006-01-02")+".org")
}

func journalHeader(id, created string, date time.Time) string {
	return ":PROPERTIES:\n" + propertyLine("ID", id) + propertyLine("CREATED", created) + ":END:\n" +
		"#+title: " + date.Format("2006-01-02") + "\n\n"
}

func capturedTime(captured string) (time.Time, error) {
	m := capturedRe.FindStringSubmatch(captured)
	if m == nil {
		return time.Time{}, fmt.Errorf(":CAPTURED: %q が [YYYY-MM-DD Day HH:MM] の形でない", captured)
	}
	hm := m[2]
	if hm == "" {
		hm = "00:00"
	}
	return time.Parse("2006-01-02 15:04", m[1]+" "+hm)
}

// laterEntry は深さ 1 の見出しのうち、:CAPTURED: が date より後の最初の
// ものの :ID: を返す。無ければ空文字列。
func laterEntry(root *org.Node, date time.Time) string {
	for _, h := range root.Children {
		if h.Type != org.TypeHeadline {
			continue
		}
		captured, ok := property(h, "CAPTURED")
		if !ok {
			continue
		}
		t, err := capturedTime(captured)
		if err != nil || !t.After(date) {
			continue
		}
		if id, ok := property(h, "ID"); ok && strings.TrimSpace(id) != "" {
			return id
		}
	}
	return ""
}
