// hover.go builds structured hover cards from the cursor subject.
package lsp

import (
	"strings"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/session"
)

// Hover returns the card at (line, UTF-8 column), or nil when there is none.
func Hover(s *session.Session, path string, line, col int) *HoverResult {
	sub, ok := subjectAt(s, path, line, col)
	if !ok {
		return nil
	}
	return hoverFrom(s, sub)
}

func hoverFrom(s *session.Session, sub subject) *HoverResult {
	switch {
	case sub.kind == "loc_value":
		h := card("loc value", "$"+sub.name+"$", "", "")
		h.Body = "format " + sub.locFilter
		return h
	case sub.kind == "script_param":
		return scriptParamHover(s, sub.name, sub.def)
	case game.IsEphemeral(sub.kind):
		return ephemeralHover(s, sub.name, sub.kind, sub.def)
	case sub.local:
		return localAssignKeyHover(s, sub.path, sub.at)
	case sub.fieldKey:
		docs := s.FieldDoc(sub.name, sub.at.kind)
		return vanillaSite(s, card(kindLabel(sub.at.kind)+" key", sub.name, docs, ""))
	case sub.def != nil && sub.def.Kind == "loc_key":
		return locHover(s, sub.name)
	case sub.def != nil:
		return defCard(s, sub.name, sub.def)
	case sub.kind != "":
		h := card(kindLabel(sub.kind), sub.name, docsOnly(s, sub.name, sub.kind), "")
		h.Usage = s.TokenUsage(sub.name)
		return vanillaSite(s, h)
	default:
		return nil
	}
}

func card(kind, key, docs, hint string) *HoverResult {
	if hint == "" {
		hint = kindHint(kind)
	}
	return &HoverResult{Kind: kind, Key: key, Hint: hint, Docs: docs}
}

func defCard(s *session.Session, word string, d *catalog.Def) *HoverResult {
	h := card(kindLabel(d.Kind), d.Key, docsOnly(s, word, d.Kind), "")
	h.Usage = s.TokenUsage(word)
	h.Body = conventionLocBody(s, d)
	attachSite(h, s, d.Path, d.Line, d.Origin, d.Start)
	attachOverlay(h, s, d)
	return h
}

// scriptParamHover is the card for a $NAME$ macro parameter.
func scriptParamHover(s *session.Session, name string, d *catalog.Def) *HoverResult {
	hint := kindHint("script_param")
	if d != nil && d.OwnerKey != "" {
		hint += " of " + d.OwnerKey
	}
	h := card("script parameter", "$"+name+"$", "", hint)
	h.Body = scriptParamBody(s, name, d)
	if d != nil {
		attachSite(h, s, d.Path, d.Line, d.Origin, d.Start)
	}
	return h
}

func scriptParamBody(s *session.Session, name string, d *catalog.Def) string {
	owner := ""
	if d != nil {
		owner = d.OwnerKey
	}
	for _, r := range s.RefsTo(name) {
		if r.Kind != "script_param" || r.Path == "" {
			continue
		}
		if owner != "" && r.OwnerKey != "" && r.OwnerKey != owner {
			continue
		}
		res := s.Parsed(r.Path)
		if res.Root == nil {
			continue
		}
		chain := jomini.NodeAtOffset(res.Root, r.Start)
		for i := len(chain) - 1; i >= 0; i-- {
			a, ok := chain[i].(*jomini.Assignment)
			if !ok || a.Key.Quoted || a.Key.Text != name {
				continue
			}
			if sc, ok := a.Value.(*jomini.Scalar); ok && !sc.Quoted {
				return sc.Text
			}
		}
	}
	return ""
}

func conventionLocBody(s *session.Session, d *catalog.Def) string {
	if d == nil {
		return ""
	}
	for _, key := range game.RequiredLocKeys(d.Kind, d.Key) {
		if text, ok := s.DefaultLoc(key); ok && text != "" {
			return unescapeLocDisplay(text)
		}
	}
	if game.CanonicalKind(d.Kind) == "message" {
		return messageTitleLoc(s, d)
	}
	return ""
}

func messageTitleLoc(s *session.Session, d *catalog.Def) string {
	res := s.Parsed(d.Path)
	if res.Root == nil {
		return ""
	}
	body := jomini.BlockForKey(res.Root, d.Key)
	if body == nil {
		return ""
	}
	for _, st := range body.Statements {
		a, ok := st.(*jomini.Assignment)
		if !ok || a.Key.Quoted || a.Key.Text != "title" {
			continue
		}
		if sc, ok := a.Value.(*jomini.Scalar); ok && !sc.Quoted {
			if text, ok := s.DefaultLoc(sc.Text); ok {
				return unescapeLocDisplay(text)
			}
		}
	}
	return ""
}

func ephemeralHover(s *session.Session, name, kind string, d *catalog.Def) *HoverResult {
	if name == "" || kind == "" {
		return nil
	}
	key := name
	if kind == "saved_scope" {
		key = "scope:" + name
	}
	h := card(kindLabel(kind), key, "", "")
	if d != nil {
		h.Body = ephemeralBody(s, d)
		attachSite(h, s, d.Path, d.Line, d.Origin, d.Start)
	}
	return h
}

