package tokenize_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Warashi/brainium-tools/internal/notesearch/tokenize"
)

func TestTokens(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "英単語は空白で切って小文字にする",
			input:    "Hello World",
			expected: []string{"hello", "world"},
		},
		{
			name:     "英数字が連なる語は 1 つの語として切る",
			input:    "bm25",
			expected: []string{"bm25"},
		},
		{
			name:     "記号は語の境界になる",
			input:    "foo-bar.baz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "日本語は 1-gram から 5-gram まで展開する",
			input:    "検索した",
			expected: []string{"検", "索", "し", "た", "検索", "索し", "した", "検索し", "索した", "検索した"},
		},
		{
			name:  "5 文字を超える日本語は 5-gram までで打ち切る",
			input: "あいうえおか",
			expected: []string{
				"あ", "い", "う", "え", "お", "か",
				"あい", "いう", "うえ", "えお", "おか",
				"あいう", "いうえ", "うえお", "えおか",
				"あいうえ", "いうえお", "うえおか",
				"あいうえお", "いうえおか",
			},
		},
		{
			name:     "日本語と英語が隣接すると別の語になる",
			input:    "org形式",
			expected: []string{"org", "形", "式", "形式"},
		},
		{
			name:     "漢字と仮名は切らずに 1 つの区間として n-gram にする",
			input:    "検索",
			expected: []string{"検", "索", "検索"},
		},
		{
			name:     "全角英数は半角に畳む",
			input:    "ＢＭ２５",
			expected: []string{"bm25"},
		},
		{
			name:     "半角カナは全角に畳む",
			input:    "ｶﾅ",
			expected: []string{"カ", "ナ", "カナ"},
		},
		{
			name:     "半角カナの濁点は合成して 1 文字にする",
			input:    "ｶﾞｽ",
			expected: []string{"ガ", "ス", "ガス"},
		},
		{
			name:     "半角カナの半濁点は合成して 1 文字にする",
			input:    "ﾊﾟﾝ",
			expected: []string{"パ", "ン", "パン"},
		},
		{
			name:     "空文字列からは語が出ない",
			input:    "",
			expected: nil,
		},
		{
			name:     "記号だけの文字列からは語が出ない",
			input:    "!!! ---",
			expected: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := tokenize.Tokens(c.input)
			slices.Sort(got)
			want := slices.Clone(c.expected)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("expected %q, got %q", want, got)
			}
		})
	}
}

func TestTokensCountsRepeatedTerms(t *testing.T) {
	t.Parallel()

	got := tokenize.Tokens("go go")
	want := []string{"go", "go"}
	if !slices.Equal(got, want) {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestGroupsPreserveLexicalSegments(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
		want [][]string
	}{
		{"slash separated Latin", "BM25/PRF", [][]string{{"bm25"}, {"prf"}}},
		{"hyphen separated Latin", "query-time", [][]string{{"query"}, {"time"}}},
		{
			"adjacent Latin and Japanese",
			"notesearchの却下",
			[][]string{{"notesearch"}, {"の", "却", "下", "の却", "却下", "の却下"}},
		},
		{"fullwidth and halfwidth folding", "ＢＭ２５/ｶﾞｽ", [][]string{{"bm25"}, {"ガ", "ス", "ガス"}}},
		{
			"natural Japanese runs",
			"検索 結果",
			[][]string{{"検", "索", "検索"}, {"結", "果", "結果"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tokenize.Groups(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Groups(%q) = %#v, want %#v", tc.text, got, tc.want)
			}
		})
	}
}

func TestTokensRetainsGroupFlatteningOrder(t *testing.T) {
	t.Parallel()
	want := []string{"bm25", "prf", "検", "索", "検索"}
	if got := tokenize.Tokens("BM25/PRF-検索"); !slices.Equal(got, want) {
		t.Fatalf("Tokens order changed: got %q, want %q", got, want)
	}
}
