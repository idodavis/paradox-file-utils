package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/session"
	"paradox-modding-tools/services/internal/wiki"
)

func TestMatchAffectedRename(t *testing.T) {
	root := t.TempDir()
	se := filepath.Join(root, "common", "scripted_effects", "e.txt")
	if err := os.MkdirAll(filepath.Dir(se), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(se, []byte("ai_war_chest = { }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sess := session.NewWithLoc("ws", "ck3", "english", &catalog.VanillaCache{
		Effects: []string{"add_gold"},
	}, nil, []catalog.ModInput{{Origin: "mod", Root: root, Name: "M"}})
	pages := []wiki.Page{{
		Title:       "Patch 1.16",
		ModdingHTML: `<ul><li>renamed ` + "`ai_war_chest`" + ` to ` + "`ai_war_gold`" + `</li></ul>`,
	}}
	rep := matchAffected(sess, "mod", pages)
	if len(rep.Likely) == 0 {
		t.Fatal("expected likely hit for renamed token used in mod")
	}
	row := rep.Likely[0]
	if row.Token != "ai_war_chest" || !strings.Contains(row.Rel, "scripted_effects") {
		t.Fatalf("row = %#v", row)
	}
	if row.Origin != "mod" || row.OriginName != "M" {
		t.Fatalf("origin fields = %#v", row)
	}
}
