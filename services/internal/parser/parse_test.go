// parse_test.go covers the parser/lexer: assignments, GUI templates, error
// recovery, tagged blocks, quoted keys, and byte-offset decoding.

package parser

import "testing"

// topKeys returns the keys of a source's top-level assignments.
func topKeys(src string) []string {
	r := Parse(src)
	tops := TopAssignments(r)
	keys := make([]string, len(tops))
	for i, a := range tops {
		keys[i] = a.Key
	}
	return keys
}

func TestParseBasicAssignments(t *testing.T) {
	src := "foo = {\n\tid = 1\n}\nbar = 2\n"
	got := topKeys(src)
	want := []string{"foo", "bar"}
	if len(got) != len(want) || got[0] != "foo" || got[1] != "bar" {
		t.Fatalf("top keys = %v want %v", got, want)
	}
}

func TestParseGUITemplateIsOneAssignment(t *testing.T) {
	// `template Foo { ... }`: `template` is a bare ValueStmt (skipped by
	// TopAssignments); `Foo { ... }` is one Op=="" assignment.
	src := "template Foo {\n\tsize = { 10 20 }\n}\n"
	r := Parse(src)
	tops := TopAssignments(r)
	if len(tops) != 1 || tops[0].Key != "Foo" {
		t.Fatalf("tops = %+v", tops)
	}
	// The Foo assignment must have Op == "".
	var foo *Assignment
	for _, st := range r.Root.Statements {
		if a, ok := st.(*Assignment); ok && a.Key.Text == "Foo" {
			foo = a
		}
	}
	if foo == nil {
		t.Fatal("no Foo assignment")
	}
	if foo.Op != "" {
		t.Fatalf("Foo op = %q want empty", foo.Op)
	}
	if _, ok := foo.Value.(*Block); !ok {
		t.Fatalf("Foo value is not a block: %T", foo.Value)
	}
}

func TestParseUnclosedBraceRecovers(t *testing.T) {
	src := "foo = {\n\tid = 1\n" // missing closing brace
	r := Parse(src)
	found := false
	for _, e := range r.Errors {
		if e.Code == ErrUnclosedBrace {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unclosed-brace error, got %+v", r.Errors)
	}
}

func TestParseUnterminatedString(t *testing.T) {
	src := "name = \"hello\n"
	r := Parse(src)
	found := false
	for _, e := range r.Errors {
		if e.Code == ErrUnterminatedString {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unterminated-string error, got %+v", r.Errors)
	}
}

func TestParseMissingValue(t *testing.T) {
	src := "foo =\nbar = 1\n"
	r := Parse(src)
	found := false
	for _, e := range r.Errors {
		if e.Code == ErrMissingValue {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing-value error, got %+v", r.Errors)
	}
	// `bar` must still parse as its own assignment.
	if keys := topKeys(src); len(keys) != 2 || keys[1] != "bar" {
		t.Fatalf("recovery failed, keys = %v", keys)
	}
}

func TestParseComments(t *testing.T) {
	src := "# header\nfoo = 1\n"
	r := Parse(src)
	if len(r.Comments) != 1 || r.Comments[0].Text != "# header" {
		t.Fatalf("comments = %+v", r.Comments)
	}
	if r.Comments[0].Line != 0 {
		t.Fatalf("comment line = %d want 0", r.Comments[0].Line)
	}
}

func TestParseQuotedKeyStripsQuotes(t *testing.T) {
	src := "\"my key\" = 1\n"
	r := Parse(src)
	tops := TopAssignments(r)
	if len(tops) != 1 || tops[0].Key != "my key" {
		t.Fatalf("tops = %+v", tops)
	}
}

func TestParseTaggedBlock(t *testing.T) {
	src := "color = rgb { 255 0 0 }\n"
	r := Parse(src)
	var a *Assignment
	for _, st := range r.Root.Statements {
		if x, ok := st.(*Assignment); ok {
			a = x
		}
	}
	if a == nil {
		t.Fatal("no assignment")
	}
	tb, ok := a.Value.(*TaggedBlock)
	if !ok {
		t.Fatalf("value is %T want *TaggedBlock", a.Value)
	}
	if tb.Tag.Text != "rgb" {
		t.Fatalf("tag = %q", tb.Tag.Text)
	}
}

func TestDecodeStripsBOMAndLatin1Fallback(t *testing.T) {
	withBOM := append([]byte{0xEF, 0xBB, 0xBF}, []byte("foo = 1")...)
	text, had := Decode(withBOM)
	if !had || text != "foo = 1" {
		t.Fatalf("BOM decode: had=%v text=%q", had, text)
	}
	// 0xE9 alone is invalid UTF-8 -> latin1 fallback (é).
	text2, had2 := Decode([]byte{'x', 0xE9, 'y'})
	if had2 || text2 != "x\u00e9y" {
		t.Fatalf("latin1 fallback: had=%v text=%q", had2, text2)
	}
}

func TestLineIndents(t *testing.T) {
	got := LineIndents("a = {\n\tb = 1\n}\n")
	if len(got) < 3 || got[0] != 0 || got[1] != 1 || got[2] != 0 {
		t.Fatalf("LineIndents = %v, want [0 1 0 ...]", got)
	}
}
