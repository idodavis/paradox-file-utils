// properties_test.go covers STRICT/BROAD/none classification of loc properties.

package loc

import "testing"

func TestClassify(t *testing.T) {
	cases := map[string]Property{
		"title":   PropStrict,
		"DESC":    PropStrict, // case-insensitive
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
}
