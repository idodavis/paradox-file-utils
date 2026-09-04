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
	blob := h.Kind + "\n" + h.Key + "\n" + h.Hint + "\n" + h.Docs + "\n" + h.Body + "\n" + h.Usage + "\n" + h.Owner
	for _, v := range h.Values {
		blob += "\n" + v.Text
	}
	for _, sub := range subs {
		if !strings.Contains(blob, sub) {
			t.Fatalf("hover kind=%#v key=%#v hint=%#v docs=%#v body=%#v usage=%#v missing %q",
				h.Kind, h.Key, h.Hint, h.Docs, h.Body, h.Usage, sub)
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

func TestDiagnoseVanillaSilent(t *testing.T) {
	install := t.TempDir()
	vEvent := write(t, install, "game/events/v.txt",
		"test.1 = {\n\ttitle = missing_key\n")
	vLoc := write(t, install, "game/localization/english/v_l_english.yml",
		"l_english:\n k:0 \"v\"\n")
	modRoot := t.TempDir()
	write(t, modRoot, "descriptor.mod", "name = \"t\"\n")
	modEvent := write(t, modRoot, "events/m.txt",
		"mod.1 = {\n\ttitle = missing_key\n}\n")

	s := session.NewWithLoc("ws", "ck3", "english", &catalog.VanillaCache{
		InstallPath: install,
	}, nil, []catalog.ModInput{{Origin: "mod", Root: modRoot, Order: 0}})
	s.DidOpen(vEvent, "test.1 = {\n\ttitle = missing_key\n")
	s.DidOpen(vLoc, "l_english:\n k:0 \"v\"\n")
	s.DidOpen(modEvent, "mod.1 = {\n\ttitle = missing_key\n}\n")

	if diags := Diagnose(s, vEvent); len(diags) != 0 {
		t.Fatalf("vanilla event diags=%v", diags)
	}
	if diags := Diagnose(s, vLoc); len(diags) != 0 {
		t.Fatalf("vanilla loc diags=%v", diags)
	}
	if !hasDiag(Diagnose(s, modEvent), "missing-required-loc") {
		t.Fatal("mod file must still get missing-required-loc")
	}
	if s.Resolve("test.1") != nil {
		t.Fatal("vanilla DidOpen must not harvest defs into workspace")
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
	wantHover(t, s, f, line, col, "localization", "Hello")
	line, col = lineCol(body, "title = type")
	valCol := col + len("title = ")
	wantHover(t, s, f, line, valCol, "localization")
	if locs := Definition(s, f, line, valCol); len(locs) != 1 || locs[0].URI != vfile {
		t.Fatalf("F12=%v want %s", locs, vfile)
	}
	line, col = lineCol(body, "immediate")
	wantHover(t, s, f, line, col, "event key", "immediate")
	if h := Hover(s, f, line, col); h == nil || h.OriginName != "Crusader Kings III" {
		t.Fatalf("immediate originName=%#v", h)
	}
	line, col = lineCol(body, "add_gold")
	wantHover(t, s, f, line, col, "effect", "add_gold", "Gives gold")
	line, col = lineCol(body, "my_trig")
	wantHover(t, s, f, line, col, "scripted trigger", "my_trig")
	line, col = lineCol(body, "scope:duel_target")
	wantHover(t, s, f, line, col+len("scope:"), "saved scope")
	line, col = lineCol(body, "save_scope_as = duel_target")
	wantHover(t, s, f, line, col+len("save_scope_as = "), "saved scope")
	line, col = lineCol(body, "scope:duel_target")
	if locs := Definition(s, f, line, col+len("scope:")); len(locs) != 1 {
		t.Fatalf("saved scope F12=%v", locs)
	}
	if refs := References(s, f, line, col+len("scope:")); len(refs) < 2 {
		t.Fatalf("saved scope refs=%v want def+use", refs)
	}
	line, col = lineCol(body, "test.1.t")
	if h := Hover(s, f, line, col); h == nil || h.Body != "Hello" {
		t.Fatalf("loc Body=%#v", h)
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
	if locs := Definition(s, tf, 0, 1); len(locs) != 1 || !session.SamePath(locs[0].URI, tf) {
		t.Fatalf("mod F12=%v want %s", locs, tf)
	}
}

func TestHoverDescriptorOrigin(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	descA := "name = \"Alpha\"\npicture = \"thumb.png\"\n"
	descB := "name = \"Beta\"\npicture = \"other.png\"\n"
	fA := write(t, rootA, "descriptor.mod", descA)
	fB := write(t, rootB, "descriptor.mod", descB)
	s := session.NewWithLoc("ws", "ck3", "english", &catalog.VanillaCache{
		FieldInfoByKind: map[string]map[string]string{
			"mod_descriptor": {"picture": "A picture for the mod."},
		},
	}, nil, []catalog.ModInput{
		{Origin: "alpha", Root: rootA, Order: 0, Name: "Alpha Mod"},
		{Origin: "beta", Root: rootB, Order: 1, Name: "Beta Mod"},
	})
	s.DidOpen(fA, descA)
	s.DidOpen(fB, descB)
	line, col := lineCol(descA, "picture")
	h := Hover(s, fA, line, col)
	if h == nil {
		t.Fatal("hover nil")
	}
	if h.Origin != "alpha" || h.OriginName != "Alpha Mod" {
		t.Fatalf("origin=%q originName=%q", h.Origin, h.OriginName)
	}
	if !strings.Contains(h.Rel, "descriptor.mod") {
		t.Fatalf("rel=%q", h.Rel)
	}
	locs := Definition(s, fA, line, col)
	if len(locs) != 1 || locs[0].URI != fA {
		t.Fatalf("definition=%v want %s", locs, fA)
	}
	if w := s.Resolve("picture"); w == nil || w.Origin != "beta" {
		t.Fatalf("workspace winner=%v want beta", w)
	}
	if h.VanillaPath != "" {
		t.Fatalf("descriptor must not report a vanilla overlay: %#v", h)
	}
}

func TestHoverVanillaOverlay(t *testing.T) {
	vbody := "overlay.1 = {\n\ttype = character_event\n}\n"
	vroot := t.TempDir()
	vpath := write(t, vroot, "events/vanilla.txt", vbody)
	body := "overlay.1 = {\n\ttype = character_event\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{"events/x.txt": body},
		&catalog.VanillaCache{
			Defs: []catalog.Def{{
				Kind: "event", Key: "overlay.1", Path: vpath, Line: 0, Start: 0, End: 9,
			}},
		}, nil)
	f := filepath.Join(root, "events", "x.txt")
	line, col := lineCol(body, "overlay.1")
	h := Hover(s, f, line, col)
	if h == nil {
		t.Fatal("hover nil")
	}
	if h.Origin != "mod" {
		t.Fatalf("origin=%q", h.Origin)
	}
	if h.VanillaPath == "" {
		t.Fatalf("want vanilla overlay path, got %#v", h)
	}
	if !session.SamePath(h.VanillaPath, vpath) {
		t.Fatalf("vanillaPath=%q want %s", h.VanillaPath, vpath)
	}
	if h.VanillaOriginName != "Crusader Kings III" {
		t.Fatalf("vanillaOriginName=%q", h.VanillaOriginName)
	}
	locs := Definition(s, f, line, col)
	if !hasLocURI(locs, f) || !hasLocURI(locs, vpath) {
		t.Fatalf("definition=%v want mod %s and vanilla %s", locs, f, vpath)
	}
	refs := References(s, f, line, col)
	if !hasLocURI(refs, vpath) {
		t.Fatalf("references=%v want vanilla %s", refs, vpath)
	}
}

func hasHoverValue(h *HoverResult, text string) bool {
	if h == nil {
		return false
	}
	for _, v := range h.Values {
		if v.Text == text {
			return true
		}
	}
	return false
}

func hasLocURI(locs []Location, path string) bool {
	for _, l := range locs {
		if session.SamePath(l.URI, path) {
			return true
		}
	}
	return false
}

func TestDefinitionCallKindName(t *testing.T) {
	trig := "scripted_trigger my_name = { always = yes }\n"
	call := "ns.1 = {\n\tmy_name = yes\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/scripted_triggers/x.txt": trig,
		"events/e.txt":                   call,
	}, nil, nil)
	tf := filepath.Join(root, "common", "scripted_triggers", "x.txt")
	ef := filepath.Join(root, "events", "e.txt")
	for _, needle := range []string{"scripted_trigger", "my_name"} {
		line, col := lineCol(trig, needle)
		locs := Definition(s, tf, line, col)
		if len(locs) == 0 {
			t.Fatalf("F12 on %s: empty", needle)
		}
		if locs[0].TargetRange == nil ||
			locs[0].TargetRange.End.Line < locs[0].Range.Start.Line {
			t.Fatalf("F12 on %s: want block targetRange, got %#v", needle, locs[0])
		}
	}
	line, col := lineCol(call, "my_name")
	locs := Definition(s, ef, line, col)
	if len(locs) == 0 {
		t.Fatalf("F12 from call: empty")
	}
	if !session.SamePath(locs[0].URI, tf) {
		t.Fatalf("F12 from call=%v want %s", locs, tf)
	}
	refs := References(s, tf, 0, strings.Index(trig, "my_name"))
	if !hasLocURI(refs, ef) {
		t.Fatalf("refs=%v want call site %s", refs, ef)
	}

	inline := "scripted_trigger inline_trig = { always = yes }\nns.1 = { inline_trig = yes }\n"
	s2, root2 := buildSession(t, "ck3", map[string]string{
		"events/birth.txt": inline,
	}, nil, nil)
	bf := filepath.Join(root2, "events", "birth.txt")
	for _, needle := range []string{"scripted_trigger", "inline_trig"} {
		line, col := lineCol(inline, needle)
		if locs := Definition(s2, bf, line, col); len(locs) == 0 {
			t.Fatalf("events-file F12 on %s: empty", needle)
		}
	}

	plain := "my_trig = { always = yes }\n"
	s3, root3 := buildSession(t, "ck3", map[string]string{
		"common/scripted_triggers/t.txt": plain,
	}, nil, nil)
	pf := filepath.Join(root3, "common", "scripted_triggers", "t.txt")
	line, col = lineCol(plain, "my_trig")
	locs = Definition(s3, pf, line, col)
	if len(locs) == 0 {
		t.Fatalf("folder-form F12 on my_trig: empty")
	}
	if locs[0].TargetRange == nil {
		t.Fatalf("folder-form F12: want targetRange, got %#v", locs[0])
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
	wantHover(t, s, tf, line, col, "scripted trigger", "my_name")
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

func TestScriptNamesNavigate(t *testing.T) {
	a := `test.1 = {
	immediate = {
		add_character_flag = used_life
		set_variable = { name = foo value = 3 }
		save_scope_as = duel_target
	}
}
`
	b := `test.2 = {
	immediate = {
		has_character_flag = used_life
		has_variable = foo
		exists = scope:duel_target
		exists = var:foo
	}
}
`
	s, root := buildSession(t, "ck3", map[string]string{
		"events/a.txt":                           a,
		"events/b.txt":                           b,
		"common/coat_of_arms/coat_of_arms/x.txt": "b_appleby = { pattern = \"p.dds\" }\n",
		"common/landed_titles/t.txt":             "k_x = { coat_of_arms = b_appleby }\n",
	}, nil, nil)
	fa := filepath.Join(root, "events", "a.txt")
	fb := filepath.Join(root, "events", "b.txt")

	t.Run("character flag", func(t *testing.T) {
		line, col := lineCol(b, "has_character_flag = used_life")
		valCol := col + len("has_character_flag = ")
		wantHover(t, s, fb, line, valCol, "character flag", "used_life")
		if locs := Definition(s, fb, line, valCol); len(locs) != 1 ||
			!session.SamePath(locs[0].URI, fa) {
			t.Fatalf("F12=%v want %s", locs, fa)
		}
		if refs := References(s, fb, line, valCol); len(refs) < 2 {
			t.Fatalf("refs=%v", refs)
		}
		if syms := DocumentSymbols(s, fa); len(syms) != 1 || syms[0].Name != "test.1" {
			t.Fatalf("outline must skip ephemeral: %v", syms)
		}
	})
	t.Run("variable", func(t *testing.T) {
		line, col := lineCol(b, "has_variable = foo")
		valCol := col + len("has_variable = ")
		wantHover(t, s, fb, line, valCol, "variable", "3")
		if h := Hover(s, fb, line, valCol); h == nil || !hasHoverValue(h, "3") {
			t.Fatalf("variable Values=%#v", h)
		}
		line, col = lineCol(b, "var:foo")
		wantHover(t, s, fb, line, col+len("var:"), "variable")
		if locs := Definition(s, fb, line, col+len("var:")); len(locs) != 1 {
			t.Fatalf("var: F12=%v", locs)
		}
		line, col = lineCol(b, "has_variable = ")
		items := Complete(s, fb, line, col+len("has_variable = "))
		if !hasComplete(items, "foo") {
			t.Fatalf("complete has_variable: %v", labels(items))
		}
	})
	t.Run("saved scope cross-file", func(t *testing.T) {
		line, col := lineCol(b, "scope:duel_target")
		wantHover(t, s, fb, line, col+len("scope:"), "saved scope", "root")
		if locs := Definition(s, fb, line, col+len("scope:")); len(locs) != 1 ||
			!session.SamePath(locs[0].URI, fa) {
			t.Fatalf("scope F12=%v", locs)
		}
	})
	t.Run("coa ref", func(t *testing.T) {
		titles := filepath.Join(root, "common", "landed_titles", "t.txt")
		body := "k_x = { coat_of_arms = b_appleby }\n"
		line, col := lineCol(body, "coat_of_arms = b_appleby")
		valCol := col + len("coat_of_arms = ")
		wantHover(t, s, titles, line, valCol, "coat of arms", "b_appleby")
		if locs := Definition(s, titles, line, valCol); len(locs) != 1 {
			t.Fatalf("coa F12=%v", locs)
		}
		items := Complete(s, titles, line, col+len("coat_of_arms = "))
		if !hasComplete(items, "b_appleby") {
			t.Fatalf("complete coa: %v", labels(items))
		}
	})
}

func TestKindMetaAndGameInfoOverride(t *testing.T) {
	body := "test.1 = {\n\ttype = character_event\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"events/x.txt": body,
	}, &catalog.VanillaCache{
		FieldInfoByKind: map[string]map[string]string{
			"event": {"type": "presentation from info"},
		},
	}, nil)
	f := filepath.Join(root, "events", "x.txt")
	line, col := lineCol(body, "type =")
	h := Hover(s, f, line, col)
	if h == nil || h.Docs != "presentation from info" {
		t.Fatalf("docs=%#v", h)
	}
	if h.Hint != "" {
		t.Fatalf("game info should hide kind hint, got %q", h.Hint)
	}

	tfBody := "brave = { category = personality }\n"
	s, root = buildSession(t, "ck3", map[string]string{
		"common/traits/00.txt": tfBody,
	}, nil, nil)
	tf := filepath.Join(root, "common", "traits", "00.txt")
	h = Hover(s, tf, 0, 1)
	if h == nil || !strings.Contains(h.Hint, "trait_") {
		t.Fatalf("trait hint=%#v", h)
	}
}

func TestLocHoverUnescape(t *testing.T) {
	locFile := "l_english:\n k.t:0 \"\\n\\nHello\"\n"
	body := "test.1 = { title = k.t }\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"localization/english/a_l_english.yml": locFile,
		"events/x.txt":                         body,
	}, nil, nil)
	f := filepath.Join(root, "events", "x.txt")
	line, col := lineCol(body, "k.t")
	h := Hover(s, f, line, col)
	if h == nil || h.Body != "Hello" {
		t.Fatalf("Body=%#v", h)
	}
	if h.Hint != "Default loc text for this key" {
		t.Fatalf("hint=%q", h.Hint)
	}
}

