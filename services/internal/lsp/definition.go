// definition.go resolves definition, references, rename, and workspace symbols.
package lsp

import (
	"strconv"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"
	"paradox-modding-tools/services/internal/session"
)

// Definition returns go-to-definition locations for the word at pos.
func Definition(s *session.Session, path string, line, col int) []Location {
	sub, ok := subjectAt(s, path, line, col)
	if !ok || sub.kind == "loc_value" || sub.fieldKey {
		return nil
	}
	if sub.kind == "loc_key" && sub.def == nil {
		if sub.spanEnd > sub.spanStart {
			return []Location{{
				URI:   path,
				Range: byteRange(jomini.NewLineIndex(s.FileText(path)), sub.spanStart, sub.spanEnd),
			}}
		}
		return nil
	}
	if sub.local {
		a := sub.at.assign
		if a == nil {
			return nil
		}
		ln := sub.at.res.Lines().PositionAt(a.Key.Range.Start).Line
		return []Location{defLocation(s, catalog.Def{
			Kind: sub.kind, Key: a.Key.Text, Path: path, Line: ln,
			Start: a.Key.Range.Start, End: a.Key.Range.End,
		})}
	}
	if sub.def != nil {
		return definitionSites(s, sub.name, sub.def)
	}
	return nil
}

// References returns every workspace site of the identifier at pos.
func References(s *session.Session, path string, line, col int) []Location {
	sub, ok := subjectAt(s, path, line, col)
	if !ok || sub.kind == "loc_value" || sub.name == "" {
		return nil
	}
	if sub.kind == "script_param" {
		return identReferencesOwned(s, sub.name, "script_param", sub.owner)
	}
	if game.IsEphemeral(sub.kind) {
		return identReferences(s, sub.name, sub.kind)
	}
	return identReferences(s, sub.name, "")
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
				if game.IsLocEngineValue(ip.Key, ip.Filter) {
					return "", 0, 0
				}
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

type locInterp struct {
	Key, Filter string
}

func locInterpAt(src string, off int) (locInterp, bool) {
	res := loc.Parse(src)
	for _, e := range res.Entries {
		for _, ip := range loc.Interps(e.Value, e.ValueRange.Start) {
			if off >= ip.WrapRange.Start && off < ip.WrapRange.End {
				return locInterp{Key: ip.Key, Filter: ip.Filter}, true
			}
		}
	}
	return locInterp{}, false
}

// callKindDefAt reports scripted_trigger NAME = (and effect/modifier) at off.
func callKindDefAt(res jomini.Result, off int) (kind, name string, assign *jomini.Assignment, ok bool) {
	if res.Root == nil {
		return "", "", nil, false
	}
	var walk func([]jomini.Statement) bool
	walk = func(stmts []jomini.Statement) bool {
		var marker string
		var markStart, markEnd int
		for _, st := range stmts {
			if vs, ok := st.(*jomini.ValueStmt); ok {
				if sc, ok := vs.Value.(*jomini.Scalar); ok && !sc.Quoted &&
					game.IsCallKind(sc.Text) {
					marker, markStart, markEnd = sc.Text, sc.Range.Start, sc.Range.End
				} else {
					marker = ""
				}
				if b := jomini.BlockOf(vs.Value); b != nil && walk(b.Statements) {
					return true
				}
				continue
			}
			a, isA := st.(*jomini.Assignment)
			if !isA {
				marker = ""
				continue
			}
			if marker != "" && !a.Key.Quoted {
				onMark := off >= markStart && off < markEnd
				onName := off >= a.Key.Range.Start && off < a.Key.Range.End
				if onMark || onName {
					kind, name, assign, ok = marker, a.Key.Text, a, true
					return true
				}
			}
			marker = ""
			if b := jomini.BlockOf(a.Value); b != nil && walk(b.Statements) {
				return true
			}
		}
		return false
	}
	ok = walk(res.Root.Statements)
	return kind, name, assign, ok
}

// definitionSites lists every def of key, FIOS/Resolve winner first.
func definitionSites(s *session.Session, key string, winner *catalog.Def) []Location {
	seen := map[string]bool{}
	var out []Location
	add := func(d catalog.Def) {
		if d.Key != key {
			return
		}
		if winner != nil && game.IsEphemeral(winner.Kind) &&
			game.CanonicalKind(d.Kind) != game.CanonicalKind(winner.Kind) {
			return
		}
		if winner != nil && winner.OwnerKey != "" && d.OwnerKey != "" &&
			d.OwnerKey != winner.OwnerKey {
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
	for _, d := range s.VanillaDefs(key) {
		if winner != nil && d.Kind != winner.Kind {
			continue
		}
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
	loc := Location{URI: d.Path}
	if d.End > d.Start {
		if src := s.FileText(d.Path); src != "" {
			li := jomini.NewLineIndex(src)
			loc.Range = byteRange(li, d.Start, d.End)
			if b := assignmentBlock(s, d); b != nil {
				tr := byteRange(li, d.Start, b.Range.End)
				loc.TargetRange = &tr
			}
			return loc
		}
	}
	loc.Range = Range{
		Start: Position{Line: d.Line, Character: 0},
		End:   Position{Line: d.Line, Character: len(d.Key)},
	}
	return loc
}

func assignmentBlock(s *session.Session, d catalog.Def) *jomini.Block {
	res := s.Parsed(d.Path)
	if res.Root == nil {
		return nil
	}
	if d.Start > 0 {
		chain := jomini.NodeAtOffset(res.Root, d.Start)
		for i := len(chain) - 1; i >= 0; i-- {
			a, ok := chain[i].(*jomini.Assignment)
			if !ok || a.Key.Quoted || a.Key.Text != d.Key {
				continue
			}
			if b := jomini.BlockOf(a.Value); b != nil {
				return b
			}
		}
	}
	return jomini.BlockForKey(res.Root, d.Key)
}

// identReferences lists defs and refs of word. kind "" keeps all kinds;
// a non-empty kind filters to CanonicalKind(kind) (ephemeral F12/rename).
func identReferences(s *session.Session, word, kind string) []Location {
	return identReferencesOwned(s, word, kind, "")
}

func identReferencesOwned(s *session.Session, word, kind, owner string) []Location {
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
	kindOK := func(k string) bool {
		return kind == "" || game.CanonicalKind(k) == game.CanonicalKind(kind)
	}
	ownerOK := func(okey string) bool {
		return owner == "" || okey == "" || okey == owner
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
		if !kindOK(d.Kind) || !ownerOK(d.OwnerKey) {
			continue
		}
		add(defLocation(s, d))
		if d.Kind == "loc_key" {
			hasLocDef = true
		}
	}
	for _, d := range s.VanillaDefs(word) {
		if !kindOK(d.Kind) || !ownerOK(d.OwnerKey) {
			continue
		}
		add(defLocation(s, d))
		if d.Kind == "loc_key" {
			hasLocDef = true
		}
	}
	if kind == "" && !hasLocDef {
		if file, ln, _, ok := s.LocSite(word); ok {
			add(defLocation(s, catalog.Def{
				Kind: "loc_key", Key: word, Path: file, Line: ln,
			}))
		}
	}
	for _, r := range s.RefsTo(word) {
		if r.Key != word || !kindOK(r.Kind) || !ownerOK(r.OwnerKey) {
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
		if game.IsEphemeral(d.Kind) {
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
			Name: d.Key, Location: defLocation(s, d), ContainerName: d.Kind,
		})
	}
	return out
}
