// hover.go builds structured hover cards from the cursor subject.
package lsp

import (
	"slices"
	"strings"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/session"
)

const hoverValueCap = 8

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
		h := card(s, "loc_value", "$"+sub.name+"$", "")
		h.Body = "format " + sub.locFilter
		return h
	case sub.kind == "script_param":
		h := card(s, "script_param", "$"+sub.name+"$", "")
		owner := sub.owner
		if owner == "" && sub.def != nil {
			owner = sub.def.OwnerKey
		}
		h.Owner = owner
		h.Values, h.More = rankHoverValues(scriptParamCounts(s, sub.name, owner))
		if sub.def != nil {
			attachSite(h, s, sub.def.Path, sub.def.Line, sub.def.Origin, sub.def.Start)
		}
		return h
	case game.IsEphemeral(sub.kind):
		return ephemeralHover(s, sub.name, sub.kind, sub.def)
	case sub.local:
		return localAssignKeyHover(s, sub.path, sub.at)
	case sub.fieldKey:
		docs := s.FieldDoc(sub.name, sub.at.kind)
		h := card(s, sub.at.kind, sub.name, docs)
		h.Kind += " key"
		return vanillaSite(s, h)
	case sub.def != nil && sub.def.Kind == "loc_key":
		return locHover(s, sub.name)
	case sub.def != nil:
		return defCard(s, sub.name, sub.def)
	case sub.kind != "":
		h := card(s, sub.kind, sub.name, docsOnly(s, sub.name, sub.kind))
		h.Usage = s.TokenUsage(sub.name)
		return vanillaSite(s, h)
	default:
		return nil
	}
}

func card(s *session.Session, rawKind, key, docs string) *HoverResult {
	h := &HoverResult{Kind: game.KindLabel(rawKind), Key: key, Docs: docs}
	if docs == "" {
		gameID := ""
		if s != nil {
			gameID = s.GameID
		}
		h.Hint = game.KindHint(gameID, rawKind)
	}
	return h
}

func defCard(s *session.Session, word string, d *catalog.Def) *HoverResult {
	h := card(s, d.Kind, d.Key, docsOnly(s, word, d.Kind))
	h.Usage = s.TokenUsage(word)
	h.Body = conventionLocBody(s, d)
	attachSite(h, s, d.Path, d.Line, d.Origin, d.Start)
	attachOverlay(h, s, d)
	return h
}

func scriptParamCounts(s *session.Session, name, owner string) map[string]int {
	counts := map[string]int{}
	if owner != "" {
		for _, r := range s.RefsTo(owner) {
			if !game.IsCallKind(r.Kind) || r.Path == "" {
				continue
			}
			if v := callArgScalar(s, r.Path, r.Start, owner, name); v != "" {
				counts[v]++
			}
		}
		if len(counts) > 0 {
			return counts
		}
	}
	for _, r := range s.RefsTo(name) {
		if r.Kind != "script_param" || r.Path == "" {
			continue
		}
		if owner != "" && r.OwnerKey != "" && r.OwnerKey != owner {
			continue
		}
		if v := assignScalarAt(s, r.Path, r.Start, name); v != "" {
			counts[v]++
		}
	}
	return counts
}

func callArgScalar(s *session.Session, path string, start int, callKey, param string) string {
	res := s.Parsed(path)
	if res.Root == nil {
		return ""
	}
	chain := jomini.NodeAtOffset(res.Root, start)
	for i := len(chain) - 1; i >= 0; i-- {
		a, ok := chain[i].(*jomini.Assignment)
		if !ok || a.Key.Quoted || a.Key.Text != callKey {
			continue
		}
		b := jomini.BlockOf(a.Value)
		if b == nil {
			return ""
		}
		for _, st := range b.Statements {
			ia, ok := st.(*jomini.Assignment)
			if !ok || ia.Key.Quoted || ia.Key.Text != param {
				continue
			}
			if sc, ok := ia.Value.(*jomini.Scalar); ok && !sc.Quoted {
				return sc.Text
			}
			return ""
		}
		return ""
	}
	return ""
}

