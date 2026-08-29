// lsp_test.go covers diagnostics, hover fieldDocs, F12 overlay and vanilla,
// create-loc BOM, .mod complete, missing-descriptor, and event references.

package lsp

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

func ck3Sess(t *testing.T, body string) (*session.Session, string) {
	t.Helper()
	root := t.TempDir()
	f := write(t, root, "events/x.txt", body)
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, body)
	return s, f
}

func TestDiagnoseParseAndMissingLoc(t *testing.T) {
	s, f := ck3Sess(t, "test.1 = {\n	title = missing_key\n")
	diags := Diagnose(s, f)
	var unclosed, missing bool
	for _, d := range diags {
		if d.Code == "unclosed-brace" {
			unclosed = true
		}
		if d.Code == "missing-required-loc" {
			missing = true
		}
		if d.Code == "wrong-scope" || d.Code == "unknown-saved-scope" {
			t.Errorf("unexpected scope lint: %+v", d)
		}
	}
	if !unclosed {
		t.Errorf("expected unclosed-brace; diags=%v", diags)
	}
	if !missing {
		t.Errorf("expected missing-required-loc; diags=%v", diags)
	}
}

func TestDiagnoseTitleScopeNotMissingLoc(t *testing.T) {
	body := "test.1 = {\n\timmediate = { TITLE = primary_title }\n}\n"
	s, f := ck3Sess(t, body)
	for _, d := range Diagnose(s, f) {
		if d.Code == "missing-required-loc" {
			t.Fatalf("TITLE = primary_title should not warn loc: %+v", d)
		}
	}
}

func TestHoverFieldDocs(t *testing.T) {
	s, f := ck3Sess(t, "brave = { category = personality }\n")
	s.DidChange(f, "brave = { category = personality }\n")
	root := filepath.Dir(filepath.Dir(f))
	cache := &model.Cache{FieldDocs: map[string]string{"category": "The trait category."}}
	s = session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, "brave = { category = personality }\n")
	h := Hover(s, f, 0, 12) // on "category"
	if h == nil || !strings.Contains(h.Contents, "trait category") {
		t.Fatalf("hover = %+v, want fieldDocs", h)
	}
}

func TestHoverSkipsComments(t *testing.T) {
	body := "# event wiki should not fire\ntest.1 = {\n\ttype = character_event\n}\n"
	s, f := ck3Sess(t, body)
	cache := &model.Cache{}
	root := filepath.Dir(filepath.Dir(f))
	s = session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, body)
	s.SetWiki(map[string]string{"event": "wiki event page"})
	col := strings.Index(body, "event")
	h := Hover(s, f, 0, col)
	if h != nil {
		t.Fatalf("comment hover = %+v, want nil", h)
	}
	if locs := Definition(s, f, 0, col); len(locs) != 0 {
		t.Fatalf("comment F12 = %v, want empty", locs)
	}
}

func TestHoverKindScopedFieldDocs(t *testing.T) {
	body := "test.1 = {\n\ttype = character_event\n\ttitle = test.1.t\n}\n"
	s, f := ck3Sess(t, body)
	root := filepath.Dir(filepath.Dir(f))
	cache := &model.Cache{
		FieldDocs: map[string]string{"type": "Invite rule type"},
		FieldDocsByKind: map[string]map[string]string{
			"event": {
				"type":  "Event presentation type",
				"title": "Dynamic loc key",
			},
		},
	}
	s = session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, body)
	lines := strings.Split(body, "\n")
	typeCol := strings.Index(lines[1], "type")
	h := Hover(s, f, 1, typeCol)
	if h == nil || !strings.Contains(h.Contents, "presentation") {
		t.Fatalf("type hover = %+v, want events.info prose", h)
	}
	if strings.Contains(h.Contents, "Invite") {
		t.Fatalf("type hover leaked rival docs: %+v", h)
	}
	titleCol := strings.Index(lines[2], "title")
	th := Hover(s, f, 2, titleCol)
	if th == nil || !strings.Contains(th.Contents, "Dynamic") {
		t.Fatalf("title hover = %+v, want events.info prose", th)
	}
	sig := SignatureAt(s, f, 1, typeCol)
	if sig == nil || !strings.Contains(sig.Documentation, "presentation") {
		t.Fatalf("signature = %+v, want events.info prose", sig)
	}
}

