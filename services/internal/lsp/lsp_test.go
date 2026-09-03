// lsp_test.go covers each LSP request type against a live session.

package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/parser/jomini"
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
	t *testing.T, game string, files map[string]string,
	cache *catalog.VanillaCache, vloc *catalog.VanillaLoc,
) (*session.Session, string) {
	t.Helper()
	if game == "" {
		game = "ck3"
	}
	root := t.TempDir()
	if game == "ck3" {
		if _, ok := files["descriptor.mod"]; !ok {
			write(t, root, "descriptor.mod", "name = \"t\"\n")
		}
	}
	for rel, body := range files {
		write(t, root, rel, body)
	}
	s := session.NewWithLoc("ws", game, "english", cache, vloc, []catalog.ModInput{
		{Origin: "mod", Root: root, Order: 0},
	})
	for rel, body := range files {
		s.DidOpen(filepath.Join(root, filepath.FromSlash(rel)), body)
	}
	return s, root
}

func ck3Sess(t *testing.T, body string) (*session.Session, string) {
	t.Helper()
	s, root := buildSession(t, "ck3", map[string]string{"events/x.txt": body}, nil, nil)
	return s, filepath.Join(root, "events", "x.txt")
}

func lineCol(src, needle string) (int, int) {
	i := strings.Index(src, needle)
	if i < 0 {
		return -1, -1
	}
	return strings.Count(src[:i], "\n"), i - (strings.LastIndex(src[:i], "\n") + 1)
}

func hasDiag(diags []Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

func wantHover(t *testing.T, s *session.Session, f string, line, col int, subs ...string) {
	t.Helper()
	h := Hover(s, f, line, col)
	if h == nil {
		t.Fatalf("hover nil at %d:%d want %v", line, col, subs)
	}
	for _, sub := range subs {
		if !strings.Contains(h.Contents, sub) {
			t.Fatalf("hover %#v missing %q", h.Contents, sub)
		}
	}
}

func TestPmtIgnore(t *testing.T) {
	sup := scanSuppressions(jomini.Parse(
		"# pmt:ignore-next-line unclosed-brace\nbad = {\nok = 1 # pmt:ignore\n").Src)
	if !sup.Covers(1, "unclosed-brace") || sup.Covers(1, "other") || !sup.Covers(2, "anything") {
		t.Fatal(sup)
	}
}

func TestDiagnose(t *testing.T) {
	tests := []struct {
		name, game, rel, body string
		want, drop            string
	}{
		{"unclosed and missing loc", "ck3", "events/x.txt",
			"test.1 = {\n	title = missing_key\n",
			"unclosed-brace", ""},
		{"TITLE arg is not loc", "ck3", "events/x.txt",
			"test.1 = {\n\timmediate = { TITLE = primary_title }\n}\n",
			"", "missing-required-loc"},
		{"metadata skips parser", "vic3", ".metadata/metadata.json", "{\n",
			"", "unclosed-brace"},
		{"missing descriptor", "vic3", "notes.txt", "foo = { }\n",
			"missing-descriptor", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, root := buildSession(t, tt.game, map[string]string{tt.rel: tt.body}, nil, nil)
			f := filepath.Join(root, filepath.FromSlash(tt.rel))
			diags := Diagnose(s, f)
			if tt.want != "" && !hasDiag(diags, tt.want) {
				t.Fatalf("missing %s: %v", tt.want, diags)
			}
			if tt.want == "unclosed-brace" && !hasDiag(diags, "missing-required-loc") {
				t.Fatalf("missing loc diag: %v", diags)
			}
			if tt.drop != "" && hasDiag(diags, tt.drop) {
				t.Fatalf("unexpected %s: %v", tt.drop, diags)
			}
		})
	}
}

