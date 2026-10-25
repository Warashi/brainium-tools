// Package orglint は brainium 形式の org リポジトリが README.md に書いた規約を
// 満たしているかを検査する。
package orglint

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

// fillColumn は本文を折り返す桁数。Emacs の fill-column に合わせる。
const fillColumn = 70

// Problem は検出した規約違反 1 件を表す。Line が 0 の違反は行を
// 特定しない。
type Problem struct {
	Path    string
	Line    int
	Message string
}

func (p Problem) String() string {
	if p.Line == 0 {
		return p.Path + ": " + p.Message
	}
	return fmt.Sprintf("%s:%d: %s", p.Path, p.Line, p.Message)
}

// skipDirs はリポジトリ直下の org ノート置き場ではないディレクトリ。
var skipDirs = map[string]bool{
	".git":     true,
	".github":  true,
	".warashi": true,
	"tools":    true,
}

type orgFile struct {
	path   string
	title  string
	fileID string
	ids    []string
	links  []string
	lines  []string
	kinds  []lineKind
}

// verbatimPat はインラインのコード表記 =…= と ~…~ の候補にマッチする。
// 区切り記号の内側が空白で始まったり終わったりしないことを求めて、
// 文中の = や ~ 単体との取り違えを避ける。実際に組まれる範囲は前後の
// 文字で決まり、中身に区切り記号を含むこともあるので、これは報告に使う
// 最短の候補であって組まれる範囲ではない。組まれる範囲は emphasisSpans
// が決める。
const verbatimPat = `=[^=\s](?:[^=]*[^=\s])?=|~[^~\s](?:[^~]*[^~\s])?~`

var (
	titleRe       = regexp.MustCompile(`(?i)^#\+title:\s*(\S.*?)\s*$`)
	idPropRe      = regexp.MustCompile(`(?i)^\s*:ID:\s*(\S+)\s*$`)
	capturedRe    = regexp.MustCompile(`(?i)^\s*:CAPTURED:\s*\S`)
	propBeginRe   = regexp.MustCompile(`(?i)^\s*:PROPERTIES:\s*$`)
	headingRe     = regexp.MustCompile(`^\*+ `)
	topHeadingRe  = regexp.MustCompile(`^\* `)
	idLinkRe      = regexp.MustCompile(`\[\[id:([^\]]+)\]`)
	slugRe        = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*\.org$`)
	journalRe     = regexp.MustCompile(`^journal/(\d{4})/((\d{4})-\d{2}-\d{2})\.org$`)
	blockBeginRe  = regexp.MustCompile(`(?i)^\s*#\+begin_(\S+)`)
	blockEndRe    = regexp.MustCompile(`(?i)^\s*#\+end_(\S+)`)
	drawerBeginRe = regexp.MustCompile(`^\s*:[\w-]+:\s*$`)
	drawerEndRe   = regexp.MustCompile(`(?i)^\s*:END:\s*$`)
	verbatimRe    = regexp.MustCompile(verbatimPat)
)

// OrgFiles は fsys 配下の検査対象になる org ファイルのパスを返す。
func OrgFiles(fsys fs.FS) ([]string, error) {
	return collect(fsys, ".org")
}

// ArchiveFiles は fsys 配下の archive ファイルのパスを返す。org-archive-subtree
// が済んだ task を移す先で、org-agenda も orglint の検査も対象にしない。
// ただし org-node が索引に使う org-mem は既定の org-mem-suffixes に
// ".org_archive" を含み、Emacs では archive 内の ID を指すリンクが生きて
// いる。orglint でだけ宙吊りにしないよう、ID の供給元としては読む。
func ArchiveFiles(fsys fs.FS) ([]string, error) {
	return collect(fsys, ".org_archive")
}

func collect(fsys fs.FS, ext string) ([]string, error) {
	var paths []string
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[p] {
				return fs.SkipDir
			}
			return nil
		}
		// 拡張子だけで拾うと、Emacs が編集中に置くロックファイル
		// (実体のないパスを指す symlink) も対象に入り、読み込みに失敗
		// して orglint 全体が落ちる。名前の規約は Emacs 側の実装詳細な
		// ので、読める実体があるかどうかで切る。
		if path.Ext(p) == ext && d.Type().IsRegular() {
			paths = append(paths, p)
		}
		return nil
	})
	return paths, err
}

