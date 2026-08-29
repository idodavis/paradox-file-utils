// graph_test.go covers defs-first orphans, via-effect hops, event detail,
// coordinates-free payload, FIOS/LIOS winners, loc coverage, and a vic3/eu5 smoke parse.

package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/session"
)

func write(t *testing.T, root, rel, body string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOrphanAndViaHops(t *testing.T) {
	root := t.TempDir()
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "events/ns.txt", `namespace = ns

ns.1 = {
	type = character_event
	immediate = {
		ns_outer_effect = yes
	}
}

ns.2 = {
	type = character_event
}

ns.9 = {
	type = character_event
}
`)
	write(t, root, "common/scripted_effects/e.txt", `ns_outer_effect = {
	ns_inner_effect = yes
}

ns_inner_effect = {
	trigger_event = ns.2
}
`)
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	g := Graph(s, EventGraphParams{Namespace: "ns"})

	ids := map[string]bool{}
	for _, n := range g.Nodes {
		ids[n.ID] = true
		if n.ID == "ns.9" && (n.Fires != 0) {
			t.Errorf("orphan ns.9 should fire nothing")
		}
	}
	if !ids["ns.1"] || !ids["ns.2"] || !ids["ns.9"] {
		t.Fatalf("defs-first nodes = %v, want ns.1, ns.2, ns.9", ids)
	}
	var via bool
	for _, e := range g.Edges {
		if e.From == "ns.1" && e.To == "ns.2" && strings.Contains(e.Label, "via") {
			via = true
		}
		if e.From == "ns.1" && e.To == "ns.2" && strings.Count(e.Label, "→") > 2 {
			t.Errorf("via hop chain too long: %q", e.Label)
		}
	}
	if !via {
		t.Fatalf("missing via-effect edge ns.1→ns.2; edges=%v", g.Edges)
	}
}

func TestNoCoordinates(t *testing.T) {
	root := t.TempDir()
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "events/x.txt", "test.1 = { type = character_event }\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	g := Graph(s, EventGraphParams{})
	if len(g.Nodes) == 0 {
		t.Fatal("expected a node")
	}
	// Compile-time: EventGraphNode has no X/Y fields. Runtime: payload is nodes+edges.
	if g.Nodes[0].ID == "" {
		t.Fatal("empty node id")
	}
}

func TestEventDetailOptionsAndTargets(t *testing.T) {
	root := t.TempDir()
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "events/x.txt", `test.1 = {
	type = character_event
	title = test.1.t
	trigger = { always = yes }
	immediate = { trigger_event = test.2 }
	option = {
		name = test.1.a
		add_gold = 2
	}
	after = { add_gold = 3 }
}

test.2 = {
	type = character_event
}
`)
	write(t, root, "localization/english/a_l_english.yml", "l_english:\n test.1.t:0 \"Hello\"\n test.1.a:0 \"OK\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	d := Detail(s, "test.1")
	if d == nil {
		t.Fatal("detail is nil")
	}
	if d.Type != "character_event" || d.Title == nil || d.Title.Text != "Hello" {
		t.Fatalf("header = type %q title %+v", d.Type, d.Title)
	}
	if len(d.Options) != 1 || d.Options[0].Name == nil || d.Options[0].Name.Text != "OK" {
		t.Fatalf("options = %+v", d.Options)
	}
	var fired bool
	for _, sec := range d.Sections {
		for _, tgt := range sec.Targets {
			if tgt.Name == "test.2" {
				fired = true
			}
		}
	}
	if !fired {
		t.Fatalf("immediate should target test.2; sections=%+v", d.Sections)
	}
}

