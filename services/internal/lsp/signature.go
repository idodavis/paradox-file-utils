// signature.go shows field-doc prose for the assignment key enclosing the cursor.

package lsp

import (
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

// SignatureAt returns a signature tooltip at pos, or nil.
func SignatureAt(s *session.Session, path string, line, col int) *SignatureHelp {
	src := fileText(s, path)
	if src == "" || isLocPath(path) {
		return nil
	}
	res := parseOf(s, path, src)
	off := offsetOf(src, line, col)
	if inComment(res, off) || res.Root == nil {
		return nil
	}
	kind := enclosingKind(s, path, src, off)
	chain := parser.NodeAtOffset(res.Root, off)
	for i := len(chain) - 1; i >= 0; i-- {
		a, ok := chain[i].(*parser.Assignment)
		if !ok {
			continue
		}
		key := a.Key.Text
		doc := docsFor(s, key, kind)
		if doc != "" {
			return &SignatureHelp{Label: key, Documentation: doc}
		}
		if d := s.Resolve(key); d != nil {
			return &SignatureHelp{Label: d.Type + " " + d.Key}
		}
		return &SignatureHelp{Label: key}
	}
	word, _, _ := wordAt(src, off)
	if word == "" {
		return nil
	}
	if doc := docsFor(s, word, kind); doc != "" {
		return &SignatureHelp{Label: word, Documentation: doc}
	}
	return nil
}