func TestLocBOMDiag(t *testing.T) {
	const body = "l_english:\n k:0 \"v\"\n"
	rel := "localization/english/a_l_english.yml"
	s, root := buildSession(t, "ck3", map[string]string{rel: body}, nil, nil)
	f := filepath.Join(root, filepath.FromSlash(rel))
	if !hasDiag(Diagnose(s, f), "missing-bom") {
		t.Fatal("want missing-bom when disk has no BOM")
	}
	raw := append([]byte{0xEF, 0xBB, 0xBF}, []byte(body)...)
	if err := os.WriteFile(f, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	s.DidOpen(f, body)
	if hasDiag(Diagnose(s, f), "missing-bom") {
		t.Fatal("disk BOM must not warn")
	}
}

func TestHover(t *testing.T) {
	vfile := write(t, t.TempDir(), "localization/english/inventory_l_english.yml",
		"l_english:\n type:0 \"Type\"\n")
	body := `# event should not hover
test.1 = {
	type = character_event
	title = test.1.t
	title = type
	immediate = {
		add_gold = 1
		my_trig = yes
		save_scope_as = duel_target
		exists = scope:duel_target
	}
}
t.1 = { title = k.t }
t.2 = { title = k.t }
`
	s, root := buildSession(t, "ck3", map[string]string{
		"events/x.txt":                    body,
		"localization/english/events.yml": "l_english:\n test.1.t:0 \"Hello\"\n",
		"common/scripted_triggers/t.txt":  "my_trig = { always = yes }\n",
		"common/traits/00.txt":            "brave = { category = personality }\n",
	}, &catalog.VanillaCache{
		FieldInfo: map[string]string{
			"type": "Invite", "add_gold": "Gives gold.",
		},
		FieldInfoByKind: map[string]map[string]string{
			"event": {"type": "presentation", "title": "Dynamic"},
		},
		Structures: map[string][]string{"event": {"immediate"}},
		Effects:    []string{"add_gold"},
		Defs:       []catalog.Def{{Kind: "loc_key", Key: "type", Path: vfile, Line: 1}},
	}, &catalog.VanillaLoc{
		Sites: map[string]catalog.LocEntry{"type": {Path: vfile, Line: 1, Value: "Type"}},
	})
	f := filepath.Join(root, "events", "x.txt")
	col := strings.Index(body, "event")
	if Hover(s, f, 0, col) != nil || len(Definition(s, f, 0, col)) != 0 {
		t.Fatal("comment token must not hover or define")
	}
	line, col := lineCol(body, "type = character_event")
	wantHover(t, s, f, line, col, "presentation")
	line, col = lineCol(body, "title = test.1.t")
	wantHover(t, s, f, line, col, "Dynamic")
	line, col = lineCol(body, "test.1.t")
	wantHover(t, s, f, line, col, "localization", `"Hello"`, "**", ">")
	line, col = lineCol(body, "title = type")
	valCol := col + len("title = ")
	wantHover(t, s, f, line, valCol, "localization")
	if locs := Definition(s, f, line, valCol); len(locs) != 1 || locs[0].URI != vfile {
		t.Fatalf("F12=%v want %s", locs, vfile)
	}
	line, col = lineCol(body, "immediate")
	wantHover(t, s, f, line, col, "event key", "`immediate`")
	if h := Hover(s, f, line, col); h == nil || h.OriginName != "Crusader Kings III" {
		t.Fatalf("immediate originName=%#v", h)
	}
	line, col = lineCol(body, "add_gold")
	wantHover(t, s, f, line, col, "**effect**", "`add_gold`", "Gives gold")
	line, col = lineCol(body, "my_trig")
	wantHover(t, s, f, line, col, "scripted trigger", "`my_trig`")
	line, col = lineCol(body, "scope:duel_target")
	wantHover(t, s, f, line, col+len("scope:"), "saved scope")
	line, col = lineCol(body, "save_scope_as = duel_target")
	wantHover(t, s, f, line, col+len("save_scope_as = "), "saved scope")
	line, col = lineCol(body, "scope:duel_target")
	if len(Definition(s, f, line, col+len("scope:"))) != 0 ||
		len(References(s, f, line, col+len("scope:"))) != 0 {
		t.Fatal("saved scope must not navigate")
	}
	line, col = lineCol(body, "k.t")
	if locs := References(s, f, line, col); len(locs) != 2 {
		t.Fatalf("refs=%v", locs)
	}
	line, col = lineCol(body, "\nt.1 =")
	if Rename(s, f, line, col+1, "t.9") == nil {
		t.Fatal("rename nil")
	}
	tf := filepath.Join(root, "common", "traits", "00.txt")
	if locs := Definition(s, tf, 0, 1); len(locs) != 1 || locs[0].URI != tf {
		t.Fatalf("mod F12=%v", locs)
	}
}

func TestHoverCallKindAndPortrait(t *testing.T) {
	trig := "scripted_trigger my_name = { always = yes }\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/scripted_triggers/x.txt": trig,
		"events/p.txt":                   "ns.1 = {\n\tright_portrait = none\n}\n",
	}, &catalog.VanillaCache{
		FieldInfo: map[string]string{"left_portrait": "Left side portrait."},
	}, nil)
	tf := filepath.Join(root, "common", "scripted_triggers", "x.txt")
	line, col := lineCol(trig, "scripted_trigger")
	wantHover(t, s, tf, line, col, "scripted trigger")
	ev := filepath.Join(root, "events", "p.txt")
	src := "ns.1 = {\n\tright_portrait = none\n}\n"
	line, col = lineCol(src, "right_portrait")
	wantHover(t, s, ev, line, col, "Left side portrait.")
}