func TestDefinitionVanillaLoc(t *testing.T) {
	root := t.TempDir()
	modFile := write(t, root, "events/e.txt", "test.1 = { title = vanilla_key }\n")
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	vanilla := t.TempDir()
	vfile := write(t, vanilla, "localization/english/v.yml",
		"l_english:\n vanilla_key:0 \"Hello\"\n")
	cache := &model.Cache{
		LocEnglish: map[string]string{"vanilla_key": "Hello"},
		LocEnglishSites: map[string]model.LocSite{
			"vanilla_key": {File: vfile, Line: 1},
		},
	}
	s := session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(modFile, "test.1 = { title = vanilla_key }\n")
	col := strings.Index("test.1 = { title = vanilla_key }\n", "vanilla_key")
	locs := Definition(s, modFile, 0, col)
	if len(locs) != 1 || locs[0].URI != vfile {
		t.Fatalf("vanilla loc F12 = %v, want %s", locs, vfile)
	}
	diags := Diagnose(s, modFile)
	for _, d := range diags {
		if d.Code == "missing-required-loc" {
			t.Fatalf("vanilla loc should satisfy lint: %+v", d)
		}
	}
}

func TestDefinitionModAndVanilla(t *testing.T) {
	root := t.TempDir()
	modFile := write(t, root, "common/traits/00.txt", "brave = { category = personality }\n")
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	vanilla := t.TempDir()
	vfile := write(t, vanilla, "common/traits/v.txt", "craven = { category = personality }\n")
	cache := &model.Cache{Defs: []model.Def{{Type: "traits", Key: "craven", Path: vfile, Line: 0}}}
	s := session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(modFile, "brave = { category = personality }\n")

	locs := Definition(s, modFile, 0, 1)
	if len(locs) != 1 || locs[0].URI != modFile {
		t.Fatalf("mod F12 = %v, want %s", locs, modFile)
	}

	refBody := "test.1 = { immediate = { craven = yes } }\n"
	refFile := write(t, root, "events/e.txt", refBody)
	s.DidOpen(refFile, refBody)
	col := strings.Index(refBody, "craven")
	if col < 0 {
		t.Fatal("fixture")
	}
	vlocs := Definition(s, refFile, 0, col)
	if len(vlocs) != 1 || vlocs[0].URI != vfile {
		t.Fatalf("vanilla F12 = %v, want %s", vlocs, vfile)
	}
}

func TestCreateLocKeyBOM(t *testing.T) {
	s, f := ck3Sess(t, "test.1 = {\n	title = brand_new_key\n}\n")
	acts := CodeActions(s, f)
	var edit *WorkspaceEdit
	for _, a := range acts {
		if strings.Contains(a.Title, "brand_new_key") {
			edit = a.Edit
		}
	}
	if edit == nil || len(edit.Create) == 0 {
		t.Fatalf("expected create-loc action, got %#v", acts)
	}
	var body string
	for _, edits := range edit.Changes {
		if len(edits) > 0 {
			body = edits[0].NewText
		}
	}
	if !strings.HasPrefix(body, "\uFEFF") {
		t.Errorf("new loc file must start with UTF-8 BOM, got %q", body)
	}
}

func TestModCompleteDependencies(t *testing.T) {
	root := t.TempDir()
	f := write(t, root, "descriptor.mod", "name = \"t\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, "name = \"t\"\n")
	items := Complete(s, f, 0, 0)
	found := false
	for _, it := range items {
		if it.Label == "dependencies" {
			found = true
		}
	}
	if !found {
		t.Errorf("complete .mod missing dependencies: %v", items)
	}
}

