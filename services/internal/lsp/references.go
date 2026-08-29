// references.go lists every indexed use and definition of the word at pos.

package lsp

import (
	"strconv"

	"paradox-modding-tools/services/internal/session"
)

// References returns every workspace site of the identifier at pos.
func References(s *session.Session, path string, line, col int) []Location {
	src := fileText(s, path)
	off := offsetOf(src, line, col)
	if inComment(parseOf(s, path, src), off) {
		return nil
	}
	if _, _, _, ok := scopeRefAt(src, off); ok {
		return nil
	}
	word, _, _ := wordAt(src, off)
	if word == "" {
		return nil
	}
	return identReferences(s, word)
}

func identReferences(s *session.Session, word string) []Location {
	idx := s.Index()
	if idx == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []Location
	add := func(loc Location) {
		k := session.CanonPath(loc.URI) + ":" +
			strconv.Itoa(loc.Range.Start.Line) + ":" +
			strconv.Itoa(loc.Range.Start.Character) + ":" +
			strconv.Itoa(loc.Range.End.Line) + ":" +
			strconv.Itoa(loc.Range.End.Character)
		if seen[k] {
			return
		}
		seen[k] = true
		out = append(out, loc)
	}
	for _, d := range idx.Defs {
		if d.Key != word || d.Type == "saved_scope" {
			continue
		}
		add(defLocation(d))
	}
	for _, r := range idx.Refs {
		if r.Key != word || r.Kind == "saved_scope" {
			continue
		}
		text := fileText(s, r.Path)
		if text == "" {
			continue
		}
		add(Location{URI: r.Path, Range: byteRange(text, r.Start, r.End)})
	}
	return out
}
