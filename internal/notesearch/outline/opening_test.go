package outline_test

import (
	"strings"
	"testing"

	"github.com/Warashi/brainium-tools/internal/notesearch/outline"
)

func TestOpening(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "メタデータを飛ばして最初の文を返す",
			source: ":PROPERTIES:\n:ID: x\n:END:\n#+title: t\n#+filetags: :a:\n\n最初の文。次の文。\n* 見出し\n",
			want:   "最初の文。",
		},
		{
			name:   "折り返した和文の行はそのままつなぐ",
			source: "#+title: t\n仕事用マシンで\nも使えるように する。\n",
			want:   "仕事用マシンでも使えるように する。",
		},
		{
			name:   "欧文どうしの折り返しは空白でつなぐ",
			source: "#+title: t\nUse the\ntools. Then rest.\n",
			want:   "Use the tools.",
		},
		{
			name:   "段落の終わりで止める",
			source: "#+title: t\n句点の無い段落\n\n次の段落。\n",
			want:   "句点の無い段落",
		},
		{
			name:   "最初の見出しより前に本文が無ければ空",
			source: "#+title: t\n\n* 見出し\n本文。\n",
			want:   "",
		},
		{
			name:   "リンクは Emacs での見え方と同じく説明文だけにする",
			source: "#+title: t\n[[id:0000][前のノート]]を受けて [[https://example.com]] を読んだ。\n",
			want:   "前のノートを受けて https://example.com を読んだ。",
		},
		{
			name:   "対応の無い :END: をドロワーの始まりとみなさない",
			source: "#+title: t\n:END:\n説明。\n",
			want:   "説明。",
		},
		{
			name:   "ブロックは本文として扱わない",
			source: "#+title: t\n#+begin_src sh\nls\n#+end_src\n説明。\n",
			want:   "説明。",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := outline.Opening(tt.source, 200); got != tt.want {
				t.Errorf("Opening() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("上限を超える文は文字の境界で切って印を付ける", func(t *testing.T) {
		t.Parallel()
		got := outline.Opening("#+title: t\n"+strings.Repeat("あ", 100)+"。\n", 10)
		if got != "あああ…" {
			t.Errorf("Opening() = %q", got)
		}
	})
}
