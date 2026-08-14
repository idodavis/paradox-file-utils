package langmodel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
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

	m, err := Build(context.Background(), "ws-test", "", []string{dir}, nil)
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

func TestBuildReportsProgress(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x = { y = 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var phases []string
	_, err := Build(context.Background(), "ws-progress", "", []string{dir},
		func(phase string, _done, _total int, _message string) {
			phases = append(phases, phase)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(phases) == 0 {
		t.Fatal("expected progress callbacks")
	}
}

func TestBuildCancelDuringWalk(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 40; i++ {
		sub := filepath.Join(dir, "nested", fmt.Sprintf("%02d", i))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sub, "f.txt"), []byte("k = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Build(ctx, "ws-cancel", "", []string{dir}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestBuildCancelStopsQuickly(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 200; i++ {
		name := filepath.Join(dir, fmt.Sprintf("%03d.txt", i))
		body := fmt.Sprintf("def_%d = {\n\tother = x\n}\n", i)
		if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := Build(ctx, "ws-cancel-fast", "", []string{dir}, nil)
		done <- err
	}()
	time.Sleep(5 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("build did not stop after cancel")
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
