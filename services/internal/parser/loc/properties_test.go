// properties_test.go covers STRICT/BROAD/none classification of loc properties.

package loc

import "testing"

func TestClassify(t *testing.T) {
	cases := map[string]Property{
		"title":             PropStrict,
		"desc":              PropStrict,
		"first":             PropStrict,
		"third":             PropStrict,
		"global":            PropStrict,
		"first_not":         PropStrict,
		"first_past":        PropStrict,
		"global_past_neg":   PropStrict,
		"localization_key":  PropBroad, // see TestLocalizationKeyIsBroadNotStrict
		"selection_tooltip": PropStrict,
		"war_name":          PropStrict,
		"cb_name":           PropStrict,
		"notification_text": PropStrict,
		"notification":      PropBroad,
		"TITLE":             PropNone, // trigger/effect arg, not event loc
		"DESC":              PropNone,
		"name":              PropBroad,
		"tooltip":           PropBroad,
		"id":                PropNone,
		"color":             PropNone,
	}
	for prop, want := range cases {
		if got := Classify(prop); got != want {
			t.Errorf("Classify(%q) = %q want %q", prop, got, want)
		}
	}
	if LooksLikeKey("yes") || LooksLikeKey("NONE") || !LooksLikeKey("evt.1.t") {
		t.Errorf("LooksLikeKey yes/none/evt.1.t")
	}
	if LooksLikeKey("scope:child") {
		t.Errorf("LooksLikeKey(scope:child) = true, want false")
	}
	if !LooksLikeKey("primary_title") {
		t.Errorf("LooksLikeKey(primary_title) = false, want true (key shape)")
	}
}

// Quoting proves nothing: Paradox quotes real keys and also writes display text
// into the same properties. The space is what separates them — a localization
// file is `key:0 "value"`, so a key is a single token.
func TestLooksLikeKeyRejectsProse(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"HRE_CONQUEST_WAR_NAME", true},
		{"evt.1.t", true},
		{"Always make coronations!", false},
		{"The Red Keep", false},
		{"Wrong culture", false},
		{"Base test value", false},
		{"trailing ", false},
		{"tab\there", false},
	}
	for _, tc := range cases {
		if got := LooksLikeKey(tc.in); got != tc.want {
			t.Errorf("LooksLikeKey(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// A property is strict only if an unresolved value is genuinely a defect.
// `localization_key` is not: customizable localization completes the stem at
// runtime, so `localization_key = CustomLoc_BR_male_` names a key that does not
// and should not exist. Measured over vanilla — where nothing is missing by
// definition — it failed to resolve 2,607 times on CK3, 11,597 on Victoria 3
// and 46,481 on EU5.
func TestLocalizationKeyIsBroadNotStrict(t *testing.T) {
	if got := Classify("localization_key"); got != PropBroad {
		t.Errorf("Classify(localization_key) = %v, want %v", got, PropBroad)
	}
	// The properties that really do demand a key keep demanding one.
	for _, p := range []string{"desc", "title", "custom_tooltip", "war_name"} {
		if got := Classify(p); got != PropStrict {
			t.Errorf("Classify(%s) = %v, want %v", p, got, PropStrict)
		}
	}
}

// TestLooksLikeKeyMacroParams pins that a key completed at call time is never
// demanded. `desc = $TT$` and `desc = $title$` both reached Workspace Health as
// missing localization on real Victoria 3 mods.
func TestLooksLikeKeyMacroParams(t *testing.T) {
	for _, s := range []string{"$TT$", "$title$", "evt_$TYPE$_desc", "$COA$"} {
		if LooksLikeKey(s) {
			t.Errorf("LooksLikeKey(%q) = true, want false", s)
		}
	}
	// Still a key when nothing is substituted.
	for _, s := range []string{"HRE_CONQUEST_WAR_NAME", "my_event.0001.desc"} {
		if !LooksLikeKey(s) {
			t.Errorf("LooksLikeKey(%q) = false, want true", s)
		}
	}
}