func assignScalarAt(s *session.Session, path string, start int, key string) string {
	res := s.Parsed(path)
	if res.Root == nil {
		return ""
	}
	chain := jomini.NodeAtOffset(res.Root, start)
	for i := len(chain) - 1; i >= 0; i-- {
		a, ok := chain[i].(*jomini.Assignment)
		if !ok || a.Key.Quoted || a.Key.Text != key {
			continue
		}
		if sc, ok := a.Value.(*jomini.Scalar); ok && !sc.Quoted {
			return sc.Text
		}
		return ""
	}
	return ""
}

func rankHoverValues(counts map[string]int) ([]HoverValue, int) {
	type pair struct {
		text string
		n    int
	}
	var ps []pair
	for t, n := range counts {
		if t != "" && n > 0 {
			ps = append(ps, pair{t, n})
		}
	}
	slices.SortFunc(ps, func(a, b pair) int {
		if a.n != b.n {
			return b.n - a.n
		}
		return strings.Compare(a.text, b.text)
	})
	more := 0
	if len(ps) > hoverValueCap {
		more = len(ps) - hoverValueCap
		ps = ps[:hoverValueCap]
	}
	out := make([]HoverValue, len(ps))
	for i, p := range ps {
		out[i] = HoverValue{Text: p.text, Count: p.n}
	}
	return out, more
}

func conventionLocBody(s *session.Session, d *catalog.Def) string {
	if d == nil {
		return ""
	}
	for _, key := range game.ConventionLocKeys(d.Kind, d.Key) {
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
	h := card(s, kind, key, "")
	h.Values, h.More = rankHoverValues(ephemeralCounts(s, name, kind))
	if d != nil {
		attachSite(h, s, d.Path, d.Line, d.Origin, d.Start)
	}
	return h
}

func ephemeralCounts(s *session.Session, name, kind string) map[string]int {
	counts := map[string]int{}
	ck := game.CanonicalKind(kind)
	for _, d := range s.FindDefsOf(name, kind, 256, false, true) {
		if d.Key != name || game.CanonicalKind(d.Kind) != ck {
			continue
		}
		v := d.Value
		if v == "" && ck == "saved_scope" {
			v = scopeTargetExpr(s, &d)
		}
		if v != "" {
			counts[v]++
		}
	}
	return counts
}

var scopeSkip = map[string]bool{
	"immediate": true, "option": true, "after": true, "effect": true,
	"limit": true, "if": true, "else": true, "else_if": true,
	"trigger": true, "potential": true, "and": true, "or": true,
	"not": true, "nor": true, "nand": true, "root": true,
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
		if scopeSkip[low] {
			if low == "root" {
				return "root"
			}
			continue
		}
		if isIteratorKey(low) {
			return iteratorLabel(a)
		}
		if strings.Contains(k, ":") || game.ScriptSlot(k) != "" {
			return k
		}
	}
	return "root"
}

func isIteratorKey(k string) bool {
	return strings.HasPrefix(k, "any_") || strings.HasPrefix(k, "every_") ||
		strings.HasPrefix(k, "random_") || strings.HasPrefix(k, "ordered_")
}

func iteratorLabel(a *jomini.Assignment) string {
	k := a.Key.Text
	b := jomini.BlockOf(a.Value)
	if b == nil {
		return k
	}
	for _, st := range b.Statements {
		ia, ok := st.(*jomini.Assignment)
		if !ok || ia.Key.Quoted || !strings.EqualFold(ia.Key.Text, "list") {
			continue
		}
		if sc, ok := ia.Value.(*jomini.Scalar); ok && !sc.Quoted && sc.Text != "" {
			return k + " - list=" + sc.Text
		}
	}
	return k
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
	h := card(s, "loc_key", key, "")
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
	h := card(s, at.kind, a.Key.Text, docs)
	h.Kind += " key"
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
