package lsp

import "testing"

// TestValueSourceOrder pins completion's resolution order. It is the second
// place the declared-beats-derived invariant lives — hover's is in
// session/resolve.go — and the two drifting apart would make a slot complete
// differently from the way it hovers.
func TestValueSourceOrder(t *testing.T) {
	want := []string{
		"fire-key",
		"localization",
		"slot-declared",
		"target-declared",
		"slot-derived",
		"macro-params",
		"macro-call-params",
		"field-enums",
	}
	if len(valueSources) != len(want) {
		t.Fatalf("source count = %d, want %d — add it to the list here too",
			len(valueSources), len(want))
	}
	for i, w := range want {
		if got := valueSources[i].name; got != w {
			t.Errorf("source %d = %q, want %q", i, got, w)
		}
	}
}

// TestValueDeclaredOutranksDerived is the invariant itself: both declared
// sources must be consulted before the install-derived field type. When
// target-declared sat below slot-derived, `has_culture_group = ` fell through
// to a derived type with too little evidence and offered yes/no.
func TestValueDeclaredOutranksDerived(t *testing.T) {
	at := func(name string) int {
		for i, s := range valueSources {
			if s.name == name {
				return i
			}
		}
		t.Fatalf("source %q missing", name)
		return -1
	}
	derived := at("slot-derived")
	for _, declared := range []string{"slot-declared", "target-declared"} {
		if i := at(declared); i > derived {
			t.Errorf("%s at %d ranks below slot-derived at %d", declared, i, derived)
		}
	}
}
