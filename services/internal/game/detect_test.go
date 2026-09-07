// detect_test.go covers version reading from launcher-settings.json.

package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
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

// InstallUpdatedAt decides whether generated script_docs still describe the
// installed build, so it must answer for every game. The obvious source,
// launcher/launcher-settings.json, does not exist on EU5 and was 11 days stale
// on CK3; the binaries are what actually track the update.
func TestInstallUpdatedAt(t *testing.T) {
	if got := InstallUpdatedAt(""); !got.IsZero() {
		t.Errorf("empty path = %v, want zero", got)
	}
	if got := InstallUpdatedAt(filepath.Join(t.TempDir(), "nope")); !got.IsZero() {
		t.Errorf("missing install = %v, want zero", got)
	}

	root := t.TempDir()
	bin := filepath.Join(root, "binaries")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	// A launcher file far in the future must not win: the binaries decide.
	launcher := filepath.Join(root, "launcher")
	if err := os.MkdirAll(launcher, 0o755); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(72 * time.Hour)
	settings := filepath.Join(launcher, "launcher-settings.json")
	if err := os.WriteFile(settings, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(settings, future, future); err != nil {
		t.Fatal(err)
	}

	exe := filepath.Join(bin, "game.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(exe, old, old); err != nil {
		t.Fatal(err)
	}
	got := InstallUpdatedAt(root)
	if got.Sub(old).Abs() > time.Minute {
		t.Fatalf("InstallUpdatedAt = %v, want the binary's %v", got, old)
	}

	// A downgrade rewrites the binaries, so the timestamp moves forward even
	// though the version went backwards. That is the behaviour we want.
	now := time.Now()
	if err := os.Chtimes(exe, now, now); err != nil {
		t.Fatal(err)
	}
	if after := InstallUpdatedAt(root); !after.After(got) {
		t.Errorf("rewriting the binary did not move the timestamp: %v then %v", got, after)
	}

	// No binaries directory: fall back to the install root's own files.
	flat := t.TempDir()
	if err := os.WriteFile(filepath.Join(flat, "game.exe"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if InstallUpdatedAt(flat).IsZero() {
		t.Error("no binaries/ dir should fall back to the install root")
	}
}

// covers the per-game table and the rules that read it:
// folder-to-extract mapping, FIOS, and the EU5 entry-mode prefixes.
func TestAll(t *testing.T) {
	got := All()
	if len(got) != 3 || got[0].ID != "ck3" || got[1].ID != "eu5" || got[2].ID != "vic3" {
		t.Fatalf("All ids: %v %v %v", got[0], got[1], got[2])
	}
	if Get("ck3") != got[0] || OriginVanilla != "vanilla" {
		t.Fatal("Get/OriginVanilla")
	}
	if Get("unknown") != nil {
		t.Error("Get(unknown) should be nil")
	}
}

func TestKeyIdentityStripsEntryModePrefix(t *testing.T) {
	cases := []struct {
		gameID, in, want string
	}{
		{"eu5", "INJECT:foo", "foo"},
		{"eu5", "REPLACE:bar", "bar"},
		{"vic3", "INJECT:ALD", "ALD"},
		{"", "INJECT:foo", "foo"},           // merge: game-agnostic strip
		{"ck3", "INJECT:foo", "INJECT:foo"}, // ck3 has no entry modes -> untouched
		{"eu5", "plain", "plain"},
		{"eu5", "\"quoted\"", "quoted"},
		{"", "  spaced  ", "spaced"},
	}
	for _, c := range cases {
		if got := KeyIdentity(c.gameID, c.in); got != c.want {
			t.Errorf("KeyIdentity(%q, %q) = %q want %q", c.gameID, c.in, got, c.want)
		}
	}
}

// An entry mode is a prefix the game spends on merge semantics, so it never
// names a database. Without this rejection EU5's `INJECT:foo` harvests a kind
// literally called "inject".
func TestParseTypedRejectsEntryModes(t *testing.T) {
	for _, c := range []struct{ gameID, text string }{
		{"eu5", "INJECT:foo"},
		{"eu5", "REPLACE_OR_CREATE:foo"},
		{"vic3", "REPLACE:foo"},
		{"", "INJECT:foo"},
	} {
		if kind, _, ok := ParseTyped(c.gameID, c.text); ok {
			t.Errorf("ParseTyped(%q, %q) = kind %q, want rejected", c.gameID, c.text, kind)
		}
	}
	// CK3 declares no entry modes, so the same prefix stays a plain cite there,
	// and an ordinary typed cite is untouched in every game.
	if kind, id, ok := ParseTyped("ck3", "culture:english"); !ok || kind != "culture" || id != "english" {
		t.Errorf("plain cite broke: kind=%q id=%q ok=%v", kind, id, ok)
	}
}

func TestMatchExtract(t *testing.T) {
	cases := []struct {
		gameID, path string
		kind         string
		mode         ExtractMode
	}{
		{"ck3", "events/my_events.txt", "event", ModeEventID},
		{"ck3", "common/traits/00_traits.txt", "traits", ModeTopLevelKey},
		{"ck3", "gui/window.gui", "gui_type", ModeGUIType},
		{"ck3", "localization/english/foo_l_english.yml", "loc_key", ModeLocKey},
		{"ck3", "descriptor.mod", "mod_descriptor", ModeTopLevelKey},
		{"ck3", "my_mod.mod", "mod_descriptor", ModeTopLevelKey},
		{"ck3", "common/religion/religion_types/00_islam.txt", "religion_types", ModeTopLevelKey},
		{"ck3", "common/on_action/activities/x.txt", "on_action", ModeTopLevelKey},
		{"eu5", "in_game/events/foo.txt", "event", ModeEventID},
		{"eu5", "in_game/common/religions/x.txt", "religions", ModeTopLevelKey},
	}
	for _, c := range cases {
		r := MatchExtract(c.gameID, c.path)
		if r.Kind != c.kind || r.Mode != c.mode {
			t.Errorf("MatchExtract(%q, %q) = %+v want {%s %s}", c.gameID, c.path, r, c.kind, c.mode)
		}
	}
}

func TestIsFIOS(t *testing.T) {
	cases := []struct {
		gameID, kind string
		want         bool
	}{
		{"ck3", "gui_type", true},
		{"ck3", "event", false},
		{"vic3", "gui_type", true},
		{"vic3", "event", true},
		{"eu5", "event", true},
		{"eu5", "gui_type", false},
		{"", "gui_type", true},
		{"", "event", false},
		{"unknown", "gui_type", true},
	}
	for _, c := range cases {
		if got := IsFIOS(c.gameID, c.kind); got != c.want {
			t.Errorf("IsFIOS(%q, %q) = %v want %v", c.gameID, c.kind, got, c.want)
		}
	}
}
