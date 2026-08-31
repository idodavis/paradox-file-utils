// detect_test.go covers version reading from launcher-settings.json.

package game

import (
	"os"
	"path/filepath"
	"testing"
)

func writeLauncher(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "launcher"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "launcher", "launcher-settings.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestReadGameVersion(t *testing.T) {
	if got := ReadGameVersion(writeLauncher(t,
		`{ "rawVersion": "1.12.5", "version": "Scythe v1.12.5" }`)); got != "1.12.5" {
		t.Fatalf("version = %q want 1.12.5", got)
	}
	if got := ReadGameVersion(t.TempDir()); got != "" {
		t.Fatalf("missing file version = %q want empty", got)
	}
	if got := ReadGameVersion(writeLauncher(t,
		`{ "version": "1.13.11 (Chamomile)" }`)); got != "1.13.11" {
		t.Fatalf("codename strip = %q want 1.13.11", got)
	}
}
