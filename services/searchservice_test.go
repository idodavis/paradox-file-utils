// searchservice_test.go covers Find-in-Files globs: include, exclude, and folder isolation.
package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSearchTree(t *testing.T, root string) {
	t.Helper()
	txt := filepath.Join(root, "events", "hit.txt")
	yml := filepath.Join(root, "events", "hit.yml")
	dds := filepath.Join(root, "gfx", "skip.dds")
	other := filepath.Join(root, "common", "other.txt")
	for _, p := range []string{txt, yml, dds, other} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(txt, []byte("title = unique_search_token\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(yml, []byte("unique_search_token yml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dds, []byte("unique_search_token in a dds\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("unique_search_token common\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTextSearchIncludeEvents(t *testing.T) {
	t.Parallel()
	if _, err := rgPath(); err != nil {
		t.Skip(err.Error())
	}
	root := t.TempDir()
	writeSearchTree(t, root)
	ss := &SearchService{}
	res, err := ss.TextSearch(context.Background(), TextSearchQuery{
		Pattern: "unique_search_token",
		Folders: []TextSearchFolder{{
			Path:     root,
			Includes: []string{"**/events/**"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 2 {
		t.Fatalf("include events hits = %+v", res.Hits)
	}
	for _, h := range res.Hits {
		if !strings.Contains(filepath.ToSlash(h.Path), "/events/") {
			t.Fatalf("leaked path %s", h.Path)
		}
	}
}

func TestTextSearchExcludeTxt(t *testing.T) {
	t.Parallel()
	if _, err := rgPath(); err != nil {
		t.Skip(err.Error())
	}
	root := t.TempDir()
	writeSearchTree(t, root)
	ss := &SearchService{}
	res, err := ss.TextSearch(context.Background(), TextSearchQuery{
		Pattern: "unique_search_token",
		Folders: []TextSearchFolder{{
			Path:     root,
			Excludes: []string{"*.txt"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range res.Hits {
		if strings.HasSuffix(filepath.ToSlash(h.Path), ".txt") {
			t.Fatalf("txt not excluded: %s", h.Path)
		}
	}
	if len(res.Hits) == 0 {
		t.Fatal("expected yml/dds hits after excluding txt")
	}
}

func TestTextSearchFoldersDoNotLeakGlobs(t *testing.T) {
	t.Parallel()
	if _, err := rgPath(); err != nil {
		t.Skip(err.Error())
	}
	a := t.TempDir()
	b := t.TempDir()
	writeSearchTree(t, a)
	writeSearchTree(t, b)
	ss := &SearchService{}
	res, err := ss.TextSearch(context.Background(), TextSearchQuery{
		Pattern: "unique_search_token",
		Folders: []TextSearchFolder{
			{Path: a, Includes: []string{"**/events/**"}},
			{Path: b, Includes: []string{"**/common/**"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var aHits, bHits int
	for _, h := range res.Hits {
		p := filepath.ToSlash(h.Path)
		switch {
		case strings.HasPrefix(p, filepath.ToSlash(a)):
			aHits++
			if !strings.Contains(p, "/events/") {
				t.Fatalf("folder a leaked %s", h.Path)
			}
		case strings.HasPrefix(p, filepath.ToSlash(b)):
			bHits++
			if !strings.Contains(p, "/common/") {
				t.Fatalf("folder b leaked %s", h.Path)
			}
		}
	}
	if aHits == 0 || bHits == 0 {
		t.Fatalf("expected hits in both folders: a=%d b=%d %+v", aHits, bHits, res.Hits)
	}
}

func TestTextSearchEmptyIncludeUsesPassedExcludes(t *testing.T) {
	t.Parallel()
	if _, err := rgPath(); err != nil {
		t.Skip(err.Error())
	}
	root := t.TempDir()
	writeSearchTree(t, root)
	ss := &SearchService{}
	res, err := ss.TextSearch(context.Background(), TextSearchQuery{
		Pattern: "unique_search_token",
		Folders: []TextSearchFolder{{
			Path:     root,
			Excludes: []string{"*.dds"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range res.Hits {
		if strings.HasSuffix(filepath.ToSlash(h.Path), ".dds") {
			t.Fatalf("dds not excluded: %s", h.Path)
		}
	}
	if len(res.Hits) == 0 {
		t.Fatal("expected script hits")
	}
}
