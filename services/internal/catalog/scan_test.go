// scan_test.go builds a tiny CK3-shaped install in a temp dir and asserts the
// corpus fixture (defs, structures, vocabulary, loc, field docs) populates with
// no script_docs present, then that a script_docs dump enriches classification.

package catalog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ck3Install writes the shared corpus fixture and returns the install path.
func ck3Install(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	writeMod(t, base, "game/common/traits/00_traits.txt", `brave = {
	category = personality
	opposites = { craven }
	ai_boldness = 30
}
`)
	writeMod(t, base, "game/common/traits/_traits.info", `# The trait category
category = personality
`)
	writeMod(t, base, "game/events/_events.info", `# Event presentation type
type = character_event
# Dynamic loc key
title = loc_key
`)
	writeMod(t, base, "game/events/test_events.txt", `namespace = test
test.1 = {
	type = character_event
	immediate = { add_gold = 10 trigger_event = test.2 }
	option = { name = test.1.a }
}
test.2 = { type = character_event }
scripted_effect my_inline = { add_prestige = 5 }
`)
	writeMod(t, base, "game/localization/english/test_l_english.yml", `l_english:
 test.1.t:0 "Test Event"
`)
	writeMod(t, base, "game/localization/english/header_only.yml", `l_english:
 header_key:0 "From header"
`)
	writeMod(t, base, "game/localization/french/test_l_french.yml", `l_french:
 french_only:0 "Bonjour"
`)
	return base
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func findDef(defs []Def, kind, key string) bool {
	for _, d := range defs {
		if d.Type == kind && d.Key == key {
			return true
		}
	}
	return false
}

func scanCK3(t *testing.T, base, docs string) (*VanillaCache, *VanillaLoc) {
	t.Helper()
	t.Cleanup(func() { _ = DropVanillaFiles("inst") })
	c, vloc, err := Scan(context.Background(), "inst", "ck3", base, "1.0", docs, "english", nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return c, vloc
}

func TestScanCorpusFixture(t *testing.T) {
	base := ck3Install(t)
	c, vloc := scanCK3(t, base, "")

	for _, d := range [][2]string{
		{"traits", "brave"}, {"event", "test.1"}, {"scripted_effect", "my_inline"},
	} {
		if !findDef(c.Defs, d[0], d[1]) {
			t.Errorf("missing %s %s; defs=%v", d[0], d[1], c.Defs)
		}
	}
	var fire bool
	for _, e := range c.Edges {
		if e.From == "test.1" && e.To == "test.2" && e.Via == "trigger_event" {
			fire = true
		}
	}
	if !fire {
		t.Errorf("missing fire edge test.1→test.2; edges=%v", c.Edges)
	}

	if got := vloc.Sites["test.1.t"].Value; got != "Test Event" {
		t.Errorf("loc[test.1.t] = %q, want %q", got, "Test Event")
	}
	if c.FieldDocs["category"] == "" {
		t.Errorf("fieldDocs[category] empty, want prose from _traits.info")
	}
	hk := vloc.Sites["header_key"]
	if hk.Value != "From header" || hk.File == "" {
		t.Errorf("header_key = %+v", hk)
	}
	if _, ok := vloc.Sites["french_only"]; ok {
		t.Errorf("french loc in default sidecar")
	}
	fr, err := LoadVanillaLoc("inst", "1.0", "french")
	if err != nil {
		t.Fatalf("french sidecar: %v", err)
	}
	if got := fr.Sites["french_only"].Value; got != "Bonjour" {
		t.Errorf("french loc[french_only] = %q", got)
	}
	if got := c.FieldDocsByKind["event"]["type"]; !strings.Contains(got, "presentation") {
		t.Errorf("fieldDocsByKind[event][type] = %q, want events.info prose", got)
	}
	if got := c.FieldDocsByKind["event"]["title"]; !strings.Contains(got, "Dynamic") {
		t.Errorf("fieldDocsByKind[event][title] = %q, want events.info prose", got)
	}

	for _, k := range []string{"category", "opposites", "ai_boldness"} {
		if !contains(c.Structures["traits"], k) {
			t.Errorf("structures[traits] missing %q: %v", k, c.Structures["traits"])
		}
	}
	for _, k := range []string{"immediate", "option"} {
		if !contains(c.Structures["event"], k) {
			t.Errorf("structures[event] missing %q: %v", k, c.Structures["event"])
		}
	}

	// Corpus vocabulary must be populated with NO script_docs dump.
	if !contains(c.Vocabulary, "add_gold") {
		t.Errorf("vocabulary missing corpus token add_gold: %v", c.Vocabulary)
	}
	if len(c.Effects) != 0 {
		t.Errorf("effects should be empty without script_docs, got %v", c.Effects)
	}
	if c.FormatVersion != CacheFormatVersion {
		t.Errorf("formatVersion = %d, want %d", c.FormatVersion, CacheFormatVersion)
	}

	for _, tt := range []struct {
		name, body, want, drop string
	}{
		{"enrichment",
			"add_gold\n\tAdds gold to the treasury.\n\nadd_prestige\n\tAdds prestige.\n",
			"add_gold", ""},
		{"nested type is not effect",
			"----\nadd_gold\n\tAdds gold.\n\n\tSupported Scopes: artifact\n\ttype = enum\n----\n",
			"add_gold", "type"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			docs := t.TempDir()
			if err := os.WriteFile(filepath.Join(docs, "effects.log"), []byte(tt.body), 0o644); err != nil {
				t.Fatal(err)
			}
			got, _ := scanCK3(t, base, docs)
			if !contains(got.Effects, tt.want) {
				t.Errorf("effects missing %s: %v", tt.want, got.Effects)
			}
			if tt.name == "enrichment" && got.FieldDocs["add_gold"] == "" {
				t.Error("fieldDocs[add_gold] empty after enrichment")
			}
			if tt.drop != "" && contains(got.Effects, tt.drop) {
				t.Fatalf("nested dump field %s leaked: %v", tt.drop, got.Effects)
			}
		})
	}
}
