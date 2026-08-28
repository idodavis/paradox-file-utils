// detect_test.go covers version reading and mod-root recognition per game.

package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadGameVersion(t *testing.T) {
	dir := t.TempDir()
	body := `{ "rawVersion": "1.12.5", "version": "Scythe v1.12.5" }`
	if err := os.WriteFile(filepath.Join(dir, "launcher-settings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReadGameVersion(dir); got != "1.12.5" {
		t.Fatalf("version = %q want 1.12.5", got)
	}
	if got := ReadGameVersion(t.TempDir()); got != "" {
		t.Fatalf("missing file version = %q want empty", got)
	}
}

func TestIsModRoot(t *testing.T) {
	// CK3: descriptor.mod
	ck3 := t.TempDir()
	if IsModRoot("ck3", ck3) {
		t.Fatal("empty dir should not be ck3 mod root")
	}
	if err := os.WriteFile(filepath.Join(ck3, "descriptor.mod"), []byte("name=\"x\""), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsModRoot("ck3", ck3) {
		t.Fatal("descriptor.mod should be a ck3 mod root")
	}

	// Vic3: .metadata/metadata.json
	vic3 := t.TempDir()
	meta := filepath.Join(vic3, ".metadata")
	if err := os.MkdirAll(meta, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(meta, "metadata.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsModRoot("vic3", vic3) {
		t.Fatal("metadata.json should be a vic3 mod root")
	}

	// EU5: stage root dir
	eu5 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(eu5, "in_game"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsModRoot("eu5", eu5) {
		t.Fatal("in_game stage dir should be an eu5 mod root")
	}
}
