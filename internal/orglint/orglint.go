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

	"github.com/Warashi/brainium-tools/internal/org"
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
	src    string
	root   *org.Node
	title  string
	fileID string
	ids    []string
	links  []string
	lines  []string
	layout layout
}

var (
	slugRe    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*\.org$`)
	journalRe = regexp.MustCompile(`^journal/(\d{4})/((\d{4})-\d{2}-\d{2})\.org$`)
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
		problems = append(problems, checkUnclosedDrawer(f)...)
		problems = append(problems, checkInbox(f)...)
		problems = append(problems, checkTaskLength(f)...)
		problems = append(problems, checkFormat(f)...)
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
	kindOther   lineKind = iota // 空行・見出し・計画行・表
	kindBody                    // 折り返しの対象になる段落の行
	kindDrawer                  // ドロワーの内側と境界行
	kindBlock                   // ブロックの内側と境界行
	kindFixed                   // 固定幅行(: で始まる行)
	kindKeyword                 // #+ で始まるキーワード行
	kindComment                 // コメント行と comment ブロックの内側・境界行
)

// layout は行の種類と、段落の行が属する段落。
type layout struct {
	kinds []lineKind
	// paras は段落の行に、その段落の番号を持たせる。段落の行でなければ -1。
	paras []int
	// indents は段落の番号ごとの継続行の字下げ。
	indents []string
	// unclosed は閉じていないドロワーの開始行。
	unclosed []int
}

// lineLayout は org の parser の木から行の種類と段落を決める。ドロワー
// と comment 以外のブロックは中の段落も折り返さない。item の中の段落の
// 継続行は、org の fill と同じく item の本文の桁
// (org-list-item-body-column)に揃える。
func lineLayout(src string, root *org.Node, lines []string) layout {
	starts := make([]int, len(lines))
	for i, off := 0, 0; i < len(lines); i++ {
		starts[i] = off
		off += len(lines[i]) + 1
	}
	index := func(off int) int {
		return sort.Search(len(starts), func(i int) bool { return starts[i] > off }) - 1
	}
	last := func(n *org.Node) int { return index(n.End-1) - n.PostBlank }
	l := layout{kinds: make([]lineKind, len(lines)), paras: make([]int, len(lines))}
	for i := range l.paras {
		l.paras[i] = -1
	}
	mark := func(n *org.Node, k lineKind) {
		for i := index(n.Begin); i <= last(n); i++ {
			l.kinds[i] = k
		}
	}
	var walk func(n *org.Node)
	walk = func(n *org.Node) {
		for _, c := range n.Children {
			switch c.Type {
			case org.TypeDrawer, org.TypePropertyDrawer:
				mark(c, kindDrawer)
			case org.TypeExampleBlock, org.TypeSrcBlock, org.TypeQuoteBlock, org.TypeSpecialBlock:
				mark(c, kindBlock)
			case org.TypeCommentBlock:
				mark(c, kindComment)
				l.commentBlockParagraphs(lines, index(c.Begin)+1, last(c)-1)
			case org.TypeFixedWidth:
				mark(c, kindFixed)
			case org.TypeKeyword:
				mark(c, kindKeyword)
			case org.TypeComment:
				mark(c, kindComment)
				l.commentParagraphs(lines, index(c.Begin), last(c))
			case org.TypeParagraph:
				// 閉じていないドロワーは org では段落になるが、折り返すと
				// :PROPERTIES: などの行がつながって壊れる
				if org.IsDrawerBegin(lines[index(c.Begin)]) {
					mark(c, kindDrawer)
					l.unclosed = append(l.unclosed, index(c.Begin))
					continue
				}
				mark(c, kindBody)
				for i := index(c.Begin); i <= last(c); i++ {
					l.paras[i] = len(l.indents)
				}
				indent := leadingSpace(lines[index(c.Begin)])
				if n.Type == org.TypeItem {
					indent = strings.Repeat(" ", itemBodyColumn(lines[index(n.Begin)]))
				}
				l.indents = append(l.indents, indent)
			default:
				walk(c)
			}
		}
	}
	walk(root)
	return l
}

var (
	// commentSeparatorRe は org-fill-element がコメントの段落の区切りと
	// する行。
	commentSeparatorRe = regexp.MustCompile(`^[ \t]*#[ \t]*$`)
	// commentPrefixRe はコメントの継続行に付ける fill-prefix。# の後ろは
	// org-mode の adaptive-fill-regexp の既定値。
	commentPrefixRe = regexp.MustCompile(`^[ \t]*#[-–!|#%;>*·•‣⁃◦ \t]*`)
)

// commentParagraphs はコメントの行 [first, last] を # だけの行で段落に
// 分ける。継続行の頭は段落の 1 行目の fill-prefix に揃える。
func (l *layout) commentParagraphs(lines []string, first, last int) {
	for i := first; i <= last; i++ {
		if commentSeparatorRe.MatchString(lines[i]) {
			continue
		}
		if i == first || commentSeparatorRe.MatchString(lines[i-1]) {
			l.indents = append(l.indents, commentPrefixRe.FindString(lines[i]))
		}
		l.paras[i] = len(l.indents) - 1
	}
}

