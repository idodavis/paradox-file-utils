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
		{"eu5", "in_game/events/foo.txt", "event", ModeEventID},
		{"eu5", "in_game/common/religions/x.txt", "religions", ModeTopLevelKey},
	}
	for _, c := range cases {
		r := MatchExtract(c.gameID, c.path)
		if r.Kind != c.kind || r.Mode != c.mode {
			t.Errorf("MatchExtract(%q, %q) = %+v want {%s %s}", c.gameID, c.path, r, c.kind, c.mode)
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
