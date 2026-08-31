// persist_test.go covers VanillaCache / loc sidecar save-load and version rejection.

package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vanilla-inst-1_0.json")
	in := &VanillaCache{
		FormatVersion: CacheFormatVersion,
		InstallID:     "inst",
		GameID:        "ck3",
		GameVersion:   "1.0",
		Defs:          []Def{{Type: "trait", Key: "brave", Path: "a.txt", Line: 3}},
		Vocabulary:    []string{"add_gold"},
	}
	if err := SaveCacheFile(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := LoadCacheFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Defs) != 1 || out.Defs[0].Key != "brave" || out.InstallID != "inst" {
		t.Errorf("round-trip mismatch: %+v", out)
	}

	for _, tt := range []struct{ name, body string }{
		{"wrong version", `{"formatVersion":999,"gameId":"ck3"}`},
		{"missing version", `{"gameId":"ck3"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "stale.json")
			if err := os.WriteFile(p, []byte(tt.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadCacheFile(p); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	thin := filepath.Join(t.TempDir(), "thin.json")
	body := fmt.Sprintf(`{"formatVersion":%d,"gameId":"ck3","gameVersion":"1"}`, CacheFormatVersion)
	if err := os.WriteFile(thin, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadCacheFile(thin)
	if err != nil {
		t.Fatal(err)
	}
	c.FieldDocs["k"] = "v"
	c.FieldDocsByKind["event"] = map[string]string{"type": "t"}
	if c.FieldDocs["k"] != "v" || c.FieldDocsByKind["event"]["type"] != "t" {
		t.Fatalf("nil maps not initialized: %+v", c)
	}

	prep := &VanillaCache{
		Effects:         []string{"add_gold"},
		Triggers:        []string{"is_adult"},
		Structures:      map[string][]string{"event": {"immediate"}},
		FieldDocs:       map[string]string{"Type": "global"},
		FieldDocsByKind: map[string]map[string]string{"event": {"Title": "kind"}},
	}
	PrepareCache(prep)
	if !prep.effectSet["add_gold"] || !prep.triggerSet["is_adult"] ||
		!prep.structureSets["event"]["immediate"] {
		t.Fatalf("sets not built")
	}
	if prep.FieldDocs["type"] != "global" || prep.FieldDocsByKind["event"]["title"] != "kind" {
		t.Fatalf("docs not normalized: %+v %+v", prep.FieldDocs, prep.FieldDocsByKind)
	}

	p, err := vanillaJSON("vanilla", "inst-1", "1.19.0")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "vanilla-inst-1-1_19_0.json" {
		t.Fatalf("cache path = %s", p)
	}
	dir := t.TempDir()
	legacy := filepath.Join(dir, "cache-ws-a.json")
	keep := filepath.Join(dir, "vanilla-x-1.json")
	if err := os.WriteFile(legacy, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	dropLegacyWorkspaceCaches(dir)
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatal("legacy cache-*.json should be removed")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("vanilla file should remain: %v", err)
	}

	locIn := &VanillaLoc{
		FormatVersion: LocFormatVersion,
		Sites:         map[string]LocEntry{"k": {File: "a.yml", Line: 1, Value: "v"}},
	}
	locPath := filepath.Join(t.TempDir(), "sidecar.json")
	if err := SaveJSON(locPath, locIn); err != nil {
		t.Fatal(err)
	}
	got, err := LoadJSON[VanillaLoc](locPath, LocFormatVersion)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sites["k"].Value != "v" || got.Sites["k"].Line != 1 {
		t.Fatalf("loc sidecar = %+v", got)
	}
}