func TestLocEngineValueHover(t *testing.T) {
	body := "l_english:\n r:0 \"Cost: $VALUE|=+0$\"\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"localization/english/a_l_english.yml": body,
	}, nil, nil)
	f := filepath.Join(root, "localization", "english", "a_l_english.yml")
	line, col := lineCol(body, "VALUE")
	h := Hover(s, f, line, col)
	if h == nil || h.Kind != "loc value" ||
		!strings.Contains(h.Hint, "Engine-supplied number") {
		t.Fatalf("hover=%#v", h)
	}
	if locs := Definition(s, f, line, col); len(locs) != 0 {
		t.Fatalf("engine VALUE F12=%v", locs)
	}
}

func TestScriptParamF12(t *testing.T) {
	trig := "my_trig = { exists = $TARGET$ }\n"
	call := "ev.1 = { my_trig = { TARGET = title:k_france.holder } }\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/scripted_triggers/t.txt": trig,
		"events/e.txt":                   call,
	}, nil, nil)
	tf := filepath.Join(root, "common", "scripted_triggers", "t.txt")
	line, col := lineCol(trig, "$TARGET$")
	if locs := Definition(s, tf, line, col+1); len(locs) != 1 {
		t.Fatalf("def F12=%v", locs)
	}
	h := Hover(s, tf, line, col+1)
	if h == nil || h.Owner != "my_trig" || !hasHoverValue(h, "title:k_france.holder") {
		t.Fatalf("param hover=%#v", h)
	}
	ef := filepath.Join(root, "events", "e.txt")
	line, col = lineCol(call, "TARGET =")
	if locs := Definition(s, ef, line, col); len(locs) < 1 {
		t.Fatalf("call F12=%v", locs)
	}
}

