// override_test.go covers the winner resolution: LIOS last-mod-wins, vanilla always
// losing to a mod, and FIOS first-mod-wins for gui_type.

package model

import "testing"

func TestWinnerLIOS(t *testing.T) {
	order := map[string]int{"a": 0, "b": 1}
	defs := []Def{
		{Type: "trait", Key: "brave", Origin: "a"},
		{Type: "trait", Key: "brave", Origin: "b"},
		{Type: "trait", Key: "brave", Origin: ""}, // vanilla
	}
	w := Winner(defs, order)
	if w == nil || w.Origin != "b" {
		t.Errorf("LIOS winner = %v, want origin b", w)
	}
}

func TestWinnerFIOSGui(t *testing.T) {
	order := map[string]int{"a": 0, "b": 1}
	defs := []Def{
		{Type: "gui_type", Key: "widget", Origin: "b"},
		{Type: "gui_type", Key: "widget", Origin: "a"},
	}
	w := Winner(defs, order)
	if w == nil || w.Origin != "a" {
		t.Errorf("FIOS winner = %v, want origin a", w)
	}
}

func TestWinnerVanillaOnly(t *testing.T) {
	w := Winner([]Def{{Type: "trait", Key: "x", Origin: ""}}, map[string]int{})
	if w == nil || w.Origin != "" {
		t.Errorf("vanilla-only winner = %v, want vanilla", w)
	}
}

func TestOverridesRow(t *testing.T) {
	idx := &Index{
		Order: []string{"a", "b"},
		Defs: []Def{
			{Type: "trait", Key: "brave", Origin: "a", Path: "a.txt", Line: 1},
			{Type: "trait", Key: "brave", Origin: "b", Path: "b.txt", Line: 2},
		},
	}
	rows := Overrides(idx, nil)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Rule != "LIOS" || rows[0].Winner != "b" || len(rows[0].Sites) != 2 {
		t.Errorf("row = %+v, want LIOS winner b with 2 sites", rows[0])
	}
}

func TestOverridesIncludesVanillaShadow(t *testing.T) {
	idx := &Index{
		Order: []string{"a", "b"},
		Defs: []Def{
			{Type: "trait", Key: "brave", Origin: "a", Path: "a.txt", Line: 1},
			{Type: "trait", Key: "brave", Origin: "b", Path: "b.txt", Line: 2},
		},
	}
	cache := &Cache{Defs: []Def{
		{Type: "trait", Key: "brave", Origin: "", Path: "vanilla.txt", Line: 9},
	}}
	rows := Overrides(idx, cache)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Winner != "b" {
		t.Errorf("winner = %q, want b", rows[0].Winner)
	}
	origins := map[string]bool{}
	for _, s := range rows[0].Sites {
		origins[s.Origin] = true
	}
	if !origins[""] || !origins["a"] || !origins["b"] {
		t.Errorf("sites = %+v, want vanilla+a+b", rows[0].Sites)
	}
}
