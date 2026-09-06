// inspect.go resolves the cursor and fills the hover/F12 card. Not a Wails RPC.

package session

import (
	"slices"
	"strings"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"
)

const inspectValueCap = 8

// InspectValue is one unique harvested RHS or save-site expression.
type InspectValue struct {
	Text  string
	Count int
}

// Inspect is the hover/F12 payload. Kind/Key/Hint/Docs/Body are the card;
// empty Kind means hover should hide. RawKind/Name/Def drive F12 and refs.
type Inspect struct {
	Kind, Key, Hint, Docs, Body, Owner, Usage  string
	Values                                     []InspectValue
	More                                       int
	Origin, OriginName, Rel, Path              string
	Line, Col                                  int
	VanillaOriginName, VanillaRel, VanillaPath string
	VanillaLine, VanillaCol                    int

	RawKind, Name string
	Def           *catalog.Def
	FieldKey      bool
	Local         bool
	SpanStart     int
	SpanEnd       int
	AssignStart   int
	AssignEnd     int
	AssignLine    int
}

type identity struct {
	path, name, kind, owner, locFilter, extractKind string
	def                                             *catalog.Def
	fieldKey, local                                 bool
	spanStart, spanEnd                              int
	assignStart, assignEnd, assignLine              int
}

// PrettyKind is the Kind line: CanonicalKind, then underscores to spaces.
func PrettyKind(kind string) string {
	k := jomini.CanonicalKind(kind)
	if k == "" {
		k = kind
	}
	return strings.ReplaceAll(k, "_", " ")
}

// Inspect resolves the identifier at (line, UTF-8 column) and fills the card.
func (s *Session) Inspect(path string, line, col int) *Inspect {
	id, ok := s.identify(path, line, col)
	if !ok {
		return nil
	}
	ins := s.fillCard(id)
	if ins == nil {
		ins = &Inspect{}
	}
	ins.RawKind = id.kind
	ins.Name = id.name
	ins.Def = id.def
	ins.FieldKey = id.fieldKey
	ins.Local = id.local
	ins.SpanStart = id.spanStart
	ins.SpanEnd = id.spanEnd
	ins.AssignStart = id.assignStart
	ins.AssignEnd = id.assignEnd
	ins.AssignLine = id.assignLine
	if ins.Owner == "" {
		ins.Owner = id.owner
	}
	return ins
}

// inspectOf fills Kind/Hint/Docs for a resolved identity.
func (s *Session) inspectOf(kind, key, docs string) *Inspect {
	out := &Inspect{Kind: PrettyKind(kind), Key: key, Docs: docs, RawKind: kind, Name: key}
	if docs == "" {
		out.Hint = s.composeHint(kind, key)
	}
	return out
}

// composeHint is the fallback subtitle when a kind has no prose of its own: the
// declared scope type first, then how the object is cited, nested and localized.
// Empty when nothing is known.
func (s *Session) composeHint(kind, _ string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ck := jomini.CanonicalKind(kind)
	if ck == "" {
		ck = kind
	}
	if languageKind(ck) {
		return ""
	}
	var parts []string
	// Lead with the declared scope type: what this object *is* to the engine
	// matters more than how PMT found it.
	if scope := s.scopeOfKindLocked(ck); scope != "" {
		parts = append(parts, PrettyKind(scope)+" scope")
	}
	if s.cache != nil {
		if p := s.cache.KindInfo[ck]; p != "" {
			parts = append(parts, firstProse(p))
		} else if p := s.cache.KindInfo[kind]; p != "" {
			parts = append(parts, firstProse(p))
		}
		if prefix := s.kindPrefix[ck]; prefix != "" {
			parts = append(parts, "cite `"+prefix+":id`")
		}
		for _, sh := range s.cache.NestedShapes {
			if jomini.CanonicalKind(sh.ChildKind) == ck {
				msg := "nested in " + PrettyKind(sh.ParentKind)
				if sh.GroupKey != "" {
					msg += " `" + sh.GroupKey + "`"
				}
				parts = append(parts, msg)
				break
			}
		}
		if loc := s.cache.LocConventions[ck]; loc != "" {
			parts = append(parts, "loc "+loc)
		}
		if jomini.IsMacroKind(ck) {
			// The real signature, when the macro takes parameters, is built by
			// macroUsage and shown as a code block; this is only the shape note
			// for one that takes none.
			parts = append(parts, "invoked by name")
		}
	}
	if game.IsFIOS(s.GameID, ck) || game.IsFIOS(s.GameID, kind) {
		parts = append(parts, "FIOS")
	}
	// The entry-mode prefixes used to be appended here. They are a property of
	// the game, not of the thing under the cursor, so they appeared on every
	// EU5 card and on many were the entire hint — "INJECT:/REPLACE:" as the sole
	// content of a modifier's hover. A line that is always present says nothing.
	return strings.Join(parts, " · ")
}

