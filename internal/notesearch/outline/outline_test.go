package outline_test

import (
	"reflect"
	"testing"

	"github.com/Warashi/brainium-tools/internal/notesearch/outline"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   []outline.Section
	}{
		{
			name:   "見出しが無ければ空",
			source: "#+title: x\n本文\n",
			want:   nil,
		},
		{
			name: "節は次の同じか浅い見出しの直前まで続く",
			source: "#+title: x\n" + // 1
				"* 目的\n" + // 2
				"本文\n" + // 3
				"** 詳細\n" + // 4
				"本文\n" + // 5
				"*** 例\n" + // 6
				"** 補足\n" + // 7
				"* 結論\n" + // 8
				"本文\n", // 9
			want: []outline.Section{
				{Level: 1, Title: "目的", StartLine: 2, EndLine: 7},
				{Level: 2, Title: "詳細", StartLine: 4, EndLine: 6},
				{Level: 3, Title: "例", StartLine: 6, EndLine: 6},
				{Level: 2, Title: "補足", StartLine: 7, EndLine: 7},
				{Level: 1, Title: "結論", StartLine: 8, EndLine: 9},
			},
		},
		{
			name:   "末尾に改行が無くても最終行で閉じる",
			source: "* a\n本文",
			want:   []outline.Section{{Level: 1, Title: "a", StartLine: 1, EndLine: 2}},
		},
		{
			name:   "星の後に空白が無い行や太字は見出しではない",
			source: "* a\n*太字*\n**\n",
			want:   []outline.Section{{Level: 1, Title: "a", StartLine: 1, EndLine: 3}},
		},
		{
			name:   "TODO キーワードやタグは見出しの文字列として残す",
			source: "** TODO 直す :tag:\n",
			want:   []outline.Section{{Level: 2, Title: "TODO 直す :tag:", StartLine: 1, EndLine: 1}},
		},
		{
			name:   "CRLF の行末を見出しに含めない",
			source: "* a\r\n本文\r\n",
			want:   []outline.Section{{Level: 1, Title: "a", StartLine: 1, EndLine: 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := outline.Parse(tt.source); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestInnermost(t *testing.T) {
	t.Parallel()

	sections := outline.Parse("#+title: x\n* a\n本文\n** b\n本文\n* c\n")

	tests := []struct {
		line int
		want int
	}{
		{line: 1, want: -1}, // 最初の見出しより前
		{line: 3, want: 0},
		{line: 5, want: 1}, // a と b の両方に入るので深い b
		{line: 6, want: 2},
	}
	for _, tt := range tests {
		if got := outline.Innermost(sections, tt.line); got != tt.want {
			t.Errorf("Innermost(%d) = %d, want %d", tt.line, got, tt.want)
		}
	}
}
