package evidence_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Warashi/brainium-tools/internal/notesearch/evidence"
)

func TestExtractMetadataAndJapaneseNestedEvidence(t *testing.T) {
	t.Parallel()
	content := ":PROPERTIES:\n:ID: note-1\n:STATUS: stable\n:VERIFIED: 2026-09-01\n:STALE_AFTER: 2027-09-01\n:END:\n" +
		"#+TiTlE: 検索の設計\n\n* 索引\n概要です。\n\n** 根拠抽出\n前半の段落です。\n\n" +
		"これは検索結果から原文の根拠を抽出する段落です。\n改変せず利用者へ表示します。\n"

	got := evidence.Extract(content, "原文の根拠を抽出")
	if got.Metadata != (evidence.Metadata{
		ID: "note-1", Title: "検索の設計", Status: "stable",
		Verified: "2026-09-01", StaleAfter: "2027-09-01",
	}) {
		t.Fatalf("metadata = %#v", got.Metadata)
	}
	if len(got.Snippets) == 0 {
		t.Fatal("no snippets")
	}
	s := got.Snippets[0]
	if s.Kind != "match" {
		t.Fatalf("kind = %q", s.Kind)
	}
	if s.StartLine != 15 || s.EndLine != 16 {
		t.Fatalf("lines = %d-%d", s.StartLine, s.EndLine)
	}
	if want := []string{"索引", "根拠抽出"}; strings.Join(s.Headings, "/") != strings.Join(want, "/") {
		t.Fatalf("headings = %#v", s.Headings)
	}
	if s.Text != "これは検索結果から原文の根拠を抽出する段落です。\n改変せず利用者へ表示します。" {
		t.Fatalf("text changed: %q", s.Text)
	}
}

func TestExtractPreservesCRLFAndFindsLateEnglishParagraph(t *testing.T) {
	t.Parallel()
	content := "#+title: Evidence\r\n\r\n* Intro\r\nordinary introduction\r\n\r\n* Details\r\n" +
		"first unrelated paragraph\r\n\r\nExact source provenance is retained.\r\nLine endings remain untouched.\r\n"
	got := evidence.Extract(content, "source provenance retained")
	if len(got.Snippets) == 0 {
		t.Fatal("no snippets")
	}
	s := got.Snippets[0]
	if s.StartLine != 9 || s.EndLine != 10 {
		t.Fatalf("lines = %d-%d", s.StartLine, s.EndLine)
	}
	if s.Text != "Exact source provenance is retained.\r\nLine endings remain untouched." {
		t.Fatalf("CRLF changed: %q", s.Text)
	}
}

func TestBlocksDoNotLeakMetadataOrHeadingAncestry(t *testing.T) {
	t.Parallel()
	content := ":PROPERTIES:\n:ID: real\n:END:\n#+title: Real title\n" +
		"#+begin_example\n:PROPERTIES:\n:ID: malicious\n:STATUS: stable\n:END:\n#+TITLE: Fake\n,* Fake heading\n#+end_example\n" +
		"\n* Real heading\nSafe body evidence.\n"
	got := evidence.Extract(content, "safe evidence")
	if got.Metadata.ID != "real" || got.Metadata.Title != "Real title" || got.Metadata.Status != "" {
		t.Fatalf("metadata leaked from block: %#v", got.Metadata)
	}
	if len(got.Snippets) != 1 || strings.Join(got.Snippets[0].Headings, "/") != "Real heading" {
		t.Fatalf("heading leaked from block: %#v", got.Snippets)
	}
}

func TestHeadingDrawerIsNeitherMetadataNorEvidence(t *testing.T) {
	t.Parallel()
	content := ":PROPERTIES:\n:ID: file-id\n:END:\n#+title: Drawer\n* Node\n" +
		":PROPERTIES:\n:ID: heading-id\n:STATUS: stable\n:END:\nActual body evidence.\n"
	got := evidence.Extract(content, "heading-id actual body evidence")
	if got.Metadata.ID != "file-id" || got.Metadata.Status != "" {
		t.Fatalf("heading properties leaked: %#v", got.Metadata)
	}
	if len(got.Snippets) != 1 || got.Snippets[0].Text != "Actual body evidence." {
		t.Fatalf("drawer selected: %#v", got.Snippets)
	}
}

func TestHeadingOnlyMatchReturnsReadableBody(t *testing.T) {
	t.Parallel()
	content := "#+title: Guide\n* Deployment checklist\nRun the ordinary validation first.\nThen publish the result.\n"
	got := evidence.Extract(content, "deployment checklist")
	if len(got.Snippets) != 1 {
		t.Fatalf("snippets = %#v", got.Snippets)
	}
	if got.Snippets[0].Text != "Run the ordinary validation first.\nThen publish the result." {
		t.Fatalf("text = %q", got.Snippets[0].Text)
	}
}