func TestMetaJSONSkipped(t *testing.T) {
	root := t.TempDir()
	f := write(t, root, ".metadata/metadata.json", "{\n")
	s := session.New("ws", "vic3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, "{\n")
	for _, d := range Diagnose(s, f) {
		if d.Code == "unclosed-brace" {
			t.Fatal("metadata.json must not go through the script parser")
		}
	}
}

func TestMissingDescriptorVic3(t *testing.T) {
	root := t.TempDir()
	f := write(t, root, "notes.txt", "foo = { }\n")
	s := session.New("ws", "vic3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, "foo = { }\n")
	diags := Diagnose(s, f)
	found := false
	for _, d := range diags {
		if d.Code == "missing-descriptor" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing-descriptor on vic3 mod root file; %v", diags)
	}
}

func TestLocDefinedFromHeaderFile(t *testing.T) {
	root := t.TempDir()
	ev := write(t, root, "events/x.txt", "namespace = t\n\nt.1 = {\n\ttitle = t.1.t\n}\n")
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "localization/english/events.yml", "l_english:\n t.1.t:0 \"Hello\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(ev, "namespace = t\n\nt.1 = {\n\ttitle = t.1.t\n}\n")
	for _, d := range Diagnose(s, ev) {
		if d.Code == "missing-required-loc" {
			t.Fatalf("unexpected missing loc: %+v", d)
		}
	}
	if v, ok := s.EnglishLoc("t.1.t"); !ok || v != "Hello" {
		t.Fatalf("EnglishLoc = %q %v", v, ok)
	}
}

func TestReferencesUniqueAfterReopen(t *testing.T) {
	root := t.TempDir()
	body := "namespace = t\n\nt.1 = {\n\ttitle = k.t\n}\n\nt.2 = {\n\ttitle = k.t\n}\n"
	ev := write(t, root, "events/x.txt", strings.ReplaceAll(body, "\n", "\r\n"))
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(ev, body)
	locs := References(s, ev, 3, 10) // on k.t in first title
	if len(locs) != 2 {
		t.Fatalf("refs = %d want 2: %+v", len(locs), locs)
	}
	for _, l := range locs {
		if l.Range.Start.Line != l.Range.End.Line {
			t.Errorf("range spilled lines: %+v", l.Range)
		}
	}
}

func lineCol(src, needle string) (line, col int) {
	i := strings.Index(src, needle)
	if i < 0 {
		return -1, -1
	}
	line = strings.Count(src[:i], "\n")
	return line, i - (strings.LastIndex(src[:i], "\n") + 1)
}

func TestHoverScopePrefixNotEventField(t *testing.T) {
	body := "test.1 = {\n\tscope = character\n\timmediate = {\n\t\texists = scope:duel_target\n\t\tsave_scope_as = duel_target\n\t}\n}\n"
	s, f := ck3Sess(t, body)
	root := filepath.Dir(filepath.Dir(f))
	cache := &model.Cache{
		FieldDocs: map[string]string{"scope": "The event root scope type."},
	}
	s = session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, body)
	line, col := lineCol(body, "scope:duel_target")
	h := Hover(s, f, line, col+len("scope:"))
	if h == nil || !strings.Contains(h.Contents, "saved scope") {
		t.Fatalf("hover = %+v, want saved scope card", h)
	}
	if strings.Contains(h.Contents, "event root") {
		t.Fatalf("hover leaked event scope field docs: %+v", h)
	}
	if h.Rel != "" || h.Origin != "" {
		t.Fatalf("hover site = origin %q rel %q, want empty", h.Origin, h.Rel)
	}
	saveLine, saveCol := lineCol(body, "save_scope_as = duel_target")
	h = Hover(s, f, saveLine, saveCol+len("save_scope_as = "))
	if h == nil || !strings.Contains(h.Contents, "saved scope") {
		t.Fatalf("save_scope_as hover = %+v, want saved scope card", h)
	}
	if h.Rel != "" || h.Origin != "" {
		t.Fatalf("save_scope_as hover site = origin %q rel %q, want empty", h.Origin, h.Rel)
	}
}

func TestHoverImmediateStructureKey(t *testing.T) {
	body := "test.1 = {\n\timmediate = { add_gold = 1 }\n}\n"
	s, f := ck3Sess(t, body)
	root := filepath.Dir(filepath.Dir(f))
	cache := &model.Cache{Structures: map[string][]string{"event": {"immediate", "option"}}}
	s = session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, body)
	line, col := lineCol(body, "immediate")
	h := Hover(s, f, line, col)
	if h == nil || !strings.Contains(h.Contents, "event key") ||
		!strings.Contains(h.Contents, "`immediate`") {
		t.Fatalf("immediate hover = %+v, want event key `immediate`", h)
	}
}

func TestDefinitionSavedScope(t *testing.T) {
	body := "test.1 = {\n\timmediate = {\n\t\tsave_scope_as = duel_target\n\t\texists = scope:duel_target\n\t}\n}\n"
	s, f := ck3Sess(t, body)
	line, col := lineCol(body, "scope:duel_target")
	if locs := Definition(s, f, line, col+len("scope:")); len(locs) != 0 {
		t.Fatalf("F12 = %+v, want no saved-scope jump", locs)
	}
	if locs := References(s, f, line, col+len("scope:")); len(locs) != 0 {
		t.Fatalf("refs = %+v, want no saved-scope jumps", locs)
	}
}

func TestReferencesEventTrigger(t *testing.T) {
	body := "namespace = t\n\nt.1 = {\n\timmediate = { trigger_event = t.2 }\n}\n\nt.2 = {\n\ttype = character_event\n}\n"
	s, f := ck3Sess(t, body)
	line, col := lineCol(body, "t.2 =")
	locs := References(s, f, line, col)
	var use bool
	for _, l := range locs {
		if l.Range.Start.Line == 3 {
			use = true
		}
	}
	if !use {
		t.Fatalf("refs = %+v, want trigger_event use site", locs)
	}
}

func TestHoverModLocValue(t *testing.T) {
	root := t.TempDir()
	body := "namespace = t\n\nt.1 = {\n\ttitle = t.1.t\n}\n"
	ev := write(t, root, "events/x.txt", body)
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	write(t, root, "localization/english/events.yml", "l_english:\n t.1.t:0 \"Hello\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(ev, body)
	line, col := lineCol(body, "t.1.t")
	h := Hover(s, ev, line, col)
	if h == nil || !strings.Contains(h.Contents, "localization") {
		t.Fatalf("hover = %+v, want localization", h)
	}
	if !strings.Contains(h.Contents, `"Hello"`) {
		t.Fatalf("hover = %+v, want quoted loc value", h)
	}
	if !strings.Contains(h.Rel, "localization/english/events.yml") {
		t.Fatalf("hover rel = %q, want relative loc path", h.Rel)
	}
}

func TestHoverVanillaLocValue(t *testing.T) {
	root := t.TempDir()
	body := "test.1 = { title = vanilla_key }\n"
	modFile := write(t, root, "events/e.txt", body)
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	vanilla := t.TempDir()
	vfile := write(t, vanilla, "localization/english/v.yml",
		"l_english:\n vanilla_key:0 \"Hello\"\n")
	cache := &model.Cache{
		InstallPath: vanilla,
		LocEnglish:  map[string]string{"vanilla_key": "Hello"},
		LocEnglishSites: map[string]model.LocSite{
			"vanilla_key": {File: vfile, Line: 1},
		},
	}
	s := session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(modFile, body)
	col := strings.Index(body, "vanilla_key")
	h := Hover(s, modFile, 0, col)
	if h == nil || !strings.Contains(h.Contents, "localization") {
		t.Fatalf("hover = %+v, want localization", h)
	}
	if !strings.Contains(h.Contents, `"Hello"`) {
		t.Fatalf("hover = %+v, want quoted loc value", h)
	}
	if !strings.Contains(h.Rel, "localization/english/v.yml") {
		t.Fatalf("hover rel = %q, want relative vanilla path", h.Rel)
	}
	if h.Origin != "vanilla" {
		t.Fatalf("hover origin = %q, want vanilla", h.Origin)
	}
}

func TestHoverScriptedTriggerKind(t *testing.T) {
	root := t.TempDir()
	write(t, root, "common/scripted_triggers/t.txt", "my_trig = { always = yes }\n")
	body := "test.1 = {\n\timmediate = { my_trig = yes }\n}\n"
	ev := write(t, root, "events/x.txt", body)
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(ev, body)
	line, col := lineCol(body, "my_trig")
	h := Hover(s, ev, line, col)
	if h == nil || !strings.Contains(h.Contents, "scripted trigger") ||
		!strings.Contains(h.Contents, "`my_trig`") {
		t.Fatalf("hover = %+v, want scripted trigger `my_trig`", h)
	}
}

func TestHoverEffectKind(t *testing.T) {
	body := "test.1 = {\n\timmediate = { add_gold = 1 }\n}\n"
	s, f := ck3Sess(t, body)
	root := filepath.Dir(filepath.Dir(f))
	cache := &model.Cache{
		Effects:   []string{"add_gold"},
		FieldDocs: map[string]string{"add_gold": "Gives gold."},
	}
	s = session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, body)
	line, col := lineCol(body, "add_gold")
	h := Hover(s, f, line, col)
	if h == nil || !strings.HasPrefix(h.Contents, "effect `add_gold`") {
		t.Fatalf("hover = %+v, want effect `add_gold`", h)
	}
	if !strings.Contains(h.Contents, "Gives gold") {
		t.Fatalf("hover = %+v, want effect docs", h)
	}
}

func TestCompletePrefixRanksStructureKeys(t *testing.T) {
	body := "test.1 = {\n\timm"
	s, f := ck3Sess(t, body)
	root := filepath.Dir(filepath.Dir(f))
	cache := &model.Cache{
		Structures: map[string][]string{"event": {"immediate", "option"}},
		Effects:    []string{"immortal", "immune"},
		Vocabulary: []string{"immune_to", "immortal"},
	}
	s = session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, body)
	line, col := lineCol(body, "imm")
	items := Complete(s, f, line, col+len("imm"))
	if len(items) == 0 || items[0].Label != "immediate" {
		t.Fatalf("complete = %+v, want immediate first", items)
	}
}

