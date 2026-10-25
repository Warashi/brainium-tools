package links

import (
	"slices"
	"testing"
	"testing/fstest"
)

func note(id, title, body string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(":PROPERTIES:\n:ID:       " + id + "\n:END:\n#+title: " + title + "\n\n" + body)}
}

func TestCollectFollowsIDLinksBothWays(t *testing.T) {
	fsys := fstest.MapFS{
		"plugin.org": note("11111111-1111-1111-1111-111111111111", "Plugin agent",
			"[[id:22222222-2222-2222-2222-222222222222][自作 agent]] と\n[[id:33333333-3333-3333-3333-333333333333]] を前提にする。\n"),
		"self-made.org": note("22222222-2222-2222-2222-222222222222", "Coding Agent を自作したい", ""),
		"go.org": note("33333333-3333-3333-3333-333333333333", "Go generics",
			"[[id:22222222-2222-2222-2222-222222222222][自作]]\n"),
	}
	g, err := Collect(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := g.To("plugin.org"), []Note{{"go.org", "Go generics"}, {"self-made.org", "Coding Agent を自作したい"}}; !slices.Equal(got, want) {
		t.Errorf("To(plugin.org) = %v, want %v", got, want)
	}
	if got, want := g.From("self-made.org"), []Note{{"go.org", "Go generics"}, {"plugin.org", "Plugin agent"}}; !slices.Equal(got, want) {
		t.Errorf("From(self-made.org) = %v, want %v", got, want)
	}
}

func TestCollectSkipsSelfUnknownAndDuplicateLinks(t *testing.T) {
	fsys := fstest.MapFS{
		"a.org": note("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "A",
			"[[id:AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA][自分]]\n[[id:bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb][B]] [[id:bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb][B again]]\n[[id:cccccccc-cccc-cccc-cccc-cccccccccccc][inbox の capture]]\n"),
		"b.org": note("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "B", ""),
	}
	g, err := Collect(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := g.To("a.org"), []Note{{"b.org", "B"}}; !slices.Equal(got, want) {
		t.Errorf("To(a.org) = %v, want %v", got, want)
	}
	if got := g.From("a.org"); len(got) != 0 {
		t.Errorf("From(a.org) = %v, want none", got)
	}
}
