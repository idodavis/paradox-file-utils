// graph_test.go covers neighborhood BFS, graph/detail/override/loc payloads.

package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
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

func buildSession(
	t *testing.T, game string, cache *catalog.VanillaCache, vloc *catalog.VanillaLoc,
	mods []catalog.ModInput,
) *session.Session {
	t.Helper()
	if game == "" {
		game = "ck3"
	}
	return session.NewWithLoc("ws", game, "english", cache, vloc, mods)
}

func oneMod(t *testing.T, game string, files map[string]string) catalog.ModInput {
	t.Helper()
	root := t.TempDir()
	if game == "ck3" || game == "" {
		write(t, root, "descriptor.mod", "name = \"t\"\n")
	} else {
		write(t, root, ".metadata/metadata.json", `{"name":"t"}`)
	}
	for rel, body := range files {
		write(t, root, rel, body)
	}
	return catalog.ModInput{Origin: "mod", Root: root, Order: 0}
}

func nodeIDs(g EventGraph) map[string]EventGraphNode {
	out := make(map[string]EventGraphNode, len(g.Nodes))
	for _, n := range g.Nodes {
		out[n.ID] = n
	}
	return out
}

func TestNeighborhood(t *testing.T) {
	assert := func(
		name string, edges []EventGraphEdge, kinds map[string]string,
		root string, want, forbid []string,
	) {
		t.Run(name, func(t *testing.T) {
			got, _, trunc, _ := neighborhood(edges, root, kinds, 80, nil)
			if trunc {
				t.Fatal("truncated")
			}
			for _, id := range want {
				if !got[id] {
					t.Fatalf("missing %s in %v", id, got)
				}
			}
			for _, id := range forbid {
				if got[id] {
					t.Fatalf("unexpected %s in %v", id, got)
				}
			}
		})
	}
	assert("one hop stops at on_action hub", []EventGraphEdge{
		{From: "a", To: "pulse"}, {From: "pulse", To: "c"},
		{From: "pulse", To: "d"}, {From: "e", To: "a"},
	}, map[string]string{
		"a": "event", "c": "event", "d": "event", "e": "event", "pulse": "on_action",
	}, "a", []string{"a", "pulse", "e"}, []string{"c", "d"})
	assert("three event hops", []EventGraphEdge{
		{From: "a", To: "b"}, {From: "b", To: "c"},
		{From: "c", To: "d"}, {From: "d", To: "e"},
	}, map[string]string{
		"a": "event", "b": "event", "c": "event", "d": "event", "e": "event",
	}, "a", []string{"a", "b", "c", "d"}, []string{"e"})
}