func TestAtMostTwoDistinctPassages(t *testing.T) {
	t.Parallel()
	content := "#+title: Search\n* One\nalpha substantial evidence here.\n\n" +
		"* Two\nalpha substantial evidence there.\n\n* Three\nalpha only.\n"
	got := evidence.Extract(content, "alpha substantial evidence")
	if len(got.Snippets) != 2 {
		t.Fatalf("got %d snippets: %#v", len(got.Snippets), got.Snippets)
	}
	if got.Snippets[0].StartLine != 3 || got.Snippets[1].StartLine != 6 {
		t.Fatalf("unexpected deterministic order: %#v", got.Snippets)
	}
}

func TestBestMatchThenOpeningContext(t *testing.T) {
	t.Parallel()
	content := "#+title: Design\n\nThis note compares retrieval approaches.\n\n" +
		"It records the experiment setup.\n\n* Results\n語彙ギャップを埋める候補と結果を比較した。\n"
	got := evidence.Extract(content, "語彙 ギャップ 候補 結果")
	if len(got.Snippets) != 2 {
		t.Fatalf("snippets = %#v", got.Snippets)
	}
	if got.Snippets[0].Kind != "match" || got.Snippets[0].StartLine != 8 {
		t.Fatalf("best match = %#v", got.Snippets[0])
	}
	if got.Snippets[1].Kind != "context" || got.Snippets[1].StartLine != 3 ||
		got.Snippets[1].EndLine != 5 ||
		got.Snippets[1].Text != "This note compares retrieval approaches.\n\nIt records the experiment setup." {
		t.Fatalf("opening context = %#v", got.Snippets[1])
	}
}

func TestOpeningDoesNotCutAnAdditionalParagraph(t *testing.T) {
	t.Parallel()
	content := "#+title: Design\n\nFirst introduction.\nSecond line.\n\n" +
		"additional one\nadditional two\nadditional three\nadditional four\nadditional five\n" +
		"additional six\nadditional seven\nadditional eight\nadditional nine\nadditional ten\n\n" +
		"* Result\nmatched result\n"
	got := evidence.Extract(content, "matched result")
	if len(got.Snippets) != 2 || got.Snippets[1].StartLine != 3 ||
		got.Snippets[1].EndLine != 4 ||
		got.Snippets[1].Text != "First introduction.\nSecond line." {
		t.Fatalf("opening context cut another paragraph: %#v", got.Snippets)
	}
}

func TestOpeningOverlapUsesSecondBestMatch(t *testing.T) {
	t.Parallel()
	content := "#+title: Design\n\nprimary retrieval evidence.\n\n* Detail\nsecondary retrieval evidence.\n"
	got := evidence.Extract(content, "retrieval evidence")
	if len(got.Snippets) != 2 || got.Snippets[0].StartLine != 3 || got.Snippets[1].StartLine != 6 {
		t.Fatalf("snippets = %#v", got.Snippets)
	}
	for _, snippet := range got.Snippets {
		if snippet.Kind != "match" {
			t.Fatalf("kind = %q", snippet.Kind)
		}
	}
}

func TestLongJapaneseComponentDoesNotDominateDistinctComponents(t *testing.T) {
	t.Parallel()
	content := "#+title: Search\n* Topic\nオブザーバビリティについて説明する。\n\n" +
		"* Decision\n設計の差を調べる。\n"
	got := evidence.Extract(content, "オブザーバビリティ 設計 差")
	if len(got.Snippets) == 0 || got.Snippets[0].StartLine != 6 {
		t.Fatalf("long topic component dominated: %#v", got.Snippets)
	}
}

func TestHeadingAndBodyContributeOneCoverageSet(t *testing.T) {
	t.Parallel()
	content := "#+title: Search\n* 分散処理\n差分を比較する。\n\n" +
		"* Other\n分散処理だけを説明する。\n"
	got := evidence.Extract(content, "分散処理 差分")
	if len(got.Snippets) == 0 || got.Snippets[0].StartLine != 3 {
		t.Fatalf("heading and body were not combined: %#v", got.Snippets)
	}
}

func TestRareDiscriminatorBreaksEqualCoverageTie(t *testing.T) {
	t.Parallel()
	content := "#+title: 保存方式\n* Motivation\n保存方式を検証する。\n\n" +
		"* Operation\n検証して精査する。\n"
	got := evidence.Extract(content, "保存方式 検証 精査")
	if len(got.Snippets) == 0 || got.Snippets[0].StartLine != 6 {
		t.Fatalf("discriminator did not break tie: %#v", got.Snippets)
	}
}

func TestIncidentalOneGramDoesNotBeatFullDiscriminator(t *testing.T) {
	t.Parallel()
	content := "#+title: project\n* オブザーバビリティ\nproject と小さな差を記す。\n\n" +
		"* オブザーバビリティ\n差分を確認する。\n"
	got := evidence.Extract(content, "project オブザーバビリティ 差分")
	if len(got.Snippets) == 0 || got.Snippets[0].StartLine != 6 {
		t.Fatalf("incidental one-gram won: %#v", got.Snippets)
	}
}

