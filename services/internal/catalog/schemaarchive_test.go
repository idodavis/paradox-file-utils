package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

const archiveEffects = `Effect Documentation:

--------------------

add_gold - Adds gold to the scoped character
Supported Scopes: character

--------------------
`

// The archive exists so a user runs script_docs once per game version, not once
// per accidental cleanup of their Documents folder.
func TestSchemaArchiveFallback(t *testing.T) {
	const installID = "archive-test"
	const version = "1.0"
	t.Cleanup(func() { _ = DropVanillaFiles(installID) })

	live := t.TempDir()
	if err := os.WriteFile(filepath.Join(live, "effects.log"),
		[]byte(archiveEffects), 0o644); err != nil {
		t.Fatal(err)
	}

	// First read: live dumps win and seed the archive.
	s, src := loadSchema(installID, version, []string{live})
	if s == nil || src != SchemaLive {
		t.Fatalf("source = %q, schema = %v; want live", src, s)
	}
	if _, ok := s.Effects["add_gold"]; !ok {
		t.Fatalf("effects = %v", s.Effects)
	}
	if !hasSchemaArchive(installID, version) {
		t.Fatal("archive not written after a live read")
	}

	// The user deletes their dumps. The archive must stand in.
	if err := os.RemoveAll(live); err != nil {
		t.Fatal(err)
	}
	s2, src2 := loadSchema(installID, version, []string{live})
	if src2 != SchemaArchive {
		t.Fatalf("source = %q, want archive", src2)
	}
	if s2 == nil {
		t.Fatal("nil schema from archive")
	}
	if _, ok := s2.Effects["add_gold"]; !ok {
		t.Errorf("archive lost the effect: %v", s2.Effects)
	}
	if s2.Source != SchemaArchive {
		t.Errorf("Source = %q, want archive", s2.Source)
	}

	// Forgetting the install drops the archive with it.
	if err := DropVanillaFiles(installID); err != nil {
		t.Fatal(err)
	}
	if hasSchemaArchive(installID, version) {
		t.Error("archive survived DropVanillaFiles")
	}
	if _, src3 := loadSchema(installID, version, []string{live}); src3 != SchemaMissing {
		t.Errorf("source = %q, want missing", src3)
	}
}

// A newer live dump must replace the archived copy, not be shadowed by it.
func TestSchemaArchiveRefreshes(t *testing.T) {
	const installID = "archive-refresh"
	const version = "1.0"
	t.Cleanup(func() { _ = DropVanillaFiles(installID) })

	live := t.TempDir()
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(live, "effects.log"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(archiveEffects)
	if _, src := loadSchema(installID, version, []string{live}); src != SchemaLive {
		t.Fatalf("source = %q", src)
	}

	write(archiveEffects + `
add_prestige - Adds prestige
Supported Scopes: character

--------------------
`)
	s, src := loadSchema(installID, version, []string{live})
	if src != SchemaLive {
		t.Fatalf("source = %q", src)
	}
	if _, ok := s.Effects["add_prestige"]; !ok {
		t.Fatalf("live re-read missed the new effect: %v", s.Effects)
	}

	if err := os.RemoveAll(live); err != nil {
		t.Fatal(err)
	}
	s2, src2 := loadSchema(installID, version, []string{live})
	if src2 != SchemaArchive {
		t.Fatalf("source = %q", src2)
	}
	if _, ok := s2.Effects["add_prestige"]; !ok {
		t.Errorf("archive was not refreshed: %v", s2.Effects)
	}
}

// Files the schema reader does not recognize must not be copied into the cache.
func TestSchemaArchiveSkipsUnrelatedFiles(t *testing.T) {
	const installID = "archive-filter"
	const version = "1.0"
	t.Cleanup(func() { _ = DropVanillaFiles(installID) })

	live := t.TempDir()
	for name, body := range map[string]string{
		"effects.log": archiveEffects,
		"error.log":   "some error spam\n",
		"game.log":    "startup noise\n",
	} {
		if err := os.WriteFile(filepath.Join(live, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	n, err := archiveScriptDocs(installID, version, []string{live})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("archived %d files, want only effects.log", n)
	}
	dir, err := schemaArchiveDir(installID, version)
	if err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if e.Name() != "effects.log" {
			t.Errorf("unexpected archived file %q", e.Name())
		}
	}
}

// With no dumps anywhere the scan proceeds without a type system.
func TestSchemaArchiveMissing(t *testing.T) {
	s, src := loadSchema("archive-none", "1.0", []string{t.TempDir()})
	if s != nil || src != SchemaMissing {
		t.Fatalf("schema = %v, source = %q; want nil/missing", s, src)
	}
}

// ReadSchemaStatus backs the Workspace Health strip, so it must answer without
// loading the model and without the side effect loadSchema has: health is
// polled, and a read that re-copies the dumps every few seconds is a bug.
func TestReadSchemaStatusIsSideEffectFree(t *testing.T) {
	const installID = "status-test"
	t.Cleanup(func() { _ = DropVanillaFiles(installID) })

	install := t.TempDir()
	docs := filepath.Join(install, "logs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "effects.log"),
		[]byte(archiveEffects), 0o644); err != nil {
		t.Fatal(err)
	}
	pinUserData(t, install, install)

	st := ReadSchemaStatus("ck3", installID, install, "1.0")
	if st.Source != SchemaLive {
		t.Fatalf("source = %q, want live", st.Source)
	}
	if st.Effects == 0 {
		t.Errorf("no effects counted: %+v", st)
	}
	if st.ReadAt == "" {
		t.Error("ReadAt empty; staleness cannot be judged")
	}
	if st.Archived {
		t.Error("reading status must not create an archive")
	}

	// With no dumps and no archive the strip must say so plainly rather than
	// report a half-loaded type system.
	if err := os.RemoveAll(docs); err != nil {
		t.Fatal(err)
	}
	gone := ReadSchemaStatus("ck3", installID, install, "1.0")
	if gone.Source != SchemaMissing || gone.Effects != 0 {
		t.Errorf("no dumps and no archive: %+v", gone)
	}
	if gone.Folder == "" {
		t.Error("the walkthrough needs a folder to point at even when nothing is there")
	}
}