func TestScriptParamOwnerScoped(t *testing.T) {
	a := "trig_a = { exists = $TARGET$ }\n"
	b := "trig_b = { exists = $TARGET$ }\n"
	call := "ev.1 = { trig_a = { TARGET = title:k_france.holder } }\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/scripted_triggers/a.txt": a,
		"common/scripted_triggers/b.txt": b,
		"events/e.txt":                   call,
	}, nil, nil)
	af := filepath.Join(root, "common", "scripted_triggers", "a.txt")
	bf := filepath.Join(root, "common", "scripted_triggers", "b.txt")
	ef := filepath.Join(root, "events", "e.txt")
	line, col := lineCol(call, "TARGET =")
	locs := Definition(s, ef, line, col)
	if len(locs) != 1 || !session.SamePath(locs[0].URI, af) {
		t.Fatalf("owner F12=%v want %s not %s", locs, af, bf)
	}
}

func TestScriptParamMultiValue(t *testing.T) {
	trig := "my_trig = { exists = $TARGET$ }\n"
	call := "ev.1 = {\n\tmy_trig = { TARGET = title:c_rouen }\n" +
		"\tmy_trig = { TARGET = scope:adventurer_target }\n" +
		"\tmy_trig = { TARGET = scope:adventurer_target }\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/scripted_triggers/t.txt": trig,
		"events/e.txt":                   call,
	}, nil, nil)
	tf := filepath.Join(root, "common", "scripted_triggers", "t.txt")
	line, col := lineCol(trig, "$TARGET$")
	h := Hover(s, tf, line, col+1)
	if h == nil || h.Owner != "my_trig" ||
		!hasHoverValue(h, "title:c_rouen") ||
		!hasHoverValue(h, "scope:adventurer_target") {
		t.Fatalf("param values=%#v", h)
	}
	var rouen, adv int
	for _, v := range h.Values {
		if v.Text == "title:c_rouen" {
			rouen = v.Count
		}
		if v.Text == "scope:adventurer_target" {
			adv = v.Count
		}
	}
	if rouen != 1 || adv != 2 {
		t.Fatalf("counts rouen=%d adv=%d values=%#v", rouen, adv, h.Values)
	}
}

