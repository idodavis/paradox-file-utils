// lsp_test.go covers diagnostics/tokens, hover fieldDocs, F12 overlay and vanilla,
// non-ASCII loc columns, create-loc BOM, .mod complete, and missing-descriptor.

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

func TestDiagnoseAndTokens(t *testing.T) {
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
	}
	if !unclosed {
		t.Errorf("expected unclosed-brace; diags=%v", diags)
	}
	if !missing {
		t.Errorf("expected missing-required-loc; diags=%v", diags)
	}
	spans := SemanticTokens(s, f)
	if len(spans) == 0 {
		t.Fatal("expected semantic tokens")
	}
}

func TestHoverFieldDocs(t *testing.T) {
	s, f := ck3Sess(t, "brave = { category = personality }\n")
	s.DidChange(f, "brave = { category = personality }\n")
	// Rebuild with a cache that has field docs. Session cache is set at New;
	// hover also resolves defs. Inject via a new session.
	root := filepath.Dir(filepath.Dir(f))
	cache := &model.Cache{FieldDocs: map[string]string{"category": "The trait category."}}
	s = session.New("ws", "ck3", cache, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, "brave = { category = personality }\n")
	h := Hover(s, f, 0, 12) // on "category"
	if h == nil || !strings.Contains(h.Contents, "trait category") {
		t.Fatalf("hover = %+v, want fieldDocs", h)
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

func TestNonASCIILocColumn(t *testing.T) {
	root := t.TempDir()
	body := "l_english:\n greet:0 \"café\"\n"
	f := write(t, root, "localization/english/a_l_english.yml", body)
	write(t, root, "descriptor.mod", "name = \"t\"\n")
	s := session.New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.DidOpen(f, body)
	spans := SemanticTokens(s, f)
	var str *SemanticSpan
	for i := range spans {
		if spans[i].Type == "string" {
			str = &spans[i]
		}
	}
	if str == nil {
		t.Fatalf("no string token in %v", spans)
	}
	// Inner value café is 5 UTF-8 bytes (é is 2), 4 UTF-16 code units.
	if str.Length != 5 {
		t.Errorf("string token length = %d, want 5 UTF-8 bytes (not UTF-16)", str.Length)
	}
	if str.Line != 1 {
		t.Errorf("string token line = %d, want 1", str.Line)
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
