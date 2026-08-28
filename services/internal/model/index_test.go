// index_test.go covers the workspace extractor: event-id defs, loc references, and
// LookupAll resolving a key from a parent mod and from vanilla.

package model

import (
	"os"
	"path/filepath"
	"testing"
)

// writeMod lays a file under a mod root and returns nothing (helper for fixtures).
func writeMod(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIndexFileEventAndLocRef(t *testing.T) {
	content := `namespace = test

test.1 = {
	title = test.1.t
	immediate = { trigger_event = test.2 }
}
`
	defs, refs, edges, _ := IndexFile("ck3", filepath.FromSlash("events/x.txt"), "events/x.txt", content, "modA")

	if !findDef(defs, "event", "test.1") {
		t.Errorf("missing event def test.1: %v", defs)
	}
	if defs[0].Origin != "modA" {
		t.Errorf("def origin = %q, want modA", defs[0].Origin)
	}
	var locRef bool
	for _, r := range refs {
		if r.Kind == "loc" && r.Key == "test.1.t" {
			locRef = true
		}
	}
	if !locRef {
		t.Errorf("missing loc ref test.1.t: %v", refs)
	}
	var edge bool
	for _, e := range edges {
		if e.Via == "trigger_event" && e.From == "test.1" && e.To == "test.2" {
			edge = true
		}
	}
	if !edge {
		t.Errorf("missing trigger_event edge test.1->test.2: %v", edges)
	}
}

func TestLookupAllParentModAndVanilla(t *testing.T) {
	parent := t.TempDir()
	child := t.TempDir()
	writeMod(t, parent, "common/traits/00.txt", "brave = { category = personality }\n")
	writeMod(t, child, "common/traits/01.txt", "craven = { category = personality }\n")

	mods := []ModInput{
		{Origin: "parent", Root: parent, Order: 0},
		{Origin: "child", Root: child, Order: 1},
	}
	cache := &Cache{Defs: []Def{{Type: "trait", Key: "vanilla_trait", Path: "v.txt"}}}
	idx := BuildIndex("ws1", "ck3", mods, cache)

	// A key defined only by the parent mod resolves from the child's perspective.
	if got := idx.LookupAll("brave", cache); len(got) == 0 || got[0].Origin != "parent" {
		t.Errorf("LookupAll(brave) = %v, want parent def", got)
	}
	// A vanilla-only key falls through to the cache.
	if got := idx.LookupAll("vanilla_trait", cache); len(got) != 1 || got[0].Origin != "" {
		t.Errorf("LookupAll(vanilla_trait) = %v, want one vanilla def", got)
	}
}
