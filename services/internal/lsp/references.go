// references.go lists every indexed use and definition of the word at pos.

package lsp

import "paradox-modding-tools/services/internal/session"

// References returns every workspace site of the identifier at pos.
func References(s *session.Session, path string, line, col int) []Location {
	src := fileText(s, path)
	word, _, _ := wordAt(src, offsetOf(src, line, col))
	if word == "" {
		return nil
	}
	idx := s.Index()
	if idx == nil {
		return nil
	}
	var out []Location
	for _, d := range idx.Defs {
		if d.Key == word {
			out = append(out, defLocation(d))
		}
	}
	for _, r := range idx.Refs {
		if r.Key != word {
			continue
		}
		text := fileText(s, r.Path)
		if text == "" {
			continue
		}
		out = append(out, Location{URI: r.Path, Range: byteRange(text, r.Start, r.End)})
	}
	return out
}
