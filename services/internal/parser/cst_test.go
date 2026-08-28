// cst_test.go covers LineIndex offset<->position math (incl. UTF-8 byte columns)
// and the WalkStatements / NodeAtOffset helpers.

package parser

import "testing"

func TestLineIndexPositionAt(t *testing.T) {
	li := NewLineIndex("ab\ncde\nf")
	cases := []struct {
		offset    int
		line, col int
	}{
		{0, 0, 0},
		{2, 0, 2}, // the \n itself sits at end of line 0
		{3, 1, 0},
		{6, 1, 3},
		{7, 2, 0},
	}
	for _, c := range cases {
		p := li.PositionAt(c.offset)
		if p.Line != c.line || p.Character != c.col {
			t.Errorf("PositionAt(%d) = %d,%d want %d,%d", c.offset, p.Line, p.Character, c.line, c.col)
		}
	}
}

func TestLineIndexOffsetAtRoundTrip(t *testing.T) {
	text := "ab\ncde\nf"
	li := NewLineIndex(text)
	for off := 0; off <= len(text); off++ {
		p := li.PositionAt(off)
		if got := li.OffsetAt(p.Line, p.Character); got != off {
			t.Errorf("round trip offset %d -> (%d,%d) -> %d", off, p.Line, p.Character, got)
		}
	}
}

func TestLineIndexUTF8ByteColumns(t *testing.T) {
	// Non-ASCII: "é" is two UTF-8 bytes. Column after it is byte column 3.
	li := NewLineIndex("a\u00e9b")
	if p := li.PositionAt(3); p.Line != 0 || p.Character != 3 {
		t.Fatalf("PositionAt(3) = %+v want line 0 col 3", p)
	}
}

func TestWalkStatementsVisitsNested(t *testing.T) {
	r := Parse("a = { b = { c = 1 } }\n")
	var keys []string
	WalkStatements(r.Root, func(st Statement) bool {
		if a, ok := st.(*Assignment); ok {
			keys = append(keys, a.Key.Text)
		}
		return true
	})
	// Expect a, b, c in DFS order.
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("walk keys = %v", keys)
	}
}

func TestNodeAtOffsetChain(t *testing.T) {
	src := "a = { b = 1 }\n"
	r := Parse(src)
	// Offset of `b` (index 6).
	path := NodeAtOffset(r.Root, 6)
	if len(path) != 2 {
		t.Fatalf("path len = %d want 2 (%v)", len(path), path)
	}
	outer, ok := path[0].(*Assignment)
	if !ok || outer.Key.Text != "a" {
		t.Fatalf("outer = %+v", path[0])
	}
	inner, ok := path[1].(*Assignment)
	if !ok || inner.Key.Text != "b" {
		t.Fatalf("inner = %+v", path[1])
	}
}
