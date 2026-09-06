// script_test.go covers the shared grammar: kind canonicalisation, trigger and
// effect slots, and the assignment keys that name an ephemeral value.

package jomini

import "testing"

func TestCanonicalKind(t *testing.T) {
	cases := []struct{ in, want string }{
		{"scripted_effects", "scripted_effects"},
		{"event namespace", "event_namespace"},
		{"landed_titles", "landed_titles"},
		{"title", "title"},
		{"culture", "culture"},
	}
	for _, c := range cases {
		if got := CanonicalKind(c.in); got != c.want {
			t.Errorf("CanonicalKind(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

func TestScriptSlot(t *testing.T) {
	cases := []struct {
		key, want string
	}{
		{"trigger", "trigger"},
		{"limit", "trigger"},
		{"AND", "trigger"},
		{"any_courtier", "trigger"},
		{"immediate", "effect"},
		{"after", "effect"},
		{"every_child", "effect"},
		{"effect", "effect"},
		{"title", ""},
		{"type", ""},
		{"test.1", ""},
	}
	for _, c := range cases {
		if got := ScriptSlot(c.key); got != c.want {
			t.Errorf("ScriptSlot(%q) = %q want %q", c.key, got, c.want)
		}
	}
}

// These names are shared Jomini, not a per-game table: 17 of the 18 are declared
// in effects.log / triggers.log on CK3, Victoria 3 and EU5 alike, and the
// eighteenth (save_temporary_value_as) is used in all three games' vanilla
// script while being declared by none of them.
func TestScriptNameAndPrefixKind(t *testing.T) {
	if r, ok := ScriptName("has_variable"); !ok || r.Kind != "var" {
		t.Fatalf("has_variable: %+v ok=%v", r, ok)
	}
	if r, ok := ScriptName("set_variable"); !ok || !r.IsDef || r.InnerKey != "name" {
		t.Fatalf("set_variable: %+v ok=%v", r, ok)
	}
	if PrefixKind("var") != "" || PrefixKind("scope") != "saved_scope" {
		t.Fatal("PrefixKind")
	}
	if !IsSaveScopeKey("save_scope_as") || IsSaveScopeKey("save_scope_value_as") {
		t.Fatal("IsSaveScopeKey")
	}
	if !IsSaveScopeValueKey("save_scope_value_as") {
		t.Fatal("IsSaveScopeValueKey")
	}
}

func TestIsEphemeral(t *testing.T) {
	for _, k := range []string{"saved_scope", "var", "global_var", "flag", "script_param"} {
		if !IsEphemeral(k) {
			t.Errorf("IsEphemeral(%q) = false", k)
		}
	}
	if IsEphemeral("traits") {
		t.Fatal("traits not ephemeral")
	}
}

func TestScriptParamSpan(t *testing.T) {
	src := `add_trait = $TRAIT$`
	name, start, end, ok := ScriptParamSpan(src, 14)
	if !ok || name != "TRAIT" || src[start:end] != "$TRAIT$" {
		t.Fatalf("span name=%q %d:%d ok=%v", name, start, end, ok)
	}
	if _, _, _, ok := ScriptParamSpan(`add_trait = brave`, 14); ok {
		t.Fatal("plain scalar is not a param span")
	}
}