func TestCompleteEmptyPrefixCapped(t *testing.T) {
	// Cursor after `{` / tab is not an identifier. Prefix must stay empty
	// (not the whole file from byte 0), or Complete walks the catalog and
	// matches nothing.
	body := "test.1 = {\n\t"
	s, f := ck3Sess(t, body)
	root := filepath.Dir(filepath.Dir(f))
	vocab := make([]string, 5000)
	for i := range vocab {
		vocab[i] = "tok_" + string(rune('a'+i%26)) + string(rune('0'+i%10))
	}
	cache := &model.Cache{
		Structures: map[string][]string{"event": {"immediate", "option"}},
		Effects:    vocab,
		Vocabulary: vocab,
	}
	s = session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, body)
	line, col := lineCol(body, "\t")
	items := Complete(s, f, line, col+1)
	if len(items) > 80 {
		t.Fatalf("complete len = %d, want ≤80", len(items))
	}
	if len(items) == 0 || items[0].Label != "immediate" {
		t.Fatalf("complete = %+v, want immediate first", items)
	}
}

func TestDiagnoseEventTypeSilent(t *testing.T) {
	body := "test.1 = {\n\ttype = character_event\n\ttitle = test.1.t\n}\n"
	s, f := ck3Sess(t, body)
	for _, d := range Diagnose(s, f) {
		if d.Code == "wrong-scope" || d.Code == "unknown-saved-scope" {
			t.Fatalf("unexpected scope lint: %+v", d)
		}
	}
}