func TestEveryLexicalSegmentInPunctuatedQueryCounts(t *testing.T) {
	t.Parallel()
	content := "#+title: BM25 Search\n* Broad\nBM25 is described here.\n\n" +
		"* Specific\nPRF is the rejected alternative.\n"
	got := evidence.Extract(content, "BM25/PRF")
	if len(got.Snippets) == 0 || got.Snippets[0].StartLine != 6 {
		t.Fatalf("slash-separated component was discarded: %#v", got.Snippets)
	}
}

func TestMixedLatinJapaneseQueryKeepsBothSegments(t *testing.T) {
	t.Parallel()
	content := "#+title: Search\n* Broad\nnotesearch is described here.\n\n" +
		"* Decision\n却下理由を記録する。\n"
	got := evidence.Extract(content, "notesearchの却下理由")
	if len(got.Snippets) != 2 || got.Snippets[1].StartLine != 6 {
		t.Fatalf("adjacent lexical segment was discarded: %#v", got.Snippets)
	}
}

func TestDuplicatePassagesAreReturnedOnce(t *testing.T) {
	t.Parallel()
	content := "#+title: Search\n* One\nexact duplicate evidence.\n\n" +
		"* Two\nexact duplicate evidence.\n"
	got := evidence.Extract(content, "duplicate evidence")
	if len(got.Snippets) != 1 {
		t.Fatalf("got duplicate snippets: %#v", got.Snippets)
	}
}

func TestFallbackAndMetadataOnly(t *testing.T) {
	t.Parallel()
	got := evidence.Extract("#+title: Slug Match\n* Start\nA readable beginning.\n\nLater text.\n", "filename-only-term")
	if len(got.Snippets) != 1 || got.Snippets[0].Text != "A readable beginning." {
		t.Fatalf("fallback = %#v", got.Snippets)
	}
	if got.Snippets[0].Kind != "context" {
		t.Fatalf("fallback kind = %q", got.Snippets[0].Kind)
	}

	only := evidence.Extract(":PROPERTIES:\n:ID: x\n:END:\n#+title: Metadata only\n", "metadata")
	if only.Metadata.ID != "x" || only.Metadata.Title != "Metadata only" || len(only.Snippets) != 0 {
		t.Fatalf("metadata-only result = %#v", only)
	}

	empty := evidence.Extract("", "anything")
	if empty.Metadata != (evidence.Metadata{}) || len(empty.Snippets) != 0 {
		t.Fatalf("empty = %#v", empty)
	}
}

func TestSelectionIsRepeatable(t *testing.T) {
	t.Parallel()
	content := "#+title: Stable\n\nOpening context.\n\n* A\n候補と結果。\n\n* B\n語彙と結果。\n"
	want := evidence.Extract(content, "語彙 候補 結果")
	for range 100 {
		if got := evidence.Extract(content, "語彙 候補 結果"); !reflect.DeepEqual(got, want) {
			t.Fatalf("selection changed:\nwant %#v\ngot  %#v", want, got)
		}
	}
}

func TestBlockEndMustMatchBeginType(t *testing.T) {
	t.Parallel()
	content := "#+title: Blocks\n#+begin_src text\n#+end_example\n,* Fake\n#+TITLE: Fake title\n" +
		"#+end_src\n* Real\nreal evidence\n"
	got := evidence.Extract(content, "real evidence")
	if got.Metadata.Title != "Blocks" {
		t.Fatalf("metadata = %#v", got.Metadata)
	}
	if len(got.Snippets) == 0 || strings.Join(got.Snippets[0].Headings, "/") != "Real" {
		t.Fatalf("mismatched block end leaked heading: %#v", got.Snippets)
	}
}

func TestMalformedLeadingDrawerDoesNotConsumeHeadingProperties(t *testing.T) {
	t.Parallel()
	content := ":PROPERTIES:\n:ID: malformed-file\n#+title: Not valid metadata\n* Node\n" +
		":PROPERTIES:\n:ID: heading-id\n:STATUS: stable\n:END:\nbody\n"
	got := evidence.Extract(content, "body")
	if got.Metadata != (evidence.Metadata{}) {
		t.Fatalf("malformed metadata leaked: %#v", got.Metadata)
	}
	if len(got.Snippets) != 1 || got.Snippets[0].Text != "body" ||
		strings.Join(got.Snippets[0].Headings, "/") != "Node" {
		t.Fatalf("body lost after malformed drawer: %#v", got.Snippets)
	}
}

func TestLongLineAndOneGramNoise(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("長", 100_000)
	content := "#+title: Long\n* Weak\n抽という一文字だけです。\n\n* Strong\n根拠抽出を正確に行います。" +
		long + "\n"
	got := evidence.Extract(content, "根拠抽出")
	if len(got.Snippets) == 0 || got.Snippets[0].StartLine != 6 {
		t.Fatalf("substantial match did not win: %#v", got.Snippets)
	}
}
