// definition.go resolves definition, references, rename, and workspace symbols.
package lsp

import (
	"strconv"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"
	"paradox-modding-tools/services/internal/session"
)

// Definition returns go-to-definition locations for the word at pos.
func Definition(s *session.Session, path string, line, col int) []Location {
	if s.KindFor(path) == "loc" {
		src := s.FileText(path)
		if src == "" {
			return nil
		}
		off := jomini.NewLineIndex(src).OffsetAt(line, col)
		key, start, end := locKeySpan(src, off)
		if key == "" {
			return nil
		}
		if file, ln, _, ok := s.LocSite(key); ok {
			return []Location{defLocation(s, catalog.Def{
				Type: "loc_key", Key: key, Path: file, Line: ln,
			})}
		}
		if end > start {
			return []Location{{
				URI:   path,
				Range: byteRange(jomini.NewLineIndex(src), start, end),
			}}
		}
		return nil
	}
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
		return definitionSites(s, at.word, d)
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
	if s.KindFor(path) == "loc" {
		src := s.FileText(path)
		if src == "" {
			return nil
		}
		off := jomini.NewLineIndex(src).OffsetAt(line, col)
		key := locKeyAt(src, off)
		if key == "" {
			return nil
		}
		return identReferences(s, key)
	}
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

// locKeyAt returns the loc key covering off (entry key or `$key$` in a value).
// A cursor after the last key character still resolves.
func locKeyAt(src string, off int) string {
	key, _, _ := locKeySpan(src, off)
	return key
}

// locKeySpan is locKeyAt plus the key's byte span in src.
func locKeySpan(src string, off int) (key string, start, end int) {
	res := loc.Parse(src)
	for _, e := range res.Entries {
		start, end = e.KeyRange.Start, e.KeyRange.End
		if off >= start && off <= end {
			return e.Key, start, end
		}
		for _, ip := range loc.Interps(e.Value, e.ValueRange.Start) {
			if off >= ip.WrapRange.Start && off < ip.WrapRange.End {
				return ip.Key, ip.KeyRange.Start, ip.KeyRange.End
			}
		}
	}
	if len(res.Entries) > 0 {
		return "", 0, 0
	}
	word, wStart, wEnd := jomini.Result{Src: src}.TokenAt(off)
	return word, wStart, wEnd
}

// definitionSites lists every def of key, FIOS/Resolve winner first.
func definitionSites(s *session.Session, key string, winner *catalog.Def) []Location {
	seen := map[string]bool{}
	var out []Location
	add := func(d catalog.Def) {
		if d.Key != key || d.Type == "saved_scope" {
			return
		}
		loc := defLocation(s, d)
		id := loc.URI + ":" + strconv.Itoa(loc.Range.Start.Line) +
			":" + strconv.Itoa(loc.Range.Start.Character)
		if seen[id] {
			return
		}
		seen[id] = true
		out = append(out, loc)
	}
	if winner != nil {
		add(*winner)
	}
	for _, d := range s.ModDefsOf(key) {
		add(d)
	}
	for _, d := range s.FindDefs(key, 64, false, true) {
		if d.Key == key {
			add(d)
		}
	}
	return out
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
	hasLocDef := false
	for _, d := range s.ModDefsOf(word) {
		add(defLocation(s, d))
		if d.Type == "loc_key" {
			hasLocDef = true
		}
	}
	if !hasLocDef {
		if file, ln, _, ok := s.LocSite(word); ok {
			add(defLocation(s, catalog.Def{
				Type: "loc_key", Key: word, Path: file, Line: ln,
			}))
		}
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