// Run は fsys をリポジトリルートとして全 org ファイルを読み、targets に
// 含まれるファイルの違反を返す。ID の重複やリンク先の存在は全ファイルを
// 見ないと判定できないので、読み込みは常に全件で行い、報告だけを絞る。
func Run(fsys fs.FS, targets ...string) ([]Problem, error) {
	files, err := readAll(fsys, OrgFiles)
	if err != nil {
		return nil, err
	}
	archives, err := readAll(fsys, ArchiveFiles)
	if err != nil {
		return nil, err
	}

	var problems []Problem
	for _, f := range files {
		problems = append(problems, checkPlacement(f)...)
		problems = append(problems, checkMetadata(f)...)
		problems = append(problems, checkMaturity(f)...)
		problems = append(problems, checkInbox(f)...)
		problems = append(problems, checkTaskLength(f)...)
		problems = append(problems, checkFormat(f)...)
		problems = append(problems, checkEmphasis(f)...)
	}
	problems = append(problems, checkIDs(files, archives)...)
	dirs, err := checkFlatDirs(fsys)
	if err != nil {
		return nil, err
	}
	problems = append(problems, dirs...)

	kept := problems[:0]
	filter := Targets(targets)
	for _, p := range problems {
		if filter.match(p.Path) {
			kept = append(kept, p)
		}
	}
	problems = kept

	sort.Slice(problems, func(i, j int) bool {
		if problems[i].Path != problems[j].Path {
			return problems[i].Path < problems[j].Path
		}
		if problems[i].Line != problems[j].Line {
			return problems[i].Line < problems[j].Line
		}
		return problems[i].Message < problems[j].Message
	})
	return problems, nil
}

// lineKind は行の種類。ブロックとドロワーの内側、および固定幅行は検査も
// 整形もしない。
type lineKind int

const (
	kindOther  lineKind = iota // 空行・見出し・キーワード行・表・コメント
	kindBody                   // 折り返しの対象になる本文行
	kindDrawer                 // ドロワーの内側
	kindBlock                  // ブロックの内側と境界行
	kindFixed                  // 固定幅行(: で始まる行)
)

// fixedWidthRe は org の固定幅行。": " で始まる行と ":" だけの行が該当
// する。ドロワーの :NAME: はコロンの直後に空白を置かないので当たらない。
var fixedWidthRe = regexp.MustCompile(`^[ \t]*:(?: .*)?$`)

// scanLines は各行の種類を判定する。ブロック(#+begin_ 〜 #+end_)と
// ドロワー(:NAME: 〜 :END:)は開始行から終了行まで状態を持ち越す。固定幅
// 行は開始・終了の記法を持たないので行ごとに判定し、連なりが結果として
// 1 つのかたまりになる。org の厳密なパースは持ち込まない。
func scanLines(lines []string) []lineKind {
	kinds := make([]lineKind, len(lines))
	openBlock := ""
	inDrawer := false
	for i, line := range lines {
		switch {
		case openBlock != "":
			kinds[i] = kindBlock
			if m := blockEndRe.FindStringSubmatch(line); m != nil && strings.EqualFold(m[1], openBlock) {
				openBlock = ""
			}
		case blockBeginRe.MatchString(line):
			kinds[i] = kindBlock
			openBlock = blockBeginRe.FindStringSubmatch(line)[1]
		case inDrawer:
			kinds[i] = kindDrawer
			if drawerEndRe.MatchString(line) {
				inDrawer = false
			}
		case drawerBeginRe.MatchString(line) && !drawerEndRe.MatchString(line):
			kinds[i] = kindDrawer
			inDrawer = true
		case fixedWidthRe.MatchString(line):
			kinds[i] = kindFixed
		case isBodyLine(line):
			kinds[i] = kindBody
		default:
			kinds[i] = kindOther
		}
	}
	return kinds
}

// isBodyLine は折り返しの対象になる行かを返す。見出し・=#+= で始まる
// キーワード行・コメント・表は org の fill でも折り返さない。
func isBodyLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	switch {
	case trimmed == "":
		return false
	case headingRe.MatchString(line):
		return false
	case strings.HasPrefix(trimmed, "#"):
		return false
	case strings.HasPrefix(trimmed, "|"):
		return false
	default:
		return true
	}
}

