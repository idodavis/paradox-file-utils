// override_test.go covers Winner (LIOS/FIOS) and Contests overlay vs conflict.

package catalog

import "testing"

func TestWinner(t *testing.T) {
	tests := []struct {
		name   string
		gameID string
		defs   []Def
		order  map[string]int
		origin string
	}{
		{
			"LIOS last mod", "",
			[]Def{
				{Kind: "trait", Key: "brave", Origin: "a"},
				{Kind: "trait", Key: "brave", Origin: "b"},
				{Kind: "trait", Key: "brave", Origin: ""},
			},
			map[string]int{"a": 0, "b": 1}, "b",
		},
		{
			"FIOS gui", "ck3",
			[]Def{
				{Kind: "gui_type", Key: "widget", Origin: "b"},
				{Kind: "gui_type", Key: "widget", Origin: "a"},
			},
			map[string]int{"a": 0, "b": 1}, "a",
		},
		{
			"vanilla only", "",
			[]Def{{Kind: "trait", Key: "x", Origin: ""}},
			map[string]int{}, "",
		},
		{
			"vic3 event FIOS", "vic3",
			[]Def{
				{Kind: "event", Key: "e.1", Origin: "b"},
				{Kind: "event", Key: "e.1", Origin: "a"},
			},
			map[string]int{"a": 0, "b": 1}, "a",
		},
		{
			"eu5 gui_type LIOS", "eu5",
			[]Def{
				{Kind: "gui_type", Key: "widget", Origin: "a"},
				{Kind: "gui_type", Key: "widget", Origin: "b"},
			},
			map[string]int{"a": 0, "b": 1}, "b",
		},
		{
			"ck3 event LIOS", "ck3",
			[]Def{
				{Kind: "event", Key: "e.1", Origin: "a"},
				{Kind: "event", Key: "e.1", Origin: "b"},
			},
			map[string]int{"a": 0, "b": 1}, "b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := Winner(tt.gameID, tt.defs, tt.order)
			if w == nil || w.Origin != tt.origin {
				t.Errorf("winner = %v, want origin %q", w, tt.origin)
			}
		})
	}
}

func TestContests(t *testing.T) {
	vanilla := []Def{{Kind: "trait", Key: "brave", Origin: "", Path: "vanilla.txt", Line: 9}}
	modPair := []Def{
		{Kind: "trait", Key: "brave", Origin: "a", Path: "a.txt", Line: 1},
		{Kind: "trait", Key: "brave", Origin: "b", Path: "b.txt", Line: 2},
	}
	tests := []struct {
		name    string
		gameID  string
		defs    []Def
		cache   []Def
		order   []string
		n       int
		overlay bool
		winner  string
		mods    int
	}{
		{"mod vs mod LIOS", "", modPair, nil, []string{"a", "b"}, 1, false, "b", 2},
		{"vanilla shadow conflict", "", modPair, vanilla, []string{"a", "b"}, 1, false, "b", 2},
		{"vanilla overlay only", "",
			[]Def{{Kind: "trait", Key: "brave", Origin: "a", Path: "a.txt", Line: 1}},
			vanilla, []string{"a"}, 1, true, "", 1},
		{
			"skip loc and isolated kinds", "",
			[]Def{
				{Kind: "loc_key", Key: "brave", Origin: "a", Path: "a.yml"},
				{Kind: "loc_key", Key: "brave", Origin: "b", Path: "b.yml"},
				{Kind: "mod_descriptor", Key: "name", Origin: "a", Path: "a.mod"},
				{Kind: "mod_descriptor", Key: "name", Origin: "b", Path: "b.mod"},
				{Kind: "trait", Key: "same", Origin: "a", Path: "t.txt"},
				{Kind: "event", Key: "same", Origin: "b", Path: "e.txt"},
			},
			nil, []string{"a", "b"}, 0, false, "", 0,
		},
		{
			"skip shared namespace", "",
			[]Def{
				{Kind: "namespace", Key: "ns", Origin: "a", Path: "a.txt"},
				{Kind: "namespace", Key: "ns", Origin: "b", Path: "b.txt"},
			},
			[]Def{{Kind: "namespace", Key: "ns", Origin: "", Path: "v.txt"}},
			[]string{"a", "b"}, 0, false, "", 0,
		},
		{
			"vic3 event FIOS", "vic3",
			[]Def{
				{Kind: "event", Key: "e.1", Origin: "a", Path: "a.txt", Line: 1},
				{Kind: "event", Key: "e.1", Origin: "b", Path: "b.txt", Line: 2},
			},
			nil, []string{"a", "b"}, 1, false, "a", 2,
		},
		{
			"eu5 gui_type LIOS", "eu5",
			[]Def{
				{Kind: "gui_type", Key: "w", Origin: "a", Path: "a.gui", Line: 1},
				{Kind: "gui_type", Key: "w", Origin: "b", Path: "b.gui", Line: 2},
			},
			nil, []string{"a", "b"}, 1, false, "b", 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := Contests(tt.gameID, tt.defs, tt.cache, tt.order)
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
			if tt.name == "vic3 event FIOS" && r.Rule != "FIOS" {
				t.Errorf("rule = %q, want FIOS", r.Rule)
			}
			if tt.name == "eu5 gui_type LIOS" && r.Rule != "LIOS" {
				t.Errorf("rule = %q, want LIOS", r.Rule)
			}
		})
	}
}
