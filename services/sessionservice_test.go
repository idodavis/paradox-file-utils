// sessionservice_test.go covers session boot when a loc sidecar format is stale.

package services

import (
	"os"
	"path/filepath"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
)

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
