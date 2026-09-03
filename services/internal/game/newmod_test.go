package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