func TestGraphViaOptionRoles(t *testing.T) {
	locFile := write(t, t.TempDir(), "v.yml", "l_english:\n vanilla_key:0 \"Hi\"\n")
	vanilla := t.TempDir()
	vfile := write(t, vanilla, "events/v.txt",
		"birth.1 = {\n	type = character_event\n	immediate = { trigger_event = mod.1 }\n}\n")
	s := buildSession(t, "ck3", &catalog.VanillaCache{Defs: []catalog.Def{
		{Type: "event", Key: "birth.1", Path: vfile, Line: 0},
	}}, &catalog.VanillaLoc{
		Sites: map[string]catalog.LocEntry{"vanilla_key": {File: locFile, Line: 1, Value: "Hi"}},
	}, []catalog.ModInput{oneMod(t, "ck3", map[string]string{
		"events/ns.txt": `namespace = ns
ns.0 = { type = character_event
	immediate = { trigger_event = ns.1 }
}
ns.1 = {
	type = character_event
	title = ns.1.t
	trigger = { always = yes }
	immediate = {
		hop_effect = yes
		trigger_event = { id = ns.4 days = 30 }
	}
	desc = used_key
	option = { name = ns.1.a
		trigger_event = ns.3
		add_gold = 2
	}
	after = { add_gold = 3 }
}
ns.5 = { type = character_event
	title = missing_key
	desc = used_key
}
ns.2 = { type = character_event }
ns.3 = { type = character_event }
ns.4 = { type = character_event }
ns.9 = { type = character_event }
mod.1 = { type = character_event
	title = vanilla_key
	immediate = { trigger_event = birth.1 }
}
`,
		"events/fan.txt":                    fanoutBody(),
		"common/scripted_effects/e.txt": "hop_effect = {\n\ttrigger_event = ns.2\n}\n",
		"localization/english/a_l_english.yml": "l_english:\n ns.1.t:0 \"Hello\"\n ns.1.a:0 \"Go\"\n used_key:0 \"Hi\"\n orphan_key:0 \"Bye\"\n",
		"localization/french/a_l_french.yml":   "l_french:\n used_key:0 \"Hi\"\n",
	})})
	g := Graph(s, EventGraphParams{Root: "ns.1"})
	ids := nodeIDs(g)
	for _, id := range []string{"ns.0", "ns.1", "ns.2", "ns.3", "ns.4"} {
		if _, ok := ids[id]; !ok {
			t.Fatalf("missing %s in %v", id, ids)
		}
	}
	if _, ok := ids["ns.9"]; ok {
		t.Fatal("orphan ns.9 should not appear")
	}
	if ids["ns.1"].Role != "root" || ids["ns.0"].Role != "caller" {
		t.Fatalf("roles root=%q caller=%q", ids["ns.1"].Role, ids["ns.0"].Role)
	}
	if ids["ns.2"].Role != "" {
		t.Fatalf("ns.2 role = %q, want empty", ids["ns.2"].Role)
	}
	var via, opt, hops int
	for _, e := range g.Edges {
		if e.From == "ns.1" && e.To == "ns.2" && e.Kind == "via" {
			via++
			if strings.Count(e.Label, "→") > 2 {
				t.Errorf("via hop chain too long: %q", e.Label)
			}
		}
		if e.From == "ns.1" && e.To == "ns.3" && e.Kind == "option" {
			opt++
		}
		if e.To == "ns.4" {
			hops++
		}
	}
	if via == 0 || opt == 0 || hops == 0 {
		t.Fatalf("via=%d option=%d delay=%d edges=%v", via, opt, hops, g.Edges)
	}
	d := Detail(s, "ns.1")
	if d == nil || d.Type != "character_event" || d.Title == nil || d.Title.Text != "Hello" {
		t.Fatalf("header = %+v", d)
	}
	if len(d.Options) != 1 || d.Options[0].Name == nil || d.Options[0].Name.Text != "Go" {
		t.Fatalf("options = %+v", d.Options)
	}
	var fired bool
	for _, sec := range d.Sections {
		for _, tgt := range sec.Targets {
			if tgt.Name == "ns.3" || tgt.Name == "ns.4" {
				fired = true
			}
		}
	}
	if !fired {
		t.Fatalf("detail should target ns.3/ns.4; sections=%+v", d.Sections)
	}
	cov := Coverage(s)
	byLang := map[string]*LocCoverage{}
	for i := range cov {
		byLang[cov[i].Language] = &cov[i]
	}
	en, fr := byLang["english"], byLang["french"]
	if en == nil || fr == nil {
		t.Fatalf("languages = %v", cov)
	}
	var missing, orphan, untr bool
	for _, m := range en.Issues {
		missing = missing || (m.Kind == "missing" && m.Key == "missing_key")
		orphan = orphan || (m.Kind == "orphaned" && m.Key == "orphan_key")
	}
	for _, u := range fr.Issues {
		untr = untr || (u.Kind == "untranslated" && u.Key == "used_key")
	}
	if !missing || !orphan || !untr {
		t.Fatalf("missing=%v orphan=%v untr=%v en=%+v fr=%+v",
			missing, orphan, untr, en.Issues, fr.Issues)
	}
	if lu := Lookup(s, "used_key"); lu == nil || lu.Text != "Hi" {
		t.Fatalf("Lookup used_key = %+v", lu)
	}

	empty := Graph(s, EventGraphParams{})
	if len(empty.Nodes) != 0 || empty.EmptyReason == "" {
		t.Fatalf("empty root nodes=%d reason=%q", len(empty.Nodes), empty.EmptyReason)
	}
	var birth, modOrig bool
	for _, it := range empty.Suggestions.IDs {
		birth = birth || (it.ID == "birth.1" && it.Origin == "")
		modOrig = modOrig || (it.ID == "mod.1" && it.Origin == "mod")
	}
	if !birth || !modOrig {
		t.Fatalf("suggestions = %+v", empty.Suggestions.IDs)
	}
	rooted := nodeIDs(Graph(s, EventGraphParams{Root: "mod.1"}))
	if rooted["mod.1"].Origin != "mod" || rooted["birth.1"].Origin != "" {
		t.Fatalf("rooted = %v", rooted)
	}
	for _, it := range Graph(s, EventGraphParams{Origins: []string{"mod"}}).Suggestions.IDs {
		if it.ID == "birth.1" {
			t.Fatal("vanilla should not be in mod-only picker")
		}
	}
	for _, it := range Graph(s, EventGraphParams{Namespace: "birth"}).Suggestions.IDs {
		if !strings.HasPrefix(it.ID, "birth.") {
			t.Fatalf("namespace filter leaked %q", it.ID)
		}
	}
	for _, row := range Coverage(s) {
		for _, m := range row.Issues {
			if m.Key == "vanilla_key" {
				t.Fatalf("vanilla_key listed as %s: %+v", m.Kind, m)
			}
		}
	}
	if lu := Lookup(s, "vanilla_key"); lu == nil || lu.Origin != "vanilla" || lu.File != locFile {
		t.Fatalf("Lookup vanilla_key = %+v", lu)
	}

	fg := Graph(s, EventGraphParams{Root: "r.1"})
	if !fg.Truncated {
		t.Fatal("expected truncated")
	}
	var more *EventGraphNode
	kids := 0
	for i := range fg.Nodes {
		n := &fg.Nodes[i]
		if n.Kind == "more" {
			more = n
		}
		if strings.HasPrefix(n.ID, "r.") && n.ID != "r.1" && n.Kind == "event" {
			kids++
		}
	}
	if more == nil || more.ID != "more:r.1" || more.Title != "+1 more" {
		t.Fatalf("more stub = %+v", more)
	}
	if kids != 12 {
		t.Fatalf("outbound children = %d, want 12", kids)
	}
	exp := Graph(s, EventGraphParams{Root: "r.1", Expand: []string{"r.1"}})
	if len(exp.Nodes) != 14 {
		t.Fatalf("expand nodes = %d, want 14", len(exp.Nodes))
	}
	for _, n := range exp.Nodes {
		if n.Kind == "more" {
			t.Fatalf("expand should omit more-stub, got %+v", n)
		}
	}

	for _, gameID := range []string{"vic3", "eu5"} {
		gs := buildSession(t, gameID, nil, nil, []catalog.ModInput{oneMod(t, gameID, map[string]string{
			"events/x.txt": "test.1 = { type = character_event }\n",
		})})
		gg := Graph(gs, EventGraphParams{Root: "test.1"})
		if len(gg.Nodes) != 1 || gg.Nodes[0].ID != "test.1" {
			t.Errorf("%s graph nodes = %v", gameID, gg.Nodes)
		}
		if Detail(gs, "test.1") == nil {
			t.Errorf("%s detail nil", gameID)
		}
	}
}

