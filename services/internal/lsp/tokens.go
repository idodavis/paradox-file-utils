// tokens.go walks the current CST and emits semantic highlight spans.

package lsp

import (
	"unicode"

	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

var tokenKeywords = map[string]bool{
	"if": true, "else": true, "else_if": true, "limit": true,
	"AND": true, "OR": true, "NOT": true, "NOR": true, "NAND": true,
	"yes": true, "no": true, "type": true, "template": true,
	"scripted_effect": true, "scripted_trigger": true,
}

// SemanticTokens returns highlight spans for path.
func SemanticTokens(s *session.Session, path string) []SemanticSpan {
	src := fileText(s, path)
	if src == "" {
		return nil
	}
	if isLocPath(path) {
		return locTokens(src)
	}
	if isMetaJSON(path) {
		return nil
	}
	res := parseOf(s, path, src)
	li := res.Lines()
	var out []SemanticSpan
	push := func(start, end int, typ string) {
		if end <= start {
			return
		}
		p := li.PositionAt(start)
		out = append(out, SemanticSpan{Line: p.Line, StartCol: p.Character, Length: end - start, Type: typ})
	}
	for _, c := range res.Comments {
		push(c.Range.Start, c.Range.End, "comment")
	}
	parser.WalkStatements(res.Root, func(st parser.Statement) bool {
		switch n := st.(type) {
		case *parser.Assignment:
			push(n.Key.Range.Start, n.Key.Range.End, "property")
			tokenValue(n.Value, push)
		case *parser.ValueStmt:
			tokenValue(n.Value, push)
		}
		return true
	})
	return out
}

func tokenValue(v parser.Value, push func(int, int, string)) {
	switch n := v.(type) {
	case *parser.Scalar:
		typ := "variable"
		if n.Quoted {
			typ = "string"
		} else if isNumber(n.Text) {
			typ = "number"
		} else if tokenKeywords[n.Text] {
			typ = "keyword"
		}
		push(n.Range.Start, n.Range.End, typ)
	case *parser.TaggedBlock:
		push(n.Tag.Range.Start, n.Tag.Range.End, "type")
	}
}

func locTokens(src string) []SemanticSpan {
	r := parseLoc(src)
	li := parser.NewLineIndex(src)
	var out []SemanticSpan
	push := func(start, end int, typ string) {
		if end <= start {
			return
		}
		p := li.PositionAt(start)
		out = append(out, SemanticSpan{Line: p.Line, StartCol: p.Character, Length: end - start, Type: typ})
	}
	for _, e := range r.Entries {
		push(e.KeyRange.Start, e.KeyRange.End, "property")
		push(e.ValueRange.Start, e.ValueRange.End, "string")
	}
	return out
}

func isNumber(s string) bool {
	if s == "" {
		return false
	}
	dots := 0
	for i, r := range s {
		if r == '-' && i == 0 {
			continue
		}
		if r == '.' {
			dots++
			if dots > 1 {
				return false
			}
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
