// rules_test.go covers KeyIdentity, MatchExtract, call/FIOS kinds, and registry Get.

package game

import "testing"

func TestKeyIdentityStripsEU5Prefix(t *testing.T) {
	cases := []struct {
		gameID, in, want string
	}{
		{"eu5", "INJECT:foo", "foo"},
		{"eu5", "REPLACE:bar", "bar"},
		{"", "INJECT:foo", "foo"},           // merge: game-agnostic strip
		{"ck3", "INJECT:foo", "INJECT:foo"}, // ck3 has no entry modes -> untouched
		{"eu5", "plain", "plain"},
		{"eu5", "\"quoted\"", "quoted"},
		{"", "  spaced  ", "spaced"},
	}
	for _, c := range cases {
		if got := KeyIdentity(c.gameID, c.in); got != c.want {
			t.Errorf("KeyIdentity(%q, %q) = %q want %q", c.gameID, c.in, got, c.want)
		}
	}
}

func TestMatchExtract(t *testing.T) {
	cases := []struct {
		gameID, path string
		kind         string
		mode         ExtractMode
	}{
		{"ck3", "events/my_events.txt", "event", ModeEventID},
		{"ck3", "common/traits/00_traits.txt", "traits", ModeTopLevelKey},
		{"ck3", "gui/window.gui", "gui_type", ModeGUIType},
		{"ck3", "localization/english/foo_l_english.yml", "loc_key", ModeLocKey},
		{"ck3", "descriptor.mod", "mod_descriptor", ModeTopLevelKey},
		{"ck3", "my_mod.mod", "mod_descriptor", ModeTopLevelKey},
		{"ck3", "common/religion/religion_types/00_islam.txt", "religion", ModeTopLevelKey},
		{"eu5", "in_game/events/foo.txt", "event", ModeEventID},
		{"eu5", "in_game/common/religions/x.txt", "religion", ModeTopLevelKey},
	}
	for _, c := range cases {
		r := MatchExtract(c.gameID, c.path)
		if r.Kind != c.kind || r.Mode != c.mode {
			t.Errorf("MatchExtract(%q, %q) = %+v want {%s %s}", c.gameID, c.path, r, c.kind, c.mode)
		}
	}
}

func TestCanonicalKind(t *testing.T) {
	cases := []struct{ in, want string }{
		{"scripted_effects", "scripted_effect"},
		{"events", "event"},
		{"event namespace", "namespace"},
		{"landed_titles", "title"},
		{"titles", "title"},
		{"cultures", "culture"},
		{"religions", "religion"},
		{"religion_types", "religion"},
		{"faiths", "faith"},
		{"title", "title"},
		{"culture", "culture"},
	}
	for _, c := range cases {
		if got := CanonicalKind(c.in); got != c.want {
			t.Errorf("CanonicalKind(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

func TestCallAndFIOSKinds(t *testing.T) {
	if !IsCallKind("scripted_effect") || IsCallKind("traits") {
		t.Fatal("IsCallKind wrong")
	}
	cases := []struct {
		gameID, kind string
		want         bool
	}{
		{"ck3", "gui_type", true},
		{"ck3", "event", false},
		{"vic3", "gui_type", true},
		{"vic3", "event", true},
		{"eu5", "event", true},
		{"eu5", "gui_type", false},
		{"", "gui_type", true},
		{"", "event", false},
		{"unknown", "gui_type", true},
	}
	for _, c := range cases {
		if got := IsFIOS(c.gameID, c.kind); got != c.want {
			t.Errorf("IsFIOS(%q, %q) = %v want %v", c.gameID, c.kind, got, c.want)
		}
	}
}

func TestGetRegistry(t *testing.T) {
	for _, id := range []string{"ck3", "vic3", "eu5"} {
		if Get(id) == nil {
			t.Errorf("Get(%q) is nil", id)
		}
	}
	if Get("unknown") != nil {
		t.Error("Get(unknown) should be nil")
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

func TestScriptNameAndRefFieldKind(t *testing.T) {
	t.Run("ck3 character flag", func(t *testing.T) {
		r, ok := ScriptName("ck3", "add_character_flag")
		if !ok || r.Kind != "character_flag" || !r.IsDef || r.InnerKey != "flag" {
			t.Fatalf("add_character_flag: %+v ok=%v", r, ok)
		}
		if RefFieldKind("ck3", "has_character_flag") != "character_flag" {
			t.Fatal("RefFieldKind has_character_flag")
		}
		if _, ok := ScriptName("vic3", "add_character_flag"); ok {
			t.Fatal("vic3 must not have character flags")
		}
	})
	t.Run("variables shared", func(t *testing.T) {
		for _, g := range []string{"ck3", "vic3", "eu5"} {
			if RefFieldKind(g, "has_variable") != "variable" {
				t.Fatalf("%s has_variable", g)
			}
			if PrefixKind("var") != "variable" || PrefixKind("global_var") != "global_variable" {
				t.Fatal("PrefixKind")
			}
		}
	})
	t.Run("coa fields", func(t *testing.T) {
		if RefFieldKind("ck3", "coat_of_arms") != "coat_of_arms" {
			t.Fatal("ck3 coat_of_arms")
		}
		if RefFieldKind("ck3", "set_coa") != "coat_of_arms" {
			t.Fatal("ck3 set_coa")
		}
		if RefFieldKind("vic3", "coa") != "coat_of_arms" {
			t.Fatal("vic3 coa")
		}
		if RefFieldKind("eu5", "change_country_flag") != "coat_of_arms" {
			t.Fatal("eu5 change_country_flag")
		}
		if RefFieldKind("ck3", "coa") != "" {
			t.Fatal("ck3 must not map coa")
		}
	})
	t.Run("namespace field", func(t *testing.T) {
		if RefFieldKind("ck3", "namespace") != "namespace" {
			t.Fatal("namespace field")
		}
	})
	t.Run("title fields", func(t *testing.T) {
		if RefFieldKind("ck3", "title") != "title" || RefFieldKind("ck3", "titles") != "title" {
			t.Fatal("title/titles field")
		}
	})
	t.Run("vic3 entry modes", func(t *testing.T) {
		g := Get("vic3")
		if g == nil || len(g.EntryModes) < 2 {
			t.Fatalf("vic3 EntryModes=%v", g)
		}
		if KeyIdentity("vic3", "INJECT:ALD") != "ALD" {
			t.Fatal("vic3 INJECT strip")
		}
	})
	t.Run("ephemeral", func(t *testing.T) {
		if !IsEphemeral("saved_scope") || !IsEphemeral("character_flag") {
			t.Fatal("IsEphemeral")
		}
		if IsEphemeral("traits") {
			t.Fatal("traits not ephemeral")
		}
	})
}

func TestRequiredLocKeysDecision(t *testing.T) {
	id := "ai_mogyer_adopt_christianity"
	if keys := RequiredLocKeys("decision", id); len(keys) != 0 {
		t.Fatalf("RequiredLocKeys(decision) = %v", keys)
	}
	got := ConventionLocKeys("decision", id)
	want := []string{
		id, id + "_desc", id + "_tooltip", id + "_confirm", id + "_tt",
	}
	if len(got) != len(want) {
		t.Fatalf("ConventionLocKeys(decision) = %v", got)
	}
	for i, k := range want {
		if got[i] != k {
			t.Fatalf("ConventionLocKeys(decision) = %v", got)
		}
	}
}
