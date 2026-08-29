// hover.go resolves the word at a position to overlay def, kind-scoped field-doc
// prose, or cached wiki text. Comments and assignment keys never hit wiki.

package lsp

import (
	"path/filepath"
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

// Hover returns hover text at (line, UTF-8 column), or nil when there is none.
func Hover(s *session.Session, path string, line, col int) *HoverResult {
	src := fileText(s, path)
	if src == "" {
		return nil
	}
	off := offsetOf(src, line, col)
	res := parseOf(s, path, src)
	if inComment(res, off) {
		return nil
	}
	kind := enclosingKind(s, path, src, off)
	if name, start, end, ok := scopeRefAt(src, off); ok {
		return savedScopeHover(name, src, start, end)
	}
	if name, start, end, ok := saveScopeNameAt(res, off); ok {
		return savedScopeHover(name, src, start, end)
	}
	if a := assignmentKeyAt(res, off); a != nil {
		if p, ok := model.ParsePrefixed(a.Key.Text); ok && model.IsSavedScopePrefix(p) {
			ns := a.Key.Range.Start + p.NameOff
			return savedScopeHover(p.Name, src, ns, ns+len(p.Name))
		}
		rg := byteRange(src, a.Key.Range.Start, a.Key.Range.End)
		if h := namedHover(s, a.Key.Text, &rg); h != nil {
			return h
		}
		docs := docsFor(s, a.Key.Text, kind)
		if docs == "" {
			docs = s.Wiki(a.Key.Text)
		}
		if docs == "" && structureKeyDoc(s, kind, a.Key.Text) == "" {
			return nil
		}
		head := kindLabel(kind) + " key `" + a.Key.Text + "`"
		return &HoverResult{Contents: hoverCard(head, "", docs), Range: &rg}
	}
	word, start, end := wordAt(src, off)
	if word == "" {
		return nil
	}
	rg := byteRange(src, start, end)
	if h := namedHover(s, word, &rg); h != nil {
		return h
	}
	if extra := docsFor(s, word, ""); extra != "" {
		return &HoverResult{Contents: extra, Range: &rg}
	}
	if w := s.Wiki(word); w != "" {
		return &HoverResult{Contents: w, Range: &rg}
	}
	return nil
}

// namedHover is a loc/def/effect/trigger card for a token, or nil.
func namedHover(s *session.Session, word string, rg *Range) *HoverResult {
	if d := s.Resolve(word); d != nil {
		if d.Type == "loc_key" {
			return locHover(s, word, rg)
		}
		if d.Type == "saved_scope" {
			return &HoverResult{
				Contents: hoverCard(kindLabel("saved_scope")+" `scope:"+word+"`", "", ""),
				Range:    rg,
			}
		}
		head := kindLabel(d.Type) + " `" + d.Key + "`"
		docs := docsFor(s, word, d.Type)
		h := &HoverResult{Contents: hoverCard(head, "", docs), Range: rg}
		attachSite(h, s, d.Path, d.Line, d.Origin)
		return h
	}
	if locDefined(s, word) {
		return locHover(s, word, rg)
	}
	if ck := catalogKind(s, word); ck != "" {
		docs := docsFor(s, word, "")
		if docs == "" {
			docs = s.Wiki(word)
		}
		return &HoverResult{
			Contents: hoverCard(ck+" `"+word+"`", "", docs),
			Range:    rg,
		}
	}
	return nil
}

func locHover(s *session.Session, key string, rg *Range) *HoverResult {
	text, _ := s.EnglishLoc(key)
	file, line, origin := locSite(s, key)
	if text == "" && file == "" {
		return nil
	}
	h := &HoverResult{
		Contents: hoverCard("localization `"+key+"`", text, ""),
		Range:    rg,
	}
	if file != "" {
		attachSite(h, s, file, line, origin)
	}
	return h
}

func locSite(s *session.Session, key string) (file string, line int, origin string) {
	idx := s.Index()
	var defs []model.Def
	if idx != nil {
		for _, d := range idx.Defs {
			if d.Type == "loc_key" && d.Key == key {
				defs = append(defs, d)
			}
		}
	}
	order := map[string]int{}
	if idx != nil {
		for i, o := range idx.Order {
			order[o] = i
		}
	}
	if w := model.Winner(defs, order); w != nil {
		return w.Path, w.Line, w.Origin
	}
	if c := s.Cache(); c != nil && c.LocEnglishSites != nil {
		if site, ok := c.LocEnglishSites[key]; ok {
			return site.File, site.Line, "vanilla"
		}
	}
	return "", 0, ""
}

func hoverCard(head, locVal, docs string) string {
	var b strings.Builder
	b.WriteString(head)
	if locVal != "" {
		b.WriteString("\n\n\"")
		b.WriteString(locVal)
		b.WriteString("\"")
	}
	if docs != "" {
		b.WriteString("\n\n")
		b.WriteString(docs)
	}
	return b.String()
}

// attachSite fills origin/rel/rootPath for the editor's location line.
func attachSite(h *HoverResult, s *session.Session, path string, line int, origin string) {
	if h == nil || path == "" {
		return
	}
	h.Rel = displayRel(s, path)
	h.Line = line
	if origin == "" {
		origin = "vanilla"
	}
	h.Origin = origin
	if locOrigin, _, ok := s.Locate(path); ok {
		for _, m := range s.Mods() {
			if m.Origin == locOrigin {
				h.RootPath = m.Root
				break
			}
		}
	}
	if h.RootPath == "" {
		if c := s.Cache(); c != nil {
			h.RootPath = c.InstallPath
		}
	}
}

// displayRel is origin-relative, else install-relative, else the path with slashes.
func displayRel(s *session.Session, path string) string {
	if path == "" {
		return ""
	}
	if _, rel, ok := s.Locate(path); ok {
		return filepath.ToSlash(rel)
	}
	if c := s.Cache(); c != nil && c.InstallPath != "" {
		if rel, err := filepath.Rel(c.InstallPath, path); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(path)
}

// kindLabel is the hover type line for a definition or catalog kind.
func kindLabel(t string) string {
	switch t {
	case "loc_key":
		return "localization"
	case "scripted_triggers", "scripted_trigger":
		return "scripted trigger"
	case "scripted_effects", "scripted_effect":
		return "scripted effect"
	case "scripted_modifiers", "scripted_modifier":
		return "scripted modifier"
	case "gui_type":
		return "gui type"
	case "saved_scope":
		return "saved scope"
	default:
		return strings.ReplaceAll(t, "_", " ")
	}
}

// catalogKind is "effect" or "trigger" when word is in the vanilla dump lists.
func catalogKind(s *session.Session, word string) string {
	c := s.Cache()
	if c == nil {
		return ""
	}
	for _, k := range c.Effects {
		if k == word || strings.EqualFold(k, word) {
			return "effect"
		}
	}
	for _, k := range c.Triggers {
		if k == word || strings.EqualFold(k, word) {
			return "trigger"
		}
	}
	return ""
}

func savedScopeHover(name, src string, start, end int) *HoverResult {
	head := kindLabel("saved_scope") + " `scope:" + name + "`"
	rg := byteRange(src, start, end)
	return &HoverResult{Contents: hoverCard(head, "", ""), Range: &rg}
}

// saveScopeNameAt is the name on `save_scope_as = X` / `save_scope_value_as = { name = X }`.
func saveScopeNameAt(res parser.Result, off int) (name string, start, end int, ok bool) {
	if res.Root == nil {
		return "", 0, 0, false
	}
	chain := parser.NodeAtOffset(res.Root, off)
	for i := len(chain) - 1; i >= 0; i-- {
		a, isA := chain[i].(*parser.Assignment)
		if !isA || a.Key.Quoted {
			continue
		}
		if game.IsSaveScopeKey(a.Key.Text) {
			sc, isS := a.Value.(*parser.Scalar)
			if isS && !sc.Quoted && off >= sc.Range.Start && off <= sc.Range.End {
				return sc.Text, sc.Range.Start, sc.Range.End, true
			}
			return "", 0, 0, false
		}
		if game.IsSaveScopeValueKey(a.Key.Text) {
			b := assignmentBlock(a)
			if b == nil {
				continue
			}
			for _, st := range b.Statements {
				ca, isC := st.(*parser.Assignment)
				if !isC || ca.Key.Text != "name" {
					continue
				}
				sc, isS := ca.Value.(*parser.Scalar)
				if isS && !sc.Quoted && off >= sc.Range.Start && off <= sc.Range.End {
					return sc.Text, sc.Range.Start, sc.Range.End, true
				}
			}
		}
	}
	return "", 0, 0, false
}

// scopeRefAt reports a `scope:name` covering off (cursor on prefix or name).
func scopeRefAt(src string, off int) (name string, start, end int, ok bool) {
	word, wStart, wEnd := wordAt(src, off)
	if word == "" {
		return "", 0, 0, false
	}
	if wEnd < len(src) && src[wEnd] == ':' && word == game.ScopePrefix {
		n, ns, ne := wordAt(src, wEnd+1)
		if n == "" {
			return "", 0, 0, false
		}
		return n, ns, ne, true
	}
	if wStart > 0 && src[wStart-1] == ':' {
		pre, _, _ := wordAt(src, wStart-2)
		if pre == game.ScopePrefix {
			return word, wStart, wEnd, true
		}
	}
	p, pok := model.ParsePrefixed(word)
	if pok && model.IsSavedScopePrefix(p) {
		ns := wStart + p.NameOff
		return p.Name, ns, ns + len(p.Name), true
	}
	return "", 0, 0, false
}

func structureKeyDoc(s *session.Session, kind, key string) string {
	c := s.Cache()
	if c == nil {
		return ""
	}
	lk := strings.ToLower(key)
	for _, k := range c.Structures[kind] {
		if k == key || strings.EqualFold(k, lk) {
			return kind + " key"
		}
	}
	return ""
}

// docsFor returns kind-scoped field prose, then the global FieldDocs fallback.
func docsFor(s *session.Session, key, kind string) string {
	c := s.Cache()
	if c == nil {
		return ""
	}
	lk := strings.ToLower(key)
	if kind != "" && c.FieldDocsByKind != nil {
		if m := c.FieldDocsByKind[kind]; m != nil {
			if d := m[lk]; d != "" {
				return d
			}
			if d := m[key]; d != "" {
				return d
			}
		}
	}
	if c.FieldDocs == nil {
		return ""
	}
	if d := c.FieldDocs[key]; d != "" {
		return d
	}
	return c.FieldDocs[lk]
}

// assignmentKeyAt returns the assignment whose key span covers off, or nil.
func assignmentKeyAt(res parser.Result, off int) *parser.Assignment {
	if res.Root == nil {
		return nil
	}
	chain := parser.NodeAtOffset(res.Root, off)
	for i := len(chain) - 1; i >= 0; i-- {
		a, ok := chain[i].(*parser.Assignment)
		if !ok {
			continue
		}
		if off >= a.Key.Range.Start && off < a.Key.Range.End {
			return a
		}
		return nil
	}
	return nil
}
