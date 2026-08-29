// persist_test.go covers Cache save/load round-tripping and the format-version
// rejection that forces a rescan instead of migrating stale files.

package model

import (
	"fmt"
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

func TestLoadCacheNilMapsAreWritable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thin.json")
	body := fmt.Sprintf(
		`{"formatVersion":%d,"gameId":"ck3","gameVersion":"1"}`,
		CacheFormatVersion,
	)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadCacheFile(path)
	if err != nil {
		t.Fatal(err)
	}
	c.FieldDocs["k"] = "v"
	c.LocEnglish["a"] = "b"
	c.FieldDocsByKind["event"] = map[string]string{"type": "t"}
	c.LocEnglishSites["a"] = LocSite{File: "x.yml", Line: 1}
	if c.FieldDocs["k"] != "v" || c.LocEnglish["a"] != "b" {
		t.Fatalf("nil maps not initialized: %+v", c)
	}
	if c.FieldDocsByKind["event"]["type"] != "t" || c.LocEnglishSites["a"].Line != 1 {
		t.Fatalf("new maps not initialized: %+v", c)
	}
}

func TestCachePathUsesWorkspaceID(t *testing.T) {
	p, err := cachePath("ws-abc")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "cache-ws-abc.json" {
		t.Fatalf("cache path = %s, want cache-ws-abc.json", p)
	}
}

func TestWorkspaceCacheFilesStayIndependent(t *testing.T) {
	dir := t.TempDir()
	a := &Cache{FormatVersion: CacheFormatVersion, WorkspaceID: "ws-a"}
	b := &Cache{FormatVersion: CacheFormatVersion, WorkspaceID: "ws-b"}
	if err := SaveCacheFile(filepath.Join(dir, "cache-ws-a.json"), a); err != nil {
		t.Fatal(err)
	}
	if err := SaveCacheFile(filepath.Join(dir, "cache-ws-b.json"), b); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ck3-1.16.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := pruneLegacyCaches(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cache-ws-a.json")); err != nil {
		t.Fatalf("ws-a cache missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cache-ws-b.json")); err != nil {
		t.Fatalf("ws-b cache missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ck3-1.16.json")); !os.IsNotExist(err) {
		t.Fatalf("legacy game-version cache still present")
	}
}