func TestOverridesWinner(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	write(t, a, "descriptor.mod", "name = \"a\"\n")
	write(t, b, "descriptor.mod", "name = \"b\"\n")
	write(t, a, "common/traits/a.txt", "brave = { category = personality }\n")
	write(t, b, "common/traits/b.txt", "brave = { category = personality }\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{
		{Origin: "modA", Root: a, Order: 0},
		{Origin: "modB", Root: b, Order: 1},
	})
	rows := OverrideRows(s)
	var hit *model.OverrideRow
	for i := range rows {
		if rows[i].Name == "brave" {
			hit = &rows[i]
		}
	}
	if hit == nil {
		t.Fatalf("no brave row: %v", rows)
	}
	if hit.Rule != "LIOS" || hit.Winner != "modB" {
		t.Fatalf("rule=%s winner=%s, want LIOS/modB", hit.Rule, hit.Winner)
	}
}

func TestLocCoverageAndLookup(t *testing.T) {
	root := t.TempDir()
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "events/x.txt", "test.1 = {\n	title = missing_key\n	desc = used_key\n}\n")
	write(t, root, "localization/english/a_l_english.yml",
		"l_english:\n used_key:0 \"Hi\"\n orphan_key:0 \"Bye\"\n")
	write(t, root, "localization/french/a_l_french.yml",
		"l_french:\n used_key:0 \"Hi\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	cov := Coverage(s)
	var en, fr *LocCoverage
	for i := range cov {
		switch cov[i].Language {
		case "english":
			en = &cov[i]
		case "french":
			fr = &cov[i]
		}
	}
	if en == nil || fr == nil {
		t.Fatalf("languages = %v", cov)
	}
	var missing, orphan bool
	for _, m := range en.Missing {
		if m.Key == "missing_key" {
			missing = true
		}
	}
	for _, o := range en.Orphaned {
		if o.Key == "orphan_key" {
			orphan = true
		}
	}
	if !missing {
		t.Errorf("english missing_key not reported: %+v", en.Missing)
	}
	if !orphan {
		t.Errorf("english orphan_key not reported: %+v", en.Orphaned)
	}
	var untr bool
	for _, u := range fr.Untranslated {
		if u.Key == "used_key" {
			untr = true
		}
	}
	if !untr {
		t.Errorf("french used_key should be untranslated: %+v", fr.Untranslated)
	}
	lu := Lookup(s, "used_key")
	if lu == nil || lu.Text != "Hi" {
		t.Fatalf("Lookup used_key = %+v", lu)
	}
}

func TestGraphSmokeVic3EU5(t *testing.T) {
	for _, gameID := range []string{"vic3", "eu5"} {
		root := t.TempDir()
		write(t, root, ".metadata/metadata.json", `{"name":"t"}`)
		write(t, root, "events/x.txt", "test.1 = { type = character_event }\n")
		s := session.New("ws", gameID, nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
		g := Graph(s, EventGraphParams{})
		if len(g.Nodes) != 1 || g.Nodes[0].ID != "test.1" {
			t.Errorf("%s graph nodes = %v", gameID, g.Nodes)
		}
		if Detail(s, "test.1") == nil {
			t.Errorf("%s detail nil", gameID)
		}
	}
}

func TestDelayAndOptionLabel(t *testing.T) {
	root := t.TempDir()
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "events/x.txt", `seq.1 = {
	type = character_event
	immediate = {
		trigger_event = { id = seq.2 days = 30 }
	}
	option = {
		name = seq.1.a
		trigger_event = seq.3
	}
}
seq.2 = { type = character_event }
seq.3 = { type = character_event }
`)
	write(t, root, "localization/english/a_l_english.yml", "l_english:\n seq.1.a:0 \"Go\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	g := Graph(s, EventGraphParams{Root: "seq.1"})
	var delay, opt bool
	for _, e := range g.Edges {
		if e.To == "seq.2" && e.Delay == "30d" {
			delay = true
		}
		if e.To == "seq.3" && strings.Contains(e.Label, "option") {
			opt = true
		}
	}
	if !delay {
		t.Errorf("missing 30d delay; edges=%v", g.Edges)
	}
	if !opt {
		t.Errorf("missing option label; edges=%v", g.Edges)
	}
}

func TestGraphIncludesVanilla(t *testing.T) {
	root := t.TempDir()
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "events/x.txt", "mod.1 = { type = character_event }\n")
	vanilla := t.TempDir()
	vfile := write(t, vanilla, "events/v.txt", "birth.1 = { type = character_event }\n")
	cache := &model.Cache{Defs: []model.Def{
		{Type: "event", Key: "birth.1", Path: vfile, Line: 0},
	}}
	s := session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	g := Graph(s, EventGraphParams{})
	ids := map[string]string{}
	for _, n := range g.Nodes {
		ids[n.ID] = n.Source
	}
	if ids["birth.1"] != "vanilla" || ids["mod.1"] != "mod" {
		t.Fatalf("nodes = %v, want vanilla birth.1 and mod mod.1", ids)
	}
	var birth, modOrig bool
	for _, it := range g.Suggestions.IDs {
		if it.ID == "birth.1" && it.Origin == "" {
			birth = true
		}
		if it.ID == "mod.1" && it.Origin == "mod" {
			modOrig = true
		}
	}
	if !birth || !modOrig {
		t.Fatalf("suggestions = %+v", g.Suggestions.IDs)
	}
	gMod := Graph(s, EventGraphParams{ModRoot: root})
	for _, n := range gMod.Nodes {
		if n.ID == "birth.1" {
			t.Fatal("vanilla should not be seeded when a mod is focused")
		}
	}
	ns := Graph(s, EventGraphParams{Namespace: "birth"})
	for _, it := range ns.Suggestions.IDs {
		if !strings.HasPrefix(it.ID, "birth.") {
			t.Fatalf("namespace filter leaked %q", it.ID)
		}
	}
}

