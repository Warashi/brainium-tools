package evidence_test

import (
	"slices"
	"testing"

	"github.com/Warashi/brainium-tools/internal/notesearch/evidence"
)

func TestHeadlinesEndSectionsEvenInsideBlockMarkers(t *testing.T) {
	for _, end := range []string{"", "#+end_src\n"} {
		t.Run(end, func(t *testing.T) {
			content := "* Alpha\n#+begin_src go\ncode alpha\n* Beta\nneedle evidence\n" + end
			got := evidence.Extract(content, "needle")
			if len(got.Snippets) == 0 {
				t.Fatal("missing evidence")
			}
			first := got.Snippets[0]
			if !slices.Equal(first.Headings, []string{"Beta"}) || first.Text != "needle evidence" || first.StartLine != 5 {
				t.Fatalf("headline inherited stale block ancestry: %+v", first)
			}
		})
	}
}

func TestEscapedHeadlinesStayInsideBlocks(t *testing.T) {
	content := "* Alpha\n#+begin_example\n,* Beta\nneedle evidence\n#+end_example\n"
	got := evidence.Extract(content, "needle")
	if len(got.Snippets) == 0 || !slices.Equal(got.Snippets[0].Headings, []string{"Alpha"}) {
		t.Fatalf("escaped headline changed ancestry: %+v", got.Snippets)
	}
}

func TestBlockMarkersCannotCarryFileMetadataPastHeadline(t *testing.T) {
	content := "#+begin_src text\n* Subtree\n#+end_src\n#+title: Not a file title\nneedle evidence\n"
	got := evidence.Extract(content, "needle")
	if got.Metadata.Title != "" {
		t.Fatalf("subtree title leaked into file metadata: %+v", got.Metadata)
	}
}