// IsMacroKind reports a kind whose definitions are invoked by name (scripted_*).
func (s *Session) IsMacroKind(kind string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return jomini.IsMacroKind(kind)
}

// IsMacroDef reports that key is a harvested scripted-macro definition.
func (s *Session) IsMacroDef(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.macroDefKinds[key]
	return ok
}

// FireKind returns the derived fire target kind for an assignment key.
// Loc properties (title, desc, …) never fire.
func (s *Session) FireKind(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if loc.Classify(key) != loc.PropNone {
		return ""
	}
	if s.cache != nil {
		if k := s.cache.FireKeys[key]; k != "" {
			return k
		}
		if k := s.cache.FireKeys[strings.ToLower(key)]; k != "" {
			return k
		}
	}
	if k := s.modFireKeys[key]; k != "" {
		return k
	}
	return s.modFireKeys[strings.ToLower(key)]
}

// ConventionLocKeys expands the derived loc pattern for one def.
func (s *Session) ConventionLocKeys(kind, id string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return nil
	}
	pat := s.cache.LocConventions[kind]
	if pat == "" {
		pat = s.cache.LocConventions[jomini.CanonicalKind(kind)]
	}
	return catalog.ConventionKeys(kind, id, pat)
}

func (s *Session) identify(path string, line, col int) (identity, bool) {
	if s.KindFor(path) == "loc" {
		return s.identifyLoc(path, line, col)
	}
	at, ok := s.probeAt(path, line, col)
	if !ok {
		return identity{}, false
	}
	if name, ok := scopeRefAt(at.res, at.off); ok {
		return s.ephemeralID(path, name, "saved_scope"), true
	}
	if at.saveOK {
		return s.ephemeralID(path, at.saveName, "saved_scope"), true
	}
	if name, pStart, pEnd, ok := jomini.ScriptParamSpan(at.src, at.off); ok {
		d := s.resolveOfKindOwner(name, "script_param", at.paramOwner)
		if d == nil {
			origin, _, _ := s.Locate(path)
			d = &catalog.Def{
				Kind: "script_param", Key: name, Path: path, Origin: origin,
				Start: pStart, End: pEnd, OwnerKey: at.paramOwner,
				Line: at.res.Lines().PositionAt(pStart).Line,
			}
		}
		return identity{
			path: path, name: name, kind: "script_param",
			def: d, owner: at.paramOwner,
		}, true
	}
	if at.paramOwner != "" {
		if d := s.resolveOfKindOwner(at.word, "script_param", at.paramOwner); d != nil {
			return identity{
				path: path, name: at.word, kind: "script_param",
				def: d, owner: at.paramOwner,
			}, true
		}
	}
	if id, ok := s.macroCallIdentity(path, at); ok {
		return id, true
	}
	if a := at.assign; a != nil {
		if p, ok := jomini.ParsePrefixed(a.Key.Text); ok && jomini.IsSavedScopePrefix(p) {
			return s.ephemeralID(path, p.Name, "saved_scope"), true
		}
		if s.isLocalDefFile(path, at.kind) {
			return identity{
				path: path, name: a.Key.Text, kind: at.kind, local: true,
				extractKind: at.kind,
				assignStart: a.Key.Range.Start, assignEnd: a.Key.Range.End,
				assignLine: at.res.Lines().PositionAt(a.Key.Range.Start).Line,
			}, true
		}
		docs := s.kindFieldDoc(a.Key.Text, at.kind)
		if docs != "" || s.isStructKey(at.kind, a.Key.Text) ||
			loc.Classify(a.Key.Text) != loc.PropNone {
			return identity{
				path: path, name: a.Key.Text, kind: at.kind, fieldKey: true,
				extractKind: at.kind,
			}, true
		}
		if d := s.resolveNonLoc(a.Key.Text); d != nil {
			return identity{path: path, name: d.Key, kind: d.Kind, def: d}, true
		}
		if ck := jomini.CanonicalKind(a.Key.Text); s.IsMacroKind(ck) {
			return identity{path: path, name: a.Key.Text, kind: ck}, true
		}
		if ck := s.vocabRole(a.Key.Text); ck != "" {
			return identity{path: path, name: a.Key.Text, kind: ck}, true
		}
	}
	if at.word == "" {
		return identity{}, false
	}
	// A bare number or boolean on the right of an assignment is a literal, and
	// resolving it against the databases finds whatever happens to share the
	// spelling. `is_ai = yes` reached the localization lookup and vanilla
	// defines a loc key called `yes`, so the card showed the player-facing
	// string "Yes". Keys are handled above, in the at.assign branch, so a
	// `random_list` weight or a numeric event id is unaffected.
	if at.slotKey != "" && jomini.IsLiteralValue(at.word) {
		return identity{}, false
	}
	if p, ok := jomini.ParsePrefixed(at.word); ok {
		kind := jomini.PrefixKind(p.Prefix)
		if kind == "" {
			kind = p.Prefix
		}
		return s.ephemeralID(path, p.Name, kind), true
	}
	wordOff := at.off - at.start
	if sp, ok := jomini.TypedSpanAt(at.word, wordOff); ok {
		kind, id, ok := game.ParseTyped(s.GameID, sp.Prefix+":"+sp.ID)
		if !ok {
			kind, id = sp.Prefix, sp.ID
		}
		if pk := s.PrefixKind(sp.Prefix); pk != "" {
			kind = pk
		} else if kind == "" {
			kind = sp.Prefix
		}
		d := s.resolveOfKind(id, kind)
		if d == nil {
			d = s.resolveNonLoc(id)
		}
		return identity{path: path, name: id, kind: kind, def: d}, true
	}
	if at.slotKey != "" {
		if r, ok := jomini.ScriptName(at.slotKey); ok {
			if jomini.IsEphemeral(r.Kind) {
				return s.ephemeralID(path, at.word, r.Kind), true
			}
			if d := s.resolveOfKind(at.word, r.Kind); d != nil {
				return identity{path: path, name: at.word, kind: r.Kind, def: d}, true
			}
		}
		if k := s.FieldValueKind(at.slotKey); k != "" {
			if jomini.IsEphemeral(k) {
				return s.ephemeralID(path, at.word, k), true
			}
			if loc.Classify(at.slotKey) == loc.PropNone {
				if d := s.resolveOfKind(at.word, k); d != nil {
					return identity{path: path, name: at.word, kind: k, def: d}, true
				}
			}
		}
	}
	if d := s.Resolve(at.word); d != nil {
		return identity{path: path, name: d.Key, kind: d.Kind, def: d}, true
	}
	// What the game declares wins over a localization entry that merely shares
	// the spelling. Paradox localizes a great many ordinary words, so with the
	// loc lookup first, hovering a declared effect or trigger showed the
	// player-facing string in a quote instead of the token's own documentation.
	// A word the type system does not declare still falls through to loc below,
	// which is what an actual loc-key reference needs.
	if ck := s.vocabRole(at.word); ck != "" {
		return identity{path: path, name: at.word, kind: ck}, true
	}
	if s.locDefined(at.word) {
		if file, ln, origin, ok := s.LocSite(at.word); ok {
			d := &catalog.Def{
				Kind: "loc_key", Key: at.word, Path: file, Line: ln, Origin: origin,
			}
			return identity{path: path, name: at.word, kind: "loc_key", def: d}, true
		}
	}
	for _, k := range s.Vocab("datafunction") {
		if k == at.word {
			return identity{path: path, name: at.word, kind: "data_function"}, true
		}
	}
	if ck := jomini.CanonicalKind(at.word); s.IsMacroKind(ck) {
		return identity{path: path, name: at.word, kind: ck}, true
	}
	if extra := s.FieldDoc(at.word, ""); extra != "" {
		return identity{path: path, name: at.word, kind: "field"}, true
	}
	if at.word != "" {
		return identity{path: path, name: at.word}, true
	}
	return identity{}, false
}

