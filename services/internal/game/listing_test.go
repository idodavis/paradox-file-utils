package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDescriptorModRoundTrip(t *testing.T) {
	root := t.TempDir()
	src := `version="1.0"
name="Old"
supported_version="1.16.*"
# keep
picture="thumbnail.png"
tags={
	"Events"
}
`
	if err := os.WriteFile(filepath.Join(root, "descriptor.mod"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadListingFields("ck3", root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Old" || got.SupportedVersion != "1.16.*" ||
		len(got.Tags) != 1 || got.Tags[0] != "Events" {
		t.Fatalf("%+v", got)
	}
	got.Name = "New"
	got.Version = "1.2.0"
	got.RemoteFileID = "99"
	got.Tags = []string{"Events", "Gameplay"}
	if err := WriteListingFields("ck3", root, got); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "descriptor.mod"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `name="New"`) ||
		!strings.Contains(body, `remote_file_id="99"`) ||
		!strings.Contains(body, "# keep") ||
		!strings.Contains(body, `"Gameplay"`) {
		t.Fatalf("%s", body)
	}
}

func TestMetadataJSONRoundTrip(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".metadata")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `{
  "name": "V",
  "id": "v",
  "version": "0.1",
  "game_id": "victoria3",
  "relationships": []
}
`
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadListingFields("vic3", root)
	if err != nil {
		t.Fatal(err)
	}
	got.Version = "0.2.0"
	got.ShortDescription = "hi"
	got.Tags = []string{"a"}
	if err := WriteListingFields("vic3", root, got); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `"version": "0.2.0"`) ||
		!strings.Contains(body, `"game_id"`) ||
		!strings.Contains(body, `"relationships"`) {
		t.Fatalf("%s", body)
	}
}

func TestEnsureDescriptionsCreatesMissingOnly(t *testing.T) {
	root := t.TempDir()
	if err := EnsureDescriptions(root, "M", "desc"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, DescMdName), []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDescriptions(root, "M", "other"); err != nil {
		t.Fatal(err)
	}
	md, err := os.ReadFile(filepath.Join(root, DescMdName))
	if err != nil {
		t.Fatal(err)
	}
	if string(md) != "kept\n" {
		t.Fatalf("%q", md)
	}
}

func TestBumpVersion(t *testing.T) {
	if BumpPatch("1.2.3") != "1.2.4" {
		t.Fatal(BumpPatch("1.2.3"))
	}
	if BumpMinor("1.2.3") != "1.3.0" {
		t.Fatal(BumpMinor("1.2.3"))
	}
}
