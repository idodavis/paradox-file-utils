// store_test.go covers JSON config round-trip and version-mismatch wipe.

package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
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

// covers session boot when a loc sidecar format is stale.
func TestEnsureSession_staleLocSidecar(t *testing.T) {
	installID := "pmt-test-stale-loc"
	dir, err := catalog.CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "vanilla-loc-"+installID+"-latest-english.json")
	if err := os.WriteFile(stale, []byte(`{"formatVersion":2,"sites":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = catalog.DropVanillaFiles(installID) })

	store := newStore(filepath.Join(t.TempDir(), "config.json"))
	if err := store.Mutate(func(c *Config) error {
		c.Installs = append(c.Installs, GameInstall{
			ID: installID, GameID: "ck3", Path: t.TempDir(), Version: "latest",
		})
		c.Workspaces = append(c.Workspaces, Workspace{
			ID: "ws", GameID: "ck3", Name: "Test", InstallID: installID,
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	svc := &SessionService{Store: store}
	h, err := svc.EnsureSession("ws")
	if err != nil {
		t.Fatalf("stale loc sidecar must not fail session: %v", err)
	}
	if h == nil || !h.IndexReady {
		t.Fatalf("want live session, got %+v", h)
	}
}

func TestConvertMarkdown(t *testing.T) {
	r := &ReleaseService{}
	got := r.Convert("# Hi\n", "bbcode")
	if !strings.Contains(got.Text, "[h1]Hi[/h1]") {
		t.Fatalf("%q", got.Text)
	}
}

func TestSteamPublishError(t *testing.T) {
	got := steamPublishError("SubmitItemUpdate EResult 9")
	if got == "" || !strings.Contains(got, "EResult 9") {
		t.Fatalf("eresult 9: %q", got)
	}
	msg := steamPublishError("SubmitItemUpdate EResult 25")
	if msg == "" || !strings.Contains(msg, "1 MB") {
		t.Fatalf("eresult 25: %q", msg)
	}
	if steamPublishError("SubmitItemUpdate EResult 2") != "" {
		t.Fatal("eresult 2 should not map")
	}
}

func TestSteamEResultCode(t *testing.T) {
	code, ok := steamEResultCode("CreateItem EResult 25")
	if !ok || code != 25 {
		t.Fatalf("got %d %v", code, ok)
	}
	if _, ok := steamEResultCode("no code here"); ok {
		t.Fatal("expected miss")
	}
}

func TestBumpVersion(t *testing.T) {
	r := &ReleaseService{}
	if got := r.BumpVersion("1.2.0", "patch"); got != "1.2.1" {
		t.Fatalf("%s", got)
	}
	if got := r.BumpVersion("1.2.3", "minor"); got != "1.3.0" {
		t.Fatalf("%s", got)
	}
}

// covers PMT-owned path checks for ResetData wipes.
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
