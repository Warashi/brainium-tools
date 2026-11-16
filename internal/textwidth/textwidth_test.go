package textwidth

import "testing"

func TestString(t *testing.T) {
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
		if got := String(tt.text); got != tt.want {
			t.Errorf("String(%q) = %d, want %d", tt.text, got, tt.want)
		}
	}
}
