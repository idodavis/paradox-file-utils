package modfile

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

func TestModSlug(t *testing.T) {
	if got := ModSlug("My Cool Mod"); got != "my_cool_mod" {
		t.Fatalf("slug: %q", got)
	}
	if got := ModSlug("   "); got != "mod" {
		t.Fatalf("empty: %q", got)
	}
}

func TestWriteNewModCK3(t *testing.T) {
	root := filepath.Join(t.TempDir(), "my_cool_mod")
	if err := WriteNewMod(NewModOpts{
		GameID: "ck3", Root: root, Name: "My Cool Mod",
		SupportedVersion: "1.16.2", LocLang: "english",
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "descriptor.mod"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `name="My Cool Mod"`) ||
		!strings.Contains(body, `supported_version="1.16.2"`) {
		t.Fatalf("descriptor: %s", body)
	}
	if _, err := os.Stat(filepath.Join(root, "common")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "events")); err != nil {
		t.Fatal(err)
	}
	md, err := os.ReadFile(filepath.Join(root, DescMdName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "# My Cool Mod") ||
		!strings.Contains(string(md), "My Cool Mod") {
		t.Fatalf("mod-description.md: %s", md)
	}
	bb, err := os.ReadFile(filepath.Join(root, DescBbName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bb), "[b]My Cool Mod[/b]") {
		t.Fatalf("mod-description.bbcode: %s", bb)
	}
	loc := filepath.Join(root, "localization", "english", "my_cool_mod_l_english.yml")
	locRaw, err := os.ReadFile(loc)
	if err != nil {
		t.Fatal(err)
	}
	if len(locRaw) < 3 || locRaw[0] != 0xEF || locRaw[1] != 0xBB || locRaw[2] != 0xBF {
		t.Fatalf("want BOM, got %q", locRaw)
	}
	if !strings.Contains(string(locRaw), "l_english:") {
		t.Fatalf("loc: %q", locRaw)
	}
	if err := WriteNewMod(NewModOpts{
		GameID: "ck3", Root: root, Name: "My Cool Mod", LocLang: "english",
	}); err == nil {
		t.Fatal("want already-exists")
	}
}

func TestWriteNewModVic3OmitsLatest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "v")
	if err := WriteNewMod(NewModOpts{
		GameID: "vic3", Root: root, Name: "V",
		SupportedVersion: "latest", LocLang: "french", Description: "A desc",
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".metadata", "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, `"name": "V"`) {
		t.Fatalf("meta: %s", s)
	}
	if strings.Contains(s, "supported_game_version") {
		t.Fatalf("latest must omit pin: %s", s)
	}
	md, err := os.ReadFile(filepath.Join(root, DescMdName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "A desc") {
		t.Fatalf("mod-description.md: %s", md)
	}
	loc := filepath.Join(root, "localization", "french", "v_l_french.yml")
	if _, err := os.Stat(loc); err != nil {
		t.Fatal(err)
	}
}

func TestWriteNewModThumbnail(t *testing.T) {
	src := filepath.Join(t.TempDir(), "art.png")
	if err := os.WriteFile(src, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "m")
	if err := WriteNewMod(NewModOpts{
		GameID: "ck3", Root: root, Name: "M",
		SupportedVersion: "1", LocLang: "english", ThumbnailSrc: src,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "thumbnail.png")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "descriptor.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `picture="thumbnail.png"`) {
		t.Fatalf("descriptor: %s", raw)
	}
}

func TestDefaultModParent(t *testing.T) {
	p := DefaultModParent("ck3")
	if p == "" || !strings.Contains(p, "Crusader Kings III") ||
		filepath.Base(p) != "mod" {
		t.Fatalf("parent: %q", p)
	}
	if DefaultModParent("nope") != "" {
		t.Fatal("unknown game")
	}
}
