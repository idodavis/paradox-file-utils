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
	chain := parser.NodeAtOffset(res.Root, offsetOf(src, line, col))
	for i := len(chain) - 1; i >= 0; i-- {
		a, ok := chain[i].(*parser.Assignment)
		if !ok {
			continue
		}
		key := a.Key.Text
		doc := docsFor(s, key)
		label := key
		if doc != "" {
			return &SignatureHelp{Label: label, Documentation: doc}
		}
		if d := s.Resolve(key); d != nil {
			return &SignatureHelp{Label: d.Type + " " + d.Key}
		}
		return &SignatureHelp{Label: key}
	}
	word, _, _ := wordAt(src, offsetOf(src, line, col))
	if word == "" {
		return nil
	}
	if doc := docsFor(s, word); doc != "" {
		return &SignatureHelp{Label: word, Documentation: doc}
	}
	return nil
}