// parseOrg は行単位の走査で ID・タイトル・id: リンクを抜き出し、整形の
// 検査に使う行と行種を持たせる。インラインのコード表記はリンクを拾う
// 前に取り除く。
func parseOrg(p string, data []byte) orgFile {
	lines := strings.Split(string(data), "\n")
	kinds := scanLines(lines)
	f := orgFile{path: p, lines: lines, kinds: kinds}
	seenHeading := false
	for i, line := range lines {
		switch kinds[i] {
		case kindBlock, kindFixed:
			continue
		case kindDrawer:
			if m := idPropRe.FindStringSubmatch(line); m != nil {
				if !seenHeading && f.fileID == "" {
					f.fileID = m[1]
				}
				f.ids = append(f.ids, m[1])
			}
			continue
		}

		if headingRe.MatchString(line) {
			seenHeading = true
		}
		if m := titleRe.FindStringSubmatch(line); m != nil && f.title == "" {
			f.title = m[1]
		}
		for _, m := range idLinkRe.FindAllStringSubmatch(stripVerbatim(line), -1) {
			f.links = append(f.links, m[1])
		}
	}
	return f
}

// stripVerbatim はコード表記として組まれる範囲を空白 1 個に置き換える。
// リンクや ID の記法をプレースホルダとしてコード表記で書いたものを
// 実体と取り違えないようにする。前後の文字の条件を満たさず組まれない
// ものは残す。Emacs がリンクとして扱うものは orglint も検査する。
func stripVerbatim(line string) string {
	spans, _ := emphasisSpans(line)
	var b strings.Builder
	prev := 0
	for _, s := range spans {
		b.WriteString(line[prev:s[0]])
		b.WriteString(" ")
		prev = s[1]
	}
	b.WriteString(line[prev:])
	return b.String()
}

func checkPlacement(f orgFile) []Problem {
	dir, base := path.Split(f.path)
	switch {
	case dir == "":
		if base == "README.org" || slugRe.MatchString(base) {
			return nil
		}
		return []Problem{{Path: f.path, Message: "ルート直下の org ファイル名は kebab-case にする"}}
	case strings.HasPrefix(f.path, "notes/"), strings.HasPrefix(f.path, "work/"):
		top, _, _ := strings.Cut(f.path, "/")
		if dir != top+"/" {
			return []Problem{{Path: f.path, Message: top + "/ にサブディレクトリは作らない"}}
		}
		if !slugRe.MatchString(base) {
			return []Problem{{Path: f.path, Message: top + "/ のファイル名は英語 kebab-case スラグにする"}}
		}
		return nil
	case strings.HasPrefix(f.path, "journal/"):
		m := journalRe.FindStringSubmatch(f.path)
		if m == nil {
			return []Problem{{Path: f.path, Message: "journal/<YYYY>/<YYYY-MM-DD>.org の形式にする"}}
		}
		if m[1] != m[3] {
			return []Problem{{Path: f.path, Message: "journal の年ディレクトリとファイル名の年を一致させる"}}
		}
		if _, err := time.Parse("2006-01-02", m[2]); err != nil {
			return []Problem{{Path: f.path, Message: "journal のファイル名が実在する日付ではない"}}
		}
		return nil
	default:
		return []Problem{{Path: f.path, Message: "org ファイルは notes/ か journal/ か work/ かルート直下に置く"}}
	}
}

// checkMetadata はファイルレベルの :ID: と #+title: を検査する。
// README.org はリポジトリの説明であってノートではないので、ID を
// 振ってリンク先にする対象から外す。
func checkMetadata(f orgFile) []Problem {
	if f.path == "README.org" {
		return nil
	}
	var problems []Problem
	if f.fileID == "" {
		problems = append(problems, Problem{Path: f.path, Message: "ファイルレベルの :ID: を必須とする"})
	}
	if f.title == "" {
		problems = append(problems, Problem{Path: f.path, Message: "#+title: を必須とする"})
	}
	return problems
}