func TestDefinitionDottedEvent(t *testing.T) {
	src := "ns.3002 = { type = character_event }\n" +
		"ns.1 = {\n\ttrigger_event = ns.3002\n}\n"
	s, f := ck3Sess(t, src)
	line, col := lineCol(src, "trigger_event = ns.3002")
	col += len("trigger_event = ")
	locs := Definition(s, f, line, col)
	if len(locs) == 0 {
		t.Fatal("Definition on ns of ns.3002")
	}
	col += len("ns.")
	if locs := Definition(s, f, line, col); len(locs) == 0 {
		t.Fatal("Definition on 3002 of ns.3002")
	}
}

func TestLocFileReferences(t *testing.T) {
	locBody := "l_english:\n used_key:0 \"Hello\"\n other_key:0 \"see $used_key$\"\n" +
		" only_loc:0 \"x\"\n ref_loc:0 \"see $only_loc|U$\"\n"
	script := "ns.1 = {\n\ttitle = used_key\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"events/x.txt":                         script,
		"localization/english/a_l_english.yml": locBody,
	}, nil, nil)
	locFile := filepath.Join(root, "localization", "english", "a_l_english.yml")
	line, col := lineCol(locBody, "used_key")
	locs := References(s, locFile, line, col)
	found := false
	for _, loc := range locs {
		if strings.Contains(loc.URI, "events") {
			found = true
		}
	}
	if !found {
		t.Fatalf("loc key refs=%v", locs)
	}
	if locs := Definition(s, locFile, line, col); len(locs) == 0 {
		t.Fatal("loc-file Definition at key")
	}
	endCol := col + len("used_key")
	if locs := References(s, locFile, line, endCol); len(locs) == 0 {
		t.Fatal("cursor at end of loc key should still find usages")
	}
	if locs := Definition(s, locFile, line, endCol); len(locs) == 0 {
		t.Fatal("loc-file Definition at end of key")
	}
	line, col = lineCol(locBody, "Hello")
	if locs := References(s, locFile, line, col); len(locs) != 0 {
		t.Fatalf("value should not steal key refs: %v", locs)
	}
	line, col = lineCol(locBody, "$used_key$")
	col++ // inner key
	if locs := References(s, locFile, line, col); len(locs) == 0 {
		t.Fatal("$used_key$ should resolve as a reference")
	}
	if locs := Definition(s, locFile, line, col); len(locs) == 0 {
		t.Fatal("$used_key$ Definition")
	}
	line, col = lineCol(locBody, "only_loc")
	interpLine, interpCol := lineCol(locBody, "$only_loc|U$")
	if !hasRefSite(References(s, locFile, line, col), interpLine, interpCol+1) {
		t.Fatal("loc-only key should find $only_loc$ reuse")
	}

	bomBody := "\ufeff" + locBody
	s.DidOpen(locFile, bomBody)
	line, col = lineCol(bomBody, "used_key")
	if locs := References(s, locFile, line, col); len(locs) == 0 {
		t.Fatal("BOM loc buffer should still find usages")
	}
	if locs := Definition(s, locFile, line, col); len(locs) == 0 {
		t.Fatal("loc-file Definition at key")
	}
	endCol = col + len("used_key")
	if locs := Definition(s, locFile, line, endCol); len(locs) == 0 {
		t.Fatal("loc-file Definition at end of key")
	}
	line, col = lineCol(bomBody, "only_loc")
	interpLine, interpCol = lineCol(bomBody, "$only_loc|U$")
	if !hasRefSite(References(s, locFile, line, col), interpLine, interpCol+1) {
		t.Fatal("BOM loc-only key should keep $only_loc$ reuse")
	}
	if locs := References(s, locFile, interpLine, interpCol); len(locs) == 0 {
		t.Fatal("cursor on $ of $only_loc|U$ should resolve")
	}
}

