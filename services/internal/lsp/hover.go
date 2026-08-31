// hover.go resolves hover cards from defs, loc, scopes, and field docs.
package lsp

import (
	"strings"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/session"
)

// Hover returns hover text at (line, UTF-8 column), or nil when there is none.
func Hover(s *session.Session, path string, line, col int) *HoverResult {
	at, ok := resolveAt(s, path, line, col)
	if !ok {
		return nil
	}
	if name, ok := scopeRefAt(at.src, at.off); ok {
		return savedScopeHover(name)
	}
	if at.saveOK {
		return savedScopeHover(at.saveName)
	}
	if a := at.assign; a != nil {
		if p, ok := game.ParsePrefixed(a.Key.Text); ok && game.IsSavedScopePrefix(p) {
			return savedScopeHover(p.Name)
		}
		if h := namedHoverAllow(s, a.Key.Text, false); h != nil {
			return h
		}
		docs := s.FieldDoc(a.Key.Text, at.kind)
		m := memberSetsFor(s, at.kind)
		if docs == "" && !m.structKeys[a.Key.Text] && !m.structKeys[strings.ToLower(a.Key.Text)] {
			return nil
		}
		head := kindLabel(at.kind) + " key `" + a.Key.Text + "`"
		return &HoverResult{Contents: hoverCard(head, "", docs)}
	}
	if at.word == "" {
		return nil
	}
	if h := namedHoverAllow(s, at.word, true); h != nil {
		return h
	}
	if extra := s.FieldDoc(at.word, ""); extra != "" {
		return &HoverResult{Contents: extra}
	}
	return nil
}

// namedHoverAllow is a loc/def/effect/trigger card for a token, or nil.
func namedHoverAllow(s *session.Session, word string, allowLoc bool) *HoverResult {
	var d *catalog.Def
	if allowLoc {
		d = s.Resolve(word)
	} else {
		d = resolveNonLoc(s, word)
	}
	if d != nil {
		if d.Type == "loc_key" {
			return locHover(s, word)
		}
		if d.Type == "saved_scope" {
			return savedScopeHover(word)
		}
		h := &HoverResult{
			Contents: hoverCard(kindLabel(d.Type)+" `"+d.Key+"`", "", s.FieldDoc(word, d.Type)),
		}
		attachSite(h, s, d.Path, d.Line, d.Origin)
		return h
	}
	if allowLoc && locDefined(s, word) {
		return locHover(s, word)
	}
	if ck := memberSetsFor(s, "").catalogKind(word); ck != "" {
		return &HoverResult{
			Contents: hoverCard(ck+" `"+word+"`", "", s.FieldDoc(word, "")),
		}
	}
	return nil
}

func locHover(s *session.Session, key string) *HoverResult {
	text, _ := s.DefaultLoc(key)
	file, line, origin, ok := s.LocSite(key)
	if text == "" && !ok {
		return nil
	}
	h := &HoverResult{Contents: hoverCard("localization `"+key+"`", text, "")}
	if ok {
		attachSite(h, s, file, line, origin)
	}
	return h
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

func attachSite(h *HoverResult, s *session.Session, path string, line int, origin string) {
	if h == nil || path == "" {
		return
	}
	h.Rel = s.DisplayRel(path)
	h.Line = line
	if origin == "" {
		origin = "vanilla"
	}
	h.Origin = origin
}

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

func savedScopeHover(name string) *HoverResult {
	return &HoverResult{
		Contents: hoverCard(kindLabel("saved_scope")+" `scope:"+name+"`", "", ""),
	}
}

func scopeRefAt(src string, off int) (name string, ok bool) {
	word, wStart, wEnd := jomini.Result{Src: src}.TokenAt(off)
	if word == "" {
		return "", false
	}
	if wEnd < len(src) && src[wEnd] == ':' && word == game.ScopePrefix {
		n, _, _ := jomini.Result{Src: src}.TokenAt(wEnd + 1)
		if n == "" {
			return "", false
		}
		return n, true
	}
	if wStart > 0 && src[wStart-1] == ':' {
		pre, _, _ := jomini.Result{Src: src}.TokenAt(wStart - 2)
		if pre == game.ScopePrefix {
			return word, true
		}
	}
	p, pok := game.ParsePrefixed(word)
	if pok && game.IsSavedScopePrefix(p) {
		return p.Name, true
	}
	return "", false
}