func (s *Session) identifyLoc(path string, line, col int) (identity, bool) {
	src := s.FileText(path)
	if src == "" {
		return identity{}, false
	}
	off := jomini.NewLineIndex(src).OffsetAt(line, col)
	hit, ok := locHitAt(src, off)
	if !ok {
		return identity{}, false
	}
	if hit.engine {
		return identity{
			path: path, name: hit.key, kind: "loc_value", locFilter: hit.filter,
		}, true
	}
	id := identity{
		path: path, name: hit.key, kind: "loc_key",
		spanStart: hit.start, spanEnd: hit.end,
	}
	if file, ln, origin, ok := s.LocSite(hit.key); ok {
		id.def = &catalog.Def{
			Kind: "loc_key", Key: hit.key, Path: file, Line: ln, Origin: origin,
		}
	}
	return id, true
}

func (s *Session) ephemeralID(path, name, kind string) identity {
	if name == "" || kind == "" {
		return identity{}
	}
	return identity{
		path: path, name: name, kind: kind,
		def: s.resolveOfKind(name, kind),
	}
}

func (s *Session) macroCallIdentity(path string, at probe) (identity, bool) {
	kind, name, a, ok := macroCallAt(at.res, at.off)
	if !ok || name == "" {
		return identity{}, false
	}
	d := s.resolveOfKind(name, kind)
	if d != nil {
		ck := jomini.CanonicalKind(kind)
		for _, cand := range s.ModDefsOf(name) {
			if jomini.CanonicalKind(cand.Kind) == ck && SamePath(cand.Path, path) {
				c := cand
				d = &c
				break
			}
		}
	}
	if d == nil && a != nil {
		origin, _, _ := s.Locate(path)
		d = &catalog.Def{
			Kind: kind, Key: name, Path: path, Origin: origin,
			Start: a.Key.Range.Start, End: a.Key.Range.End,
			Line: at.res.Lines().PositionAt(a.Key.Range.Start).Line,
		}
	}
	return identity{path: path, name: name, kind: kind, def: d}, true
}