func TestTriggerLocalizationReferences(t *testing.T) {
	locBody := "l_english:\n IS_ADULT_TRIGGER:0 \"Is an adult\"\n"
	trig := "is_adult = {\n\tglobal = IS_ADULT_TRIGGER\n\tfirst = I_AM_ADULT_TRIGGER\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/trigger_localization/x.txt":    trig,
		"localization/english/a_l_english.yml": locBody,
	}, nil, nil)
	locFile := filepath.Join(root, "localization", "english", "a_l_english.yml")
	line, col := lineCol(locBody, "IS_ADULT_TRIGGER")
	locs := References(s, locFile, line, col)
	if !hasURI(locs, "trigger_localization") {
		t.Fatalf("trigger loc key refs=%v", locs)
	}
	trigFile := filepath.Join(root, "common", "trigger_localization", "x.txt")
	tLine, tCol := lineCol(trig, "IS_ADULT_TRIGGER")
	if locs := Definition(s, trigFile, tLine, tCol); !hasURI(locs, "a_l_english.yml") {
		t.Fatalf("script-side Definition=%v", locs)
	}
}

func hasURI(locs []Location, sub string) bool {
	for _, loc := range locs {
		if strings.Contains(loc.URI, sub) {
			return true
		}
	}
	return false
}

func hasRefSite(locs []Location, line, col int) bool {
	for _, loc := range locs {
		if loc.Range.Start.Line == line && loc.Range.Start.Character == col {
			return true
		}
	}
	return false
}

func TestCompleteActionsSymbols(t *testing.T) {
	body := "test.1 = {\n  type = character_event\n}\n"
	s, f := ck3Sess(t, body)
	if edits := FormatDocument(s, f); len(edits) == 0 {
		t.Fatal("want indent edits")
	}
	if folds := FoldingRanges(s, f); len(folds) == 0 {
		t.Fatal("want fold range")
	}
	syms := DocumentSymbols(s, f)
	if len(syms) == 0 || syms[0].Name != "test.1" {
		t.Fatalf("doc symbols=%v", syms)
	}
	if ws := WorkspaceSymbols(s, "test"); len(ws) == 0 {
		t.Fatal("workspace symbols empty")
	}

	s, f = ck3Sess(t, "test.1 = {\n	title = brand_new_key\n}\n")
	var edit *WorkspaceEdit
	for _, a := range CodeActions(s, f) {
		if strings.Contains(a.Title, "brand_new_key") {
			edit = a.Edit
		}
	}
	if edit == nil || len(edit.Create) == 0 {
		t.Fatal("expected create-loc")
	}
	for _, edits := range edit.Changes {
		if len(edits) > 0 && !strings.HasPrefix(edits[0].NewText, "\uFEFF") {
			t.Fatal("loc create must include BOM")
		}
	}

	s, root := buildSession(t, "ck3", map[string]string{"descriptor.mod": "name = \"t\"\n"}, nil, nil)
	f = filepath.Join(root, "descriptor.mod")
	found := false
	for _, it := range Complete(s, f, 0, 0) {
		found = found || it.Label == "dependencies"
	}
	if !found {
		t.Fatal("missing dependencies completion")
	}

	imm := "test.1 = {\n\timm"
	s, root = buildSession(t, "ck3", map[string]string{"events/x.txt": imm}, &catalog.VanillaCache{
		Structures: map[string][]string{"event": {"immediate", "option"}},
		Effects:    []string{"immortal", "immune"},
		Vocabulary: []string{"immune_to", "immortal"},
	}, nil)
	f = filepath.Join(root, "events", "x.txt")
	line, col := lineCol(imm, "imm")
	if items := Complete(s, f, line, col+len("imm")); len(items) == 0 || items[0].Label != "immediate" {
		t.Fatalf("complete=%v", items)
	}
	vocab := make([]string, 5000)
	for i := range vocab {
		vocab[i] = "tok_" + string(rune('a'+i%26))
	}
	s, root = buildSession(t, "ck3", map[string]string{"events/x.txt": "test.1 = {\n\t"}, &catalog.VanillaCache{
		Structures: map[string][]string{"event": {"immediate", "option"}},
		Effects:    vocab, Vocabulary: vocab,
	}, nil)
	f = filepath.Join(root, "events", "x.txt")
	items := Complete(s, f, 1, 1)
	if len(items) > 80 || len(items) == 0 || items[0].Label != "immediate" {
		t.Fatalf("complete len=%d first=%q", len(items), items[0].Label)
	}
}

