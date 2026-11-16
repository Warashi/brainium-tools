package orgedit

import (
	"strconv"
	"testing"
)

func TestMoveMatchesEmacs(t *testing.T) {
	files := map[string]string{"tasks.org": tasksOrg, "inbox.org": inboxOrg}
	refile := func(name, file, id, targetFile, targetID string, first bool) emacsCase {
		target := "nil"
		if targetID != "" {
			target = strconv.Quote(targetID)
		}
		firstArg := "nil"
		if first {
			firstArg = "t"
		}
		dst := Destination{ParentID: targetID, First: first}
		if targetID == "" {
			dst.File = targetFile
		}
		return emacsCase{
			name:  name,
			files: files,
			run: func(root string, _ Env) error {
				_, err := Move(root, id, dst)
				return err
			},
			elisp: `(edit-refile "` + file + `" "` + id + `" "` + targetFile + `" ` + target + ` ` + firstArg + `)`,
		}
	}
	before := func(name, file, id, siblingFile, siblingID string) emacsCase {
		return emacsCase{
			name:  name,
			files: files,
			run: func(root string, _ Env) error {
				_, err := Move(root, id, Destination{BeforeID: siblingID})
				return err
			},
			elisp: `(edit-move-before "` + file + `" "` + id + `" "` + siblingFile + `" "` + siblingID + `")`,
		}
	}
	runEmacsCases(t, []emacsCase{
		refile("別のファイルの節の末尾へ深さを下げて移す", "inbox.org", "c-new", "tasks.org", "sec-next", false),
		refile("改行で終わらないファイルの末尾の節へ移す", "inbox.org", "c-old", "tasks.org", "sec-someday", false),
		refile("同じファイルの節の先頭へ移す", "tasks.org", "t-last", "tasks.org", "sec-next", true),
		refile("同じファイルの後ろの節の末尾へ移す", "tasks.org", "t-sched", "tasks.org", "sec-someday", false),
		refile("子を持つ見出しを後ろの節の先頭へ移す", "tasks.org", "t-logged", "tasks.org", "sec-someday", true),
		refile("深さを上げて節の末尾へ移す", "tasks.org", "t-child", "tasks.org", "sec-next", false),
		refile("別のファイルの深さ 1 の末尾へ移す", "inbox.org", "c-new", "tasks.org", "", false),
		refile("深さ 1 の末尾へ深さを上げて移す", "tasks.org", "t-tag", "inbox.org", "", false),
		before("後ろの兄弟の手前へ移す", "tasks.org", "t-tag", "tasks.org", "t-done"),
		before("前の兄弟の手前へ移す", "tasks.org", "t-done", "tasks.org", "t-tag"),
		before("別のファイルの見出しの手前へ移す", "inbox.org", "c-old", "tasks.org", "t-sched"),
	})
}
