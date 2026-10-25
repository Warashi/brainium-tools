package orglint

import (
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func run(t *testing.T, files map[string]string) []Problem {
	t.Helper()
	fsys := fstest.MapFS{}
	for p, body := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(body)}
	}
	problems, err := Run(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return problems
}

func runTargets(t *testing.T, files map[string]string, targets ...string) []Problem {
	t.Helper()
	fsys := fstest.MapFS{}
	for p, body := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(body)}
	}
	problems, err := Run(fsys, targets...)
	if err != nil {
		t.Fatal(err)
	}
	return problems
}

func assertProblem(t *testing.T, problems []Problem, path, substr string) {
	t.Helper()
	for _, p := range problems {
		if p.Path == path && strings.Contains(p.Message, substr) {
			return
		}
	}
	t.Errorf("problem %q containing %q not found in %v", path, substr, problems)
}

const validNote = `:PROPERTIES:
:ID:       aaaaaaaa-0000-0000-0000-000000000001
:STATUS:   draft
:END:
#+title: Nix の flake
`

const validNote2 = `:PROPERTIES:
:ID:       aaaaaaaa-0000-0000-0000-000000000002
:STATUS:   draft
:END:
#+title: 別のノート
`

func TestRun(t *testing.T) {
	valid := map[string]string{
		"README.org": "#+title: brainium\n",
		"inbox.org": ":PROPERTIES:\n:ID: inbox-id\n:END:\n#+title: Inbox\n" +
			"* 試したいこと\n:PROPERTIES:\n:ID: capture-id\n:CAPTURED: [2026-07-28 Tue 20:48]\n:END:\n本文\n",
		"tasks.org":           ":PROPERTIES:\n:ID: task-id\n:END:\n#+title: Tasks\n",
		"notes/nix-flake.org": validNote + "\n[[id:task-id][タスク]]\n",
		"journal/2026/2026-07-19.org": ":PROPERTIES:\n:ID: journal-file-id\n:END:\n#+title: 2026-07-19\n" +
			"* メモ\n:PROPERTIES:\n:ID: journal-heading-id\n:END:\n" +
			"[[id:aaaaaaaa-0000-0000-0000-000000000001]]\n",
	}

	t.Run("規約を満たすリポジトリでは違反を報告しない", func(t *testing.T) {
		if got := run(t, valid); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("notes のファイルレベル ID の欠落を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"notes/no-id.org": "#+title: ID なし\n"})
		assertProblem(t, got, "notes/no-id.org", ":ID:")
	})

	t.Run("notes の title の欠落を報告する", func(t *testing.T) {
		got := run(t, map[string]string{
			"notes/no-title.org": ":PROPERTIES:\n:ID: x\n:END:\n",
		})
		assertProblem(t, got, "notes/no-title.org", "#+title:")
	})

	t.Run("notes の kebab-case でないファイル名を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"notes/Nix_Flake.org": validNote})
		assertProblem(t, got, "notes/Nix_Flake.org", "kebab-case")
	})

	t.Run("notes のサブディレクトリ配置を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"notes/sub/nested.org": validNote})
		assertProblem(t, got, "notes/sub/nested.org", "サブディレクトリ")
	})

	t.Run("journal のファイルレベル ID の欠落を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"journal/2026/2026-07-19.org": "#+title: ID なし\n"})
		assertProblem(t, got, "journal/2026/2026-07-19.org", ":ID:")
	})

	t.Run("journal の title の欠落を報告する", func(t *testing.T) {
		got := run(t, map[string]string{
			"journal/2026/2026-07-19.org": ":PROPERTIES:\n:ID: x\n:END:\n",
		})
		assertProblem(t, got, "journal/2026/2026-07-19.org", "#+title:")
	})

	t.Run("journal の年ディレクトリ欠落を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"journal/2026-07-19.org": "#+title: x\n"})
		assertProblem(t, got, "journal/2026-07-19.org", "形式")
	})

	t.Run("journal の年ディレクトリとファイル名の不一致を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"journal/2025/2026-07-19.org": "#+title: x\n"})
		assertProblem(t, got, "journal/2025/2026-07-19.org", "一致")
	})

	t.Run("journal の実在しない日付を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"journal/2026/2026-13-01.org": "#+title: x\n"})
		assertProblem(t, got, "journal/2026/2026-13-01.org", "実在する日付")
	})

	t.Run("inbox のファイルレベル ID の欠落を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"inbox.org": "#+title: Inbox\n"})
		assertProblem(t, got, "inbox.org", "ファイルレベルの :ID:")
	})

	t.Run("旧書式の capture では :ID: と :CAPTURED: の欠落を報告する", func(t *testing.T) {
		got := run(t, map[string]string{
			"inbox.org": ":PROPERTIES:\n:ID: inbox-id\n:END:\n#+title: Inbox\n" +
				"* 旧書式の capture\n[2026-07-25 Sat 06:37]\n\n本文\n",
		})
		assertProblem(t, got, "inbox.org", "capture の見出しに :ID:")
		assertProblem(t, got, "inbox.org", "capture の見出しに :CAPTURED:")
	})

	t.Run("capture の :CAPTURED: だけの欠落を報告する", func(t *testing.T) {
		got := run(t, map[string]string{
			"inbox.org": ":PROPERTIES:\n:ID: inbox-id\n:END:\n#+title: Inbox\n" +
				"* capture\n:PROPERTIES:\n:ID: capture-id\n:END:\n本文\n",
		})
		assertProblem(t, got, "inbox.org", "capture の見出しに :CAPTURED:")
	})

	t.Run("capture の見出しと離れた PROPERTIES は property とみなさない", func(t *testing.T) {
		got := run(t, map[string]string{
			"inbox.org": ":PROPERTIES:\n:ID: inbox-id\n:END:\n#+title: Inbox\n" +
				"* capture\n\n:PROPERTIES:\n:ID: capture-id\n:CAPTURED: [2026-07-28 Tue 20:48]\n:END:\n",
		})
		assertProblem(t, got, "inbox.org", "capture の見出しに :ID:")
	})

	t.Run("inbox 以外のルート直下ファイルの見出しは検査しない", func(t *testing.T) {
		got := run(t, map[string]string{
			"tasks.org": ":PROPERTIES:\n:ID: task-id\n:END:\n#+title: Tasks\n* Next\n** TODO やる\n",
		})
		if len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("README はファイルレベル ID を必須としない", func(t *testing.T) {
		got := run(t, map[string]string{"README.org": "#+title: brainium\n"})
		if len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("ルート直下の kebab-case でないファイル名を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"Someday.org": "#+title: x\n"})
		assertProblem(t, got, "Someday.org", "kebab-case")
	})

	t.Run("work/ 直下の kebab-case の org ファイルを許可する", func(t *testing.T) {
		got := run(t, map[string]string{
			"work/pr-triage.org": ":PROPERTIES:\n:ID: aaaaaaaa-0000-0000-0000-000000000002\n:END:\n#+title: x\n",
		})
		if len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("work/ のサブディレクトリを報告する", func(t *testing.T) {
		got := run(t, map[string]string{"work/sub/pr-triage.org": "#+title: x\n"})
		assertProblem(t, got, "work/sub/pr-triage.org", "サブディレクトリ")
	})

	t.Run("work/ の kebab-case でないファイル名を報告する", func(t *testing.T) {
		got := run(t, map[string]string{"work/PR_triage.org": "#+title: x\n"})
		assertProblem(t, got, "work/PR_triage.org", "kebab-case")
	})

	t.Run("置き場所でないディレクトリの org ファイルを報告する", func(t *testing.T) {
		got := run(t, map[string]string{"docs/stray.org": "#+title: x\n"})
		assertProblem(t, got, "docs/stray.org", "置く")
	})

	t.Run("ファイルをまたぐ ID の重複を両方のファイルで報告する", func(t *testing.T) {
		got := run(t, map[string]string{
			"notes/first.org":  ":PROPERTIES:\n:ID: dup\n:END:\n#+title: 一\n",
			"notes/second.org": ":PROPERTIES:\n:ID: dup\n:END:\n#+title: 二\n",
		})
		assertProblem(t, got, "notes/first.org", "重複")
		assertProblem(t, got, "notes/second.org", "重複")
	})

	t.Run("存在しない ID へのリンクを報告する", func(t *testing.T) {
		got := run(t, map[string]string{
			"notes/broken.org": validNote + "\n[[id:missing-id][どこにもない]]\n",
		})
		assertProblem(t, got, "notes/broken.org", "missing-id")
	})

	t.Run("コード表記で書いた id リンクは実リンクとみなさない", func(t *testing.T) {
		got := run(t, map[string]string{
			"notes/verbatim.org": validNote + "\n=[[id:placeholder]]= と ~[[id:another]]~ は記法の例示\n",
		})
		if len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("組まれないコード表記の中の id リンクは実リンクとみなす", func(t *testing.T) {
		got := run(t, map[string]string{
			"notes/dangling.org": validNote + "\nx=[[id:missing-id]]=y\n",
		})
		assertProblem(t, got, "notes/dangling.org", "missing-id")
	})

	t.Run("ソースブロック内の id リンクと :ID: は検査しない", func(t *testing.T) {
		got := run(t, map[string]string{
			"notes/block.org": validNote +
				"\n#+begin_src org\n:PROPERTIES:\n:ID: aaaaaaaa-0000-0000-0000-000000000001\n:END:\n[[id:missing]]\n#+end_src\n",
		})
		if len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("tools と .github 配下の org ファイルは検査しない", func(t *testing.T) {
		got := run(t, map[string]string{
			"tools/testdata/fixture.org": "invalid placement\n",
			".github/stray.org":          "invalid placement\n",
		})
		if len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})
}

// org-archive-subtree は済んだ task を tasks.org_archive へ移す。org-node
// が索引に使う org-mem は既定の org-mem-suffixes に ".org_archive" を含む
// ので、Emacs では archive 内の ID を指すリンクが生きている。orglint も
// ID の供給元としてだけ archive を読み、検査はしない。
func TestRunArchive(t *testing.T) {
	const archived = "* DONE 済んだ task\n:PROPERTIES:\n:ID: archived-id\n:END:\n"

	t.Run("archive 内の ID を指すリンクを違反としない", func(t *testing.T) {
		got := run(t, map[string]string{
			"notes/nix-flake.org": validNote + "\n[[id:archived-id][済んだ task]]\n",
			"tasks.org_archive":   archived,
		})
		if len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("archive 自体は配置もメタデータも折り返しも検査しない", func(t *testing.T) {
		got := run(t, map[string]string{
			"tasks.org_archive": archived + strings.Repeat("あ", 80) + "\n",
		})
		if len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("archive から出る宙吊りリンクは報告しない", func(t *testing.T) {
		got := run(t, map[string]string{
			"tasks.org_archive": archived + "[[id:missing-id][もう無い]]\n",
		})
		if len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("archive と重複する ID を .org 側で報告する", func(t *testing.T) {
		got := run(t, map[string]string{
			"notes/dup.org":     ":PROPERTIES:\n:ID: archived-id\n:END:\n#+title: 重複\n",
			"tasks.org_archive": archived,
		})
		assertProblem(t, got, "notes/dup.org", "重複")
	})

	t.Run("OrgFiles は archive を返さない", func(t *testing.T) {
		fsys := fstest.MapFS{
			"notes/nix-flake.org": &fstest.MapFile{Data: []byte(validNote)},
			"tasks.org_archive":   &fstest.MapFile{Data: []byte(archived)},
		}
		paths, err := OrgFiles(fsys)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(paths, []string{"notes/nix-flake.org"}) {
			t.Errorf("OrgFiles = %v, want only notes/nix-flake.org", paths)
		}
	})
}

// Emacs は編集中のファイルの隣に =.#foo.org= という、実体のないパスを指
// す symlink を置く。拡張子だけで拾うと読み込みに失敗して orglint 全体が
// 落ちるので、通常ファイルでないものは検査対象から外れる。
func TestOrgFilesSkipsNonRegular(t *testing.T) {
	fsys := fstest.MapFS{
		"notes/nix-flake.org":   &fstest.MapFile{Data: []byte(validNote)},
		"notes/.#nix-flake.org": &fstest.MapFile{Data: []byte("warashi@host.1234:1234"), Mode: fs.ModeSymlink},
	}
	paths, err := OrgFiles(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(paths, []string{"notes/nix-flake.org"}) {
		t.Errorf("OrgFiles = %v, want only notes/nix-flake.org", paths)
	}

	if _, err := Run(fsys); err != nil {
		t.Errorf("Run with an Emacs lock file: %v", err)
	}
}

func TestParseOrg(t *testing.T) {
	t.Run("最初の見出しより前の :ID: をファイルレベル ID とする", func(t *testing.T) {
		f := parseOrg("a.org", []byte(":PROPERTIES:\n:ID: file-id\n:END:\n* 見出し\n:PROPERTIES:\n:ID: heading-id\n:END:\n"))
		if f.fileID != "file-id" {
			t.Errorf("fileID = %q, want %q", f.fileID, "file-id")
		}
		if len(f.ids) != 2 {
			t.Errorf("ids = %v, want 2 ids", f.ids)
		}
	})

	t.Run("見出し配下しか :ID: がなければファイルレベル ID なしとする", func(t *testing.T) {
		f := parseOrg("a.org", []byte("* 見出し\n:PROPERTIES:\n:ID: heading-id\n:END:\n"))
		if f.fileID != "" {
			t.Errorf("fileID = %q, want empty", f.fileID)
		}
	})

	t.Run("説明付きと説明なし両方の id: リンクを抽出する", func(t *testing.T) {
		f := parseOrg("a.org", []byte("[[id:one][説明]] と [[id:two]]\n"))
		if len(f.links) != 2 || f.links[0] != "one" || f.links[1] != "two" {
			t.Errorf("links = %v, want [one two]", f.links)
		}
	})

	t.Run("#+TITLE: の大文字小文字を区別しない", func(t *testing.T) {
		f := parseOrg("a.org", []byte("#+TITLE: 大文字\n"))
		if f.title != "大文字" {
			t.Errorf("title = %q, want %q", f.title, "大文字")
		}
	})

	t.Run("固定幅行の id: リンクは実体として扱わない", func(t *testing.T) {
		f := parseOrg("a.org", []byte(": 例: [[id:one][説明]]\n"))
		if len(f.links) != 0 {
			t.Errorf("links = %v, want empty", f.links)
		}
	})

	t.Run("ドロワーの外の :ID: 行はファイルレベル ID とみなさない", func(t *testing.T) {
		f := parseOrg("a.org", []byte(":ID: 例として書いた行\n"))
		if f.fileID != "" || len(f.ids) != 0 {
			t.Errorf("fileID = %q, ids = %v, want empty", f.fileID, f.ids)
		}
	})
}

func TestCheckFormat(t *testing.T) {
	// 全角 1 文字 2 桁なので、あ 35 個で 70 桁ちょうどになる。
	const full70 = "あああああああああああああああああああああああああああああああああああ"

	widthProblems := func(t *testing.T, body string) []Problem {
		t.Helper()
		return checkFormat(parseOrg("notes/a.org", []byte(body)))
	}

	t.Run("70 桁ちょうどの行は報告しない", func(t *testing.T) {
		if got := widthProblems(t, full70+"\n"); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("70 桁を超える行を段落の先頭の行番号で報告する", func(t *testing.T) {
		got := widthProblems(t, "\n"+full70+"あい\n")
		if len(got) != 1 {
			t.Fatalf("want 1 problem, got %v", got)
		}
		if got[0].Line != 2 {
			t.Errorf("Line = %d, want 2", got[0].Line)
		}
	})

	t.Run("次の行の語を繰り上げられる段落を報告する", func(t *testing.T) {
		// どの行も 70 桁に収まるが、2 行目の語は 1 行目に載る。
		got := widthProblems(t, "あいうえお\nかきくけこ\n")
		if len(got) != 1 {
			t.Fatalf("want 1 problem, got %v", got)
		}
		if got[0].Line != 1 {
			t.Errorf("Line = %d, want 1", got[0].Line)
		}
	})

	t.Run("段落を跨いで繰り上げはしない", func(t *testing.T) {
		body := "あいうえお。\n\n- かきくけこ\n- さしすせそ\n"
		if got := widthProblems(t, body); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("行頭に置けない文字が 1 文字はみ出しただけの行は報告しない", func(t *testing.T) {
		for _, tail := range []string{"。", ")", "、"} {
			if got := widthProblems(t, full70+tail+"\n"); len(got) != 0 {
				t.Errorf("tail %q: want no problems, got %v", tail, got)
			}
		}
	})

	t.Run("行末を伸ばしきれない禁則は手前で折り返せるので報告する", func(t *testing.T) {
		// 「ャ」1 文字なら 72 桁まで伸ばして飲み込めるが、「ャッ」で
		// 74 桁になると伸ばせず、手前の折り返し点まで戻ることになる。
		if got := widthProblems(t, full70+"ャ\n"); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
		if got := widthProblems(t, full70+"ャッ\n"); len(got) != 1 {
			t.Errorf("want 1 problem, got %v", got)
		}
	})

	t.Run("折り返せない長い URL の行は報告しない", func(t *testing.T) {
		body := "https://example.com/" + strings.Repeat("a", 80) + "\n"
		if got := widthProblems(t, body); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("URL の前に折り返せる本文があれば報告する", func(t *testing.T) {
		body := full70 + " https://example.com/" + strings.Repeat("a", 80) + "\n"
		if got := widthProblems(t, body); len(got) != 1 {
			t.Errorf("want 1 problem, got %v", got)
		}
	})

	t.Run("折り返せない長い英単語の行は報告しない", func(t *testing.T) {
		if got := widthProblems(t, strings.Repeat("a", 80)+"\n"); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("見出し・キーワード行・表・ドロワー・ブロックは報告しない", func(t *testing.T) {
		body := "* " + full70 + "あ\n" +
			"#+caption: " + full70 + "あ\n" +
			"| " + full70 + "あ |\n" +
			":PROPERTIES:\n:CUSTOM: " + full70 + "あ\n:END:\n" +
			"#+begin_src go\n// " + full70 + "あ\n#+end_src\n"
		if got := widthProblems(t, body); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	// linkPath は 39 桁、linkDesc 付きのリンクは 10 桁として表示される。
	const linkPath = "[[id:0f1e2d3c-4b5a-4968-8778-a6b5c4d3e2f1]]"
	const linkDesc = "[[id:0f1e2d3c-4b5a-4968-8778-a6b5c4d3e2f1][org の行幅]]"

	t.Run("id リンクは説明文の幅だけを数える", func(t *testing.T) {
		if got := widthProblems(t, linkDesc+strings.Repeat("あ", 30)+"\n"); len(got) != 0 {
			t.Errorf("70 桁ちょうど: want no problems, got %v", got)
		}
		if got := widthProblems(t, linkDesc+strings.Repeat("あ", 32)+"\n"); len(got) != 1 {
			t.Errorf("74 桁: want 1 problem, got %v", got)
		}
	})

	t.Run("説明のない id リンクはパスの幅を数える", func(t *testing.T) {
		if got := widthProblems(t, linkPath+strings.Repeat("あ", 15)+"\n"); len(got) != 0 {
			t.Errorf("69 桁: want no problems, got %v", got)
		}
		if got := widthProblems(t, linkPath+strings.Repeat("あ", 18)+"\n"); len(got) != 1 {
			t.Errorf("75 桁: want 1 problem, got %v", got)
		}
	})

	t.Run("コード表記の中の id リンクは記法をそのまま数える", func(t *testing.T) {
		// org はコード表記の中のリンクを組み直さないので記法が見える。
		// 記法を隠すと 66 桁で、そのまま数えると 76 桁になる。
		body := "あ =[[id:x][説明]]= " + strings.Repeat("あ", 28) + "\n"
		if got := widthProblems(t, body); len(got) != 1 {
			t.Errorf("want 1 problem, got %v", got)
		}
	})

	t.Run("折り返しで割れたリンクの ]] も隠れているものとして数える", func(t *testing.T) {
		// Emacs は段落を 1 行につないだ状態で不可視属性を付けてから
		// 折り返すので、2 行に割れたリンクの記法も隠れたままになる。
		if got := widthProblems(t, "  行幅]]"+strings.Repeat("あ", 32)+"\n"); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("リンクの直前の空白で折り返した継続行は字下げを数える", func(t *testing.T) {
		link := "[[id:0f1e2d3c-4b5a-4968-8778-a6b5c4d3e2f1][行幅]]"
		stable := "- " + strings.Repeat("あ", 34) + "\n  " + link + strings.Repeat("あ", 32) + "\n  あ\n"
		if got := widthProblems(t, stable); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
		overflow := "- " + strings.Repeat("あ", 34) + "\n  " + link + strings.Repeat("あ", 33) + "\n"
		if got := widthProblems(t, overflow); len(got) != 1 {
			t.Errorf("want 1 problem, got %v", got)
		}
	})
}

func TestCheckEmphasis(t *testing.T) {
	problems := func(t *testing.T, body string) []Problem {
		t.Helper()
		return checkEmphasis(parseOrg("notes/a.org", []byte(body)))
	}

	t.Run("前後の文字が条件を満たさないコード表記を報告する", func(t *testing.T) {
		got := problems(t, "* 見出し\n書式は=aaa=b のようになる\n")
		if len(got) != 1 {
			t.Fatalf("want 1 problem, got %v", got)
		}
		if got[0].Line != 2 {
			t.Errorf("Line = %d, want 2", got[0].Line)
		}
		if !strings.Contains(got[0].Message, "=aaa=") {
			t.Errorf("Message = %q, want to contain %q", got[0].Message, "=aaa=")
		}
	})

	t.Run("和文の句読点の直後も報告する", func(t *testing.T) {
		got := problems(t, "書式は、=aaa= のようになる。\n")
		if len(got) != 1 {
			t.Fatalf("want 1 problem, got %v", got)
		}
	})

	t.Run("空白で区切ったコード表記は報告しない", func(t *testing.T) {
		if got := problems(t, "書式は、 =aaa= のようになる。 =bbb= も同じ。\n"); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("中に区切り記号を含むコード表記は報告しない", func(t *testing.T) {
		if got := problems(t, "ルートへコピーし =--opt=val-<name>= で外から渡す。\n"); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("ブロック・ドロワー・キーワード行は報告しない", func(t *testing.T) {
		body := "#+title: x=aaa=b\n:PROPERTIES:\n:ID: x=aaa=b\n:END:\n" +
			"#+begin_src sh\nx=aaa=b\n#+end_src\n# x=aaa=b\n"
		if got := problems(t, body); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("固定幅行は報告しない", func(t *testing.T) {
		if got := problems(t, "* 見出し\n: $ x=aaa=b\n:\n"); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})
}

func TestEmphasisSpans(t *testing.T) {
	texts := func(line string, spans [][]int) []string {
		out := []string{}
		for _, s := range spans {
			out = append(out, line[s[0]:s[1]])
		}
		return out
	}
	check := func(t *testing.T, line string, wantTypeset, wantDangling []string) {
		t.Helper()
		typeset, dangling := emphasisSpans(line)
		if got := texts(line, typeset); !slices.Equal(got, wantTypeset) {
			t.Errorf("typeset = %q, want %q", got, wantTypeset)
		}
		if got := texts(line, dangling); !slices.Equal(got, wantDangling) {
			t.Errorf("dangling = %q, want %q", got, wantDangling)
		}
	}

	t.Run("行頭と行末に接するものは組まれる", func(t *testing.T) {
		check(t, "=a=", []string{"=a="}, nil)
	})

	t.Run("空白で区切られた候補が続いても後ろを取りこぼさない", func(t *testing.T) {
		check(t, "=a= =b= ~c~", []string{"=a=", "=b=", "~c~"}, nil)
	})

	t.Run("前が半角記号だと組まれない", func(t *testing.T) {
		check(t, "HTTP/JSON/=os/exec= だけ", nil, []string{"=os/exec="})
	})

	t.Run("後ろが半角の英数だと組まれない", func(t *testing.T) {
		check(t, "=a=b", nil, []string{"=a="})
	})

	t.Run("前後が和文だと組まれない", func(t *testing.T) {
		check(t, "例えば=aaa=は組まれない。", nil, []string{"=aaa="})
		check(t, "実装した。=orglint= を通す", nil, []string{"=orglint="})
	})

	t.Run("和文でも空白で区切れば組まれる", func(t *testing.T) {
		check(t, "実装した。 =orglint= を通す", []string{"=orglint="}, nil)
	})

	// Emacs の org-verbatim-re は中身を非貪欲に取り、後ろの文字が条件を
	// 満たす最初の閉じ記号で閉じる。中身に区切り記号が入っても構わない。
	t.Run("中に区切り記号を含んでも後ろの条件を満たす閉じ記号まで組まれる", func(t *testing.T) {
		check(t, "コピーし =--opt=val-<name>= で渡す", []string{"=--opt=val-<name>="}, nil)
		check(t, "=a=b =c= d", []string{"=a=b =c="}, nil)
		check(t, "~a=b~ と =a~b= は別", []string{"~a=b~", "=a~b="}, nil)
	})

	t.Run("前の条件を外れた開き記号は次の候補を邪魔しない", func(t *testing.T) {
		check(t, "x=a =b=", []string{"=b="}, nil)
	})

	t.Run("閉じ記号の直前が空白なら閉じない", func(t *testing.T) {
		check(t, "=a = b", nil, nil)
	})
}

func TestDisplayWidth(t *testing.T) {
	for _, tt := range []struct {
		text string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"あ", 2},
		{"ｱ", 1},
		{"Ａ", 2},
		{"漢字 mixed", 4 + 1 + 5},
		{"　", 2},
		// East Asian Ambiguous は Emacs の既定と同じく 1 桁。
		{"※①→…℃", 5},
		{"±×─■⇒Ⅰ‐—“”", 10},
		{"😀", 2},
		// Ambiguous の → を挟み、和欧が混ざって 70 桁ちょうどになる行。
		{"draftに書く → Claude で verify する → ひととおり読んで Claude で清書す", 70},
	} {
		if got := displayWidth(tt.text); got != tt.want {
			t.Errorf("displayWidth(%q) = %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestTargets(t *testing.T) {
	files := map[string]string{
		"tasks.org":                   ":PROPERTIES:\n:ID: task-id\n:END:\n#+title: Tasks\n",
		"notes/nix-flake.org":         validNote + "\n[[id:task-id][タスク]]\n",
		"notes/Bad_Name.org":          validNote2,
		"journal/2026/2026-07-19.org": "#+title: ID なし\n",
	}

	t.Run("対象を指定すると指定ファイルの違反だけを返す", func(t *testing.T) {
		got := runTargets(t, files, "notes/Bad_Name.org")
		assertProblem(t, got, "notes/Bad_Name.org", "kebab-case")
		for _, p := range got {
			if p.Path != "notes/Bad_Name.org" {
				t.Errorf("対象外の違反を返した: %v", p)
			}
		}
	})

	t.Run("対象外のファイルが持つ ID へのリンクを誤検出しない", func(t *testing.T) {
		if got := runTargets(t, files, "notes/nix-flake.org"); len(got) != 0 {
			t.Errorf("want no problems, got %v", got)
		}
	})

	t.Run("ディレクトリを指定すると配下のファイルを対象にする", func(t *testing.T) {
		got := runTargets(t, files, "notes")
		if len(got) == 0 {
			t.Fatal("notes 配下の違反を返さなかった")
		}
		for _, p := range got {
			if !strings.HasPrefix(p.Path, "notes/") {
				t.Errorf("notes 配下でない違反を返した: %v", p)
			}
		}
	})

	t.Run("対象を指定しなければ全ファイルを検査する", func(t *testing.T) {
		got := runTargets(t, files)
		assertProblem(t, got, "journal/2026/2026-07-19.org", ":ID:")
		assertProblem(t, got, "notes/Bad_Name.org", "kebab-case")
	})
}

func TestTargetsUnmatched(t *testing.T) {
	paths := []string{"tasks.org", "notes/nix-flake.org"}

	t.Run("どのファイルにも一致しない対象を返す", func(t *testing.T) {
		got := Targets{"notes/typo.org", "tools"}.Unmatched(paths)
		want := []string{"notes/typo.org", "tools"}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("一致する対象は返さない", func(t *testing.T) {
		if got := (Targets{"notes", "tasks.org"}).Unmatched(paths); len(got) != 0 {
			t.Errorf("got %v, want none", got)
		}
	})
}