func hasComplete(items []CompletionItem, label string) bool {
	for _, it := range items {
		if it.Label == label {
			return true
		}
	}
	return false
}

func TestCompleteSlots(t *testing.T) {
	cache := &catalog.VanillaCache{
		Structures: map[string][]string{"event": {"immediate", "option", "title"}},
		Effects:    []string{"add_gold"},
		Triggers:   []string{"is_adult"},
		FieldInfo:  map[string]string{"immediate": "runs first"},
	}
	vloc := &catalog.VanillaLoc{Sites: map[string]catalog.LocEntry{
		"test.1.t": {Value: "Hi"},
		"other":    {},
	}}

	imm := "test.1 = {\n\timmediate = {\n\t\t\n\t}\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{"events/x.txt": imm}, cache, nil)
	f := filepath.Join(root, "events", "x.txt")
	line, _ := lineCol(imm, "immediate = {")
	items := Complete(s, f, line+1, 2)
	if !hasComplete(items, "add_gold") {
		t.Fatalf("immediate slot missing add_gold: %v", labels(items))
	}
	if hasComplete(items, "option") || hasComplete(items, "title") {
		t.Fatalf("immediate slot leaked structures: %v", labels(items))
	}

	trig := "test.1 = {\n\ttrigger = {\n\t\t\n\t}\n}\n"
	s, root = buildSession(t, "ck3", map[string]string{"events/x.txt": trig}, cache, nil)
	f = filepath.Join(root, "events", "x.txt")
	line, _ = lineCol(trig, "trigger = {")
	items = Complete(s, f, line+1, 2)
	if !hasComplete(items, "is_adult") {
		t.Fatalf("trigger slot missing is_adult: %v", labels(items))
	}
	if hasComplete(items, "add_gold") {
		t.Fatalf("trigger slot leaked effect: %v", labels(items))
	}

	title := "test.1 = {\n\ttitle = te\n}\n"
	s, root = buildSession(t, "ck3", map[string]string{"events/x.txt": title}, cache, vloc)
	f = filepath.Join(root, "events", "x.txt")
	line, col := lineCol(title, "title = te")
	items = Complete(s, f, line, col+len("title = te"))
	if !hasComplete(items, "test.1.t") || hasComplete(items, "other") {
		t.Fatalf("title value loc: %v", labels(items))
	}
	if hasComplete(items, "add_gold") {
		t.Fatalf("title value leaked effect: %v", labels(items))
	}

	fire := "test.1 = {\n\timmediate = { trigger_event = te }\n}\ntest.2 = { type = character_event }\n"
	s, root = buildSession(t, "ck3", map[string]string{"events/x.txt": fire}, cache, nil)
	f = filepath.Join(root, "events", "x.txt")
	line, col = lineCol(fire, "trigger_event = te")
	items = Complete(s, f, line, col+len("trigger_event = te"))
	if !hasComplete(items, "test.1") && !hasComplete(items, "test.2") {
		t.Fatalf("trigger_event value: %v", labels(items))
	}

	body := "test.1 = {\n\timm"
	s, root = buildSession(t, "ck3", map[string]string{"events/x.txt": body}, cache, nil)
	f = filepath.Join(root, "events", "x.txt")
	line, col = lineCol(body, "imm")
	items = Complete(s, f, line, col+len("imm"))
	if len(items) == 0 || items[0].Label != "immediate" {
		t.Fatalf("structure key: %v", labels(items))
	}
	if items[0].Documentation != "runs first" {
		t.Fatalf("FieldDoc: %#v", items[0].Documentation)
	}
	if items[0].InsertText != "immediate = " {
		t.Fatalf("scalar insert=%q", items[0].InsertText)
	}
	if items[0].Range.Start.Line != line {
		t.Fatalf("range=%v want line %d", items[0].Range, line)
	}

	theme := "test.1 = {\n\ttheme = s\n}\n"
	s, root = buildSession(t, "ck3", map[string]string{
		"events/x.txt":               theme,
		"common/event_themes/00.txt": "seduction = { }\n",
	}, &catalog.VanillaCache{
		FieldValueKinds: map[string]string{"theme": "event_themes"},
		Defs: []catalog.Def{
			{Kind: "event_themes", Key: "seduction", Path: "t.txt", Line: 0},
		},
	}, nil)
	f = filepath.Join(root, "events", "x.txt")
	line, col = lineCol(theme, "theme = s")
	items = Complete(s, f, line, col+len("theme = s"))
	if !hasComplete(items, "seduction") {
		t.Fatalf("theme value: %v", labels(items))
	}
	if hasComplete(items, "add_gold") {
		t.Fatalf("theme leaked effect: %v", labels(items))
	}
}