func TestSavedScopeIteratorValue(t *testing.T) {
	src := "ev.1 = {\n" +
		"\ttitle:c_rouen = { save_scope_as = adventurer_target }\n" +
		"\trandom_in_list = {\n\t\tlist = western_scandi_targets_list\n" +
		"\t\tsave_scope_as = adventurer_target\n\t}\n" +
		"\trandom_in_list = {\n\t\tlist = western_scandi_targets_list\n" +
		"\t\tsave_scope_as = adventurer_target\n\t}\n" +
		"\texists = scope:adventurer_target\n}\n"
	s, f := ck3Sess(t, src)
	line, col := lineCol(src, "scope:adventurer_target")
	h := Hover(s, f, line, col+len("scope:"))
	if h == nil || !hasHoverValue(h, "title:c_rouen") ||
		!hasHoverValue(h, "random_in_list list=western_scandi_targets_list") {
		t.Fatalf("scope values=%#v", h)
	}
}

func TestDottedScriptParamF12(t *testing.T) {
	trig := "my_trig = {\n\t$TARGET$.holder = { is_ai = yes }\n}\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/scripted_triggers/t.txt": trig,
	}, nil, nil)
	tf := filepath.Join(root, "common", "scripted_triggers", "t.txt")
	line, col := lineCol(trig, "$TARGET$")
	if locs := Definition(s, tf, line, col+1); len(locs) == 0 {
		t.Fatalf("dotted $TARGET$ F12 empty")
	}
}