// commentBlockParagraphs は comment ブロックの中身の行 [first, last] を
// 空行で段落に分ける。org-fill-element は中身を org の要素として読まず
// 素の文章として fill するので、リストやキーワード行もつながる。継続行
// の頭は fill-context-prefix と同じく、2 行目(1 行だけなら 1 行目)の
// 字下げに揃える。
func (l *layout) commentBlockParagraphs(lines []string, first, last int) {
	for i := first; i <= last; {
		if strings.TrimSpace(lines[i]) == "" {
			i++
			continue
		}
		end := i
		for end+1 <= last && strings.TrimSpace(lines[end+1]) != "" {
			end++
		}
		for j := i; j <= end; j++ {
			l.paras[j] = len(l.indents)
		}
		l.indents = append(l.indents, leadingSpace(lines[min(i+1, end)]))
		i = end + 1
	}
}

// itemBodyColumn は item の行の本文が始まる桁を返す
// (org-list-item-body-column)。マーカーの直後に空白 1 個分を足す。
func itemBodyColumn(line string) int {
	rest := strings.TrimLeft(line, " \t")
	marker := len(line) - len(rest)
	if i := strings.IndexAny(rest, " \t"); i >= 0 {
		marker += i
	} else {
		marker += len(rest)
	}
	return len([]rune(line[:marker])) + 1
}

// parseOrg は ID・ファイルレベルのタイトル・id: リンクを org の parser
// で読み、整形の検査に使う行と行種を持たせる。
func parseOrg(p string, data []byte) orgFile {
	lines := strings.Split(string(data), "\n")
	src := string(data)
	root := org.Parse(src)
	f := orgFile{path: p, src: src, root: root, lines: lines, layout: lineLayout(src, root, lines), title: org.FileTitle(root)}
	if drawer := org.FileProperties(root); drawer != nil {
		f.fileID = propertyID(drawer)
	}
	f.ids = collectIDs(root, nil)
	for _, l := range org.Links(root) {
		if l.LinkType == "id" {
			f.links = append(f.links, l.Path)
		}
	}
	return f
}

// collectIDs は n 以下の PROPERTIES ドロワーにある :ID: の値を ids に
// 足して返す。org-node が索引に使う org-mem も、PROPERTIES ドロワーの
// :ID: だけを拾う。
func collectIDs(n *org.Node, ids []string) []string {
	if n.Type == org.TypePropertyDrawer {
		if id := propertyID(n); id != "" {
			ids = append(ids, id)
		}
		return ids
	}
	for _, c := range n.Children {
		ids = collectIDs(c, ids)
	}
	return ids
}

// propertyID は PROPERTIES ドロワーの :ID: の値を返す。無ければ空文字
// 列。複数あれば org-entry-get と org-mem と同じく後ろの値を取る。
func propertyID(drawer *org.Node) string {
	var id string
	for _, n := range drawer.Children {
		if strings.EqualFold(n.Key, "ID") {
			id = n.Value
		}
	}
	return id
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

// checkUnclosedDrawer は :END: で閉じていないドロワーを報告する。org は
// それをドロワーではなく段落として読み、PROPERTIES なら見出しの
// property も失われる。
func checkUnclosedDrawer(f orgFile) []Problem {
	var problems []Problem
	for _, i := range f.layout.unclosed {
		problems = append(problems, Problem{Path: f.path, Line: i + 1, Message: "ドロワーが :END: で閉じていない"})
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
	for _, h := range f.root.Children {
		if h.Type != org.TypeHeadline || h.Level != 1 {
			continue
		}
		hasID, hasCaptured := headlineProps(h)
		line := f.lineAt(h.Begin)
		if !hasID {
			problems = append(problems, Problem{Path: f.path, Line: line, Message: "capture の見出しに :ID: を必須とする"})
		}
		if !hasCaptured {
			problems = append(problems, Problem{Path: f.path, Line: line, Message: "capture の見出しに :CAPTURED: を必須とする"})
		}
	}
	return problems
}

// headlineProps は見出しの PROPERTIES ドロワーに値のある :ID: と
// :CAPTURED: があるかを返す。
func headlineProps(h *org.Node) (hasID, hasCaptured bool) {
	drawer := org.HeadlineProperties(h)
	if drawer == nil {
		return false, false
	}
	for _, n := range drawer.Children {
		if n.Value == "" {
			continue
		}
		switch strings.ToUpper(n.Key) {
		case "ID":
			hasID = true
		case "CAPTURED":
			hasCaptured = true
		}
	}
	return hasID, hasCaptured
}

// lineAt は source のバイト位置 offset を含む行の 1 始まりの行番号を返す。
func (f orgFile) lineAt(offset int) int {
	return strings.Count(f.src[:offset], "\n") + 1
}

// lineCount は行頭から始まる [begin, end) が占める行数を返す。end が
// 改行で終わらないファイル末尾なら、最後の行も数える。
func (f orgFile) lineCount(begin, end int) int {
	n := strings.Count(f.src[begin:end], "\n")
	if end == len(f.src) && end > begin && !strings.HasSuffix(f.src, "\n") {
		n++
	}
	return n
}

// checkFormat は本文の段落が fillColumn での折り返し結果と一致しない
// ものを報告する。桁数の超過だけでなく、折り返しが早すぎて次の行の語を
// 繰り上げられる段落も違反になる。
func checkFormat(f orgFile) []Problem {
	_, changed := formatLines(f.lines, f.layout)
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
