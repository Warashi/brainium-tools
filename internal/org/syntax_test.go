package org

import (
	"encoding/json"
	"slices"
	"testing"
)

// TestSyntaxMatchesEmacs は文字の分類の表が Emacs の org-mode の構文表と
// 一致することを確かめる。
func TestSyntaxMatchesEmacs(t *testing.T) {
	out := runEmacs(t, lookEmacs(t), "-l", "testdata/syntax.el")
	var want map[string][][2]rune
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatalf("syntax.el の出力を読めない: %v", err)
	}
	for name, got := range map[string][][2]rune{
		"space":              spaceRanges,
		"word":               wordRanges,
		"latin_word":         latinWordRanges,
		"latex_fragment_end": latexFragmentEndRanges,
	} {
		if !slices.Equal(got, want[name]) {
			t.Errorf("%s の表が Emacs と違う。testdata/syntax.el の出力で作り直す", name)
		}
	}
}