func TestGameRuleSettingHover(t *testing.T) {
	rules := "secret_unbeliever_found_rule = {\n\tdefault = a\n\tsuf_quieter = { }\n}\n"
	loc := "l_english:\n setting_suf_quieter:0 \"Quieter\"\n"
	use := "ev.1 = { limit = { has_game_rule = suf_quieter } }\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/game_rules/x.txt":              rules,
		"localization/english/a_l_english.yml": loc,
		"events/e.txt":                         use,
	}, nil, &catalog.VanillaLoc{
		Sites: map[string]catalog.LocEntry{
			"setting_suf_quieter": {Path: "loc", Line: 1, Value: "Quieter"},
		},
	})
	ef := filepath.Join(root, "events", "e.txt")
	line, col := lineCol(use, "suf_quieter")
	h := Hover(s, ef, line, col)
	if h == nil || h.Body != "Quieter" {
		t.Fatalf("hover body=%#v", h)
	}
	if locs := Definition(s, ef, line, col); len(locs) != 1 {
		t.Fatalf("F12=%v", locs)
	}
}

func ck3VanillaSess(t *testing.T, install string, cache *catalog.VanillaCache) *session.Session {
	t.Helper()
	if cache == nil {
		cache = &catalog.VanillaCache{}
	}
	cache.InstallPath = install
	if cache.GameID == "" {
		cache.GameID = "ck3"
	}
	mod := t.TempDir()
	write(t, mod, "descriptor.mod", "name = \"t\"\n")
	return session.NewWithLoc("ws", "ck3", "english", cache, nil, []catalog.ModInput{
		{Origin: "mod", Root: mod, Order: 0},
	})
}

