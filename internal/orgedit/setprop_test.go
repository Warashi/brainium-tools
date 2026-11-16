package orgedit

import "testing"

func TestSetPropertyMatchesEmacs(t *testing.T) {
	put := func(name, id, key, value string) emacsCase {
		return emacsCase{
			name:  name,
			files: map[string]string{"tasks.org": tasksOrg},
			run: func(root string, _ Env) error {
				_, err := SetProperty(root, id, key, value)
				return err
			},
			elisp: `(edit-goto-id "tasks.org" "` + id + `") (org-entry-put nil "` + key + `" "` + value + `")`,
		}
	}
	del := func(name, id, key string) emacsCase {
		return emacsCase{
			name:  name,
			files: map[string]string{"tasks.org": tasksOrg},
			run: func(root string, _ Env) error {
				_, err := SetProperty(root, id, key, "")
				return err
			},
			elisp: `(edit-goto-id "tasks.org" "` + id + `") (org-entry-delete nil "` + key + `")`,
		}
	}
	runEmacsCases(t, []emacsCase{
		put("足す", "t-tag", "WORKING", "[2026-11-16 Mon 21:15]"),
		put("計画行のある見出しに足す", "t-sched", "CURRENT_STEP", "Claude に「テストを足して」と打つ"),
		put("置き換える", "t-logged", "CURRENT_STEP", "別の一手"),
		put("小文字の key で置き換える", "t-logged", "current_step", "別の一手"),
		put("長い key", "t-tag", "ARCHIVE_CATEGORY", "Next"),
		put("ファイル末尾の見出し", "t-last", "WORKING", "[2026-11-16 Mon 21:15]"),
		del("消す", "t-logged", "CURRENT_STEP"),
		del("最後の 1 つを消すとドロワーも消える", "t-last", "ID"),
		del("無い key", "t-tag", "WORKING"),
	})
}
