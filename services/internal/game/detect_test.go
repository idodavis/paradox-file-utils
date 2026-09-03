// detect_test.go covers version reading from launcher-settings.json.

package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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

func TestUserDataDirGameDataPath(t *testing.T) {
	ud := t.TempDir()
	raw, err := json.Marshal(map[string]string{"gameDataPath": ud})
	if err != nil {
		t.Fatal(err)
	}
	install := writeLauncher(t, string(raw))
	got := UserDataDir("ck3", install)
	if got != filepath.Clean(ud) {
		t.Fatalf("UserDataDir = %q want %q", got, ud)
	}
}

func TestUserDataDirXDGBeatsMissingDocuments(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG is Linux")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	xdgHome := filepath.Join(home, "xdg")
	t.Setenv("XDG_DATA_HOME", xdgHome)
	want := filepath.Join(xdgHome, "Paradox Interactive", "Crusader Kings III")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	got := UserDataDir("ck3", "")
	if got != want {
		t.Fatalf("UserDataDir = %q want %q", got, want)
	}
}

func TestUserDataDirProtonWhenXDGAbsent(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Proton prefixes are Linux")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	want := filepath.Join(
		home, ".local", "share", "Steam", "steamapps", "compatdata", "1158310",
		"pfx", "drive_c", "users", "steamuser",
		"Documents", "Paradox Interactive", "Crusader Kings III",
	)
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	got := UserDataDir("ck3", "")
	if got != want {
		t.Fatalf("UserDataDir = %q want %q", got, want)
	}
}
