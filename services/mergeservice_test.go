// Merge goldens: TopAssignments slice contract, PREFER comments, GUI template, EU5 keys.
package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseFileObjects(t *testing.T) {
	tests := []struct {
		name, file, src string
		keys            []string
		prefer          string
	}{
		{
			"prefer comment", "script.txt",
			"foo = {\n\tid = 1\n}\n# PREFER: B\nbar = {\n\tid = 2\n}\n",
			[]string{"foo", "bar"}, "B",
		},
		{
			"gui template", "window.gui",
			"template Foo {\n\tsize = { 10 20 }\n}\n",
			[]string{"Foo"}, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs, err := parseFileObjects(writeTemp(t, t.TempDir(), tt.file, tt.src))
			if err != nil {
				t.Fatal(err)
			}
			if len(objs) != len(tt.keys) {
				t.Fatalf("objects %d keys %#v", len(objs), mergeKeys(objs))
			}
			for i, k := range tt.keys {
				if objs[i].Key != k {
					t.Fatalf("keys %#v want %#v", mergeKeys(objs), tt.keys)
				}
			}
			got := ""
			for _, o := range objs {
				got += o.RawText
			}
			if got != tt.src {
				t.Fatalf("reconstruct\n got %q\nwant %q", got, tt.src)
			}
			if tt.prefer != "" && objs[1].PreferSide != tt.prefer {
				t.Fatalf("prefer %q", objs[1].PreferSide)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name, a, b, aName, bName string
		key                      string
		want, drop               string
		preferBar                bool
	}{
		{
			"inject matches bare",
			"INJECT:foo = {\n\tx = 1\n}\n", "foo = {\n\tx = 2\n}\n",
			"a.txt", "b.txt", "foo", "x = 1", "x = 2", false,
		},
		{
			"prefer B keeps comment",
			"foo = { x = 1 }\n# PREFER: B\nbar = { y = 1 }\n",
			"foo = { x = 9 }\n# PREFER: B\nbar = { y = 9 }\n",
			"a.txt", "b.txt", "", "# PREFER: B", "", true,
		},
		{
			"gui template A wins",
			"template Foo {\n\tsize = { 10 20 }\n}\n",
			"template Foo {\n\tsize = { 30 40 }\n}\n",
			"a.gui", "b.gui", "Foo", "template Foo", "30 40", false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			m := &MergeService{}
			mr, err := m.performMerge(
				writeTemp(t, dir, tt.aName, tt.a),
				writeTemp(t, dir, tt.bName, tt.b),
				MergerOptions{},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.key != "" {
				if len(mr.ResolvedConflicts) != 1 || mr.ResolvedConflicts[0].Key != tt.key {
					t.Fatalf("conflicts %#v", mr.ResolvedConflicts)
				}
			}
			if tt.want != "" && !strings.Contains(mr.Content, tt.want) {
				t.Fatalf("missing %q in %q", tt.want, mr.Content)
			}
			if tt.drop != "" && strings.Contains(mr.Content, tt.drop) {
				t.Fatalf("unexpected %q in %q", tt.drop, mr.Content)
			}
			if tt.preferBar && !strings.Contains(mr.Content, "y = 9") {
				t.Fatalf("PREFER B not applied: %q", mr.Content)
			}
		})
	}
}

func mergeKeys(objs []scriptObject) []string {
	out := make([]string, len(objs))
	for i, o := range objs {
		out[i] = o.Key
	}
	return out
}
