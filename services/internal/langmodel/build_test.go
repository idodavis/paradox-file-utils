package langmodel

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildExtractsDefsAndEdges(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "events.txt")
	content := "namespace = test\n" +
		"foo = {\n\tid = foo\n\ttrigger_event = bar\n}\n" +
		"bar = {\n\tid = bar\n}\n"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Build(context.Background(), "ws-test", "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Definitions) < 2 {
		t.Fatalf("expected >=2 defs, got %d", len(m.Definitions))
	}
	keys := map[string]bool{}
	for _, d := range m.Definitions {
		keys[d.Key] = true
	}
	if !keys["foo"] || !keys["bar"] {
		t.Fatalf("missing foo/bar defs: %#v", keys)
	}
	foundEdge := false
	for _, e := range m.Edges {
		if e.FromKey == "foo" && e.ToKey == "bar" {
			foundEdge = true
			break
		}
	}
	if !foundEdge {
		t.Fatalf("expected foo->bar edge, got %#v", m.Edges)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	m := &Model{
		WorkspaceID: "roundtrip-" + filepath.Base(t.TempDir()),
		Definitions: []Def{{Type: "object", Key: "k", FilePath: "a.txt", Line: 1}},
		Edges:       []Edge{{FromKey: "k", ToKey: "k", EdgeType: "ref"}},
	}
	if err := Save(m); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if p, err := ModelPath(m.WorkspaceID); err == nil {
			_ = os.Remove(p)
		}
	})
	loaded, err := Load(m.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Definitions) != 1 || loaded.Definitions[0].Key != "k" {
		t.Fatalf("bad load: %#v", loaded)
	}
}