func TestVanillaScriptParamIgnoresFieldDoc(t *testing.T) {
	trig := "can_fight = {\n\tsubject = $ARMY_OWNER$\n}\n"
	call := "ev.1 = {\n\tcan_fight = { ARMY_OWNER = scope:potential }\n}\n"
	trigRel := "game/common/scripted_triggers/00_religious_triggers.txt"
	callRel := "game/events/dlc/mpo/mpo_events_ariana.txt"
	install := t.TempDir()
	tf := write(t, install, trigRel, trig)
	ef := write(t, install, callRel, call)
	s := ck3VanillaSess(t, install, &catalog.VanillaCache{
		Defs: []catalog.Def{{
			Kind: "scripted_trigger", Key: "can_fight", Path: tf,
			Start: 0, End: len("can_fight"),
		}},
		FieldInfo: map[string]string{
			"ARMY_OWNER": "Get owner of scoped army",
		},
	})
	s.DidOpen(tf, trig)
	s.DidOpen(ef, call)
	line, col := lineCol(trig, "$ARMY_OWNER$")
	h := Hover(s, tf, line, col+1)
	if h == nil || h.Kind != "script parameter" ||
		!hasHoverValue(h, "scope:potential") {
		t.Fatalf("hover=%#v", h)
	}
}

