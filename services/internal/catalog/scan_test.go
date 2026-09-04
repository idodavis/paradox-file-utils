// scan_test.go builds a tiny CK3-shaped install in a temp dir and asserts the
// corpus fixture (defs, structures, vocabulary, loc, field docs) populates with
// no script_docs present, then that a script_docs dump enriches classification.

package catalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/game"
)

// ck3Install writes the shared corpus fixture and returns install + isolated user data.
func ck3Install(t *testing.T) (base, userData string) {
	t.Helper()
	base = t.TempDir()
	userData = filepath.Join(base, "userdata")
	if err := os.MkdirAll(userData, 0o755); err != nil {
		t.Fatal(err)
	}
	pinUserData(t, base, userData)
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
	immediate = {
		add_gold = 10
		trigger_event = test.2
		save_scope_as = duel_target
		my_trig = { TARGET = root }
	}
	option = { name = test.1.a }
}
test.2 = { type = character_event }
scripted_effect my_inline = { add_prestige = 5 }
`)
	writeMod(t, base, "game/common/scripted_triggers/t.txt",
		"my_trig = { exists = $TARGET$ }\n")
	writeMod(t, base, "game/localization/english/test_l_english.yml", `l_english:
 test.1.t:0 "Test Event"
`)
	writeMod(t, base, "game/localization/english/header_only.yml", `l_english:
 header_key:0 "From header"
`)
	writeMod(t, base, "game/localization/french/test_l_french.yml", `l_french:
 french_only:0 "Bonjour"
