// hover.go resolves hover cards from defs, loc, scopes, and field docs.
package lsp

import (
	"html"
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
		if isLocalDefFile(s, path, at.kind) {
			return localAssignKeyHover(s, path, at, a)
		}
		if h := namedHoverAllow(s, a.Key.Text, false); h != nil {
			return h
		}
		if ck := game.CanonicalKind(a.Key.Text); game.IsCallKind(ck) {
			return vanillaSite(s, &HoverResult{
				Contents: hoverCard(kindLabel(ck), a.Key.Text, "", hoverExtra(s, a.Key.Text, ck)),
			})
		}
		docs := s.FieldDoc(a.Key.Text, at.kind)
		m := memberSetsFor(s, at.kind)
		if docs == "" && !m.structKeys[a.Key.Text] && !m.structKeys[strings.ToLower(a.Key.Text)] {
			return nil
		}
		head := kindLabel(at.kind) + " key"
		return vanillaSite(s, &HoverResult{Contents: hoverCard(head, a.Key.Text, "", docs)})
	}
	if at.word == "" {
		return nil
	}
	if h := namedHoverAllow(s, at.word, true); h != nil {
		return h
	}
	if extra := s.FieldDoc(at.word, ""); extra != "" {
		return vanillaSite(s, &HoverResult{Contents: hoverCard("field", at.word, "", extra)})
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
		if d.Kind == "loc_key" {
			return locHover(s, word)
		}
		if d.Kind == "saved_scope" {
			return savedScopeHover(word)
		}
		h := &HoverResult{
			Contents: hoverCard(kindLabel(d.Kind), d.Key, "", hoverExtra(s, word, d.Kind)),
		}
		attachSite(h, s, d.Path, d.Line, d.Origin, d.Start)
		attachOverlay(h, s, d)
		return h
	}
	if allowLoc && locDefined(s, word) {
		return locHover(s, word)
	}
	if ck := memberSetsFor(s, "").catalogKind(word); ck != "" {
		return vanillaSite(s, &HoverResult{
			Contents: hoverCard(ck, word, "", hoverExtra(s, word, "")),
		})
	}
	for _, k := range s.Vocab("datafunction") {
		if k == word {
			return vanillaSite(s, &HoverResult{
				Contents: hoverCard("data function", word, "", hoverExtra(s, word, "")),
			})
		}
	}
	if ck := game.CanonicalKind(word); game.IsCallKind(ck) {
		return vanillaSite(s, &HoverResult{
			Contents: hoverCard(kindLabel(ck), word, "", hoverExtra(s, word, ck)),
		})
	}
	return nil
}

func hoverExtra(s *session.Session, word, kind string) string {
	doc := s.FieldDoc(word, kind)
	if sc := s.TokenScopes(word); sc != "" {
		if doc != "" {
			doc += "\n\n"
		}
		doc += "Scope here: " + sc
	}
	if u := s.TokenUsage(word); u != "" {
		if doc != "" {
			doc += "\n\n"
		}
		doc += "```\n" + u + "\n```"
	}
	return doc
}

func locHover(s *session.Session, key string) *HoverResult {
	text, _ := s.DefaultLoc(key)
	file, line, origin, ok := s.LocSite(key)
	if text == "" && !ok {
		return nil
	}
	h := &HoverResult{Contents: hoverCard("localization", key, text, "")}
	if ok {
		attachSite(h, s, file, line, origin, 0)
		attachOverlay(h, s, &catalog.Def{
			Kind: "loc_key", Key: key, Path: file, Line: line, Origin: origin,
		})
	}
	return h
}

func hoverCard(kind, key, locVal, docs string) string {
	var b strings.Builder
	b.WriteString("**")
	b.WriteString(mdEscape(kind))
	b.WriteString("** `")
	b.WriteString(mdEscape(key))
	b.WriteString("`")
	if locVal != "" {
		b.WriteString("\n\n")
		b.WriteString(quoteBlock(locVal))
	}
	if docs != "" {
		b.WriteString("\n\n")
		b.WriteString(mdEscape(docs))
	}
	return b.String()
}

func quoteBlock(s string) string {
	body := strings.ReplaceAll(html.EscapeString(s), "\n", "<br>")
	return `<blockquote style="border-left:3px solid #6b7280;` +
		`background-color:rgba(127,127,127,0.16);` +
		`padding:6px 10px;margin:8px 0;">` + body + `</blockquote>`
}

func mdEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "`", "\\`")
	if !strings.Contains(s, "\n") {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		if strings.HasPrefix(ln, ">") {
			lines[i] = `\` + ln
		}
	}
	return strings.Join(lines, "\n")
}

func localAssignKeyHover(
	s *session.Session, path string, at atPos, a *jomini.Assignment,
) *HoverResult {
	docs := s.FieldDoc(a.Key.Text, at.kind)
	m := memberSetsFor(s, at.kind)
	if docs == "" && !m.structKeys[a.Key.Text] && !m.structKeys[strings.ToLower(a.Key.Text)] {
		return nil
	}
	head := kindLabel(at.kind) + " key"
	h := &HoverResult{Contents: hoverCard(head, a.Key.Text, "", docs)}
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
	gameName := s.OriginName(game.OriginVanilla)
	h.Contents += "\n\nThis " + kindLabel(d.Kind) + " is overriding " +
		mdEscape(gameName) + "'s version of `" + mdEscape(d.Key) + "`."
	h.VanillaPath = v.Path
	h.VanillaRel = s.DisplayRel(v.Path)
	h.VanillaLine = v.Line
	h.VanillaOriginName = gameName
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
		Contents: hoverCard(kindLabel("saved_scope"), "scope:"+name, "", ""),
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