func twoMods(t *testing.T, game, aBody, bBody string) *session.Session {
	t.Helper()
	a, b := t.TempDir(), t.TempDir()
	desc, body := "descriptor.mod", "name = \"t\"\n"
	if game != "ck3" {
		desc, body = ".metadata/metadata.json", `{"name":"t"}`
	}
	write(t, a, desc, body)
	write(t, b, desc, body)
	write(t, a, "common/traits/a.txt", aBody)
	write(t, b, "common/traits/b.txt", bBody)
	return session.NewWithLoc("ws", game, "english", nil, nil, []catalog.ModInput{
		{Origin: "modA", Root: a, Order: 0},
		{Origin: "modB", Root: b, Order: 1},
	})
}

func TestOverrideRows(t *testing.T) {
	tests := []struct {
		name, game, a, b, rule, winner string
		sites                          int
	}{
		{"LIOS last mod", "ck3",
			"brave = { category = personality }\n",
			"brave = { category = personality }\n",
			"LIOS", "modB", 0},
		{"INJECT REPLACE collide", "eu5",
			"INJECT:brave = { category = personality }\n",
			"REPLACE:brave = { category = personality }\n",
			"", "modB", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := OverrideRows(twoMods(t, tt.game, tt.a, tt.b))
			var hit *OverrideRow
			for i := range rows {
				if rows[i].Name == "brave" {
					hit = &rows[i]
				}
			}
			if hit == nil || hit.Overlay {
				t.Fatalf("brave row = %+v", hit)
			}
			if tt.rule != "" && (hit.Rule != tt.rule || hit.Winner != tt.winner) {
				t.Fatalf("rule=%s winner=%s", hit.Rule, hit.Winner)
			}
			if tt.sites > 0 {
				n := 0
				for _, site := range hit.Sites {
					if site.Origin != "" {
						n++
					}
				}
				if n != tt.sites {
					t.Fatalf("mod sites = %d, want %d", n, tt.sites)
				}
			}
		})
	}
}

func fanoutBody() string {
	var b strings.Builder
	b.WriteString("r.1 = {\n\ttype = character_event\n\tevents = {\n")
	for i := 2; i <= 14; i++ {
		fmt.Fprintf(&b, "\t\tr.%d\n", i)
	}
	b.WriteString("\t}\n}\n")
	for i := 2; i <= 14; i++ {
		fmt.Fprintf(&b, "r.%d = { type = character_event }\n", i)
	}
	return b.String()
}
