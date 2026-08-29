// dependencies.go lists mod sites that reference a definition and the named
// definitions its own body points at. Vanilla callers are not indexed.

package graph

import (
	"cmp"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

// Deps returns dependents and inner dependencies of name (optional kind filter).
func Deps(s *session.Session, name, kind string) Dependencies {
	d := lookupDef(s, name)
	if d != nil && kind != "" && d.Type != kind &&
		graphKind(d.Type) != kind && !kindMatch(d.Type, kind) {
		d = nil
		if idx := s.Index(); idx != nil {
			for i := range idx.Defs {
				if idx.Defs[i].Key == name && kindMatch(idx.Defs[i].Type, kind) {
					dd := idx.Defs[i]
					d = &dd
					break
				}
			}
		}
	}
	if d == nil {
		return Dependencies{}
	}
	return Dependencies{
		Def:          &DependencyDef{Name: d.Key, Kind: d.Type, File: d.Path, Line: d.Line},
		Dependents:   collectDependents(s, *d),
		Dependencies: collectDependencies(s, *d),
	}
}

func kindMatch(got, want string) bool {
	return got == want || graphKind(got) == want ||
		strings.TrimSuffix(got, "s") == want ||
		got+"s" == want
}

type depSite struct {
	file string
	line int
}

func collectDependents(s *session.Session, def model.Def) []DependencyGroup {
	var sites []depSite
	idx := s.Index()
	if idx != nil {
		for _, r := range idx.Refs {
			if r.Key == def.Key {
				sites = append(sites, depSite{r.Path, r.Line})
			}
		}
		for _, e := range idx.Edges {
			if e.To == def.Key {
				sites = append(sites, depSite{e.Path, e.Line})
			}
		}
	}
	if isCallKind(def.Type) {
		sites = append(sites, scanCallSites(s, def)...)
	}
	groups := map[string]map[string]DependencyItem{}
	for _, site := range sites {
		container := containingDef(s, site.file, site.line)
		kind, item := "file", filepath.Base(site.file)
		if container != nil {
			kind, item = container.Type, container.Key
		}
		dedupe := item + " " + site.file
		byItem := groups[kind]
		if byItem == nil {
			byItem = map[string]DependencyItem{}
			groups[kind] = byItem
		}
		if _, ok := byItem[dedupe]; !ok {
			byItem[dedupe] = DependencyItem{Name: item, File: site.file, Line: site.line}
		}
	}
	return toGroups(groups)
}

func scanCallSites(s *session.Session, def model.Def) []depSite {
	idx := s.Index()
	if idx == nil {
		return nil
	}
	files := map[string]bool{}
	for _, d := range idx.Defs {
		if d.Origin != "" && strings.HasSuffix(strings.ToLower(d.Path), ".txt") {
			files[d.Path] = true
		}
	}
	var sites []depSite
	seen := map[string]bool{}
	for path := range files {
		src := s.FileText(path)
		if !strings.Contains(src, def.Key) {
			continue
		}
		res := parseOf(s, path)
		if res.Root == nil {
			continue
		}
		li := res.Lines()
		parser.WalkStatements(res.Root, func(st parser.Statement) bool {
			a, ok := st.(*parser.Assignment)
			if !ok || a.Key.Quoted || a.Key.Text != def.Key {
				return true
			}
			line := li.PositionAt(a.Key.Range.Start).Line
			if path == def.Path && line == def.Line {
				return true
			}
			key := path + " " + strconv.Itoa(line)
			if seen[key] {
				return true
			}
			seen[key] = true
			sites = append(sites, depSite{path, line})
			return true
		})
	}
	return sites
}

func containingDef(s *session.Session, file string, line int) *model.Def {
	idx := s.Index()
	if idx == nil {
		return nil
	}
	var best *model.Def
	for i := range idx.Defs {
		d := &idx.Defs[i]
		if d.Path != file {
			continue
		}
		if d.Line <= line && (best == nil || d.Line > best.Line) {
			best = d
		}
	}
	return best
}

func collectDependencies(s *session.Session, def model.Def) []DependencyGroup {
	res := parseOf(s, def.Path)
	if res.Root == nil {
		return nil
	}
	li := res.Lines()
	var block *parser.Block
	for _, st := range res.Root.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok || a.Key.Text != def.Key {
			continue
		}
		if li.PositionAt(a.Key.Range.Start).Line != def.Line {
			continue
		}
		block = blockOf(a.Value)
		break
	}
	if block == nil {
		return nil
	}
	groups := map[string]map[string]DependencyItem{}
	add := func(t model.Def) {
		if t.Key == def.Key {
			return
		}
		dedupe := t.Type + " " + t.Key
		byItem := groups[t.Type]
		if byItem == nil {
			byItem = map[string]DependencyItem{}
			groups[t.Type] = byItem
		}
		if _, ok := byItem[dedupe]; !ok {
			byItem[dedupe] = DependencyItem{Name: t.Key, File: t.Path, Line: t.Line}
		}
	}
	walkScalars(block, func(word string, isKey bool) {
		if word == def.Key {
			return
		}
		d := lookupDef(s, word)
		if d == nil {
			return
		}
		if isKey && isCallKind(d.Type) {
			add(*d)
		} else if !isKey {
			add(*d)
		}
	})
	return toGroups(groups)
}

func walkScalars(block *parser.Block, cb func(word string, isKey bool)) {
	for _, st := range block.Statements {
		if a, ok := st.(*parser.Assignment); ok {
			if !a.Key.Quoted {
				cb(a.Key.Text, true)
			}
			if sc, ok := a.Value.(*parser.Scalar); ok && !sc.Quoted {
				cb(sc.Text, false)
			}
			if sub := blockOf(a.Value); sub != nil {
				walkScalars(sub, cb)
			}
		} else if vs, ok := st.(*parser.ValueStmt); ok {
			if sc, ok := vs.Value.(*parser.Scalar); ok && !sc.Quoted {
				cb(sc.Text, false)
			} else if inner := blockOf(vs.Value); inner != nil {
				walkScalars(inner, cb)
			}
		}
	}
}

func toGroups(groups map[string]map[string]DependencyItem) []DependencyGroup {
	out := make([]DependencyGroup, 0, len(groups))
	for kind, byItem := range groups {
		items := make([]DependencyItem, 0, len(byItem))
		for _, it := range byItem {
			items = append(items, it)
		}
		slices.SortFunc(items, func(a, b DependencyItem) int { return cmp.Compare(a.Name, b.Name) })
		out = append(out, DependencyGroup{Kind: kind, Items: items})
	}
	slices.SortFunc(out, func(a, b DependencyGroup) int { return cmp.Compare(a.Kind, b.Kind) })
	return out
}
