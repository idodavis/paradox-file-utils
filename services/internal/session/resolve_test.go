package session

import "testing"

// TestResolutionOrder pins the order itself. Every LSP bug found so far was a
// derived source ranked above a declared one, and each fix moved one branch of
// a 200-line if-ladder where the move was invisible in review. Naming the order
// here makes inserting a source a deliberate edit to this list.
func TestResolutionOrder(t *testing.T) {
	want := []string{
		"language-prefix",
		"typed-cite",
		"slot-declared",
		"slot-derived",
		"definition",
		"vocabulary",
		"localization",
		"data-function",
		"macro-kind",
		"field-doc",
		"unresolved",
	}
	if len(wordResolvers) != len(want) {
		t.Fatalf("resolver count = %d, want %d — add it to the list here too",
			len(wordResolvers), len(want))
	}
	for i, w := range want {
		if got := wordResolvers[i].name; got != w {
			t.Errorf("resolver %d = %q, want %q", i, got, w)
		}
	}
}

// TestDeclaredOutranksDerived is the invariant the order exists to hold: what
// the game declares about a slot is consulted before what PMT inferred about
// it. The derived pass carries coverage floors, so it is silent exactly where
// the declared answer was available all along.
func TestDeclaredOutranksDerived(t *testing.T) {
	declared, derived := -1, -1
	for i, r := range wordResolvers {
		switch r.name {
		case "slot-declared":
			declared = i
		case "slot-derived":
			derived = i
		}
	}
	if declared < 0 || derived < 0 {
		t.Fatal("slot resolvers missing")
	}
	if declared > derived {
		t.Errorf("slot-declared at %d ranks below slot-derived at %d", declared, derived)
	}
}

// TestUnresolvedIsLast pins the floor. Every other resolver may decline; this
// one answers for anything left, so a source added below it would never run.
func TestUnresolvedIsLast(t *testing.T) {
	last := wordResolvers[len(wordResolvers)-1]
	if last.name != "unresolved" {
		t.Fatalf("last resolver = %q, want %q", last.name, "unresolved")
	}
	for i, r := range wordResolvers[:len(wordResolvers)-1] {
		if r.name == "unresolved" {
			t.Errorf("unresolved also at %d", i)
		}
	}
}

// TestVocabularyOutranksLocalization pins the fix for the most visible symptom
// of the ordering fault: a declared effect or trigger hovering as the quoted
// player-facing string, because Paradox localizes a great many ordinary words.
func TestVocabularyOutranksLocalization(t *testing.T) {
	vocab, locz := -1, -1
	for i, r := range wordResolvers {
		switch r.name {
		case "vocabulary":
			vocab = i
		case "localization":
			locz = i
		}
	}
	if vocab < 0 || locz < 0 {
		t.Fatal("resolvers missing")
	}
	if vocab > locz {
		t.Errorf("vocabulary at %d ranks below localization at %d", vocab, locz)
	}
}
