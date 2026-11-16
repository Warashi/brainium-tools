package orgedit

import (
	"strconv"
	"testing"
)

func TestCaptureMatchesEmacs(t *testing.T) {
	capture := func(name, inbox, title, body string) emacsCase {
		return emacsCase{
			name:  name,
			files: map[string]string{"inbox.org": inbox},
			run: func(root string, env Env) error {
				_, _, err := Capture(root, env, title, body)
				return err
			},
			elisp: `(edit-capture ` + strconv.Quote(title) + ` ` + strconv.Quote(body) + `)`,
		}
	}
	const header = ":PROPERTIES:\n:ID:       file-inbox\n:END:\n#+title: Inbox\n\n# org-capture の投入先。\n"
	runEmacsCases(t, []emacsCase{
		capture("先頭に足す", inboxOrg, "作業の記録", "[[id:t-tag][task]] の一手。\n\n- 箇条 1\n- 箇条 2"),
		capture("本文なし", inboxOrg, "本文の無い capture", ""),
		capture("子の見出し", inboxOrg, "レビュー", "対象: [[id:t-tag][task]]\n\n** Pros\n- 良い\n** Cons\n- 悪い"),
		capture("空の inbox", header, "最初の capture", "本文。"),
		capture("改行で終わらない空の inbox", header[:len(header)-1], "最初の capture", "本文。"),
		capture("空行で終わる空の inbox", header+"\n", "最初の capture", "本文。"),
	})
}

func TestCaptureFillsTheBody(t *testing.T) {
	home := t.TempDir()
	root := writeRepo(t, home, map[string]string{"inbox.org": inboxOrg})
	body := "長い本文は orglint と同じ幅で折り返す。長い本文は orglint と同じ幅で折り返す。長い本文は orglint と同じ幅で折り返す。"
	id, _, err := Capture(root, testEnv(home), "折り返し", body)
	if err != nil {
		t.Fatal(err)
	}
	if id != testIDs[0] {
		t.Errorf("id = %s", id)
	}
	got := readRepo(t, root)["inbox.org"]
	want := "* 折り返し\n:PROPERTIES:\n:ID:       " + testIDs[0] + "\n:CAPTURED: [2026-11-16 Mon 21:08]\n:END:\n\n" +
		"長い本文は orglint と同じ幅で折り返す。長い本文は orglint と同じ幅で折\nり返す。長い本文は orglint と同じ幅で折り返す。\n\n"
	if !contains(got, want) {
		t.Errorf("inbox.org:\n%s", got)
	}
}

func TestCaptureRejectsTopLevelHeadingsInTheBody(t *testing.T) {
	home := t.TempDir()
	root := writeRepo(t, home, map[string]string{"inbox.org": inboxOrg})
	if _, _, err := Capture(root, testEnv(home), "見出し", "本文\n* 見出しの行"); err == nil {
		t.Error("本文の深さ 1 の見出しを受け付けた")
	}
}
