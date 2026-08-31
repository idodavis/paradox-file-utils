// definition.go resolves definition, references, rename, and workspace symbols.
package lsp

import (
	"strconv"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/session"
)

// Definition returns go-to-definition locations for the word at pos.
func Definition(s *session.Session, path string, line, col int) []Location {
	at, ok := resolveAt(s, path, line, col)
	if !ok {
		return nil
	}
	if _, ok := scopeRefAt(at.src, at.off); ok {
		return nil
	}
	if a := at.assign; a != nil {
		if d := resolveNonLoc(s, a.Key.Text); d != nil {
			return []Location{defLocation(s, *d)}
		}
		return nil
	}
	if at.word == "" {
		return nil
	}
	if d := s.Resolve(at.word); d != nil {
		if d.Type == "saved_scope" {
			return nil
		}
		return []Location{defLocation(s, *d)}
	}
	if locDefined(s, at.word) {
		if file, line, _, ok := s.LocSite(at.word); ok {
			return []Location{defLocation(s, catalog.Def{
				Type: "loc_key", Key: at.word, Path: file, Line: line,
			})}
		}
	}
	return nil
}

// References returns every workspace site of the identifier at pos.
func References(s *session.Session, path string, line, col int) []Location {
	at, ok := resolveAt(s, path, line, col)
	if !ok {
		return nil
	}
	if _, ok := scopeRefAt(at.src, at.off); ok {
		return nil
	}
	if at.word == "" {
		return nil
	}
	return identReferences(s, at.word)
}

func defLocation(s *session.Session, d catalog.Def) Location {
	if d.End > d.Start {
		if src := s.FileText(d.Path); src != "" {
			return Location{URI: d.Path, Range: byteRange(jomini.NewLineIndex(src), d.Start, d.End)}
		}
	}
	return Location{
		URI: d.Path,
		Range: Range{
			Start: Position{Line: d.Line, Character: 0},
			End:   Position{Line: d.Line, Character: len(d.Key)},
		},
	}
}

func identReferences(s *session.Session, word string) []Location {
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
	lis := map[string]*jomini.LineIndex{}
	lineIndex := func(p string) *jomini.LineIndex {
		if li, ok := lis[p]; ok {
			return li
		}
		src := s.FileText(p)
		if r, ok := s.Result(p); ok && r.Src == src {
			lis[p] = r.Lines()
			return lis[p]
		}
		lis[p] = jomini.NewLineIndex(src)
		return lis[p]
	}
	for _, d := range s.ModDefsOf(word) {
		add(defLocation(s, d))
	}
	for _, r := range s.RefsTo(word) {
		if r.Key != word || r.Kind == "saved_scope" {
			continue
		}
		add(Location{URI: r.Path, Range: byteRange(lineIndex(r.Path), r.Start, r.End)})
	}
	return out
}

// Rename returns edits to rename the identifier at pos, or nil if none.
func Rename(s *session.Session, path string, line, col int, newName string) *WorkspaceEdit {
	if newName == "" {
		return nil
	}
	locs := References(s, path, line, col)
	if len(locs) == 0 {
		return nil
	}
	changes := map[string][]TextEdit{}
	for _, loc := range locs {
		if _, _, ok := s.Locate(loc.URI); !ok {
			continue
		}
		changes[loc.URI] = append(changes[loc.URI], TextEdit{Range: loc.Range, NewText: newName})
	}
	if len(changes) == 0 {
		return nil
	}
	return &WorkspaceEdit{Changes: changes}
}

const maxWorkspaceSymbols = 200

// DocumentSymbols returns definitions declared in path.
func DocumentSymbols(s *session.Session, path string) []SymbolInformation {
	var out []SymbolInformation
	for _, d := range s.DefsInFile(path) {
		if d.Type == "saved_scope" {
			continue
		}
		out = append(out, SymbolInformation{Name: d.Key, Location: defLocation(s, d)})
	}
	return out
}

// WorkspaceSymbols searches definitions whose keys contain query.
func WorkspaceSymbols(s *session.Session, query string) []SymbolInformation {
	var out []SymbolInformation
	for _, d := range s.FindDefs(query, maxWorkspaceSymbols, false, false) {
		out = append(out, SymbolInformation{
			Name: d.Key, Location: defLocation(s, d), ContainerName: d.Type,
		})
	}
	return out
}
