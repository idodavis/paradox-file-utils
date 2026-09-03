package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkshopPreviewAddRemove(t *testing.T) {
	r, root := testRelease(t)
	src := filepath.Join(t.TempDir(), "shot.png")
	if err := os.WriteFile(src, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	listing, err := r.AddWorkshopPreview("w", "m", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Previews) != 1 || listing.Previews[0].Kind != "image" {
		t.Fatalf("%+v", listing.Previews)
	}
	rel := listing.Previews[0].Rel
	if !strings.HasPrefix(rel, "workshop/previews/") {
		t.Fatalf("rel %s", rel)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		t.Fatal(err)
	}
	listing, err = r.RemoveWorkshopPreview("w", "m", rel)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Previews) != 0 {
		t.Fatalf("%+v", listing.Previews)
	}
}

func TestWorkshopVideoAddRemove(t *testing.T) {
	r, root := testRelease(t)
	const id = "dQw4w9WgXcQ"
	listing, err := r.AddWorkshopVideo("w", "m", "https://youtu.be/"+id)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Previews) != 1 || listing.Previews[0].ID != id {
		t.Fatalf("%+v", listing.Previews)
	}
	again, err := r.AddWorkshopVideo("w", "m", id)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Previews) != 1 {
		t.Fatalf("duplicate: %+v", again.Previews)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash("workshop/videos.txt")))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != id {
		t.Fatalf("%q", raw)
	}
	listing, err = r.RemoveWorkshopVideo("w", "m", id)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Previews) != 0 {
		t.Fatalf("%+v", listing.Previews)
	}
}

func TestWorkshopExtrasCap(t *testing.T) {
	r, _ := testRelease(t)
	dir := t.TempDir()
	for i := 0; i < 10; i++ {
		src := filepath.Join(dir, "shot-"+string(rune('a'+i))+".png")
		if err := os.WriteFile(src, []byte("png"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := r.AddWorkshopPreview("w", "m", src); err != nil {
			t.Fatal(err)
		}
	}
	src := filepath.Join(dir, "overflow.png")
	if err := os.WriteFile(src, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.AddWorkshopPreview("w", "m", src); err == nil {
		t.Fatal("want cap error")
	}
	if _, err := r.AddWorkshopVideo("w", "m", "dQw4w9WgXcQ"); err == nil {
		t.Fatal("want cap error")
	}
}

func TestListingThumbAbs(t *testing.T) {
	r, root := testRelease(t)
	thumb := filepath.Join(root, "thumbnail.png")
	if err := os.WriteFile(thumb, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	listing, err := r.LoadListing("w", "m")
	if err != nil {
		t.Fatal(err)
	}
	if listing.ThumbnailRel != "thumbnail.png" || listing.ThumbnailAbs != thumb {
		t.Fatalf("%q %q", listing.ThumbnailRel, listing.ThumbnailAbs)
	}
}

func testRelease(t *testing.T) (*ReleaseService, string) {
	t.Helper()
	root := t.TempDir()
	body := "name=\"T\"\nversion=\"1.0.0\"\nsupported_version=\"1.16.*\"\n"
	if err := os.WriteFile(filepath.Join(root, "descriptor.mod"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	st := &Store{cfg: Config{
		FormatVersion: 1,
		Workspaces: []Workspace{{
			ID: "w", GameID: "ck3", Name: "W",
			Mods: []WorkspaceMod{{ID: "m", Name: "T", Path: root}},
		}},
	}}
	return &ReleaseService{Store: st}, root
}
