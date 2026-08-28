// Merge goldens: TopAssignments slice contract, PREFER comments, GUI template, EU5 keys.
package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseFileObjectsPreferComment(t *testing.T) {
	dir := t.TempDir()
	src := "foo = {\n\tid = 1\n}\n# PREFER: B\nbar = {\n\tid = 2\n}\n"
	path := writeTemp(t, dir, "script.txt", src)
	objs, _, err := parseFileObjects(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 2 {
		t.Fatalf("objects %d keys %#v", len(objs), mergeKeys(objs))
	}
	if objs[0].Key != "foo" || objs[1].Key != "bar" {
		t.Fatalf("keys %#v", mergeKeys(objs))
	}
	if objs[1].PreferSide != "B" {
		t.Fatalf("prefer %q raw %q", objs[1].PreferSide, objs[1].RawText)
	}
	if !strings.Contains(objs[1].RawText, "# PREFER: B") {
		t.Fatalf("comment missing from gap: %q", objs[1].RawText)
	}
	got := objs[0].RawText + objs[1].RawText
	if got != src {
		t.Fatalf("reconstruct\n got %q\nwant %q", got, src)
	}
}

func TestParseFileObjectsGUITemplate(t *testing.T) {
	dir := t.TempDir()
	src := "template Foo {\n\tsize = { 10 20 }\n}\n"
	path := writeTemp(t, dir, "window.gui", src)
	objs, _, err := parseFileObjects(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 1 {
		t.Fatalf("want 1 object, got %d keys %#v", len(objs), mergeKeys(objs))
	}
	if objs[0].Key != "Foo" {
		t.Fatalf("key %q", objs[0].Key)
	}
	if objs[0].RawText != src {
		t.Fatalf("split or reformatted: %q", objs[0].RawText)
	}
}

func TestMergeInjectMatchesBareKey(t *testing.T) {
	dir := t.TempDir()
	pathA := writeTemp(t, dir, "a.txt", "INJECT:foo = {\n\tx = 1\n}\n")
	pathB := writeTemp(t, dir, "b.txt", "foo = {\n\tx = 2\n}\n")
	m := &MergeService{}
	mr, err := m.performMerge(pathA, pathB, MergerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.ResolvedConflicts) != 1 {
		t.Fatalf("conflicts %#v changed %#v", mr.ResolvedConflicts, mr.EntriesChanged)
	}
	if mr.ResolvedConflicts[0].Key != "foo" {
		t.Fatalf("key %q", mr.ResolvedConflicts[0].Key)
	}
	if !strings.Contains(mr.Content, "x = 1") {
		t.Fatalf("default A lost: %q", mr.Content)
	}
	if strings.Contains(mr.Content, "x = 2") {
		t.Fatalf("took B: %q", mr.Content)
	}
}

func TestMergePreferBKeepsCommentSlice(t *testing.T) {
	dir := t.TempDir()
	pathA := writeTemp(t, dir, "a.txt",
		"foo = { x = 1 }\n# PREFER: B\nbar = { y = 1 }\n")
	pathB := writeTemp(t, dir, "b.txt",
		"foo = { x = 9 }\n# PREFER: B\nbar = { y = 9 }\n")
	m := &MergeService{}
	mr, err := m.performMerge(pathA, pathB, MergerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mr.Content, "# PREFER: B") {
		t.Fatalf("lost PREFER comment: %q", mr.Content)
	}
	if !strings.Contains(mr.Content, "y = 9") {
		t.Fatalf("PREFER B not applied to bar: %q", mr.Content)
	}
}

func TestMergeGUITemplateConflict(t *testing.T) {
	dir := t.TempDir()
	pathA := writeTemp(t, dir, "a.gui", "template Foo {\n\tsize = { 10 20 }\n}\n")
	pathB := writeTemp(t, dir, "b.gui", "template Foo {\n\tsize = { 30 40 }\n}\n")
	m := &MergeService{}
	mr, err := m.performMerge(pathA, pathB, MergerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(mr.ResolvedConflicts) != 1 || mr.ResolvedConflicts[0].Key != "Foo" {
		t.Fatalf("conflicts %#v", mr.ResolvedConflicts)
	}
	if strings.Contains(mr.Content, "30 40") {
		t.Fatalf("default A should win: %q", mr.Content)
	}
	if !strings.Contains(mr.Content, "template Foo") {
		t.Fatalf("template keyword dropped: %q", mr.Content)
	}
}

func mergeKeys(objs []scriptObject) []string {
	out := make([]string, len(objs))
	for i, o := range objs {
		out[i] = o.Key
	}
	return out
}
