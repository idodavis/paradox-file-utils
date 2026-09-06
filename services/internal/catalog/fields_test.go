// fields_test.go covers field typing: a field takes one def kind, or a small
// closed set of scalars, and mods overlay vanilla.

package catalog

import (
	"regexp"
	"testing"
)

func TestDeriveFieldValueKinds(t *testing.T) {
	defs := []Def{
		{Kind: "event_theme", Key: "seduction"},
		{Kind: "trait", Key: "brave"},
	}
	ok := deriveFieldValueKinds(map[string]map[string]bool{
		"theme": {"seduction": true},
	}, defs)
	if ok["theme"] != "event_theme" {
		t.Fatalf("theme = %v", ok)
	}
	conflict := deriveFieldValueKinds(map[string]map[string]bool{
		"theme": {"seduction": true, "brave": true},
	}, defs)
	if _, hit := conflict["theme"]; hit {
		t.Fatalf("conflict should omit theme: %v", conflict)
	}
}

func TestDeriveFieldEnums(t *testing.T) {
	voted := map[string]string{"theme": "event_theme"}
	got := deriveFieldEnums(map[string]map[string]map[string]bool{
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

// Unanimity is not evidence. Values that resolve to nothing are skipped, so
// without a coverage floor one coincidental hit typed the whole field — and
// every later use of it became a reference of that kind which could never
// resolve. On vanilla, `has_cultural_parameter` was typed `ep_3` on 1 of 432
// values and `remove_variable` was typed `achievements` on 2 of 392.
func TestDeriveFieldValueKindsNeedsCoverage(t *testing.T) {
	defs := []Def{{Kind: "achievements", Key: "some_achievement"}}
	thin := map[string]map[string]bool{"remove_variable": {"some_achievement": true}}
	for i := range 30 {
		thin["remove_variable"][string(rune('a'+i%26))+"_var"+string(rune('0'+i/26))] = true
	}
	if got := deriveFieldValueKinds(thin, defs); got["remove_variable"] != "" {
		t.Errorf("typed on 1 of %d values: %q", len(thin["remove_variable"]), got["remove_variable"])
	}

	// A field whose values really are that kind still types.
	rich := map[string]map[string]bool{"give_achievement": {}}
	var many []Def
	for i := range 10 {
		key := "ach_" + string(rune('a'+i))
		many = append(many, Def{Kind: "achievements", Key: key})
		rich["give_achievement"][key] = true
	}
	if got := deriveFieldValueKinds(rich, many); got["give_achievement"] != "achievements" {
		t.Errorf("10/10 coverage did not type the field: %v", got)
	}
}

// harvestGetSet replaced a regex that ran over every byte of every script file.
// The scanner must agree with the pattern it replaced, boundaries included.
func TestHarvestGetSetMatchesPattern(t *testing.T) {
	re := regexp.MustCompile(`\b((?:Get|Set)[A-Za-z0-9_]+)\b`)
	for _, src := range []string{
		"",
		"Get",
		"GetName",
		"[GetPlayer.GetName]",
		"desc = \"[ROOT.Char.GetTitledFirstName]\"",
		"xGetName and _SetFoo and forgetName",
		"SetVariable = { name = x } get_lower = 1",
		"GetName2|E] [SetY_1]",
		"Set",
		"GetA.SetB.GetC_9",
	} {
		want := map[string]bool{}
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			want[m[1]] = true
		}
		got := harvestGetSet(src)
		if len(got) != len(want) {
			t.Errorf("harvestGetSet(%q) = %v, regex = %v", src, got, want)
			continue
		}
		for k := range want {
			if !got[k] {
				t.Errorf("harvestGetSet(%q) missing %q (regex = %v)", src, k, want)
			}
		}
	}
}
