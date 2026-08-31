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
		"events/x.txt": body,
		"localization/english/events.yml": "l_english:\n test.1.t:0 \"Hello\"\n",
		"common/scripted_triggers/t.txt":  "my_trig = { always = yes }\n",
		"common/traits/00.txt":            "brave = { category = personality }\n",
	}, &catalog.VanillaCache{
		FieldDocs: map[string]string{
			"type": "Invite", "add_gold": "Gives gold.",
		},
		FieldDocsByKind: map[string]map[string]string{
			"event": {"type": "presentation", "title": "Dynamic"},
		},
		Structures: map[string][]string{"event": {"immediate"}},
		Effects:    []string{"add_gold"},
		Defs:       []catalog.Def{{Type: "loc_key", Key: "type", Path: vfile, Line: 1}},
	}, &catalog.VanillaLoc{
		Sites: map[string]catalog.LocEntry{"type": {File: vfile, Line: 1, Value: "Type"}},
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
	wantHover(t, s, f, line, col, "localization", `"Hello"`)
	line, col = lineCol(body, "title = type")
	valCol := col + len("title = ")
	wantHover(t, s, f, line, valCol, "localization")
	if locs := Definition(s, f, line, valCol); len(locs) != 1 || locs[0].URI != vfile {
		t.Fatalf("F12=%v want %s", locs, vfile)
	}
	line, col = lineCol(body, "immediate")
	wantHover(t, s, f, line, col, "event key", "`immediate`")
	line, col = lineCol(body, "add_gold")
	wantHover(t, s, f, line, col, "effect `add_gold`", "Gives gold")
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