func TestCoverageSkipsVanillaKeys(t *testing.T) {
	root := t.TempDir()
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "events/x.txt", "test.1 = {\n	title = vanilla_key\n}\n")
	write(t, root, "localization/english/a_l_english.yml", "l_english:\n mod_orphan:0 \"X\"\n")
	vfile := write(t, t.TempDir(), "v.yml", "l_english:\n vanilla_key:0 \"Hi\"\n")
	cache := &model.Cache{
		LocEnglish: map[string]string{"vanilla_key": "Hi"},
		LocEnglishSites: map[string]model.LocSite{
			"vanilla_key": {File: vfile, Line: 1},
		},
	}
	s := session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	cov := Coverage(s)
	for _, row := range cov {
		for _, m := range row.Missing {
			if m.Key == "vanilla_key" {
				t.Fatalf("vanilla_key listed as missing: %+v", m)
			}
		}
		for _, o := range row.Orphaned {
			if o.Origin != "mod" && o.Key == "mod_orphan" {
				t.Errorf("orphan origin = %q, want mod", o.Origin)
			}
			if o.Key == "vanilla_key" {
				t.Fatal("vanilla loc must not appear as coverage rows")
			}
		}
	}
	lu := Lookup(s, "vanilla_key")
	if lu == nil || lu.Origin != "vanilla" || lu.File != vfile {
		t.Fatalf("Lookup vanilla_key = %+v", lu)
	}
}

func TestOverridesInjectKeysCollide(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	write(t, a, ".metadata/metadata.json", `{"name":"a"}`)
	write(t, b, ".metadata/metadata.json", `{"name":"b"}`)
	write(t, a, "common/traits/a.txt", "INJECT:brave = { category = personality }\n")
	write(t, b, "common/traits/b.txt", "REPLACE:brave = { category = personality }\n")
	s := session.New("ws", "eu5", nil, []model.ModInput{
		{Origin: "modA", Root: a, Order: 0},
		{Origin: "modB", Root: b, Order: 1},
	})
	rows := OverrideRows(s)
	var hit *model.OverrideRow
	for i := range rows {
		if rows[i].Name == "brave" {
			hit = &rows[i]
		}
	}
	if hit == nil || hit.Overlay || hit.ModCount != 2 {
		t.Fatalf("INJECT/REPLACE collide = %+v", rows)
	}
}