func (s *Session) fillCard(id identity) *Inspect {
	switch {
	case id.kind == "loc_value":
		ins := s.inspectOf("loc_value", "$"+id.name+"$", "")
		ins.Body = "format " + id.locFilter
		return ins
	case id.kind == "script_param":
		ins := s.inspectOf("script_param", "$"+id.name+"$", "")
		owner := id.owner
		if owner == "" && id.def != nil {
			owner = id.def.OwnerKey
		}
		ins.Owner = owner
		ins.Values, ins.More = rankInspectValues(s.scriptParamCounts(id.name, owner))
		if id.def != nil {
			s.attachSite(ins, id.def.Path, id.def.Line, id.def.Origin, id.def.Start)
		}
		return ins
	case jomini.IsEphemeral(id.kind):
		return s.ephemeralCard(id.name, id.kind, id.def)
	case id.local:
		return s.localAssignCard(id)
	case id.fieldKey:
		docs := s.FieldDoc(id.name, id.extractKind)
		ins := s.inspectOf(id.extractKind, id.name, docs)
		ins.Kind += " key"
		return s.vanillaSite(ins)
	case id.def != nil && id.def.Kind == "loc_key":
		return s.locCard(id.name)
	case id.def != nil:
		return s.defCard(id.name, id.def)
	case id.kind != "":
		ins := s.inspectOf(id.kind, id.name, s.docsOnly(id.name, id.kind))
		ins.Usage = s.TokenUsage(id.name)
		return s.vanillaSite(ins)
	default:
		return nil
	}
}

func (s *Session) defCard(word string, d *catalog.Def) *Inspect {
	docs := s.docsOnly(word, d.Kind)
	// A named script body has no other description: the author's comment above
	// it is the only one there will ever be.
	if docs == "" && d.Doc != "" {
		docs = d.Doc
	}
	ins := s.inspectOf(d.Kind, d.Key, docs)
	ins.Usage = s.TokenUsage(word)
	if ins.Usage == "" {
		ins.Usage = s.macroUsage(d)
	}
	ins.Body = s.conventionLocBody(d)
	s.attachSite(ins, d.Path, d.Line, d.Origin, d.Start)
	s.attachOverlay(ins, d)
	return ins
}

