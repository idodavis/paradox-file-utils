// index_test.go covers ExtractFile harvest and BuildIndex across mods.

package catalog

import (
	"context"
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

func TestEventTitleSingleLocRef(t *testing.T) {
	body := "namespace = test\n\ntest.1 = {\n\ttitle = k.t\n}\n"
	ex := extractCK3("events/x.txt", body, "mod")
	var kt []Ref
	for _, r := range ex.Refs {
		if r.Key == "k.t" {
			kt = append(kt, r)
		}
	}
	if len(kt) != 1 {
		t.Fatalf("k.t refs=%v", kt)
	}
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
	t.Run("saved scope indexed", func(t *testing.T) {
		ex := extractCK3("events/x.txt", saved, "modA")
		if !findDef(ex.Defs, "saved_scope", "duel_target") {
			t.Fatalf("missing save_scope_as def: %v", ex.Defs)
		}
		if !findDef(ex.Defs, "saved_scope", "gold_amt") {
			t.Fatalf("missing save_scope_value_as def: %v", ex.Defs)
		}
		if !hasRef(ex.Refs, "saved_scope", "duel_target") {
			t.Fatalf("missing scope: ref: %v", ex.Refs)
		}
		if !hasRef(ex.Refs, "loc", "test.1.t") {
			t.Fatalf("missing loc ref: %v", ex.Refs)
		}
	})
	t.Run("character flag and variable", func(t *testing.T) {
		src := `test.1 = {
	immediate = {
		add_character_flag = used_life
		has_character_flag = used_life
		set_variable = { name = foo value = 3 }
		has_variable = foo
		exists = var:foo
	}
}
`
		ex := extractCK3("events/x.txt", src, "modA")
		if !findDef(ex.Defs, "character_flag", "used_life") {
			t.Fatalf("missing character_flag def: %v", ex.Defs)
		}
		if !hasRef(ex.Refs, "character_flag", "used_life") {
			t.Fatalf("missing character_flag ref: %v", ex.Refs)
		}
		if !findDef(ex.Defs, "variable", "foo") {
			t.Fatalf("missing variable def: %v", ex.Defs)
		}
		for _, d := range ex.Defs {
			if d.Kind == "variable" && d.Key == "foo" && d.Value != "3" {
				t.Fatalf("variable Value=%q want 3", d.Value)
			}
			if d.Kind == "character_flag" && d.Key == "used_life" && d.Value != "yes" {
				t.Fatalf("flag Value=%q want yes", d.Value)
			}
		}
		if !hasRef(ex.Refs, "variable", "foo") {
			t.Fatalf("missing variable ref: %v", ex.Refs)
		}
	})
	t.Run("vic3 has no character_flag", func(t *testing.T) {
		src := "e = { immediate = { add_character_flag = x set_variable = y } }\n"
		ex := ExtractFile("vic3", "events/x.txt", "events/x.txt", "m", src, false)
		for _, d := range ex.Defs {
			if d.Kind == "character_flag" {
				t.Fatalf("vic3 must not harvest character_flag: %+v", d)
			}
		}
		if !findDef(ex.Defs, "variable", "y") {
			t.Fatalf("vic3 must still harvest variable: %v", ex.Defs)
		}
	})
	t.Run("coa and flag_definition kinds", func(t *testing.T) {
		coa := ExtractFile("ck3", "common/coat_of_arms/coat_of_arms/x.txt",
			"common/coat_of_arms/coat_of_arms/x.txt", "m",
			"b_appleby = { pattern = \"pattern_solid.dds\" }\n", false)
		if !findDef(coa.Defs, "coat_of_arms", "b_appleby") {
			t.Fatalf("coa defs=%v", coa.Defs)
		}
		fd := ExtractFile("vic3", "common/flag_definitions/00.txt",
			"common/flag_definitions/00.txt", "m",
			"ENG = { flag_definition = { } }\n", false)
		if !findDef(fd.Defs, "flag_definition", "ENG") &&
			!findDef(fd.Defs, "flag_definitions", "ENG") {
			t.Fatalf("flag_definitions defs=%v", fd.Defs)
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
		refs := ApplyCallRefs(ex.Cands, CallKindSet(ex.Defs, nil))
		if !hasRef(refs, "scripted_effect", "real_effect") {
			t.Fatalf("call refs=%v want real_effect", refs)
		}
		if hasRef(refs, "scripted_effect", "not_an_effect") {
			t.Fatalf("engine-like key must not be a call ref: %v", refs)
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
	t.Run("script_value skips prefixed RHS", func(t *testing.T) {
		ex := extractCK3("events/x.txt",
			"e.1 = { immediate = { set_variable = { name = x value = scope:foo } } }\n",
			"mod")
		if hasRef(ex.Refs, "script_value", "scope:foo") ||
			hasRef(ex.Refs, "script_value", "x") {
			t.Fatalf("noisy script_value refs: %v", ex.Refs)
		}
	})
	t.Run("game_rule_setting defs", func(t *testing.T) {
		ex := extractCK3("common/game_rules/x.txt",
			"my_rule = {\n\tdefault = a\n\tsuf_quieter = { }\n}\n", "mod")
		if !findDef(ex.Defs, "game_rule_setting", "suf_quieter") {
			t.Fatalf("missing setting def: %v", ex.Defs)
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
	t.Run("trigger localization slots are loc refs", func(t *testing.T) {
		ex := extractCK3("common/trigger_localization/x.txt",
			"is_adult = {\n\tglobal = IS_ADULT_TRIGGER\n\tfirst = I_AM_ADULT_TRIGGER\n"+
				"\tthird = THEY_ARE_ADULT_TRIGGER\n\tnone = yes\n}\n",
			"mod")
		if !hasRef(ex.Refs, "loc", "IS_ADULT_TRIGGER") ||
			!hasRef(ex.Refs, "loc", "I_AM_ADULT_TRIGGER") ||
			!hasRef(ex.Refs, "loc", "THEY_ARE_ADULT_TRIGGER") {
			t.Fatalf("missing trigger loc refs: %v", ex.Refs)
		}
		if hasRef(ex.Refs, "loc", "yes") {
			t.Fatalf("none = yes harvested: %v", ex.Refs)
		}
	})
	t.Run("effect localization slots are loc refs", func(t *testing.T) {
		ex := extractCK3("common/effect_localization/x.txt",
			"add_gold = {\n\tfirst = I_GAIN_GOLD_EFFECT\n\tfirst_past = I_GAINED_GOLD_EFFECT\n}\n",
			"mod")
		if !hasRef(ex.Refs, "loc", "I_GAIN_GOLD_EFFECT") ||
			!hasRef(ex.Refs, "loc", "I_GAINED_GOLD_EFFECT") {
			t.Fatalf("missing effect loc refs: %v", ex.Refs)
		}
	})
	t.Run("customizable localization_key is loc", func(t *testing.T) {
		ex := extractCK3("common/customizable_localization/x.txt",
			"LifestyleFocus = {\n\ttext = {\n\t\tlocalization_key = LifestyleFocus_martial\n\t}\n}\n",
			"mod")
		if !hasRef(ex.Refs, "loc", "LifestyleFocus_martial") {
			t.Fatalf("missing localization_key ref: %v", ex.Refs)
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
	cache := &VanillaCache{Defs: []Def{{Kind: "trait", Key: "vanilla_trait", Path: "v.txt"}}}
	idx, err := BuildIndex(context.Background(), "ck3", []ModInput{
		{Origin: "parent", Root: parent, Order: 0},
		{Origin: "child", Root: child, Order: 1},
	}, cache)
	if err != nil {
		t.Fatal(err)
	}
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
	locIdx, err := BuildIndex(context.Background(), "ck3", []ModInput{{Origin: "mod", Root: root, Order: 0}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := locIdx.Loc["english"]["test.1.t"].Value; got != "Hello" {
		t.Fatalf("Loc from header = %v", locIdx.Loc)
	}
}

func TestVoteFieldValueKinds(t *testing.T) {
	defs := []Def{
		{Kind: "event_theme", Key: "seduction"},
		{Kind: "trait", Key: "brave"},
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