func TestVanillaTriggerCallRefs(t *testing.T) {
	trig := "is_wrong_gender_in_faith_trigger = { always = yes }\n"
	call := "mpo_events_ariana.0100 = {\n\ttrigger = {\n" +
		"\t\tis_wrong_gender_in_faith_trigger = { FAITH = root.faith }\n\t}\n}\n"
	trigRel := "game/common/scripted_triggers/00_religious_triggers.txt"
	callRel := "game/events/dlc/mpo/mpo_events_ariana.txt"
	key := "is_wrong_gender_in_faith_trigger"
	t.Run("live open", func(t *testing.T) {
		install := t.TempDir()
		tf := write(t, install, trigRel, trig)
		ef := write(t, install, callRel, call)
		s := ck3VanillaSess(t, install, &catalog.VanillaCache{
			Defs: []catalog.Def{{
				Kind: "scripted_trigger", Key: key, Path: tf,
				Start: 0, End: len(key),
			}},
		})
		s.DidOpen(tf, trig)
		s.DidOpen(ef, call)
		line, col := lineCol(trig, key)
		if refs := References(s, tf, line, col); !hasLocURI(refs, ef) {
			t.Fatalf("refs=%v want %s", refs, ef)
		}
		line, col = lineCol(call, key)
		if locs := Definition(s, ef, line, col); !hasLocURI(locs, tf) {
			t.Fatalf("F12=%v want %s", locs, tf)
		}
	})
	t.Run("persisted call refs", func(t *testing.T) {
		install := t.TempDir()
		tf := write(t, install, trigRel, trig)
		ef := write(t, install, callRel, call)
		i := strings.Index(call, key)
		s := ck3VanillaSess(t, install, &catalog.VanillaCache{
			Defs: []catalog.Def{{
				Kind: "scripted_trigger", Key: key, Path: tf,
				Start: 0, End: len(key),
			}},
			CallRefs: []catalog.Ref{{
				Key: key, Kind: "scripted_trigger", Path: ef,
				Line:  strings.Count(call[:i], "\n"),
				Start: i, End: i + len(key),
			}},
		})
		s.DidOpen(tf, trig)
		line, col := lineCol(trig, key)
		if refs := References(s, tf, line, col); !hasLocURI(refs, ef) {
			t.Fatalf("refs=%v want persisted %s", refs, ef)
		}
	})
}

func TestScriptValueRef(t *testing.T) {
	vals := "hre_conquest_ai_score_value = 100\n"
	use := "cb = { value = hre_conquest_ai_score_value }\n"
	s, root := buildSession(t, "ck3", map[string]string{
		"common/script_values/v.txt":      vals,
		"common/casus_belli_types/cb.txt": use,
	}, nil, nil)
	vf := filepath.Join(root, "common", "script_values", "v.txt")
	uf := filepath.Join(root, "common", "casus_belli_types", "cb.txt")
	line, col := lineCol(use, "hre_conquest_ai_score_value")
	if locs := Definition(s, uf, line, col); len(locs) != 1 ||
		!session.SamePath(locs[0].URI, vf) {
		t.Fatalf("F12=%v want %s", locs, vf)
	}
}
