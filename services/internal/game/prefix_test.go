// prefix_test.go covers ParsePrefixed (ephemeral) and ParseTyped (type:id).

package game

import "testing"

func TestParsePrefixedUnchanged(t *testing.T) {
	t.Parallel()
	p, ok := ParsePrefixed("scope:foo")
	if !ok || p.Prefix != "scope" || p.Name != "foo" {
		t.Fatalf("scope:foo = %+v ok=%v", p, ok)
	}
	if _, ok := ParsePrefixed("culture:english"); ok {
		t.Fatal("culture:english must not be ParsePrefixed")
	}
}

func TestParseTyped(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, game, in, kind, id string
		ok                       bool
	}{
		{"culture cite", "ck3", "culture:english", "culture", "english", true},
		{"bare culture", "ck3", "english", "", "", false},
		{"dotted link", "ck3", "capital_province.culture", "", "", false},
		{"dollar interp", "ck3", "$COA$", "", "", false},
		{"scope ephemeral", "ck3", "scope:foo", "", "", false},
		{"var ephemeral", "ck3", "var:gold", "", "", false},
		{"eu5 inject", "eu5", "INJECT:foo", "", "", false},
		{"trait cite", "ck3", "trait:brave", "traits", "brave", true},
		{"coa cite", "ck3", "coat_of_arms:b_appleby", "coat_of_arms", "b_appleby", true},
		{"titles cite", "ck3", "titles:e_hre", "title", "e_hre", true},
		{"title cite", "ck3", "title:k_france", "title", "k_france", true},
		{"faith cite", "ck3", "faith:catholic", "faith", "catholic", true},
		{"script_value cite", "ck3", "script_value:max_soldiers", "script_value", "max_soldiers", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, id, ok := ParseTyped(c.game, c.in)
			if ok != c.ok || kind != c.kind || id != c.id {
				t.Fatalf("ParseTyped(%q) = %q %q %v want %q %q %v",
					c.in, kind, id, ok, c.kind, c.id, c.ok)
			}
		})
	}
}

func TestSkipObjectRHS(t *testing.T) {
	t.Parallel()
	if !SkipObjectRHS("$COA$") || !SkipObjectRHS("capital_province.culture") {
		t.Fatal("expected skip")
	}
	if SkipObjectRHS("english") || SkipObjectRHS("culture:english") {
		t.Fatal("ids must not skip")
	}
}

func TestSkipFieldRHS(t *testing.T) {
	t.Parallel()
	if !SkipFieldRHS("title", "target_titles") || !SkipFieldRHS("title", "none") {
		t.Fatal("CB target selectors must skip")
	}
	if !SkipFieldRHS("landed_titles", "target_titles") {
		t.Fatal("folder alias must fold to title")
	}
	if SkipFieldRHS("title", "e_hre") || SkipFieldRHS("faith", "catholic") {
		t.Fatal("real ids must not skip")
	}
}
