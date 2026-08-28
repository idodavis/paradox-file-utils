// folding.go emits a fold range for every CST block that spans more than one line.

package lsp

import (
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

// FoldingRanges returns foldable block ranges for path.
func FoldingRanges(s *session.Session, path string) []FoldingRange {
	src := fileText(s, path)
	if src == "" || isLocPath(path) || isMetaJSON(path) {
		return nil
	}
	res := parseOf(s, path, src)
	li := res.Lines()
	var out []FoldingRange
	parser.WalkStatements(res.Root, func(st parser.Statement) bool {
		var b *parser.Block
		switch n := st.(type) {
		case *parser.Assignment:
			switch v := n.Value.(type) {
			case *parser.Block:
				b = v
			case *parser.TaggedBlock:
				b = &v.Block
			}
		case *parser.ValueStmt:
			if blk, ok := n.Value.(*parser.Block); ok {
				b = blk
			}
		}
		if b == nil || b.CloseBrace < 0 {
			return true
		}
		start := li.PositionAt(b.OpenBrace).Line
		end := li.PositionAt(b.CloseBrace).Line
		if end > start {
			out = append(out, FoldingRange{StartLine: start, EndLine: end})
		}
		return true
	})
	return out
}
