package eval

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Warashi/brainium-tools/internal/notesearch"
)

func TestLoadSkipsHeaderAndParsesGap(t *testing.T) {
	tsv := "query\texpected\tsplit\tgap\tbilingual\n" +
		"foo\ta.org\ttrain\t1\tfoo bar\nbar\tb.org\ttest\t0\tbar\n"
	qs, err := Load(strings.NewReader(tsv))
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 2 {
		t.Fatalf("2 件のはずが %d 件", len(qs))
	}
	if qs[0] != (Query{"foo", "a.org", "train", true, "foo bar"}) {
		t.Errorf("1 件目が %+v", qs[0])
	}
	if qs[1].Gap {
		t.Error("gap=0 が true になっている")
	}
}

func TestLoadRejectsWrongColumnCount(t *testing.T) {
	if _, err := Load(strings.NewReader("h\nfoo\ta.org\ttrain\t1\n")); err == nil {
		t.Error("列数不正がエラーにならない")
	}
}

func TestSelectFiltersBySplit(t *testing.T) {
	qs := []Query{{Split: "train"}, {Split: "test"}, {Split: "train"}}
	if n := len(Select(qs, "train")); n != 2 {
		t.Errorf("train が %d 件", n)
	}
	if n := len(Select(qs, "all")); n != 3 {
		t.Errorf("all が %d 件", n)
	}
}

func TestEvaluateAveragesReciprocalRank(t *testing.T) {
	ix, err := notesearch.Build(fstest.MapFS{
		"apple.org":  {Data: []byte("apple pie recipe")},
		"banana.org": {Data: []byte("banana bread recipe")},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	qs := []Query{
		{Text: "apple", Want: "apple.org", Gap: true},
		{Text: "recipe banana", Want: "banana.org"},
		{Text: "cherry", Want: "apple.org"},
	}
	all, gap := Evaluate(ix, qs, notesearch.Params{}, Rank)
	if all.N != 3 || all.Hit1 != 2 || all.Hit10 != 2 {
		t.Errorf("全体 %+v", all)
	}
	if want := 2.0 / 3; all.MRR < want-1e-9 || all.MRR > want+1e-9 {
		t.Errorf("MRR=%v want %v", all.MRR, want)
	}
	if gap.N != 1 || gap.Hit1 != 1 || gap.MRR != 1 {
		t.Errorf("gap %+v", gap)
	}
}