// macroUsage builds the call signature of a scripted macro from the $PARAM$
// names harvested inside its own body.
//
// No dump can supply this: script_docs documents the engine API, and of CK3's
// 11,708 scripted macros exactly 4 appear there — name collisions with real
// engine tokens. The definition is the only source, and it is a good one:
// 2,536 CK3 macros, 468 EU5 and 44 Vic3 take parameters, and before this every
// one of them got the same fixed string on its card.
//
// A macro with no parameters gets nothing here; `invoked by name` in the hint
// already says all there is to say about it.
func (s *Session) macroUsage(d *catalog.Def) string {
	if d == nil || d.Key == "" || !jomini.IsMacroKind(d.Kind) {
		return ""
	}
	names := s.FindParamsOf(d.Key, maxMacroParams)
	if len(names) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(d.Key)
	b.WriteString(" = {\n")
	for _, n := range names {
		b.WriteString("\t")
		b.WriteString(n)
		b.WriteString(" = <value>\n")
	}
	b.WriteString("}")
	return b.String()
}

// maxMacroParams caps one macro's rendered signature. The largest measured on
// the three installs takes 6.
const maxMacroParams = 32

func (s *Session) ephemeralCard(name, kind string, d *catalog.Def) *Inspect {
	if name == "" || kind == "" {
		return nil
	}
	key := name
	if kind == "saved_scope" {
		key = "scope:" + name
	}
	ins := s.inspectOf(kind, key, "")
	ins.Values, ins.More = rankInspectValues(s.ephemeralCounts(name, kind))
	if d != nil {
		s.attachSite(ins, d.Path, d.Line, d.Origin, d.Start)
	}
	return ins
}

func (s *Session) locCard(key string) *Inspect {
	text, _ := s.DefaultLoc(key)
	file, line, origin, ok := s.LocSite(key)
	if text == "" && !ok {
		return nil
	}
	ins := s.inspectOf("loc_key", key, "")
	ins.Body = unescapeLocDisplay(text)
	if ok {
		s.attachSite(ins, file, line, origin, 0)
		s.attachOverlay(ins, &catalog.Def{
			Kind: "loc_key", Key: key, Path: file, Line: line, Origin: origin,
		})
	}
	return ins
}

func (s *Session) localAssignCard(id identity) *Inspect {
	docs := s.FieldDoc(id.name, id.extractKind)
	if docs == "" && !s.isStructKey(id.extractKind, id.name) {
		return nil
	}
	ins := s.inspectOf(id.extractKind, id.name, docs)
	ins.Kind += " key"
	origin, _, _ := s.Locate(id.path)
	s.attachSite(ins, id.path, id.assignLine, origin, id.assignStart)
	return ins
}