func ephemeralBody(s *session.Session, d *catalog.Def) string {
	if d.Value != "" {
		return d.Value
	}
	if game.CanonicalKind(d.Kind) == "saved_scope" {
		return scopeTargetExpr(s, d)
	}
	return ""
}

// scopeTargetExpr is the current-scope expression at the save site (else "root").
func scopeTargetExpr(s *session.Session, d *catalog.Def) string {
	res := s.Parsed(d.Path)
	if res.Root == nil || d.Start <= 0 {
		return "root"
	}
	chain := jomini.NodeAtOffset(res.Root, d.Start)
	for i := len(chain) - 1; i >= 0; i-- {
		a, ok := chain[i].(*jomini.Assignment)
		if !ok || a.Key.Quoted {
			continue
		}
		k := a.Key.Text
		if game.IsSaveScopeKey(k) || game.IsSaveScopeValueKey(k) {
			continue
		}
		low := strings.ToLower(k)
		switch low {
		case "immediate", "option", "after", "effect", "limit", "if",
			"else", "else_if", "trigger", "potential", "and", "or", "not",
			"nor", "nand", "root":
			if low == "root" {
				return "root"
			}
			continue
		}
		if strings.Contains(k, ":") || game.ScriptSlot(k) != "" {
			return k
		}
	}
	return "root"
}

func docsOnly(s *session.Session, word, kind string) string {
	doc := s.FieldDoc(word, kind)
	if sc := s.TokenScopes(word); sc != "" {
		if doc != "" {
			doc += "\n\n"
		}
		doc += "Scope here: " + sc
	}
	return doc
}

func locHover(s *session.Session, key string) *HoverResult {
	text, _ := s.DefaultLoc(key)
	file, line, origin, ok := s.LocSite(key)
	if text == "" && !ok {
		return nil
	}
	h := card("localization", key, "", "")
	h.Body = unescapeLocDisplay(text)
	if ok {
		attachSite(h, s, file, line, origin, 0)
		attachOverlay(h, s, &catalog.Def{
			Kind: "loc_key", Key: key, Path: file, Line: line, Origin: origin,
		})
	}
	return h
}

func unescapeLocDisplay(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		switch s[i+1] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case '\\':
			b.WriteByte('\\')
		case '"':
			b.WriteByte('"')
		default:
			b.WriteByte(s[i])
			continue
		}
		i++
	}
	return strings.TrimSpace(b.String())
}

func localAssignKeyHover(s *session.Session, path string, at atPos) *HoverResult {
	a := at.assign
	if a == nil {
		return nil
	}
	docs := s.FieldDoc(a.Key.Text, at.kind)
	m := memberSetsFor(s, at.kind)
	if docs == "" && !m.structKeys[a.Key.Text] && !m.structKeys[strings.ToLower(a.Key.Text)] {
		return nil
	}
	h := card(kindLabel(at.kind)+" key", a.Key.Text, docs, "")
	origin, _, _ := s.Locate(path)
	line := at.res.Lines().PositionAt(a.Key.Range.Start).Line
	attachSite(h, s, path, line, origin, a.Key.Range.Start)
	return h
}

func attachOverlay(h *HoverResult, s *session.Session, d *catalog.Def) {
	if h == nil || d == nil || isVanillaOrigin(d.Origin) || d.Kind == "mod_descriptor" {
		return
	}
	var v *catalog.Def
	for _, cand := range s.VanillaDefs(d.Key) {
		if cand.Kind == d.Kind {
			c := cand
			v = &c
			break
		}
	}
	if v == nil || v.Path == "" || session.SamePath(v.Path, d.Path) {
		return
	}
	h.VanillaPath = v.Path
	h.VanillaRel = s.DisplayRel(v.Path)
	h.VanillaLine = v.Line
	h.VanillaOriginName = s.OriginName(game.OriginVanilla)
	if v.Start > 0 {
		h.VanillaCol = s.Parsed(v.Path).Lines().PositionAt(v.Start).Character
	}
}

func isVanillaOrigin(origin string) bool {
	return origin == "" || origin == game.OriginVanilla
}

func vanillaSite(s *session.Session, h *HoverResult) *HoverResult {
	if h == nil {
		return nil
	}
	h.Origin = game.OriginVanilla
	h.OriginName = s.OriginName(game.OriginVanilla)
	return h
}

func attachSite(h *HoverResult, s *session.Session, path string, line int, origin string, start int) {
	if h == nil || path == "" {
		return
	}
	h.Path = path
	h.Rel = s.DisplayRel(path)
	h.Line = line
	if origin == "" {
		origin = game.OriginVanilla
	}
	h.Origin = origin
	h.OriginName = s.OriginName(origin)
	if start > 0 {
		h.Col = s.Parsed(path).Lines().PositionAt(start).Character
	}
}
