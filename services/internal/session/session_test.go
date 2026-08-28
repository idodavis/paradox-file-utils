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
