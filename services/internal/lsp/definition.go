// definition.go and vanilla on-demand parse: overlay winner first, then a single
// vanilla file parse when the key exists only in the cache.

package lsp

import (
	"os"

	"paradox-modding-tools/services/internal/loc"
	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

// Definition returns go-to-definition locations for the word at pos.
func Definition(s *session.Session, path string, line, col int) []Location {
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
	if d := s.Resolve(word); d != nil {
		if d.Type == "saved_scope" {
			return nil
		}
		return []Location{defLocation(*d)}
	}
	if v := model.LookupVanillaDef(s.Cache(), word); v != nil {
		return []Location{defLocation(*v)}
	}
	if locDefined(s, word) {
		idx := s.Index()
		if idx != nil {
			for _, d := range idx.Defs {
				if d.Type == "loc_key" && d.Key == word {
					return []Location{defLocation(d)}
				}
			}
		}
		if c := s.Cache(); c != nil && c.LocEnglishSites != nil {
			if site, ok := c.LocEnglishSites[word]; ok {
				return []Location{defLocation(model.Def{
					Type: "loc_key", Key: word, Path: site.File, Line: site.Line,
				})}
			}
		}
	}
	return nil
}

func defLocation(d model.Def) Location {
	if d.Type == "loc_key" {
		if r, err := loc.ParseFile(d.Path); err == nil {
			for _, e := range r.Entries {
				if e.Key != d.Key {
					continue
				}
				raw, err := os.ReadFile(d.Path)
				if err != nil {
					break
				}
				text, _ := parser.Decode(raw)
				return Location{URI: d.Path, Range: byteRange(text, e.KeyRange.Start, e.KeyRange.End)}
			}
		}
	}
	line, start, end := model.PreciseDefRange(d)
	return Location{
		URI: d.Path,
		Range: Range{
			Start: Position{Line: line, Character: start},
			End:   Position{Line: line, Character: end},
		},
	}
}
