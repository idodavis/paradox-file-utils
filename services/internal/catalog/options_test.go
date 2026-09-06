// options_test.go covers the field-to-nested-block join that finds option
// databases — game-rule options, cultural parameters — which are referenced by
// a field rather than a `prefix:id` cite.

package catalog

import (
	"fmt"
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/parser/jomini"
)

func parseAs(rel, body string) parsedScript {
	return parsedScript{
		f:   fileRef{rel: rel, abs: rel},
		res: jomini.Parse(jomini.Normalize(body)),
	}
}

// ruleFixture writes n game rules, each with two options plus the ordinary
// fields a rule carries, and a file referencing every option.
func ruleFixture(n int) (files []parsedScript, refs map[string]bool) {
	var rules, uses strings.Builder
	refs = map[string]bool{}
	for i := range n {
		rule := fmt.Sprintf("rule_%d", i)
		on := fmt.Sprintf("opt_%d_on", i)
		off := fmt.Sprintf("opt_%d_off", i)
		fmt.Fprintf(&rules, "%s = {\n\tcategories = { c }\n\tdefault = %s\n"+
			"\t%s = { flag = f }\n\t%s = { flag = f }\n}\n", rule, on, on, off)
		fmt.Fprintf(&uses, "e_%d = { limit = { has_game_rule = %s } }\n", i, on)
		fmt.Fprintf(&uses, "e_%db = { limit = { has_game_rule = %s } }\n", i, off)
		refs[on], refs[off] = true, true
	}
	return []parsedScript{
		parseAs("common/game_rules/00_rules.txt", rules.String()),
		parseAs("common/scripted_triggers/use.txt", uses.String()),
	}, refs
}

func fieldRHSOf(files []parsedScript) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, f := range files {
		for field, vals := range extractFieldRHS(f.res.Root) {
			if out[field] == nil {
				out[field] = map[string]bool{}
			}
			for v := range vals {
				out[field][v] = true
			}
		}
	}
	return out
}

func TestDeriveNestedOptionsGameRules(t *testing.T) {
	files, refs := ruleFixture(6) // 12 options, over minOptionRefs
	var defs []Def
	for _, f := range files {
		defs = append(defs, extractForms("ck3", f.f.abs, f.f.rel, "", f.res, false).Defs...)
	}
	shapes := deriveNestedOptions("ck3", defs, fieldRHSOf(files), ownersOf(files))

	var got *NestedShape
	for i := range shapes {
		if shapes[i].ParentKind == "game_rules" {
			got = &shapes[i]
		}
	}
	if got == nil {
		t.Fatalf("no game_rules option shape; shapes=%+v", shapes)
	}
	if got.ChildKind != "game_rule" {
		t.Errorf("ChildKind = %q, want game_rule (named from has_game_rule)", got.ChildKind)
	}
	if !got.IsOption {
		t.Error("shape must be marked IsOption")
	}
	// The complement is stored, so the ordinary fields are what gets skipped.
	skip := map[string]bool{}
	for _, s := range got.SkipKeys {
		skip[s] = true
	}
	for _, want := range []string{"categories", "default"} {
		if !skip[want] {
			t.Errorf("%q should be skipped as an ordinary field; skip=%v", want, got.SkipKeys)
		}
	}
	for name := range refs {
		if skip[name] {
			t.Errorf("option %q was skipped", name)
		}
	}

	// Harvesting the rules file yields the options and not the fields.
	rules := files[0]
	harvested := applyNestedShape("ck3", rules.f.abs, "", rules.res.Root,
		rules.res.Lines(), *got, nil)
	byKey := map[string]bool{}
	for _, d := range harvested {
		if d.Kind != "game_rule" {
			t.Errorf("unexpected kind %q for %q", d.Kind, d.Key)
		}
		byKey[d.Key] = true
	}
	for name := range refs {
		if !byKey[name] {
			t.Errorf("option %q not harvested; got %v", name, byKey)
		}
	}
	for _, no := range []string{"categories", "default"} {
		if byKey[no] {
			t.Errorf("ordinary field %q harvested as an option", no)
		}
	}
}

