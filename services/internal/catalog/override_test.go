// override_test.go covers Winner (LIOS/FIOS) and Contests overlay vs conflict.

package catalog

import "testing"

func TestWinner(t *testing.T) {
	tests := []struct {
		name   string
		defs   []Def
		order  map[string]int
		origin string
	}{
		{
			"LIOS last mod",
			[]Def{
				{Type: "trait", Key: "brave", Origin: "a"},
				{Type: "trait", Key: "brave", Origin: "b"},
				{Type: "trait", Key: "brave", Origin: ""},
			},
			map[string]int{"a": 0, "b": 1}, "b",
		},
		{
			"FIOS gui",
			[]Def{
				{Type: "gui_type", Key: "widget", Origin: "b"},
				{Type: "gui_type", Key: "widget", Origin: "a"},
			},
			map[string]int{"a": 0, "b": 1}, "a",
		},
		{
			"vanilla only",
			[]Def{{Type: "trait", Key: "x", Origin: ""}},
			map[string]int{}, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := Winner(tt.defs, tt.order)
			if w == nil || w.Origin != tt.origin {
				t.Errorf("winner = %v, want origin %q", w, tt.origin)
			}
		})
	}
}

func TestContests(t *testing.T) {
	vanilla := []Def{{Type: "trait", Key: "brave", Origin: "", Path: "vanilla.txt", Line: 9}}
	modPair := []Def{
		{Type: "trait", Key: "brave", Origin: "a", Path: "a.txt", Line: 1},
		{Type: "trait", Key: "brave", Origin: "b", Path: "b.txt", Line: 2},
	}
	tests := []struct {
		name    string
		defs    []Def
		cache   []Def
		order   []string
		n       int
		overlay bool
		winner  string
		mods    int
	}{
		{"mod vs mod LIOS", modPair, nil, []string{"a", "b"}, 1, false, "b", 2},
		{"vanilla shadow conflict", modPair, vanilla, []string{"a", "b"}, 1, false, "b", 2},
		{"vanilla overlay only",
			[]Def{{Type: "trait", Key: "brave", Origin: "a", Path: "a.txt", Line: 1}},
			vanilla, []string{"a"}, 1, true, "", 1},
		{
			"skip loc and isolated kinds",
			[]Def{
				{Type: "loc_key", Key: "brave", Origin: "a", Path: "a.yml"},
				{Type: "loc_key", Key: "brave", Origin: "b", Path: "b.yml"},
				{Type: "mod_descriptor", Key: "name", Origin: "a", Path: "a.mod"},
				{Type: "mod_descriptor", Key: "name", Origin: "b", Path: "b.mod"},
				{Type: "trait", Key: "same", Origin: "a", Path: "t.txt"},
				{Type: "event", Key: "same", Origin: "b", Path: "e.txt"},
			},
			nil, []string{"a", "b"}, 0, false, "", 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := Contests(tt.defs, tt.cache, tt.order)
			if len(rows) != tt.n {
				t.Fatalf("rows = %d, want %d (%+v)", len(rows), tt.n, rows)
			}
			if tt.n == 0 {
				return
			}
			r := rows[0]
			if r.Overlay != tt.overlay || r.ModCount != tt.mods {
				t.Errorf("overlay=%v mods=%d", r.Overlay, r.ModCount)
			}
			if tt.winner != "" && r.Winner != tt.winner {
				t.Errorf("winner = %q, want %s", r.Winner, tt.winner)
			}
		})
	}
}
