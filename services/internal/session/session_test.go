// session_test.go covers reindex, resolution, and the exported query surface.

package session

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/parser/jomini"
)

func must(t *testing.T, ok bool, msg string) {
	t.Helper()
	if !ok {
		t.Fatal(msg)
	}
}

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestReindexCoalesces(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"common/traits/00.txt": "brave = { category = personality }\n",
	})
	file := filepath.Join(root, "common", "traits", "00.txt")
	s := NewWithLoc("ws", "ck3", "english", nil, nil, []catalog.ModInput{
		{Origin: "mod", Root: root, Order: 0},
	})
	s.DidSave(file)
	if n := len(s.DefsInFile(file)); n != 1 {
		t.Fatalf("defs after save = %d, want 1", n)
	}
	s.DidSave(file)
	if n := len(s.DefsInFile(file)); n != 1 {
		t.Fatalf("defs after no-op save = %d, want 1", n)
	}
	s.DidChange(file, "other = { category = education }\n")
	defs := s.DefsInFile(file)
	if len(defs) != 1 || defs[0].Key != "other" {
		t.Fatalf("defs after change = %+v, want other", defs)
	}
}

func TestDidOpenDropsDiskRefs(t *testing.T) {
	crlf := "namespace = test\r\n\r\ntest.1 = {\r\n\ttitle = k.t\r\n}\r\n"
	root := writeFiles(t, map[string]string{"events/x.txt": crlf})
	ev := filepath.Join(root, "events", "x.txt")
	s := NewWithLoc("ws", "ck3", "english", nil, nil, []catalog.ModInput{
		{Origin: "mod", Root: root, Order: 0},
	})
	s.DidOpen(ev, jomini.Normalize(crlf))
	if n := len(s.RefsTo("k.t")); n != 1 {
		t.Fatalf("k.t refs after DidOpen = %d, want 1", n)
	}
}

func TestQueries(t *testing.T) {
	body := "namespace = t\n\nt.1 = {\n\ttitle = k.t\n\timmediate = { trigger_event = t.2 }\n}\n" +
		"t.2 = { type = character_event }\n"
	root := writeFiles(t, map[string]string{
		"events/x.txt":                         body,
		"localization/english/a_l_english.yml": "l_english:\n k.t:0 \"Hi\"\n",
	})
	ev := filepath.Join(root, "events", "x.txt")
	loc := filepath.Join(root, "localization", "english", "a_l_english.yml")
	cache := &catalog.VanillaCache{
		InstallPath: "/game", GameVersion: "1.16",
		Structures: map[string][]string{"event": {"immediate"}},
		Vocabulary: []string{"add_gold"},
		Effects:    []string{"add_gold"},
		FieldDocs:  map[string]string{"immediate": "runs first"},
	}
	s := NewWithLoc("ws", "ck3", "english", cache, nil, []catalog.ModInput{
		{Origin: "mod", Root: root, Name: "M", Order: 0},
	})
	s.locByLang = nil
	s.DidOpen(loc, "l_english:\n k.t:0 \"Hi\"\n")

	must(t, s.DefaultLang() == "english" && s.DefCount() >= 2 && s.Resolve("t.1") != nil, "defs")
	must(t, s.ResolveMatching("t.1", func(d catalog.Def) bool { return d.Type == "event" }) != nil,
		"ResolveMatching")
	must(t, len(s.ModDefsOf("t.1")) > 0 && len(s.FindDefs("t.1", 10, true, false)) > 0 &&
		len(s.DefsInFile(ev)) > 0, "ModDefsOf/FindDefs/DefsInFile")
	_, _, _, locOK := s.LocSite("k.t")
	v, locValOK := s.DefaultLoc("k.t")
	_, locFileOK := s.LocFile("mod", "english")
	must(t, locOK && locValOK && v == "Hi" && locFileOK, "loc site/default/file")
	must(t, len(s.LocKeys(false)) > 0 && s.LocByLang()["english"]["k.t"].Value == "Hi",
		"LocKeys/LocByLang")
	must(t, len(s.RefsTo("k.t")) > 0 && len(s.RefsInFile(ev)) > 0 && len(s.LocRefs()) > 0, "refs")
	must(t, len(s.EdgesFrom("t.1")) > 0 && len(s.EdgesTo("t.2")) > 0, "edges")
	must(t, s.FileText(ev) != "" && s.Parsed(ev).Root != nil, "FileText/Parsed")
	origin, _, located := s.Locate(ev)
	must(t, located && origin == "mod" && s.DisplayRel(ev) != "" && s.OriginName("mod") == "M",
		"locate")
	must(t, s.OriginName("") == "Crusader Kings III" &&
		s.OriginName("vanilla") == "Crusader Kings III", "OriginName vanilla")
	must(t, len(s.Mods()) == 1 && s.KindFor(ev) != "", "Mods/KindFor")
	picker, kinds, _ := s.GraphCatalog(nil)
	must(t, kinds["t.1"] != "" || picker["t.1"] != "", "GraphCatalog")
	must(t, len(s.Structures("event")) > 0 && len(s.Vocab("vocabulary")) > 0 &&
		s.FieldDoc("immediate", "event") != "", "Structures/Vocab/FieldDoc")
	_, effects, _ := s.MemberSets("event")
	must(t, effects != nil, "MemberSets")
	inst, _, ver, _ := s.CacheInfo()
	must(t, inst == "/game" && ver == "1.16", "CacheInfo")
	_ = s.Contests([]string{"mod"})
	s.ReplaceCache(&catalog.VanillaCache{InstallPath: "/y", GameVersion: "2"})
	inst, _, _, _ = s.CacheInfo()
	must(t, inst == "/y", "ReplaceCache")
}

func TestEdgesFromConcurrent(t *testing.T) {
	body := "namespace = t\n\nt.1 = {\n\timmediate = { trigger_event = t.2 }\n}\n" +
		"t.2 = { type = character_event }\n"
	root := writeFiles(t, map[string]string{"events/x.txt": body})
	cache := &catalog.VanillaCache{
		InstallPath: "/game", GameVersion: "1",
		Edges: []catalog.Edge{{From: "t.1", To: "t.2"}},
	}
	s := NewWithLoc("ws", "ck3", "english", cache, nil, []catalog.ModInput{
		{Origin: "mod", Root: root, Order: 0},
	})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = s.EdgesFrom("")
			_ = s.EdgesTo("t.2")
		}()
		go func() {
			defer wg.Done()
			s.ReplaceCache(cache)
		}()
	}
	wg.Wait()
}
