package orgedit

import "testing"

func TestArchiveMatchesEmacs(t *testing.T) {
	const archived = "#+todo: TODO | DONE CANCELLED\n\n* DONE 前に archive した task\nCLOSED: [2026-09-01 Tue 18:47]\n:PROPERTIES:\n:ARCHIVE_TIME: 2026-09-01 Tue 18:47\n:END:\n\n本文。\n"
	archive := func(name string, files map[string]string, file, id string) emacsCase {
		return emacsCase{
			name:  name,
			files: files,
			run: func(root string, env Env) error {
				_, err := Archive(root, env, id)
				return err
			},
			elisp: `(edit-archive "` + file + `" "` + id + `")`,
		}
	}
	withArchive := map[string]string{"tasks.org": tasksOrg, "tasks.org_archive": archived}
	runEmacsCases(t, []emacsCase{
		archive("完了した task", withArchive, "tasks.org", "t-done"),
		archive("タグを揃え直す", withArchive, "tasks.org", "t-tag"),
		archive("祖先のタグと見出しを残す", withArchive, "tasks.org", "t-child"),
		archive("子を持つ task", withArchive, "tasks.org", "t-logged"),
		archive("ファイル末尾の task", withArchive, "tasks.org", "t-last"),
		archive("archive のファイルが無い", map[string]string{"tasks.org": tasksOrg}, "tasks.org", "t-done"),
		archive("CATEGORY の無いファイル", map[string]string{"inbox.org": inboxOrg}, "inbox.org", "c-new"),
	})
}

func TestArchiveFileRoot(t *testing.T) {
	env := Env{Home: "/home/u", ArchiveFileRoot: "~/ghq/example/repo/"}
	if got := archiveFileName("/srv/repo", env, "tasks.org"); got != "~/ghq/example/repo/tasks.org" {
		t.Errorf("got %s", got)
	}
	env.ArchiveFileRoot = ""
	if got := archiveFileName("/home/u/repo", env, "tasks.org"); got != "~/repo/tasks.org" {
		t.Errorf("got %s", got)
	}
	if got := archiveFileName("/srv/repo", env, "tasks.org"); got != "/srv/repo/tasks.org" {
		t.Errorf("got %s", got)
	}
}
