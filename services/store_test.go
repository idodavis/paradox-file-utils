// store_test.go covers JSON config round-trip and version-mismatch wipe.

package services

import (
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
}