// checkInbox は inbox.org のトップレベル見出しが org-capture テンプレート
// と同じメタデータを持つかを検査する。inbox に直接書き込む経路(skill や
// エージェント)が旧書式に戻るのを止めるのが目的なので、テンプレートを
// 通さない書き方に例外は設けない。
func checkInbox(f orgFile) []Problem {
	if f.path != "inbox.org" {
		return nil
	}
	var problems []Problem
	for i, line := range f.lines {
		if f.kinds[i] == kindBlock || !topHeadingRe.MatchString(line) {
			continue
		}
		hasID, hasCaptured := headingProps(f, i)
		if !hasID {
			problems = append(problems, Problem{Path: f.path, Line: i + 1, Message: "capture の見出しに :ID: を必須とする"})
		}
		if !hasCaptured {
			problems = append(problems, Problem{Path: f.path, Line: i + 1, Message: "capture の見出しに :CAPTURED: を必須とする"})
		}
	}
	return problems
}

// headingProps は heading 行の直後に置かれた PROPERTIES ドロワーに
// :ID: と :CAPTURED: があるかを返す。見出しと離れたドロワーは org 自身が
// 見出しの property として扱わないので、直後にあることも合わせて見る。
func headingProps(f orgFile, heading int) (hasID, hasCaptured bool) {
	i := heading + 1
	if i >= len(f.lines) || !propBeginRe.MatchString(f.lines[i]) {
		return false, false
	}
	for i++; i < len(f.lines) && !drawerEndRe.MatchString(f.lines[i]); i++ {
		switch {
		case idPropRe.MatchString(f.lines[i]):
			hasID = true
		case capturedRe.MatchString(f.lines[i]):
			hasCaptured = true
		}
	}
	return hasID, hasCaptured
}

// checkFormat は本文の段落が fillColumn での折り返し結果と一致しない
// ものを報告する。桁数の超過だけでなく、折り返しが早すぎて次の行の語を
// 繰り上げられる段落も違反になる。
func checkFormat(f orgFile) []Problem {
	_, changed := formatLines(f.lines, f.kinds)
	var problems []Problem
	for _, line := range changed {
		problems = append(problems, Problem{
			Path:    f.path,
			Line:    line,
			Message: fmt.Sprintf("段落の折り返しが %d 桁の整形結果と一致しない", fillColumn),
		})
	}
	return problems
}

// checkEmphasis は org がコード表記として組み直さない =…= と ~…~ を
// 報告する。前後に置ける文字を外れた記法は色も等幅も付かず、中の
// リンクも組み立てられないまま素の文字として表示される。ブロックと
// ドロワーと固定幅行の中、および #+ 行とコメントは org 自身が組み直さ
// ないので対象にしない。
func checkEmphasis(f orgFile) []Problem {
	var problems []Problem
	for i, line := range f.lines {
		if f.kinds[i] == kindBlock || f.kinds[i] == kindDrawer || f.kinds[i] == kindFixed {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		_, dangling := emphasisSpans(line)
		for _, s := range dangling {
			problems = append(problems, Problem{
				Path:    f.path,
				Line:    i + 1,
				Message: fmt.Sprintf("コード表記 %s の前後の文字が条件を満たさず組まれない", line[s[0]:s[1]]),
			})
		}
	}
	return problems
}

// readAll は list が返したパスを読んで解析する。
func readAll(fsys fs.FS, list func(fs.FS) ([]string, error)) ([]orgFile, error) {
	paths, err := list(fsys)
	if err != nil {
		return nil, err
	}
	var files []orgFile
	for _, p := range paths {
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return nil, err
		}
		files = append(files, parseOrg(p, data))
	}
	return files, nil
}

// checkIDs は ID の重複とリンク先の存在を検査する。archives は ID を
// 提供するだけで、そこから出るリンクは検査しない。archive に残る過去の
// task は当時のリンクをそのまま抱えており、リンク先を消すたびに済んだ
// task を書き換える羽目になるため。
func checkIDs(files, archives []orgFile) []Problem {
	owners := map[string][]string{}
	for _, f := range slices.Concat(files, archives) {
		for _, id := range f.ids {
			owners[id] = append(owners[id], f.path)
		}
	}
	var problems []Problem
	for _, f := range files {
		for _, id := range f.ids {
			if len(owners[id]) > 1 {
				problems = append(problems, Problem{Path: f.path, Message: fmt.Sprintf("ID %s が %d 箇所で重複している", id, len(owners[id]))})
			}
		}
		for _, target := range f.links {
			if len(owners[target]) == 0 {
				problems = append(problems, Problem{Path: f.path, Message: fmt.Sprintf("リンク先 ID %s が存在しない", target)})
			}
		}
	}
	return problems
}