func TestCompleteEmptyPrefixAndFallback(t *testing.T) {
	cache := &catalog.VanillaCache{
		Structures: map[string][]string{"event": {"immediate", "option"}},
		Defs: []catalog.Def{
			{Kind: "trait", Key: "ambitious", Path: "t.txt", Line: 0},
		},
	}
	vloc := &catalog.VanillaLoc{Sites: map[string]catalog.LocEntry{
		"test.1.t": {Value: "Hi"},
	}}

	title := "test.1 = {\n\ttitle = \n}\n"
	s, root := buildSession(t, "ck3", map[string]string{"events/x.txt": title}, cache, vloc)
	f := filepath.Join(root, "events", "x.txt")
	line, col := lineCol(title, "title = ")
	items := Complete(s, f, line, col+len("title = "))
	if !hasComplete(items, "test.1.t") {
		t.Fatalf("empty title loc: %v", labels(items))
	}

	body := "test.1 = { \n}\n"
	s, root = buildSession(t, "ck3", map[string]string{"events/x.txt": body}, cache, nil)
	f = filepath.Join(root, "events", "x.txt")
	line, col = lineCol(body, "{ ")
	items = Complete(s, f, line, col+len("{ "))
	if !hasComplete(items, "immediate") {
		t.Fatalf("structure key after space: %v", labels(items))
	}

	unk := "test.1 = {\n\tunknown_field = \n}\n"
	s, root = buildSession(t, "ck3", map[string]string{"events/x.txt": unk}, cache, nil)
	f = filepath.Join(root, "events", "x.txt")
	line, col = lineCol(unk, "unknown_field = ")
	items = Complete(s, f, line, col+len("unknown_field = "))
	if len(items) == 0 || !hasComplete(items, "yes") {
		t.Fatalf("unknown field value: %v", labels(items))
	}
}

func TestSignatureHelpAndSchemaDiags(t *testing.T) {
	cache := &catalog.VanillaCache{
		TokenUsage: map[string]string{"add_gold": "add_gold = { $VALUE$ }"},
		FieldInfo:  map[string]string{"add_gold": "gold"},
	}
	s, root := buildSession(t, "ck3", map[string]string{
		"events/x.txt":         "namespace = ns\nns.1 = {\n\timmediate = { add_gold = {  trigger_event = ns.99 }\n}\n",
		"common/traits/00.txt": "ambitious = { }\n",
	}, cache, nil)
	f := filepath.Join(root, "events", "x.txt")
	line, col := lineCol(
		"namespace = ns\nns.1 = {\n\timmediate = { add_gold = {  trigger_event = ns.99 }\n}\n",
		"add_gold = {",
	)
	h := SignatureHelp(s, f, line, col+len("add_gold = {"))
	if h == nil || !strings.Contains(h.Label, "add_gold") {
		t.Fatalf("signature: %#v", h)
	}

	tf := filepath.Join(root, "common", "traits", "00.txt")
	if !hasDiag(Diagnose(s, tf), "required-loc") {
		t.Fatal("trait required-loc missing")
	}
	if !hasDiag(Diagnose(s, f), "unknown-event") {
		t.Fatal("unknown-event missing")
	}
}

func labels(items []CompletionItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Label
	}
	return out
}

