package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func internFixture() *VanillaCache {
	const a = `C:\Games\CK3\game\common\traits\00_traits.txt`
	const b = `C:\Games\CK3\game\events\yearly_events.txt`
	return &VanillaCache{
		FormatVersion: CacheFormatVersion,
		InstallID:     "i", GameID: "ck3", GameVersion: "1.0",
		Defs: []Def{
			{Kind: "traits", Key: "brave", Path: a, Line: 1},
			{Kind: "traits", Key: "craven", Path: a, Line: 9},
			{Kind: "event", Key: "y.1", Path: b, Line: 3},
		},
		LocRefs:  []Ref{{Key: "brave", Kind: "loc", Path: a, Line: 2}},
		CallRefs: []Ref{{Key: "my_effect", Kind: "scripted_effects", Path: b, Line: 4}},
		Edges:    []Edge{{From: "y.1", To: "y.2", Via: "trigger_event", Path: b, Line: 5}},
	}
}

// Paths must survive the save/load round trip exactly: they are what go-to
// definition navigates to.
func TestInternPathsRoundTrip(t *testing.T) {
	src := internFixture()
	dir := t.TempDir()
	p := filepath.Join(dir, "vanilla-test.json")
	if err := saveCacheFile(p, src); err != nil {
		t.Fatal(err)
	}
	got, err := loadCacheFile(p)
	if err != nil {
		t.Fatal(err)
	}
	for i, d := range src.Defs {
		if got.Defs[i].Path != d.Path {
			t.Errorf("def %d path = %q, want %q", i, got.Defs[i].Path, d.Path)
		}
		if got.Defs[i].Key != d.Key || got.Defs[i].Line != d.Line {
			t.Errorf("def %d = %+v, want %+v", i, got.Defs[i], d)
		}
	}
	if got.LocRefs[0].Path != src.LocRefs[0].Path {
		t.Errorf("locRef path = %q", got.LocRefs[0].Path)
	}
	if got.CallRefs[0].Path != src.CallRefs[0].Path {
		t.Errorf("callRef path = %q", got.CallRefs[0].Path)
	}
	if got.Edges[0].Path != src.Edges[0].Path {
		t.Errorf("edge path = %q", got.Edges[0].Path)
	}
	if len(got.Paths) != 0 {
		t.Errorf("Paths table leaked into the loaded cache: %v", got.Paths)
	}
}

// Saving must not disturb the in-memory cache: a live session may hold it while
// the scan writes.
func TestInternPathsDoesNotMutateSource(t *testing.T) {
	src := internFixture()
	want := src.Defs[0].Path
	dir := t.TempDir()
	if err := saveCacheFile(filepath.Join(dir, "v.json"), src); err != nil {
		t.Fatal(err)
	}
	if src.Defs[0].Path != want {
		t.Errorf("source mutated: %q, want %q", src.Defs[0].Path, want)
	}
	if len(src.Paths) != 0 {
		t.Errorf("source gained a Paths table: %v", src.Paths)
	}
}

// The whole point: the install path appears once, not once per record.
func TestInternPathsShrinksFile(t *testing.T) {
	src := internFixture()
	dir := t.TempDir()
	p := filepath.Join(dir, "v.json")
	if err := saveCacheFile(p, src); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	const a = `common\\traits\\00_traits.txt`
	if n := strings.Count(string(raw), a); n != 1 {
		t.Errorf("traits path written %d times, want 1:\n%s", n, raw)
	}
}

// A cache written before interning has no table and must still load.
func TestExpandPathsWithoutTable(t *testing.T) {
	c := internFixture()
	expandPaths(c)
	if !strings.HasSuffix(c.Defs[0].Path, "00_traits.txt") {
		t.Errorf("path mangled without a table: %q", c.Defs[0].Path)
	}
}
