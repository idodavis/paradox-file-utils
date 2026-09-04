// kind_test.go covers KindLabel / KindHint blurbs and per-game overlays.
package game

import (
	"strings"
	"testing"
)

func TestKindHint(t *testing.T) {
	for _, k := range []string{
		"traits", "effect", "trigger", "scope", "buildings", "cultures",
		"script_values", "event", "gui_type", "flag_definition",
		"country_definitions", "generic_actions", "journal_entries",
		"interest_groups", "advances", "situations", "animation",
	} {
		h := KindHint("", k)
		if h == "" {
			t.Errorf("missing kind hint for %s", k)
		}
		if strings.Contains(h, "common/") || strings.Contains(h, "key = { }") {
			t.Errorf("%s hint still looks like a folder/path: %q", k, h)
		}
	}
	if !strings.Contains(KindHint("eu5", "event"), "first ID wins") {
		t.Fatalf("eu5 event hint=%q", KindHint("eu5", "event"))
	}
	if !strings.Contains(KindHint("vic3", "event"), "never self-fire") {
		t.Fatalf("vic3 event hint=%q", KindHint("vic3", "event"))
	}
	if KindLabel("coat_of_arms") != "coat of arms" {
		t.Fatalf("label=%q", KindLabel("coat_of_arms"))
	}
	if !strings.Contains(KindHint("", "animation"), "animation = key") {
		t.Fatalf("animation hint=%q", KindHint("", "animation"))
	}
}
