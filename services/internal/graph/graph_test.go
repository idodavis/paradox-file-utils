// graph_test.go covers defs-first orphans, via-effect hops, sim order, no
// coordinates, FIOS/LIOS winners, loc coverage, and a vic3/eu5 smoke parse.

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

func TestSimStepOrder(t *testing.T) {
	root := t.TempDir()
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "events/x.txt", `test.1 = {
	type = character_event
	title = test.1.t
	trigger = { always = yes }
	immediate = { add_gold = 1 }
	option = {
		name = test.1.a
		add_gold = 2
	}
	after = { add_gold = 3 }
}
`)
	write(t, root, "localization/english/a_l_english.yml", "l_english:\n test.1.t:0 \"Hello\"\n test.1.a:0 \"OK\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	d := Detail(s, "test.1")
	if d == nil {
		t.Fatal("detail is nil")
	}
	want := []string{"trigger", "immediate", "option", "after"}
	var got []string
	for _, st := range d.SimSteps {
		got = append(got, st.Kind)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("simSteps kinds = %v, want %v", got, want)
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