`)
	return base, userData
}

func pinUserData(t *testing.T, install, userData string) {
	t.Helper()
	if err := os.MkdirAll(userData, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(install, "launcher")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]string{"gameDataPath": userData})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "launcher-settings.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
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
		if d.Kind == kind && d.Key == key {
			return true
		}
	}
	return false
}

func scanCK3(t *testing.T, base string) (*VanillaCache, *VanillaLoc) {
	t.Helper()
	t.Cleanup(func() { _ = DropVanillaFiles("inst") })
	c, vloc, err := Scan(context.Background(), ScanRequest{
		InstallID: "inst", GameID: "ck3", InstallPath: base,
		Version: "1.0", LocLang: "english",
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return c, vloc
}

func TestScanCorpusFixture(t *testing.T) {
	base, userData := ck3Install(t)
	c, vloc := scanCK3(t, base)

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
	if c.FieldInfo["category"] == "" {
		t.Errorf("fieldDocs[category] empty, want prose from _traits.info")
	}
	hk := vloc.Sites["header_key"]
	if hk.Value != "From header" || hk.Path == "" {
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
	if got := c.FieldInfoByKind["event"]["type"]; !strings.Contains(got, "presentation") {
		t.Errorf("fieldDocsByKind[event][type] = %q, want events.info prose", got)
	}
	if got := c.FieldInfoByKind["event"]["title"]; !strings.Contains(got, "Dynamic") {
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
	if got := c.Structures["event"]; len(got) == 0 || got[0] != "type" {
		t.Errorf("structures[event] freq = %v, want type first", got)
	}
	if !contains(c.StructureBlocks["event"], "immediate") ||
		!contains(c.StructureBlocks["event"], "option") {
		t.Errorf("structureBlocks[event] = %v", c.StructureBlocks["event"])
	}
	if contains(c.StructureBlocks["event"], "type") {
		t.Errorf("type should be scalar: %v", c.StructureBlocks["event"])
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
	gotParam, gotCall := false, false
	for _, d := range c.Defs {
		k := game.CanonicalKind(d.Kind)
		if game.IsEphemeral(k) && k != "script_param" {
			t.Errorf("scan persisted ephemeral %s %s", d.Kind, d.Key)
		}
		if k == "script_param" && d.Key == "TARGET" {
			gotParam = true
		}
	}
	for _, r := range c.CallRefs {
		if r.Key == "my_trig" && game.CanonicalKind(r.Kind) == "scripted_trigger" {
			gotCall = true
		}
	}
	if !gotParam || !gotCall {
		t.Errorf("vanilla $NAME$/call persist param=%v call=%v refs=%v",
			gotParam, gotCall, c.CallRefs)
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
			logs := filepath.Join(userData, "logs")
			if err := os.MkdirAll(logs, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(logs, "effects.log"), []byte(tt.body), 0o644); err != nil {
				t.Fatal(err)
			}
			got, _ := scanCK3(t, base)
			if !contains(got.Effects, tt.want) {
				t.Errorf("effects missing %s: %v", tt.want, got.Effects)
			}
			if tt.name == "enrichment" && got.TokenDoc["add_gold"] == "" {
				t.Error("tokenDoc[add_gold] empty after enrichment")
			}
			if tt.drop != "" && contains(got.Effects, tt.drop) {
				t.Fatalf("nested dump field %s leaked: %v", tt.drop, got.Effects)
			}
		})
	}
}

func TestClassifyRelShippedDocs(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{"_traits.info", "info"},
		{"readme.txt", "info"},
		{"__readme.txt", "info"},
		{"____Info.txt", "info"},
		{"notes.md", "info"},
		{"00_traits.txt", "script"},
		{"_default.txt", ""},
		{"_hardcoded.txt", ""},
		{"_advances_template.txt", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyRel("common/x/"+tt.name, tt.name); got != tt.want {
				t.Fatalf("ClassifyRel(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestHarvestDocFile(t *testing.T) {
	tests := []struct {
		name, src, key, want string
	}{
		{
			name: "ck3 above comment",
			src:  "# The trait category\ncategory = personality\n",
			key:  "category", want: "trait category",
		},
		{
			name: "below block comment",
			src:  "allow = {\n\t# must also pass category\n}\n",
			key:  "allow", want: "must also pass",
		},
		{
			name: "below does not steal next above",
			src: "# Event presentation type\ntype = character_event\n" +
				"# Dynamic loc key\ntitle = loc_key\n",
			key: "title", want: "Dynamic",
		},
		{
			name: "eu5 attribute bullets",
			src: "# ATTRIBUTES\n" +
				"# - build_time: <integer> building time in days\n" +
				"# - allow: <trigger> can the building be built\n",
			key: "build_time", want: "building time",
		},
		{
			name: "commented template inline",
			src:  "#\ttype = country_event\t\t# Defines the scope\n",
			key:  "type", want: "Defines the scope",
		},
		{
			name: "inline plus below block",
			src:  "chance = { # weight\n\t# extra lines\n}\n",
			key:  "chance", want: "weight extra lines",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := harvestDocFile(tt.src)
			if !strings.Contains(got[tt.key], tt.want) {
				t.Fatalf("harvestDocFile()[%q] = %q, want substring %q (all=%v)",
					tt.key, got[tt.key], tt.want, got)
			}
		})
	}
	t.Run("ck3 type keeps above comment", func(t *testing.T) {
		got := harvestDocFile(
			"# Event presentation type\ntype = character_event\n" +
				"# Dynamic loc key\ntitle = loc_key\n",
		)
		if !strings.Contains(got["type"], "presentation") {
			t.Fatalf("type = %q, want presentation", got["type"])
		}
	})
}

func TestScanEU5ShippedDocs(t *testing.T) {
	base := t.TempDir()
	writeMod(t, base, "game/in_game/common/traits/00_traits.txt", `brave = {
	category = ruler
	allow = { always = yes }
}
`)
	writeMod(t, base, "game/in_game/common/traits/_traits.info", `trait_name = {
	# Which type of character can use this trait?
	category = ruler
	allow = {
		# What else does the character need
	}
}
`)
	writeMod(t, base, "game/in_game/common/building_types/readme.txt", `# ATTRIBUTES
