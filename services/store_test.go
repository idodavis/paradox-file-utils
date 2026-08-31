// store_test.go covers JSON config round-trip, version-mismatch wipe, and run cap.

package services

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s := newStore(path)
	if err := s.Mutate(func(c *Config) error {
		c.Installs = append(c.Installs, GameInstall{ID: "i1", Name: "CK3"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	s2 := newStore(path)
	var n int
	s2.Read(func(c *Config) { n = len(c.Installs) })
	if n != 1 {
		t.Fatalf("got %d installs", n)
	}

	stale := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(stale, []byte(`{"formatVersion":99,"installs":[{"id":"x"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var wiped int
	newStore(stale).Read(func(c *Config) { wiped = len(c.Installs) })
	if wiped != 0 {
		t.Fatalf("want empty, got %d", wiped)
	}

	capStore := newStore(filepath.Join(t.TempDir(), "c.json"))
	if err := capStore.Mutate(func(c *Config) error {
		for i := 0; i < 25; i++ {
			c.PatchRuns = append(c.PatchRuns, PatchRun{ID: fmt.Sprintf("%d", i)})
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var last string
	capStore.Read(func(c *Config) {
		n = len(c.PatchRuns)
		last = c.PatchRuns[n-1].ID
	})
	if n != 20 || last != "24" {
		t.Fatalf("n=%d last=%s", n, last)
	}
}
