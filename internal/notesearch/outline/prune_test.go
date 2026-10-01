package outline_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Warashi/brainium-tools/internal/notesearch/outline"
)

// body は n バイトちょうどの本文行を作る(改行込み)。
func body(n int) string {
	return strings.Repeat("x", n-1) + "\n"
}

func titles(nodes []outline.Node) []string {
	var out []string
	for _, n := range nodes {
		out = append(out, n.Title)
	}
	return out
}

func TestPrune(t *testing.T) {
	t.Parallel()

	t.Run("たどる手間より丸ごと読むほうが安い文書は畳み、見出しを key items に残す", func(t *testing.T) {
		t.Parallel()
		// 全体 40 バイト。たどれば R=100 が掛かるので畳む
		source := "* a\n" + body(10) + "** b\n" + body(10) + "* c\n" + body(8)
		got := outline.Prune(source, 100)
		if !got.Collapsed || len(got.Sections) != 0 {
			t.Fatalf("got %+v, want collapsed", got)
		}
		if want := []string{"a", "b", "c"}; !reflect.DeepEqual(got.KeyItems, want) {
			t.Errorf("KeyItems = %v, want %v", got.KeyItems, want)
		}
	})

	t.Run("大きな節に分かれた文書は木を残す", func(t *testing.T) {
		t.Parallel()
		// 全体 2008 バイト。たどれば 10 + 1004 で済む
		source := "* a\n" + body(1000) + "* b\n" + body(1000)
		got := outline.Prune(source, 10)
		if got.Collapsed {
			t.Fatalf("got collapsed, want tree")
		}
		if want := []string{"a", "b"}; !reflect.DeepEqual(titles(got.Sections), want) {
			t.Errorf("sections = %v, want %v", titles(got.Sections), want)
		}
		if got.Sections[0].Bytes != 1004 || got.Bytes != 2008 {
			t.Errorf("bytes = %d / %d, want 1004 / 2008", got.Sections[0].Bytes, got.Bytes)
		}
	})

	t.Run("小さな部分木だけを畳む", func(t *testing.T) {
		t.Parallel()
		source := "* big\n" + body(1000) + "* big2\n" + body(1000) +
			"* small\n" + body(20) + "** x\n" + body(20) + "** y\n" + body(20)
		got := outline.Prune(source, 100)
		if want := []string{"big", "big2", "small"}; !reflect.DeepEqual(titles(got.Sections), want) {
			t.Fatalf("sections = %v, want %v", titles(got.Sections), want)
		}
		if want := []string{"x", "y"}; !reflect.DeepEqual(got.Sections[2].KeyItems, want) {
			t.Errorf("small KeyItems = %v, want %v", got.Sections[2].KeyItems, want)
		}
		if got.Sections[0].KeyItems != nil {
			t.Errorf("leaf has KeyItems %v", got.Sections[0].KeyItems)
		}
	})

	t.Run("最初の見出しより前の本文が大きければ、たどっても安くならない", func(t *testing.T) {
		t.Parallel()
		source := body(1000) + "* a\n" + body(500) + "* b\n" + body(500)
		got := outline.Prune(source, 10)
		// 全体 2008 <= 10 + max(1000, 504) にはならないので木は残る
		if got.Collapsed {
			t.Fatalf("got collapsed")
		}
		got = outline.Prune(source, 1100)
		if !got.Collapsed {
			t.Errorf("R=1100 should collapse: 2008 <= 1100+1000")
		}
	})

	t.Run("見出しが無ければ畳んだ扱いで key items も無い", func(t *testing.T) {
		t.Parallel()
		got := outline.Prune("#+title: x\n本文\n", 10)
		if !got.Collapsed || got.KeyItems != nil || got.Sections != nil {
			t.Errorf("got %+v", got)
		}
	})
}
