// settingsservice_test.go covers PMT-owned path checks for ResetData wipes.

package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOwnedByPMT(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	inside := filepath.Join(root, "workspaces", "ws1")
	if !ownedByPMT(root, inside) {
		t.Fatal("want subdirectory owned")
	}
	if ownedByPMT(root, root) {
		t.Fatal("root itself must not be owned")
	}
	if ownedByPMT(root, "") {
		t.Fatal("empty path")
	}
	outside := t.TempDir()
	if ownedByPMT(root, outside) {
		t.Fatal("foreign temp dir must not be owned")
	}
	escape := filepath.Join(root, "..", filepath.Base(outside))
	if ownedByPMT(root, escape) {
		t.Fatal("path escaping root must not be owned")
	}
}

func TestRemoveOwnedDirs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	owned := filepath.Join(root, "workspaces", "ws1")
	if err := os.MkdirAll(owned, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(owned, "x.txt")
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	keep := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(keep, []byte("k"), 0o644); err != nil {
		t.Fatal(err)
	}
	removeOwnedDirs(root, []string{owned, outside, "", owned})
	if _, err := os.Stat(owned); !os.IsNotExist(err) {
		t.Fatalf("owned dir still present: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("outside file was deleted: %v", err)
	}
}
