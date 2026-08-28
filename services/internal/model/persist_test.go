// persist_test.go covers Cache save/load round-tripping and the format-version
// rejection that forces a rescan instead of migrating stale files.

package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ck3-1.0.json")
	in := &Cache{
		FormatVersion: CacheFormatVersion,
		GameID:        "ck3",
		GameVersion:   "1.0",
		Defs:          []Def{{Type: "trait", Key: "brave", Path: "a.txt", Line: 3}},
		LocEnglish:    map[string]string{"k": "v"},
		Vocabulary:    []string{"add_gold"},
	}
	if err := SaveCacheFile(path, in); err != nil {
		t.Fatalf("SaveCacheFile: %v", err)
	}
	out, err := LoadCacheFile(path)
	if err != nil {
		t.Fatalf("LoadCacheFile: %v", err)
	}
	if len(out.Defs) != 1 || out.Defs[0].Key != "brave" || out.LocEnglish["k"] != "v" {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestLoadRejectsWrongVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stale.json")
	if err := os.WriteFile(path, []byte(`{"formatVersion":999,"gameId":"ck3"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCacheFile(path); err == nil {
		t.Fatalf("expected version-mismatch error, got nil")
	}
}

func TestLoadRejectsMissingFormatVersion(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "old-cache.json")
	if err := os.WriteFile(cachePath, []byte(`{"gameId":"ck3"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCacheFile(cachePath); err == nil {
		t.Fatal("expected missing cache formatVersion to fail")
	}
	indexPath := filepath.Join(dir, "old-index.json")
	if err := os.WriteFile(indexPath, []byte(`{"workspaceId":"ws"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadIndexFile(indexPath); err == nil {
		t.Fatal("expected missing index formatVersion to fail")
	}
}
