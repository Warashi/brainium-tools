// Package orglint の対象絞り込み。
package orglint

import "strings"

// Targets は検査・整形の対象を絞り込むルート相対のスラッシュ区切り
// パス。ファイルとディレクトリのどちらも書ける。空なら全件が対象。
type Targets []string

// match は p が対象に含まれるかを返す。
func (t Targets) match(p string) bool {
	if len(t) == 0 {
		return true
	}
	for _, target := range t {
		if target == "." || p == target || strings.HasPrefix(p, target+"/") {
			return true
		}
	}
	return false
}

// Filter は paths のうち対象に含まれるものだけを返す。
func (t Targets) Filter(paths []string) []string {
	var kept []string
	for _, p := range paths {
		if t.match(p) {
			kept = append(kept, p)
		}
	}
	return kept
}

// Unmatched は paths のどれにも一致しなかった対象を返す。打ち間違えた
// パスを黙って「違反なし」と見せないために呼び出し側が検査する。
func (t Targets) Unmatched(paths []string) []string {
	var missing []string
	for _, target := range t {
		found := false
		for _, p := range paths {
			if (Targets{target}).match(p) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, target)
		}
	}
	return missing
}
