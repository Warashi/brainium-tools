package maturity

import (
	"strings"
	"testing"
	"time"
)

func note(props string) string {
	return ":PROPERTIES:\n:ID:       aaaaaaaa-0000-0000-0000-000000000001\n" + props + ":END:\n#+title: ノート\n\n* 見出し\n:PROPERTIES:\n:STATUS:   bogus\n:END:\n"
}

func TestParse(t *testing.T) {
	p := Parse(note(":STATUS:   stable\n:VERIFIED: [2026-09-26 Sat]\n:STALE_AFTER: [2026-11-26 Thu]\n"))
	if p.Status.Value != "stable" || p.Status.Line != 3 {
		t.Errorf("status = %+v", p.Status)
	}
	if p.Verified.Value != "[2026-09-26 Sat]" || p.Verified.Line != 4 {
		t.Errorf("verified = %+v", p.Verified)
	}
	if p.StaleAfter.Value != "[2026-11-26 Thu]" || p.StaleAfter.Line != 5 {
		t.Errorf("stale_after = %+v", p.StaleAfter)
	}
}

func TestParseIgnoresHeadingProperties(t *testing.T) {
	if p := Parse(note("")); p.Status.Value != "" {
		t.Errorf("見出しの PROPERTIES を拾った: %+v", p.Status)
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name  string
		props string
		want  []string
	}{
		{"draft は通る", ":STATUS:   draft\n", nil},
		{"検証日付きの stable は通る", ":STATUS:   stable\n:VERIFIED: [2026-09-22]\n", nil},
		{"deprecated は通る", ":STATUS:   deprecated\n", nil},
		{"STATUS が無い", "", []string{":STATUS: を必須とする"}},
		{"語彙に無い STATUS", ":STATUS:   wip\n", []string{"draft / stable / deprecated"}},
		{"検証日の無い stable", ":STATUS:   stable\n", []string{"stable には :VERIFIED:"}},
		{"日付として読めない VERIFIED", ":STATUS:   stable\n:VERIFIED: 2026-09-26\n", []string{":VERIFIED: が"}},
		{"日付として読めない STALE_AFTER", ":STATUS:   draft\n:STALE_AFTER: [2026-13-01]\n", []string{":STALE_AFTER: が"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Check(Parse(note(tt.props)))
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %d problems", got, len(tt.want))
			}
			for i, w := range tt.want {
				if !strings.Contains(got[i].Message, w) {
					t.Errorf("problem %q does not contain %q", got[i].Message, w)
				}
			}
		})
	}
}

func TestPending(t *testing.T) {
	today := time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		props string
		want  []string
	}{
		{"draft は精錬待ち", ":STATUS:   draft\n", []string{"draft"}},
		{"期限内の stable は対象外", ":STATUS:   stable\n:VERIFIED: [2026-09-26 Sat]\n:STALE_AFTER: [2026-11-27 Fri]\n", nil},
		{"期限を過ぎた stable は再検証待ち", ":STATUS:   stable\n:VERIFIED: [2026-09-26 Sat]\n:STALE_AFTER: [2026-11-26 Thu]\n", []string{"stale_after [2026-11-26 Thu]"}},
		{"期限を過ぎた draft は両方", ":STATUS:   draft\n:STALE_AFTER: [2026-11-26 Thu]\n", []string{"draft", "stale_after"}},
		{"deprecated は期限を過ぎても対象外", ":STATUS:   deprecated\n:STALE_AFTER: [2026-11-26 Thu]\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Pending(Parse(note(tt.props)), today)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i, w := range tt.want {
				if !strings.Contains(got[i], w) {
					t.Errorf("reason %q does not contain %q", got[i], w)
				}
			}
		})
	}
}
