// index.go is the single workspace extractor: it turns each mod file into defs,
// references, and event edges (via one CST walk) and assembles them into an Index.
// The same per-file routine (IndexFile) is used both for a full BuildIndex and for
// single-file reindexing by the session. Definition extraction is shared with the
// install scan (extractTopLevel/extractEvents/extractGUI) so there is one extractor.

package model

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/loc"
	"paradox-modding-tools/services/internal/parser"
)

// BuildIndex walks every mod in load order and indexes its files into one Index.
func BuildIndex(workspaceID, gameID string, mods []ModInput, cache *Cache) *Index {
	idx := &Index{
		FormatVersion: IndexFormatVersion,
		WorkspaceID:   workspaceID,
		BuiltAt:       time.Now().UTC().Format(time.RFC3339),
		Loc:           map[string]string{},
	}
	ordered := append([]ModInput{}, mods...)
	sortByOrder(ordered)
	for _, m := range ordered {
		idx.Order = append(idx.Order, m.Origin)
		for _, f := range modFiles(m.Root) {
			raw, err := os.ReadFile(f.abs)
			if err != nil {
				continue
			}
			text, _ := parser.Decode(raw)
			defs, refs, edges, locEng := IndexFile(gameID, f.abs, f.rel, text, m.Origin)
			idx.Defs = append(idx.Defs, defs...)
			idx.Refs = append(idx.Refs, refs...)
			idx.Edges = append(idx.Edges, edges...)
			for k, v := range locEng {
				idx.Loc[k] = v
			}
		}
	}
	return idx
}

// IndexFile extracts one file's defs, loc references, event edges, and (for english
// loc files) its key->value map. content is the buffer text so unsaved edits index.
func IndexFile(gameID, absPath, rel, content, origin string) (defs []Def, refs []Ref, edges []Edge, locEng map[string]string) {
	absPath = filepath.Clean(absPath)
	content = parser.Normalize(content)
	rule := game.MatchExtract(gameID, rel)
	if rule.Mode == game.ModeLocKey {
		return indexLoc(absPath, content, origin)
	}
	res := parser.Parse(content)
	li := res.Lines()
	defs = defsFor(res.Root, li, gameID, rule, absPath, origin)
	refs = extractScriptRefs(res.Root, li, absPath)
	edges = extractEdges(res.Root, li, absPath)
	return defs, refs, edges, nil
}

// indexLoc turns a localization file into loc_key defs and, for english, a value map.
func indexLoc(absPath, content, origin string) (defs []Def, refs []Ref, edges []Edge, locEng map[string]string) {
	r := loc.Parse(content)
	english := loc.LanguageOf(absPath, r.Language) == "english"
	for _, e := range r.Entries {
		defs = append(defs, Def{Type: "loc_key", Key: e.Key, Path: absPath, Line: e.Line, Origin: origin})
		if english {
			if locEng == nil {
				locEng = map[string]string{}
			}
			v := e.Value
			if len(v) > locValueLimit {
				v = v[:locValueLimit]
			}
			locEng[e.Key] = v
		}
	}
	return defs, nil, nil, locEng
}

// defsFor dispatches definition extraction by extract mode, stamping each def with
// its origin. It reuses the same extractors as the install scan.
func defsFor(root *parser.Root, li *parser.LineIndex, gameID string, rule game.ExtractRule, path, origin string) []Def {
	var defs []Def
	switch rule.Mode {
	case game.ModeTopLevelKey:
		defs = extractTopLevel(root, li, gameID, rule.Kind, path)
	case game.ModeEventID:
		defs = extractEvents(root, li, gameID, path)
	case game.ModeGUIType:
		defs = extractGUI(root, li, path).defs
	case game.ModeLocKey:
		return nil
	}
	for i := range defs {
		defs[i].Origin = origin
	}
	return defs
}

// extractScriptRefs walks the CST once for loc refs and event/on_action fire sites.
func extractScriptRefs(root *parser.Root, li *parser.LineIndex, path string) []Ref {
	var refs []Ref
	parser.WalkStatements(root, func(st parser.Statement) bool {
		a, ok := st.(*parser.Assignment)
		if !ok || a.Key.Quoted {
			return true
		}
		key := a.Key.Text

		prop := loc.Classify(key)
		if prop != loc.PropNone {
			if sc, ok := a.Value.(*parser.Scalar); ok && sc.Text != "" && loc.LooksLikeKey(sc.Text) {
				kind := "loc-broad"
				if prop == loc.PropStrict {
					kind = "loc"
				}
				refs = append(refs, Ref{
					Key: sc.Text, Kind: kind, Path: path,
					Line:  li.PositionAt(sc.Range.Start).Line,
					Start: sc.Range.Start, End: sc.Range.End,
				})
			}
		}

		if fk := game.FireKind(key); fk != "" {
			refs = appendFireRefs(refs, a.Value, fk, li, path)
		}

		return true
	})
	return refs
}

