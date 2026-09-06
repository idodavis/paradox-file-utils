package lsp

import (
	"path/filepath"
	"testing"
)

// `key = ` parses to an Assignment with a nil Value. Every request must survive
// it: this is the state the buffer is in on every keystroke while typing a
// value, so a panic here breaks the editor mid-word.
func TestZNilValueRequests(t *testing.T) {
	for _, body := range []string{
		"test.1 = {\n\ttype = \n}\n",
		"test.1 = {\n\timmediate = {\n\t\tadd_gold = \n\t}\n}\n",
		"test.1 = \n",
		"a = = \n",
		"test.1 = {\n\tculture_group:\n}\n",
		"test.1 = {\n\thas_culture_group = culture_group:\n}\n",
		"x = {\n\ty =",
	} {
		s, root := buildSession(t, "ck3", map[string]string{"events/x.txt": body}, nil, nil)
		f := filepath.Join(root, "events", "x.txt")
		for line := range 4 {
			for col := range 24 {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Fatalf("panic at %d:%d in %q: %v", line, col, body, r)
						}
					}()
					_ = Hover(s, f, line, col)
					_ = Complete(s, f, line, col)
					_ = Definition(s, f, line, col)
					_ = References(s, f, line, col)
					_ = Diagnose(s, f)
					_ = FormatDocument(s, f)
				}()
			}
		}
	}
}