func TestCompleteRangeAndInsert(t *testing.T) {
	cache := &catalog.VanillaCache{
		Structures: map[string][]string{
			"event": {"title", "immediate", "rare_flag"},
		},
		StructureBlocks: map[string][]string{"event": {"immediate"}},
	}
	body := "test.1 = {\n\t\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{"events/x.txt": body}, cache, nil)
	f := filepath.Join(root, "events", "x.txt")
	items := Complete(s, f, 1, 1)
	if len(items) < 3 || items[0].Label != "title" || items[2].Label != "rare_flag" {
		t.Fatalf("freq order: %v", labels(items))
	}
	var imm, title CompletionItem
	for _, it := range items {
		switch it.Label {
		case "immediate":
			imm = it
		case "title":
			title = it
		}
	}
	if imm.InsertText != "immediate = { $0 }" {
		t.Fatalf("block insert=%q", imm.InsertText)
	}
	if title.InsertText != "title = " {
		t.Fatalf("scalar insert=%q", title.InsertText)
	}
}

func TestCompleteDottedReplace(t *testing.T) {
	src := "namespace = hostile_scheme_discovery\n" +
		"hostile_scheme_discovery.3002 = { type = character_event }\n" +
		"hostile_scheme_discovery.1 = {\n" +
		"	trigger_event = hostile_scheme_discovery.3\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{"events/x.txt": src}, nil, nil)
	f := filepath.Join(root, "events", "x.txt")
	line, col := lineCol(src, "trigger_event = hostile_scheme_discovery.3")
	col += len("trigger_event = ")
	items := Complete(s, f, line, col+len("hostile_scheme_discovery.3"))
	var hit CompletionItem
	for _, it := range items {
		if it.Label == "hostile_scheme_discovery.3002" {
			hit = it
			break
		}
	}
	if hit.Label == "" {
		t.Fatalf("missing 3002: %v", labels(items))
	}
	if hit.Range.Start.Line != line || hit.Range.Start.Character != col {
		t.Fatalf("range start=%v want %d:%d", hit.Range.Start, line, col)
	}
	end := col + len("hostile_scheme_discovery.3")
	if hit.Range.End.Line != line || hit.Range.End.Character != end {
		t.Fatalf("range end=%v want %d:%d", hit.Range.End, line, end)
	}
}

func TestDiagnoseMessageConventionNotMissing(t *testing.T) {
	s, root := buildSession(t, "ck3", map[string]string{
		"common/messages/x.txt": "quieter_events_neutral = {\n" +
			"	title = event_message_title\n}\n",
		"localization/english/a_l_english.yml": "l_english:\n event_message_title:0 \"T\"\n",
	}, nil, nil)
	f := filepath.Join(root, "common", "messages", "x.txt")
	for _, d := range Diagnose(s, f) {
		if strings.Contains(d.Message, "quieter_events_neutral") ||
			strings.Contains(d.Message, "rule_quieter") ||
			strings.Contains(d.Message, "setting_quieter") {
			t.Fatalf("invented loc diag: %+v", d)
		}
	}
}

func TestCompleteRootAndEnums(t *testing.T) {
	src := "script"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/scripted_triggers/x.txt": src,
	}, nil, nil)
	f := filepath.Join(root, "common", "scripted_triggers", "x.txt")
	items := Complete(s, f, 0, len("script"))
	if !hasComplete(items, "scripted_trigger") {
		t.Fatalf("root scripted_trigger: %v", labels(items))
	}

	cache := &catalog.VanillaCache{
		Structures: map[string][]string{"event": {"type", "immediate"}},
		FieldEnumsByKind: map[string]map[string][]string{
			"event": {"type": {"character_event", "letter_event"}},
		},
	}
	body := "test.1 = {\n\ttype = c\n}\n"
	s, root = buildSession(t, "ck3", map[string]string{"events/x.txt": body}, cache, nil)
	f = filepath.Join(root, "events", "x.txt")
	line, col := lineCol(body, "type = c")
	items = Complete(s, f, line, col+len("type = c"))
	if !hasComplete(items, "character_event") {
		t.Fatalf("type enum: %v", labels(items))
	}

	unk := "test.1 = {\n\tunknown_field = \n}\n"
	s, root = buildSession(t, "ck3", map[string]string{
		"events/x.txt": unk,
		"events/y.txt": "other.1 = { type = character_event }\n",
	}, cache, nil)
	f = filepath.Join(root, "events", "x.txt")
	line, col = lineCol(unk, "unknown_field = ")
	items = Complete(s, f, line, col+len("unknown_field = "))
	if !hasComplete(items, "yes") {
		t.Fatalf("unknown_field yes: %v", labels(items))
	}
	for _, it := range items {
		if it.Label != "yes" && it.Label != "no" {
			t.Fatalf("unknown_field dumped %q: %v", it.Label, labels(items))
		}
	}
}