func appendFireRefs(
	refs []Ref, v parser.Value, kind string, li *parser.LineIndex, path string,
) []Ref {
	add := func(name string, start, end int) {
		if !fireTargetOK(name) {
			return
		}
		refs = append(refs, Ref{
			Key: name, Kind: kind, Path: path,
			Line: li.PositionAt(start).Line, Start: start, End: end,
		})
	}
	switch t := v.(type) {
	case *parser.Scalar:
		if !t.Quoted {
			add(t.Text, t.Range.Start, t.Range.End)
		}
	case *parser.Block, *parser.TaggedBlock:
		b := blockOf(v)
		if b == nil {
			return refs
		}
		for _, st := range b.Statements {
			switch n := st.(type) {
			case *parser.ValueStmt:
				if sc, ok := n.Value.(*parser.Scalar); ok && !sc.Quoted {
					add(sc.Text, sc.Range.Start, sc.Range.End)
				} else if inner := blockOf(n.Value); inner != nil {
					refs = appendFireRefs(refs, n.Value, kind, li, path)
				}
			case *parser.Assignment:
				key := strings.ToLower(n.Key.Text)
				own := game.FireKind(n.Key.Text)
				if sc, ok := n.Value.(*parser.Scalar); ok && !sc.Quoted &&
					(key == "id" || isDigits(key) || own != "") {
					add(sc.Text, sc.Range.Start, sc.Range.End)
					continue
				}
				if own != "" || key == "id" || isDigits(key) {
					refs = appendFireRefs(refs, n.Value, kind, li, path)
				}
			}
		}
	}
	return refs
}

func fireTargetOK(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	if c < 'A' || (c > 'Z' && c < 'a') || c > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c = s[i]
		ok := c == '_' || c == '.' || c == '-' ||
			(c >= '0' && c <= '9') ||
			(c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		if !ok {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := range s {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// extractEdges collects `trigger_event` links, tagging each with the enclosing
// top-level event id as the edge source.
func extractEdges(root *parser.Root, li *parser.LineIndex, path string) []Edge {
	var edges []Edge
	var cur string
	var walk func(stmts []parser.Statement, depth int)
	walk = func(stmts []parser.Statement, depth int) {
		for _, st := range stmts {
			if a, ok := st.(*parser.Assignment); ok {
				if depth == 0 && eventIDRe.MatchString(a.Key.Text) {
					cur = a.Key.Text
				}
				if a.Key.Text == "trigger_event" {
					if to := edgeTarget(a.Value); to != "" {
						edges = append(edges, Edge{From: cur, To: to, Via: "trigger_event", Path: path, Line: li.PositionAt(a.Key.Range.Start).Line})
					}
				}
				if b := blockOf(a.Value); b != nil {
					walk(b.Statements, depth+1)
				}
				continue
			}
			if vs, ok := st.(*parser.ValueStmt); ok {
				if b := blockOf(vs.Value); b != nil {
					walk(b.Statements, depth+1)
				}
			}
		}
	}
	walk(root.Statements, 0)
	return edges
}

// edgeTarget resolves a trigger_event value: a bare `evt.1` scalar or a block with
// an `id = evt.1` child.
func edgeTarget(v parser.Value) string {
	switch t := v.(type) {
	case *parser.Scalar:
		return t.Text
	case *parser.Block:
		for _, st := range t.Statements {
			if a, ok := st.(*parser.Assignment); ok && a.Key.Text == "id" {
				if sc, ok := a.Value.(*parser.Scalar); ok {
					return sc.Text
				}
			}
		}
	}
	return ""
}

// LookupAll returns every definition of key across the workspace (any origin) and
// then vanilla, so callers can pick a winner or list all sites.
func (idx *Index) LookupAll(key string, cache *Cache) []Def {
	var out []Def
	for _, d := range idx.Defs {
		if d.Key == key {
			out = append(out, d)
		}
	}
	if cache != nil {
		for _, d := range cache.Defs {
			if d.Key == key {
				out = append(out, d)
			}
		}
	}
	return out
}

// modFiles lists the indexable files under a mod root: script (.txt, non `_`),
// GUI (.gui), and localization YAML, each with its root-relative path.
func modFiles(root string) []fileRef {
	var out []fileRef
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		lower := strings.ToLower(name)
		rel, _ := filepath.Rel(root, p)
		p = filepath.Clean(p)
		switch {
		case strings.HasSuffix(lower, ".mod"):
			out = append(out, fileRef{p, rel})
		case strings.HasSuffix(lower, ".gui"):
			out = append(out, fileRef{p, rel})
		case strings.HasSuffix(lower, ".yml"), strings.HasSuffix(lower, ".yaml"):
			if strings.Contains(strings.ReplaceAll(rel, "\\", "/"), "localization") {
				out = append(out, fileRef{p, rel})
			}
		case strings.HasSuffix(lower, ".txt") && !strings.HasPrefix(name, "_"):
			out = append(out, fileRef{p, rel})
		}
		return nil
	})
	return out
}

// sortByOrder sorts mods ascending by load order (stable to preserve input order
// among equal Order values).
func sortByOrder(mods []ModInput) {
	slices.SortStableFunc(mods, func(a, b ModInput) int {
		return cmp.Compare(a.Order, b.Order)
	})
}

// LookupVanillaDef returns the first vanilla cache def for key, or nil.
func LookupVanillaDef(cache *Cache, key string) *Def {
	if cache == nil {
		return nil
	}
	for i := range cache.Defs {
		if cache.Defs[i].Key == key {
			return &cache.Defs[i]
		}
	}
	return nil
}

// PreciseDefRange re-parses def.Path to locate the key's UTF-8 column span.
// Falls back to the stored line if the file cannot be read.
func PreciseDefRange(d Def) (line, startCol, endCol int) {
	line = d.Line
	endCol = len(d.Key)
	res, err := parser.ParseFile(d.Path)
	if err != nil {
		return
	}
	for _, st := range res.Root.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok {
			continue
		}
		if a.Key.Text != d.Key && game.KeyIdentity("", a.Key.Text) != d.Key {
			continue
		}
		pos := res.Lines().PositionAt(a.Key.Range.Start)
		end := res.Lines().PositionAt(a.Key.Range.End)
		return pos.Line, pos.Character, end.Character
	}
	return
}
