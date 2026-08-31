// properties_test.go covers STRICT/BROAD/none classification of loc properties.

package loc

import "testing"

func TestClassify(t *testing.T) {
	cases := map[string]Property{
		"title":   PropStrict,
		"desc":    PropStrict,
		"TITLE":   PropNone, // trigger/effect arg, not event loc
		"DESC":    PropNone,
		"name":    PropBroad,
		"tooltip": PropBroad,
		"id":      PropNone,
		"color":   PropNone,
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
