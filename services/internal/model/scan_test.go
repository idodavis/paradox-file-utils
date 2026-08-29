// scan_test.go builds a tiny CK3-shaped install in a temp dir and asserts the
// corpus baseline (defs, structures, vocabulary, loc, field docs) populates with
// no script_docs present, then that a script_docs dump enriches classification.

package model

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture lays a file down at rel under base, creating parent dirs.
func writeFixture(t *testing.T, base, rel, content string) {
	t.Helper()
	p := filepath.Join(base, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ck3Install writes the shared corpus fixture and returns the install path.
func ck3Install(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	writeFixture(t, base, "game/common/traits/00_traits.txt", `brave = {
	category = personality
	opposites = { craven }
	ai_boldness = 30
}
`)
	writeFixture(t, base, "game/common/traits/_traits.info", `# The trait category
category = personality
`)
	writeFixture(t, base, "game/events/_events.info", `# Event presentation type
type = character_event
# Dynamic loc key
title = loc_key
`)
	writeFixture(t, base, "game/common/activities/_invite_rules.info", `# Invite rule type
type = friend
`)
	writeFixture(t, base, "game/events/test_events.txt", `namespace = test

test.1 = {
	type = character_event
	immediate = {
		add_gold = 10
	}
	option = {
		name = test.1.a
	}
}

scripted_effect my_inline = {
	add_prestige = 5
}
`)
	writeFixture(t, base, "game/localization/english/test_l_english.yml", `l_english:
 test.1.t:0 "Test Event"
 brave_desc: "Brave"
`)
	writeFixture(t, base, "game/localization/english/header_only.yml", `l_english:
 header_key:0 "From header"
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

func containsDoc(got, needle string) bool {
	return strings.Contains(got, needle)
}

func findDef(defs []Def, kind, key string) bool {
	for _, d := range defs {
		if d.Type == kind && d.Key == key {
			return true
		}
	}
	return false
}

func TestScanCorpusBaseline(t *testing.T) {
	base := ck3Install(t)
	c, err := Scan(context.Background(), "ws", "ck3", base, "", nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if !findDef(c.Defs, "traits", "brave") {
		t.Errorf("missing trait def brave; defs=%v", c.Defs)
	}
	if !findDef(c.Defs, "event", "test.1") {
		t.Errorf("missing event def test.1")
	}
	if !findDef(c.Defs, "scripted_effect", "my_inline") {
		t.Errorf("missing inline scripted_effect my_inline")
	}

	if got := c.LocEnglish["test.1.t"]; got != "Test Event" {
		t.Errorf("locEnglish[test.1.t] = %q, want %q", got, "Test Event")
	}
	if c.FieldDocs["category"] == "" {
		t.Errorf("fieldDocs[category] empty, want prose from _traits.info")
	}
	if got := c.LocEnglish["header_key"]; got != "From header" {
		t.Errorf("locEnglish[header_key] = %q, want header-only english", got)
	}
	if site, ok := c.LocEnglishSites["header_key"]; !ok || site.File == "" {
		t.Errorf("locEnglishSites[header_key] missing: %+v", c.LocEnglishSites)
	}
	if got := c.FieldDocsByKind["event"]["type"]; !containsDoc(got, "presentation") {
		t.Errorf("fieldDocsByKind[event][type] = %q, want events.info prose", got)
	}
	if got := c.FieldDocsByKind["event"]["title"]; !containsDoc(got, "Dynamic") {
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
}

func TestScanScriptDocsEnrichment(t *testing.T) {
	base := ck3Install(t)
	docs := t.TempDir()
	if err := os.WriteFile(filepath.Join(docs, "effects.log"),
		[]byte("add_gold\n\tAdds gold to the treasury.\n\nadd_prestige\n\tAdds prestige.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := Scan(context.Background(), "ws", "ck3", base, docs, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !contains(c.Effects, "add_gold") {
		t.Errorf("effects missing add_gold after enrichment: %v", c.Effects)
	}
	if c.FieldDocs["add_gold"] == "" {
		t.Errorf("fieldDocs[add_gold] empty after enrichment")
	}
}

func TestClassicDumpNestedTypeIsNotEffect(t *testing.T) {
	base := ck3Install(t)
	docs := t.TempDir()
	body := "----\nadd_gold\n\tAdds gold.\n\n\tSupported Scopes: artifact\n\ttype = enum\n----\n"
	if err := os.WriteFile(filepath.Join(docs, "effects.log"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Scan(context.Background(), "ws", "ck3", base, docs, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !contains(c.Effects, "add_gold") {
		t.Errorf("effects missing add_gold: %v", c.Effects)
	}
	if contains(c.Effects, "type") {
		t.Fatalf("nested dump field type leaked into effects: %v", c.Effects)
	}
}
