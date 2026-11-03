package org

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

// diffNode は org-element.el が出す節点。位置は 0 始まりの文字単位。
type diffNode struct {
	Type          string            `json:"type"`
	Begin         int               `json:"begin"`
	End           int               `json:"end"`
	ContentsBegin *int              `json:"contents_begin"`
	ContentsEnd   *int              `json:"contents_end"`
	PostBlank     int               `json:"post_blank"`
	Attrs         map[string]string `json:"attrs"`
	Children      []diffNode        `json:"children"`
}

// TestParseMatchesOrgElement は Parse の木が Emacs の org-element と一致
// することを確かめる。Emacs が無ければ飛ばすが、ORG_DIFF_REQUIRE=1 なら
// 失敗にして、正解を出せないまま通るのを防ぐ。ORG_DIFF_CORPUS にディレ
// クトリを渡すと、その配下の org ファイルも比べる。
func TestParseMatchesOrgElement(t *testing.T) {
	emacs := lookEmacs(t)
	files := orgFiles(t, "testdata/cases")
	if corpus := os.Getenv("ORG_DIFF_CORPUS"); corpus != "" {
		files = append(files, orgFiles(t, corpus)...)
	}
	want := runOrgElement(t, emacs, files)
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			source := string(data)
			got := toDiffNode(Parse(source), runeOffsets(source))
			if diff := firstDiff(got, want[file], got.Type); diff != "" {
				t.Error(diff)
			}
		})
	}
}

// lookEmacs は emacs のパスを返す。無ければテストを飛ばすが、
// ORG_DIFF_REQUIRE=1 なら失敗にする。
func lookEmacs(t *testing.T) string {
	t.Helper()
	emacs, err := exec.LookPath("emacs")
	if err != nil {
		if os.Getenv("ORG_DIFF_REQUIRE") == "1" {
			t.Fatal("ORG_DIFF_REQUIRE=1 だが emacs が PATH に無い")
		}
		t.Skip("emacs が PATH に無い")
	}
	return emacs
}

func runEmacs(t *testing.T, emacs string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(emacs, append([]string{"-Q", "--batch"}, args...)...)
	// org-persist などが HOME に書こうとするので、書ける場所に向ける
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("emacs: %v\n%s", err, stderr.String())
	}
	return out
}

func orgFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && p != dir {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(p, ".org") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func runOrgElement(t *testing.T, emacs string, files []string) map[string]diffNode {
	t.Helper()
	out := runEmacs(t, emacs, append([]string{"-l", "testdata/org-element.el"}, files...)...)
	var result struct {
		OrgVersion string              `json:"org_version"`
		Files      map[string]diffNode `json:"files"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("org-element.el の出力を読めない: %v", err)
	}
	t.Logf("org-version %s", result.OrgVersion)
	return result.Files
}

// runeOffsets はバイト位置から文字位置への対応表を返す。
func runeOffsets(source string) []int {
	offsets := make([]int, len(source)+1)
	for i := range len(source) {
		offsets[i+1] = offsets[i]
		if utf8.RuneStart(source[i]) {
			offsets[i+1]++
		}
	}
	return offsets
}

func toDiffNode(n *Node, offsets []int) diffNode {
	d := diffNode{
		Type:      string(n.Type),
		Begin:     offsets[n.Begin],
		End:       offsets[n.End],
		PostBlank: n.PostBlank,
		Attrs:     attributes(n),
		Children:  []diffNode{},
	}
	if n.ContentsBegin >= 0 {
		b, e := offsets[n.ContentsBegin], offsets[n.ContentsEnd]
		d.ContentsBegin, d.ContentsEnd = &b, &e
	}
	for _, c := range n.Children {
		d.Children = append(d.Children, toDiffNode(c, offsets))
	}
	return d
}

// attributes は org-element.el の org-diff--attributes と同じ属性を取り
// 出す。
func attributes(n *Node) map[string]string {
	switch n.Type {
	case TypeNodeProperty, TypeKeyword:
		return map[string]string{"key": n.Key, "value": n.Value}
	}
	return map[string]string{}
}

func firstDiff(got, want diffNode, path string) string {
	field := func(name string, g, w any) string {
		return fmt.Sprintf("%s: %s = %v, org-element は %v", path, name, g, w)
	}
	switch {
	case got.Type != want.Type:
		return field("type", got.Type, want.Type)
	case got.Begin != want.Begin:
		return field("begin", got.Begin, want.Begin)
	case got.End != want.End:
		return field("end", got.End, want.End)
	case fmtPos(got.ContentsBegin) != fmtPos(want.ContentsBegin):
		return field("contents_begin", fmtPos(got.ContentsBegin), fmtPos(want.ContentsBegin))
	case fmtPos(got.ContentsEnd) != fmtPos(want.ContentsEnd):
		return field("contents_end", fmtPos(got.ContentsEnd), fmtPos(want.ContentsEnd))
	case got.PostBlank != want.PostBlank:
		return field("post_blank", got.PostBlank, want.PostBlank)
	case !maps.Equal(got.Attrs, want.Attrs):
		return field("attrs", got.Attrs, want.Attrs)
	}
	for i := range min(len(got.Children), len(want.Children)) {
		child := fmt.Sprintf("%s/%d:%s", path, i, want.Children[i].Type)
		if diff := firstDiff(got.Children[i], want.Children[i], child); diff != "" {
			return diff
		}
	}
	if len(got.Children) != len(want.Children) {
		return field("children の数", len(got.Children), len(want.Children))
	}
	return ""
}

func fmtPos(p *int) string {
	if p == nil {
		return "null"
	}
	return fmt.Sprint(*p)
}
