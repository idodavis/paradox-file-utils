// index_test.go covers ExtractFile harvest and BuildIndex across mods.

package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMod(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func extractCK3(rel, content, origin string) FileExtract {
	return ExtractFile("ck3", filepath.FromSlash(rel), rel, origin, content, false)
}

func hasRef(refs []Ref, kind, key string) bool {
	for _, r := range refs {
		if r.Kind == kind && r.Key == key {
			return true
		}
	}
	return false
}

func hasEdge(edges []Edge, from, to, kind string) bool {
	for _, e := range edges {
		if e.From == from && e.To == to && (kind == "" || e.Kind == kind || e.Via == kind) {
			return true
		}
	}
	return false
}

func TestExtractFile(t *testing.T) {
	eventSrc := `namespace = test
test.1 = {
	title = test.1.t
	immediate = { trigger_event = test.2 }
	option = { name = test.1.a
		trigger_event = test.3
	}
	events = { test.4 }
}
`
	titleArg := "test.1 = {\n\timmediate = { TITLE = primary_title }\n\ttitle = test.1.t\n}\n"
	saved := `test.1 = {
	immediate = {
		save_scope_as = duel_target
		exists = scope:duel_target
		save_scope_value_as = { name = gold_amt }
	}
	title = test.1.t
}
`
	t.Run("event loc and fire", func(t *testing.T) {
		ex := extractCK3("events/x.txt", eventSrc, "modA")
		if !findDef(ex.Defs, "event", "test.1") || ex.Defs[0].Origin != "modA" {
			t.Fatalf("defs=%v", ex.Defs)
		}
		if !hasRef(ex.Refs, "loc", "test.1.t") || !hasRef(ex.Refs, "event", "test.2") {
			t.Fatalf("refs=%v", ex.Refs)
		}
		if !hasEdge(ex.Edges, "test.1", "test.2", "trigger_event") {
			t.Fatalf("edges=%v", ex.Edges)
		}
		if !hasEdge(ex.Edges, "test.1", "test.3", "option") ||
			!hasEdge(ex.Edges, "test.1", "test.4", "events") {
			t.Fatalf("container edges=%v", ex.Edges)
		}
	})
	t.Run("TITLE arg is not loc", func(t *testing.T) {
		refs := extractCK3("events/x.txt", titleArg, "modA").Refs
		if hasRef(refs, "loc", "primary_title") {
			t.Fatalf("TITLE = primary_title should not be loc: %v", refs)
		}
		if !hasRef(refs, "loc", "test.1.t") {
			t.Fatalf("missing loc ref: %v", refs)
		}
	})
	t.Run("saved scope not indexed", func(t *testing.T) {
		ex := extractCK3("events/x.txt", saved, "modA")
		for _, d := range ex.Defs {
			if d.Type == "saved_scope" {
				t.Fatalf("saved_scope def: %+v", d)
			}
		}
		for _, r := range ex.Refs {
			if r.Kind == "saved_scope" {
				t.Fatalf("saved_scope ref: %+v", r)
			}
		}
		if !hasRef(ex.Refs, "loc", "test.1.t") {
			t.Fatalf("missing loc ref: %v", ex.Refs)
		}
	})
	t.Run("on_action container", func(t *testing.T) {
		oa := extractCK3("common/on_action/x.txt", "on_birth = {\n\tevents = { test.5 }\n}\n", "modA")
		if !hasEdge(oa.Edges, "on_birth", "test.5", "events") {
			t.Fatalf("edges=%v", oa.Edges)
		}
	})
	t.Run("call edges only for effects", func(t *testing.T) {
		content := `test.1 = {
	immediate = {
		real_effect = yes
		not_an_effect = yes
	}
}
real_effect = {
	trigger_event = test.2
}
`
		rel := "common/scripted_effects/e.txt"
		ex := extractCK3(rel, content, "modA")
		edges := append(append([]Edge{}, ex.Edges...), ApplyCallEdges(ex.Cands, EffectSet(ex.Defs, nil))...)
		var call, fake, fire bool
		for _, e := range edges {
			call = call || (e.Kind == EdgeKindCall && e.From == "test.1" && e.To == "real_effect")
			fake = fake || (e.Kind == EdgeKindCall && e.To == "not_an_effect")
			fire = fire || (e.From == "real_effect" && e.To == "test.2")
		}
		if !call || fake || !fire {
			t.Fatalf("call=%v fake=%v fire=%v edges=%v", call, fake, fire, edges)
		}
		for _, e := range ex.Edges {
			if e.Kind == EdgeKindCall {
				t.Fatalf("nil effectSet should omit calls: %v", ex.Edges)
			}
		}
	})
	t.Run("message title is loc not convention id", func(t *testing.T) {
		ex := extractCK3("common/messages/x.txt",
			"quieter_events_neutral = {\n\ttitle = event_message_title\n}\n", "mod")
		if hasRef(ex.Refs, "loc", "quieter_events_neutral") ||
			hasRef(ex.Refs, "loc-convention", "quieter_events_neutral") {
			t.Fatalf("invented message loc: %v", ex.Refs)
		}
		if !hasRef(ex.Refs, "loc", "event_message_title") {
			t.Fatalf("missing title loc: %v", ex.Refs)
		}
	})
	t.Run("game_rules convention keys", func(t *testing.T) {
		ex := extractCK3("common/game_rules/x.txt",
			"my_rule = {\n\tdefault = a\n\ta = { }\n}\n", "mod")
		if hasRef(ex.Refs, "loc", "my_rule") || hasRef(ex.Refs, "loc-convention", "my_rule") {
			t.Fatalf("bare rule id: %v", ex.Refs)
		}
		if !hasRef(ex.Refs, "loc-convention", "rule_my_rule") {
			t.Fatalf("missing rule_ key: %v", ex.Refs)
		}
		if !hasRef(ex.Refs, "loc-convention", "setting_a") {
			t.Fatalf("missing setting_ key: %v", ex.Refs)
		}
	})
	t.Run("loc file interpolations", func(t *testing.T) {
		bom := "\ufeffl_english:\n used:0 \"Hi\"\n other:0 \"see $used$ and $used|U$\"\n"
		ex := extractCK3("localization/english/a_l_english.yml", bom, "mod")
		if !findDef(ex.Defs, "loc_key", "used") || !findDef(ex.Defs, "loc_key", "other") {
			t.Fatalf("defs=%v", ex.Defs)
		}
		if !hasRef(ex.Refs, "loc", "used") {
			t.Fatalf("missing $used$ ref: %v", ex.Refs)
		}
	})
}

func TestBuildIndex(t *testing.T) {
	parent, child := t.TempDir(), t.TempDir()
	writeMod(t, parent, "common/traits/00.txt", "brave = { category = personality }\n")
	writeMod(t, child, "common/traits/01.txt", "craven = { category = personality }\n")
	cache := &VanillaCache{Defs: []Def{{Type: "trait", Key: "vanilla_trait", Path: "v.txt"}}}
	idx := BuildIndex("ck3", []ModInput{
		{Origin: "parent", Root: parent, Order: 0},
		{Origin: "child", Root: child, Order: 1},
	}, cache)
	var braveOrigin string
	for _, d := range idx.Defs {
		if d.Key == "brave" {
			braveOrigin = d.Origin
		}
	}
	if braveOrigin != "parent" {
		t.Errorf("brave origin = %q, want parent", braveOrigin)
	}
	if !findDef(cache.Defs, "trait", "vanilla_trait") {
		t.Error("missing vanilla_trait in cache defs")
	}

	root := t.TempDir()
	writeMod(t, root, "localization/english/events.yml", "l_english:\n test.1.t:0 \"Hello\"\n")
	locIdx := BuildIndex("ck3", []ModInput{{Origin: "mod", Root: root, Order: 0}}, nil)
	if got := locIdx.Loc["english"]["test.1.t"].Value; got != "Hello" {
		t.Fatalf("Loc from header = %v", locIdx.Loc)
	}
}

func TestVoteFieldValueKinds(t *testing.T) {
	defs := []Def{
		{Type: "event_theme", Key: "seduction"},
		{Type: "trait", Key: "brave"},
	}
	ok := VoteFieldValueKinds(map[string]map[string]bool{
		"theme": {"seduction": true},
	}, defs)
	if ok["theme"] != "event_theme" {
		t.Fatalf("theme = %v", ok)
	}
	conflict := VoteFieldValueKinds(map[string]map[string]bool{
		"theme": {"seduction": true, "brave": true},
	}, defs)
	if _, hit := conflict["theme"]; hit {
		t.Fatalf("conflict should omit theme: %v", conflict)
	}
}

func TestVoteFieldEnums(t *testing.T) {
	voted := map[string]string{"theme": "event_theme"}
	got := VoteFieldEnums(map[string]map[string]map[string]bool{
		"event": {
			"type":  {"character_event": true, "letter_event": true},
			"theme": {"seduction": true, "default": true},
			"id":    {"a": true},
		},
	}, voted)
	if len(got["event"]["type"]) != 2 {
		t.Fatalf("type enums: %v", got)
	}
	if _, hit := got["event"]["theme"]; hit {
		t.Fatalf("voted field should be omitted: %v", got)
	}
	if _, hit := got["event"]["id"]; hit {
		t.Fatalf("single value should be omitted: %v", got)
	}
}
