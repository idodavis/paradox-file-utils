// session_test.go covers the single-file reindex path (identical bytes coalesce to
// one reindex), override resolution through model.Winner, and inline suppressions.

package session

import (
	"os"
	"path/filepath"
	"testing"

	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/parser"
)

// modFixture writes one trait file into a temp mod root and returns (root, file).
func modFixture(t *testing.T, body string) (root, file string) {
	t.Helper()
	root = t.TempDir()
	file = filepath.Join(root, "common", "traits", "00.txt")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, file
}

func TestReindexCoalesces(t *testing.T) {
	root, file := modFixture(t, "brave = { category = personality }\n")
	s := New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})

	s.DidSave(file)
	if got := s.ReindexCount(); got != 1 {
		t.Fatalf("reindex after save = %d, want 1", got)
	}
	s.DidSave(file) // identical bytes -> coalesced, no extra reindex
	if got := s.ReindexCount(); got != 1 {
		t.Fatalf("reindex after no-op save = %d, want 1", got)
	}
	s.DidChange(file, "brave = { category = education }\n")
	if got := s.ReindexCount(); got != 2 {
		t.Fatalf("reindex after change = %d, want 2", got)
	}
}

func TestResolveWinner(t *testing.T) {
	root, _ := modFixture(t, "brave = { category = personality }\n")
	s := New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	d := s.Resolve("brave")
	if d == nil || d.Origin != "mod" || d.Type != "traits" {
		t.Errorf("Resolve(brave) = %v, want mod trait", d)
	}
}

func TestScanSuppressions(t *testing.T) {
	res := parser.Parse("# pmt:ignore-next-line unclosed-brace\nbad = {\nok = 1 # pmt:ignore\n")
	sup := ScanSuppressions(res.Src)
	if !sup.Covers(1, "unclosed-brace") {
		t.Errorf("line 1 should suppress unclosed-brace")
	}
	if sup.Covers(1, "other-code") {
		t.Errorf("line 1 should NOT suppress an unlisted code")
	}
	if !sup.Covers(2, "anything") {
		t.Errorf("bare ignore on line 2 should suppress all codes")
	}
}

func TestReindexNilLocMap(t *testing.T) {
	root := t.TempDir()
	locFile := filepath.Join(root, "localization", "english", "foo.yml")
	if err := os.MkdirAll(filepath.Dir(locFile), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "l_english:\n k.t:0 \"Hi\"\n"
	if err := os.WriteFile(locFile, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	s.index.Loc = nil
	s.DidOpen(locFile, body)
	if v, ok := s.EnglishLoc("k.t"); !ok || v != "Hi" {
		t.Fatalf("EnglishLoc after nil Loc = %q %v", v, ok)
	}
}

func TestDidOpenDropsDiskRefs(t *testing.T) {
	root := t.TempDir()
	ev := filepath.Join(root, "events", "x.txt")
	if err := os.MkdirAll(filepath.Dir(ev), 0o755); err != nil {
		t.Fatal(err)
	}
	crlf := "namespace = test\r\n\r\ntest.1 = {\r\n\ttitle = k.t\r\n}\r\n"
	if err := os.WriteFile(ev, []byte(crlf), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	lf := parser.Normalize(crlf)
	s.DidOpen(ev, lf)
	n := 0
	for _, r := range s.Index().Refs {
		if r.Key == "k.t" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("k.t refs after DidOpen = %d, want 1", n)
	}
}

func TestReplaceCache(t *testing.T) {
	root, _ := modFixture(t, "brave = { category = personality }\n")
	s := New("ws", "ck3", nil, []model.ModInput{{Origin: "mod", Root: root, Order: 0}})
	if s.Cache() != nil {
		t.Fatal("expected nil cache")
	}
	c := &model.Cache{WorkspaceID: "ws", GameVersion: "1.16"}
	s.ReplaceCache(c)
	got := s.Cache()
	if got == nil || got.GameVersion != "1.16" {
		t.Fatalf("ReplaceCache = %+v", got)
	}
}