func (s *Session) scriptParamCounts(name, owner string) map[string]int {
	counts := map[string]int{}
	if owner != "" {
		for _, r := range s.RefsTo(owner) {
			if !s.IsMacroKind(r.Kind) || r.Path == "" {
				continue
			}
			if v := s.callArgScalar(r.Path, r.Start, owner, name); v != "" {
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
		if v := s.assignScalarAt(r.Path, r.Start, name); v != "" {
			counts[v]++
		}
	}
	return counts
}

func (s *Session) callArgScalar(path string, start int, callKey, param string) string {
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

func (s *Session) assignScalarAt(path string, start int, key string) string {
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

func (s *Session) ephemeralCounts(name, kind string) map[string]int {
	counts := map[string]int{}
	ck := jomini.CanonicalKind(kind)
	for _, d := range s.FindDefsOf(name, kind, 256, false, true) {
		if d.Key != name || jomini.CanonicalKind(d.Kind) != ck {
			continue
		}
		v := d.Value
		if v == "" && ck == "saved_scope" {
			v = s.scopeTargetExpr(&d)
		}
		if v != "" {
			counts[v]++
		}
	}
	return counts
}

func (s *Session) scopeTargetExpr(d *catalog.Def) string {
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
		if jomini.IsSaveScopeKey(k) || jomini.IsSaveScopeValueKey(k) {
			continue
		}
		low := strings.ToLower(k)
		if isIteratorKey(low) {
			return iteratorLabel(a)
		}
		if catalog.IsStop(k) || jomini.ScriptSlot(k) != "" {
			if low == "root" {
				return "root"
			}
			continue
		}
		if strings.Contains(k, ":") || jomini.ScriptSlot(k) != "" {
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

func (s *Session) conventionLocBody(d *catalog.Def) string {
	if d == nil {
		return ""
	}
	for _, key := range s.ConventionLocKeys(d.Kind, d.Key) {
		if text, ok := s.DefaultLoc(key); ok && text != "" {
			return unescapeLocDisplay(text)
		}
	}
	return ""
}

func (s *Session) docsOnly(word, kind string) string {
	doc := s.FieldDoc(word, kind)
	if doc == "" {
		// Install prose and dump prose both land in TokenDoc during a scan, but
		// the schema carries its own description; use it rather than relying on
		// that map having been filled.
		doc = s.schemaDoc(word)
	}
	line := s.schemaSignature(word)
	if line == "" {
		// No declared type system: fall back to the raw scopes text.
		if sc := s.tokenScopes(word); sc != "" {
			line = "Scope here: " + sc
		}
	}
	if line == "" {
		return doc
	}
	if doc != "" {
		doc += "\n\n"
	}
	return doc + line
}

// schemaDoc returns the description the game ships for an engine token or link.
func (s *Session) schemaDoc(word string) string {
	if t, _, ok := s.engineToken(word); ok && t.Doc != "" {
		return t.Doc
	}
	if l, ok := s.scopeLink(word); ok {
		return l.Doc
	}
	return ""
}

// schemaSignature renders what the game itself declares about an engine token
// or a scope link: where it runs, and what it takes or yields. This replaces
// the old raw "character; targets: culture" scopes string.
func (s *Session) schemaSignature(word string) string {
	if t, role, ok := s.engineToken(word); ok {
		parts := []string{strings.ToUpper(role[:1]) + role[1:]}
		if in := scopeList(t.In); in != "" {
			parts = append(parts, "runs on "+in)
		}
		if t.Target != "" {
			parts = append(parts, "takes a "+PrettyKind(t.Target))
		}
		return strings.Join(parts, " · ")
	}
	if l, ok := s.scopeLink(word); ok && l.Out != "" {
		var parts []string
		if in := scopeList(l.In); in != "" {
			parts = append(parts, "from "+in)
		} else if l.Global {
			parts = append(parts, "Global link")
		}
		parts = append(parts, "yields "+PrettyKind(l.Out))
		if l.Data {
			parts = append(parts, "cite `"+word+":id`")
		}
		return strings.Join(parts, " · ")
	}
	if m, ok := s.modifierToken(word); ok {
		return modifierSignature(m)
	}
	return ""
}

// modifierSignature renders the one line a declared modifier gets on its hover
// card, from what script_docs actually states about it.
//
// This is the last unrendered part of the schema. A modifier's `Area` is the
// category list the game prints on the header line — CK3 "character and
// province", EU5 "location, all", Vic3 a Mask — and for EU5 it is the *only*
// thing declared: all 2,436 modifiers carry a category and none carry prose.
// Before this the card for `local_trades_per_burgher` was empty.
func modifierSignature(m catalog.Modifier) string {
	areas := modifierAreas(m.Area)
	if len(areas) == 0 {
		return ""
	}
	return "Modifier · applies to " + strings.Join(areas, " or ")
}

// modifierAreas splits a declared category list into the scope types it names.
//
// The three games phrase it differently — CK3 "character and province", EU5
// "location, all", Vic3 a Mask — so the separators of all three are accepted.
// "all" is dropped: EU5 stamps it on every one of its 2,436 modifiers, so it
// distinguishes nothing and would only push the useful half off the line.
func modifierAreas(area string) []string {
	if area == "" {
		return nil
	}
	fields := strings.FieldsFunc(area, func(r rune) bool {
		return r == ',' || r == ';' || r == '/' || r == '|'
	})
	out := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, f := range fields {
		for _, part := range strings.Split(strings.TrimSpace(f), " and ") {
			p := strings.ToLower(strings.TrimSpace(part))
			if p == "" || p == "all" || p == "none" || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, PrettyKind(p))
		}
	}
	return out
}

// scopeList renders declared input scopes. "none" means unrestricted, so it
// carries no information worth showing.
func scopeList(in []string) string {
	keep := make([]string, 0, len(in))
	for _, sc := range in {
		if sc == "none" || sc == "" {
			continue
		}
		keep = append(keep, PrettyKind(sc))
	}
	return strings.Join(keep, " or ")
}

func (s *Session) attachOverlay(ins *Inspect, d *catalog.Def) {
	if ins == nil || d == nil || isVanillaOrigin(d.Origin) || d.Kind == "mod_descriptor" {
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
	if v == nil || v.Path == "" || SamePath(v.Path, d.Path) {
		return
	}
	ins.VanillaPath = v.Path
	ins.VanillaRel = s.DisplayRel(v.Path)
	ins.VanillaLine = v.Line
	ins.VanillaOriginName = s.OriginName(game.OriginVanilla)
	if v.Start > 0 {
		ins.VanillaCol = s.Parsed(v.Path).Lines().PositionAt(v.Start).Character
	}
}

func isVanillaOrigin(origin string) bool {
	return origin == "" || origin == game.OriginVanilla
}

func (s *Session) vanillaSite(ins *Inspect) *Inspect {
	if ins == nil {
		return nil
	}
	ins.Origin = game.OriginVanilla
	ins.OriginName = s.OriginName(game.OriginVanilla)
	return ins
}

func (s *Session) attachSite(ins *Inspect, path string, line int, origin string, start int) {
	if ins == nil || path == "" {
		return
	}
	ins.Path = path
	ins.Rel = s.DisplayRel(path)
	ins.Line = line
	if origin == "" {
		origin = game.OriginVanilla
	}
	ins.Origin = origin
	ins.OriginName = s.OriginName(origin)
	if start > 0 {
		ins.Col = s.Parsed(path).Lines().PositionAt(start).Character
	}
}

func (s *Session) resolveNonLoc(word string) *catalog.Def {
	return s.ResolveMatching(word, func(d catalog.Def) bool {
		return d.Kind != "loc_key" && !jomini.IsEphemeral(d.Kind)
	})
}

func (s *Session) resolveOfKind(word, kind string) *catalog.Def {
	return s.resolveOfKindOwner(word, kind, "")
}

func (s *Session) resolveOfKindOwner(word, kind, owner string) *catalog.Def {
	if word == "" || kind == "" {
		return nil
	}
	ck := jomini.CanonicalKind(kind)
	return s.ResolveMatching(word, func(d catalog.Def) bool {
		if jomini.CanonicalKind(d.Kind) != ck {
			return false
		}
		if owner != "" && d.OwnerKey != "" && d.OwnerKey != owner {
			return false
		}
		return true
	})
}

func (s *Session) locDefined(key string) bool {
	if _, ok := s.DefaultLoc(key); ok {
		return true
	}
	_, _, _, ok := s.LocSite(key)
	return ok
}

func (s *Session) isStructKey(kind, key string) bool {
	sk, _, _ := s.MemberSets(kind)
	return sk[key] || sk[strings.ToLower(key)]
}

// vocabRole names an engine token's role. The declared type system wins over
// the harvested vocabulary: it distinguishes effects, triggers and scope links,
// which a flat name list cannot.
func (s *Session) vocabRole(word string) string {
	if _, role, ok := s.engineToken(word); ok {
		return role
	}
	if l, ok := s.scopeLink(word); ok && l.Out != "" {
		return "scope_link"
	}
	// A modifier is declared too. Without this it only hovered where the install
	// happened to also define it as a row (EU5's modifier_type_definitions), so
	// the same token in a mod's own file produced no card at all.
	if _, ok := s.modifierToken(word); ok {
		return "modifier"
	}
	_, ef, tr := s.MemberSets("")
	if ef[word] || ef[strings.ToLower(word)] {
		return "effect"
	}
	if tr[word] || tr[strings.ToLower(word)] {
		return "trigger"
	}
	return ""
}

func rankInspectValues(counts map[string]int) ([]InspectValue, int) {
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
	if len(ps) > inspectValueCap {
		more = len(ps) - inspectValueCap
		ps = ps[:inspectValueCap]
	}
	out := make([]InspectValue, len(ps))
	for i, p := range ps {
		out[i] = InspectValue{Text: p.text, Count: p.n}
	}
	return out, more
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

func languageKind(k string) bool {
	switch k {
	case "effect", "trigger", "scope", "field", "loc_key", "loc_value",
		"script_param", "saved_scope", "data_function", "mod_descriptor":
		return true
	}
	return false
}

func firstProse(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
