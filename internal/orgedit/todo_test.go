package orgedit

import "testing"

func TestSetTodoMatchesEmacs(t *testing.T) {
	todo := func(name, file, id, state string) emacsCase {
		files := map[string]string{"tasks.org": tasksOrg, "inbox.org": inboxOrg}
		arg := `"` + state + `"`
		if state == "" {
			arg = `'none`
		}
		return emacsCase{
			name:  name,
			files: files,
			run: func(root string, env Env) error {
				_, err := SetTodo(root, env, id, state)
				return err
			},
			elisp: `(edit-goto-id "` + file + `" "` + id + `") (edit-todo ` + arg + `)`,
		}
	}
	runEmacsCases(t, []emacsCase{
		todo("完了にするとタグを揃え直す", "tasks.org", "t-tag", "DONE"),
		todo("SCHEDULED の前に CLOSED を入れる", "tasks.org", "t-sched", "DONE"),
		todo("既存の LOGBOOK の先頭に足す", "tasks.org", "t-logged", "DONE"),
		todo("完了から戻すと CLOSED を消す", "tasks.org", "t-done", "TODO"),
		todo("完了から完了", "tasks.org", "t-done", "CANCELLED"),
		todo("未完了から未完了", "tasks.org", "t-tag", "WIP"),
		todo("キーワードを外す", "tasks.org", "t-done", ""),
		todo("次の見出しの手前に LOGBOOK を作る", "tasks.org", "t-child", "DONE"),
		todo("ファイル末尾の見出し", "tasks.org", "t-last", "DONE"),
		todo("記録の設定が無いファイル", "inbox.org", "c-old", "TODO"),
		todo("記録の設定が無いファイルで完了にする", "inbox.org", "c-new", "DONE"),
	})
}