# - build_time: <integer> building time in days
# - allow: <trigger> can the building be built
`)
	pinUserData(t, base, filepath.Join(t.TempDir(), "ud"))
	t.Cleanup(func() { _ = DropVanillaFiles("eu5docs") })
	c, _, err := Scan(context.Background(), ScanRequest{
		InstallID: "eu5docs", GameID: "eu5", InstallPath: base,
		Version: "1.0", LocLang: "english",
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got := c.FieldInfoByKind["traits"]["allow"]; !strings.Contains(got, "character need") {
		t.Fatalf("traits allow doc = %q", got)
	}
	if got := c.FieldInfoByKind["traits"]["category"]; !strings.Contains(got, "Which type") {
		t.Fatalf("traits category doc = %q", got)
	}
	if got := c.FieldInfoByKind["building_types"]["build_time"]; !strings.Contains(got, "building time") {
		t.Fatalf("building_types build_time = %q", got)
	}
}

func TestScanEU5InstallUsesGameFolder(t *testing.T) {
	base := t.TempDir()
	writeMod(t, base, "game/in_game/events/x.txt", `namespace = t
t.1 = { type = country_event }
`)
	writeMod(t, base, "in_game/events/wrong.txt", `namespace = w
w.1 = { type = country_event }
`)
	pinUserData(t, base, filepath.Join(t.TempDir(), "ud"))
	t.Cleanup(func() { _ = DropVanillaFiles("eu5inst") })
	c, _, err := Scan(context.Background(), ScanRequest{
		InstallID: "eu5inst", GameID: "eu5", InstallPath: base,
		Version: "1.0", LocLang: "english",
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !findDef(c.Defs, "event", "t.1") {
		t.Fatalf("missing install event t.1; defs=%v", c.Defs)
	}
	if findDef(c.Defs, "event", "w.1") {
		t.Fatalf("scanned install-root in_game, not game/: %v", c.Defs)
	}
}

func TestScanScriptDocsFixtures(t *testing.T) {
	t.Run("ck3", func(t *testing.T) {
		base, ud := ck3Install(t)
		raw, err := os.ReadFile(filepath.Join("testdata", "ck3-effects.log"))
		if err != nil {
			t.Fatal(err)
		}
		logs := filepath.Join(ud, "logs")
		if err := os.MkdirAll(logs, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(logs, "effects.log"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
		c, _ := scanCK3(t, base)
		if !contains(c.Effects, "add_diplomacy_skill") {
			t.Fatalf("effects missing add_diplomacy_skill: %v", c.Effects)
		}
		if contains(c.Effects, "Effect") {
			t.Fatalf("header token leaked: %v", c.Effects)
		}
		if !strings.Contains(c.TokenDoc["add_diplomacy_skill"], "diplomacy") {
			t.Fatalf("tokenDoc = %q", c.TokenDoc["add_diplomacy_skill"])
		}
	})
	t.Run("eu5", func(t *testing.T) {
		base := t.TempDir()
		ud := filepath.Join(base, "userdata")
		pinUserData(t, base, ud)
		writeMod(t, base, "game/in_game/events/x.txt", `namespace = t
t.1 = { type = country_event }
`)
		raw, err := os.ReadFile(filepath.Join("testdata", "eu5-effects.log"))
		if err != nil {
			t.Fatal(err)
		}
		docs := filepath.Join(ud, "docs")
		if err := os.MkdirAll(docs, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(docs, "effects.log"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = DropVanillaFiles("eu5fix") })
		c, _, err := Scan(context.Background(), ScanRequest{
			InstallID: "eu5fix", GameID: "eu5", InstallPath: base,
			Version: "1.0", LocLang: "english",
		})
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if !contains(c.Effects, "abandon_colonial_charter") {
			t.Fatalf("effects missing abandon_colonial_charter: %v", c.Effects)
		}
		if contains(c.Effects, "Effect") {
			t.Fatalf("header token leaked: %v", c.Effects)
		}
		if !strings.Contains(c.TokenDoc["abandon_colonial_charter"], "colonial") {
			t.Fatalf("tokenDoc = %q", c.TokenDoc["abandon_colonial_charter"])
		}
		if !strings.Contains(c.TokenScopes["abandon_colonial_charter"], "none") {
			t.Fatalf("scopes = %q", c.TokenScopes["abandon_colonial_charter"])
		}
	})
}