// The complement is stored precisely so a mod's own new option is a row by
// default. A name list derived from vanilla would freeze it out, which is what
// left `events_karling_decline_rule` dangling on a real CK3 mod.
func TestOptionShapeHarvestsModAddedOption(t *testing.T) {
	files, _ := ruleFixture(6)
	var defs []Def
	for _, f := range files {
		defs = append(defs, extractForms("ck3", f.f.abs, f.f.rel, "", f.res, false).Defs...)
	}
	shapes := deriveNestedOptions("ck3", defs, fieldRHSOf(files), ownersOf(files))
	var shape NestedShape
	for _, s := range shapes {
		if s.ParentKind == "game_rules" {
			shape = s
		}
	}
	mod := parseAs("common/game_rules/zz_mod.txt",
		"my_new_rule = {\n\tcategories = { c }\n\tdefault = mod_quiet\n"+
			"\tmod_quiet = { flag = f }\n}\n")
	got := applyNestedShape("ck3", mod.f.abs, "mod", mod.res.Root,
		mod.res.Lines(), shape, nil)
	if !findDef(got, "game_rule", "mod_quiet") {
		t.Fatalf("mod-added option not harvested: %v", got)
	}
	if findDef(got, "game_rule", "categories") || findDef(got, "game_rule", "default") {
		t.Errorf("ordinary fields harvested from the mod file: %v", got)
	}
}

// A cultural parameter is `name = yes` — a scalar. Requiring a block value
// harvested none of them.
func TestOptionShapeHarvestsScalarRows(t *testing.T) {
	var trads, uses strings.Builder
	for i := range 5 {
		fmt.Fprintf(&trads, "trad_%d = {\n\tcategory = c\n\tparameters = {\n"+
			"\t\tparam_%da = yes\n\t\tparam_%db = yes\n\t}\n}\n", i, i, i)
		fmt.Fprintf(&uses, "t_%d = { culture = { has_cultural_parameter = param_%da } }\n", i, i)
		fmt.Fprintf(&uses, "t_%db = { culture = { has_cultural_parameter = param_%db } }\n", i, i)
	}
	files := []parsedScript{
		parseAs("common/culture/traditions/00_t.txt", trads.String()),
		parseAs("common/scripted_triggers/use.txt", uses.String()),
	}
	var defs []Def
	for _, f := range files {
		defs = append(defs, extractForms("ck3", f.f.abs, f.f.rel, "", f.res, false).Defs...)
	}
	shapes := deriveNestedOptions("ck3", defs, fieldRHSOf(files), ownersOf(files))
	var shape NestedShape
	for _, s := range shapes {
		if s.ChildKind == "cultural_parameter" {
			shape = s
		}
	}
	if shape.ParentKind != "traditions" || shape.GroupKey != "parameters" {
		t.Fatalf("shape = %+v, want traditions/parameters -> cultural_parameter", shape)
	}
	got := applyNestedShape("ck3", files[0].f.abs, "", files[0].res.Root,
		files[0].res.Lines(), shape, nil)
	if !findDef(got, "cultural_parameter", "param_0a") ||
		!findDef(got, "cultural_parameter", "param_4b") {
		t.Fatalf("scalar rows not harvested: %v", got)
	}
	if findDef(got, "cultural_parameter", "category") {
		t.Errorf("field outside the group harvested: %v", got)
	}
}

// Below minOptionRefs a match is coincidence, which is how unrelated blocks
// were claimed on every install measured.
func TestDeriveNestedOptionsIgnoresThinEvidence(t *testing.T) {
	files, _ := ruleFixture(2) // 4 options, under minOptionRefs
	var defs []Def
	for _, f := range files {
		defs = append(defs, extractForms("ck3", f.f.abs, f.f.rel, "", f.res, false).Defs...)
	}
	if got := deriveNestedOptions("ck3", defs, fieldRHSOf(files), ownersOf(files)); len(got) != 0 {
		t.Fatalf("claimed a database on %d references: %+v", 4, got)
	}
}

// ownersOf indexes the fixture the way the shared derive walk does.
func ownersOf(files []parsedScript) map[optionOwner]*ownerKeys {
	out := map[optionOwner]*ownerKeys{}
	for _, f := range files {
		addOptionOwners("ck3", f.f.rel, f.res.Root, out)
	}
	return out
}
