// derive.go works out what the install implies but never states: fire keys,
// nested databases, setup wrappers, field value kinds, loc conventions, and —
// only when script_docs is absent — prefixes and engine tokens. Everything the
// game declares outright comes from schema.go instead.
//
// These derivations resolve conflicts by dropping, so a single odd usage
// anywhere in the corpus silently deletes a fact. That is tolerable for the
// conventions left here and was not tolerable for the type system, which is why
// the type system is now read rather than inferred.
//
// Scan persists the results; BuildIndex merges mod overlays on top.

package catalog

import (
	"maps"
	"slices"
	"strings"
	"sync"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"
)

// Derived is the in-memory overlay ExtractParsed / extractUses consume.
// PrefixKinds and KindScope normally come from bindKinds joining the game's
// declared types to the harvested databases; the rest is inferred from the
// corpus. PrefixKinds falls back to inference only without script_docs.
type Derived struct {
	PrefixKinds     map[string]string
	KindScope       map[string]string
	FireKeys        map[string]string
	NestedShapes    []NestedShape
	Wrappers        map[string]bool
	FieldValueKinds map[string]string
	LocConventions  map[string]string
}

// derivedFromCache copies the persisted derived maps so a live reindex can
// overlay one file onto them. Nil cache yields empty derived.
func derivedFromCache(c *VanillaCache) *Derived {
	v := &Derived{
		PrefixKinds:     map[string]string{},
		KindScope:       map[string]string{},
		FireKeys:        map[string]string{},
		Wrappers:        map[string]bool{},
		FieldValueKinds: map[string]string{},
		LocConventions:  map[string]string{},
	}
	if c == nil {
		return v
	}
	maps.Copy(v.KindScope, c.KindScope)
	maps.Copy(v.PrefixKinds, c.PrefixKinds)
	maps.Copy(v.FireKeys, c.FireKeys)
	maps.Copy(v.FieldValueKinds, c.FieldValueKinds)
	maps.Copy(v.LocConventions, c.LocConventions)
	for _, w := range c.Wrappers {
		v.Wrappers[w] = true
	}
	v.NestedShapes = slices.Clone(c.NestedShapes)
	return v
}

func (v *Derived) fireKind(key string) string {
	if v == nil || loc.Classify(key) != loc.PropNone {
		return ""
	}
	if k := v.FireKeys[key]; k != "" {
		return k
	}
	return v.FireKeys[strings.ToLower(key)]
}

func (v *Derived) fieldKind(key string) string {
	if v == nil {
		return ""
	}
	if k := v.FieldValueKinds[key]; k != "" {
		return k
	}
	return v.FieldValueKinds[strings.ToLower(key)]
}

func (v *Derived) locConvention(kind string) string {
	if v == nil {
		return ""
	}
	if p := v.LocConventions[kind]; p != "" {
		return p
	}
	return v.LocConventions[jomini.CanonicalKind(kind)]
}

// ConventionKeys expands a derived loc pattern (id / id_desc / kind_id) for one def.
func ConventionKeys(kind, id, pattern string) []string {
	if id == "" || pattern == "" {
		return nil
	}
	switch pattern {
	case "id":
		return []string{id}
	case "id_desc":
		return []string{id, id + "_desc"}
	case "kind_id":
		stem := strings.TrimSuffix(kind, "s")
		out := []string{kind + "_" + id}
		if stem != kind {
			out = append(out, stem+"_"+id)
		}
		return out
	default:
		return nil
	}
}

// IsStop reports a grammar/logic word that is never an object key.
func IsStop(key string) bool {
	return stoplist[strings.ToLower(key)]
}

func (v *Derived) mergeLocal(local *Derived) {
	if local == nil {
		return
	}
	v.PrefixKinds = mergeStringMap(v.PrefixKinds, local.PrefixKinds)
	v.FireKeys = mergeStringMap(v.FireKeys, local.FireKeys)
	v.FieldValueKinds = mergeStringMap(v.FieldValueKinds, local.FieldValueKinds)
	v.LocConventions = mergeStringMap(v.LocConventions, local.LocConventions)
	for k, ok := range local.Wrappers {
		if ok {
			v.Wrappers[k] = true
		}
	}
	v.NestedShapes = mergeShapes(v.NestedShapes, local.NestedShapes)
}

// mergeStringMap overlays mods on vanilla; a value conflict drops the key.
func mergeStringMap(vanilla, mods map[string]string) map[string]string {
	if len(vanilla) == 0 && len(mods) == 0 {
		return nil
	}
	out := maps.Clone(vanilla)
	if out == nil {
		out = map[string]string{}
	}
	for k, v := range mods {
		if prev, ok := out[k]; ok && prev != v {
			delete(out, k)
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeShapes(a, b []NestedShape) []NestedShape {
	if len(a) == 0 {
		return slices.Clone(b)
	}
	seen := map[string]bool{}
	var out []NestedShape
	add := func(s NestedShape) {
		k := s.ParentKind + "\x00" + s.ChildKind + "\x00" + s.GroupKey
		if seen[k] {
			return
		}
		seen[k] = true
		out = append(out, s)
	}
	for _, s := range a {
		add(s)
	}
	for _, s := range b {
		add(s)
	}
	return out
}

func collectFieldCites(root *jomini.Root) []typedCite {
	if root == nil {
		return nil
	}
	var out []typedCite
	jomini.Walk(root, func(st jomini.Statement, _ int, _ *jomini.Block) bool {
		a, ok := st.(*jomini.Assignment)
		if !ok || a.Key.Quoted {
			return true
		}
		sc, ok := a.Value.(*jomini.Scalar)
		if !ok || sc.Quoted || sc.Text == "" || jomini.SkipObjectRHS(sc.Text) {
			return true
		}
		if _, _, ok := game.ParseTyped("", sc.Text); ok {
			return true
		}
		if _, ok := jomini.ParsePrefixed(sc.Text); ok {
			return true
		}
		if !defNameRe.MatchString(sc.Text) || IsStop(sc.Text) {
			return true
		}
		out = append(out, typedCite{prefix: a.Key.Text, id: sc.Text})
		return true
	})
	return out
}

// fireCount is the evidence for one assignment key: how often its RHS named an
// event or on_action, against how often it named anything at all. The
// denominator is what stops a key being called a fire site on one coincidence.
type fireCount struct {
	kinds map[string]int
	named int
}

// deriveFireKeys counts, per assignment key, how many of the names on its RHS
// are harvested event or on_action ids (or the event-id form `ns.1`).
func deriveFireKeys(root *jomini.Root, eventIDs, onActionIDs map[string]bool) map[string]*fireCount {
	if root == nil {
		return nil
	}
	out := map[string]*fireCount{}
	var walk func([]jomini.Statement)
	walk = func(stmts []jomini.Statement) {
		for _, st := range stmts {
			a, ok := st.(*jomini.Assignment)
			if !ok {
				if vs, ok := st.(*jomini.ValueStmt); ok {
					if b := jomini.BlockOf(vs.Value); b != nil {
						walk(b.Statements)
					}
				}
				continue
			}
			if !a.Key.Quoted && !stoplist[strings.ToLower(a.Key.Text)] &&
				loc.Classify(a.Key.Text) == loc.PropNone &&
				!strings.EqualFold(a.Key.Text, "id") {
				for _, to := range fireTargets(a.Value) {
					m := out[a.Key.Text]
					if m == nil {
						m = &fireCount{kinds: map[string]int{}}
						out[a.Key.Text] = m
					}
					m.named++
					if kind := classifyFireTarget(to, eventIDs, onActionIDs); kind != "" {
						m.kinds[kind]++
					}
				}
			}
			if b := jomini.BlockOf(a.Value); b != nil {
				walk(b.Statements)
			}
		}
	}
	walk(root.Statements)
	return out
}

// fireTargets is the assignment's own RHS: a scalar, list values, or
// inner id=/digit keys. Nested fire sites stay on their own keys.
func fireTargets(v jomini.Value) []string {
	var out []string
	if sc, ok := v.(*jomini.Scalar); ok {
		if !sc.Quoted && fireTargetOK(sc.Text) {
			out = append(out, sc.Text)
		}
		return out
	}
	b := jomini.BlockOf(v)
	if b == nil {
		return out
	}
	for _, st := range b.Statements {
		switch n := st.(type) {
		case *jomini.ValueStmt:
			if sc, ok := n.Value.(*jomini.Scalar); ok && !sc.Quoted &&
				fireTargetOK(sc.Text) {
				out = append(out, sc.Text)
			}
		case *jomini.Assignment:
			key := strings.ToLower(n.Key.Text)
			if sc, ok := n.Value.(*jomini.Scalar); ok && !sc.Quoted &&
				fireTargetOK(sc.Text) && (key == "id" || isDigitKey(key)) {
				out = append(out, sc.Text)
			}
		}
	}
	return out
}

func classifyFireTarget(s string, eventIDs, onActionIDs map[string]bool) string {
	if eventIDs[s] || eventIDRe.MatchString(s) {
		return "event"
	}
	if onActionIDs[s] {
		return "on_action"
	}
	return ""
}

const (
	// fireKeyCoverage is the share of a key's named RHS values that must be
	// events or on_actions before the key is treated as firing them.
	//
	// Unanimity alone was not enough, the same flaw the field typing had: names
	// that matched nothing were not counted, so one coincidence made a key a
	// fire site and every later use of it emitted a reference that could never
	// resolve. On one real Vic3 mod that produced 4,033 false dangling rows
	// under kind `on_action`, for keys like `alert` and `character`.
	fireKeyCoverage = 50
	// minFireNames is how many names a key must carry before coverage is worth
	// consulting at all.
	//
	// This matters more here than elsewhere, because for fire keys a *low* hit
	// rate is often the very thing worth reporting: a mod whose `trigger_event`
	// targets mostly do not exist has broken references, and silencing the key
	// would hide exactly the defect Workspace Health is for. Coverage may only
	// veto a key once there is enough evidence that a low rate means "this is
	// not a fire key" rather than "this mod is broken".
	minFireNames = 8
)

func resolveFireKinds(counts map[string]*fireCount) map[string]string {
	out := map[string]string{}
	for key, c := range counts {
		if c == nil || len(c.kinds) != 1 {
			continue
		}
		hits := 0
		for _, n := range c.kinds {
			hits += n
		}
		if c.named >= minFireNames && hits*100 < c.named*fireKeyCoverage {
			continue
		}
		for k := range c.kinds {
			out[key] = k
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeFireCounts(dst, src map[string]*fireCount) map[string]*fireCount {
	if dst == nil {
		dst = map[string]*fireCount{}
	}
	for key, c := range src {
		m := dst[key]
		if m == nil {
			m = &fireCount{kinds: map[string]int{}}
			dst[key] = m
		}
		m.named += c.named
		for k, n := range c.kinds {
			m.kinds[k] += n
		}
	}
	return dst
}

func eventOnActionIDs(defs []Def) (events, onActions map[string]bool) {
	events, onActions = map[string]bool{}, map[string]bool{}
	for _, d := range defs {
		switch jomini.CanonicalKind(d.Kind) {
		case "event":
			events[d.Key] = true
		case "on_action":
			onActions[d.Key] = true
		}
	}
	return events, onActions
}
func nestedChildKind(gameID, prefix, id string, prefixKinds map[string]string) string {
	if prefixKinds != nil {
		if k := prefixKinds[prefix]; k != "" {
			return k
		}
	}
	if k, _, ok := game.ParseTyped(gameID, prefix+":"+id); ok {
		return k
	}
	return jomini.CanonicalKind(prefix)
}

func addNestedCite(
	dst map[string][]string, defKeys map[string]bool, gameID string,
	c typedCite, prefixKinds map[string]string,
) {
	if c.id == "" || defKeys[c.id] || stoplist[strings.ToLower(c.id)] {
		return
	}
	childKind := nestedChildKind(gameID, c.prefix, c.id, prefixKinds)
	if childKind == "" || childKind == "type" {
		return
	}
	for _, have := range dst[c.id] {
		if have == childKind {
			return
		}
	}
	dst[c.id] = append(dst[c.id], childKind)
}

// walkNestedWanted visits nested object keys (depth > 0) that are in want.
func walkNestedWanted(
	gameID, rel string, root *jomini.Root, want map[string][]string,
	fn func(parentKind, groupKey, childKey, childKind string),
) {
	if root == nil || len(want) == 0 {
		return
	}
	parentKind := game.MatchExtract(gameID, rel).Kind
	var walk func(stmts []jomini.Statement, wrapper string, depth int)
	walk = func(stmts []jomini.Statement, wrapper string, depth int) {
		for _, st := range stmts {
			a, ok := st.(*jomini.Assignment)
			if !ok {
				if vs, ok := st.(*jomini.ValueStmt); ok {
					if b := jomini.BlockOf(vs.Value); b != nil {
						walk(b.Statements, wrapper, depth+1)
					}
				}
				continue
			}
			name := game.KeyIdentity(gameID, a.Key.Text)
			b := jomini.BlockOf(a.Value)
			if b != nil && depth > 0 {
				for _, ck := range want[name] {
					fn(parentKind, wrapper, name, ck)
				}
			}
			if b != nil {
				next := wrapper
				if depth == 0 {
					next = ""
				} else if wrapper == "" {
					next = name
				}
				walk(b.Statements, next, depth+1)
			}
		}
	}
	walk(root.Statements, "", 0)
}

// deriveNestedShapes records parent→child harvest patterns from unresolved cites.
// One CST walk per file (index unresolved ids, then visit matching nested keys).
// The child kind must be something the game declares as a type. Without that
// test the pass harvests any repeated inner key as a database: on CK3 it minted
// 310,044 defs of kinds named `SKILL`, `TIER`, `NUMBER` and `$SKILL$` — script
// *parameter* names — which was 64% of the entire model. EU5 produced 35,458
// under `CALL_FOR_PEACE_WARSCORE_LIMIT` and `var:chinese_expedition_opinion`.
//
// Filtering on Schema.Declares leaves CK3 6 shapes, Vic3 4, EU5 15 — including
// `religion_types -> faith`, the case this pass exists for. Anything dropped
// either was not an object at all, or already has its own folder and needs no
// nested harvest (EU5 `production_methods` is `in_game/common/production_methods`).
func deriveNestedShapes(
	gameID string, corp corpus, defs []Def,
	prefixKinds map[string]string, schema *Schema,
	cites, fieldCites []typedCite,
) []NestedShape {
	defKeys := map[string]bool{}
	for _, d := range defs {
		if d.Key != "" && !jomini.IsEphemeral(d.Kind) && d.Kind != "loc_key" {
			defKeys[d.Key] = true
		}
	}
	idChild := map[string][]string{}
	for _, c := range cites {
		addNestedCite(idChild, defKeys, gameID, c, prefixKinds)
	}
	for _, c := range fieldCites {
		addNestedCite(idChild, defKeys, gameID, c, prefixKinds)
	}
	if len(idChild) == 0 {
		return nil
	}
	type acc struct {
		parent, group, prefix string
		ok, clash             bool
		letters               map[byte]bool
	}
	byChild := map[string]*acc{}
	// A second walk: where those ids sit is only answerable once every citation
	// is known, so this cannot fold into the pass above.
	var mu sync.Mutex
	_ = corp.walk(func(f fileRef, res jomini.Result) error {
		mu.Lock()
		defer mu.Unlock()
		walkNestedWanted(gameID, f.rel, res.Root, idChild,
			func(parent, group, childKey, childKind string) {
				a := byChild[childKind]
				if a == nil {
					a = &acc{
						parent: parent, group: group, ok: true,
						letters: map[byte]bool{},
					}
					byChild[childKind] = a
				}
				if a.parent != parent {
					a.clash = true
				}
				if a.group != group {
					a.group = ""
				}
				if a.letters != nil && len(childKey) > 2 && childKey[1] == '_' {
					a.letters[childKey[0]] = true
				} else if a.letters != nil {
					a.letters = nil
				}
			})
		return nil
	})
	var out []NestedShape
	for child, a := range byChild {
		if !a.ok || a.clash || a.parent == "" {
			continue
		}
		s := NestedShape{ParentKind: a.parent, ChildKind: child, GroupKey: a.group}
		if len(a.letters) > 0 {
			var b strings.Builder
			keys := slices.Collect(maps.Keys(a.letters))
			slices.Sort(keys)
			for _, c := range keys {
				b.WriteByte(c)
			}
			s.KeyPrefix = b.String()
		}
		if !schema.Declares(s.ChildKind) {
			continue
		}
		out = append(out, s)
	}
	slices.SortFunc(out, func(a, b NestedShape) int {
		if a.ParentKind != b.ParentKind {
			return strings.Compare(a.ParentKind, b.ParentKind)
		}
		return strings.Compare(a.ChildKind, b.ChildKind)
	})
	return out
}

func keyPrefixOK(key, prefixSet string) bool {
	if prefixSet == "" || len(key) < 3 || key[1] != '_' {
		return prefixSet == ""
	}
	return strings.IndexByte(prefixSet, key[0]) >= 0
}

func looksLikeObject(v jomini.Value) bool {
	b := jomini.BlockOf(v)
	if b == nil {
		return false
	}
	if len(b.Statements) == 0 {
		return true
	}
	for _, st := range b.Statements {
		if _, ok := st.(*jomini.Assignment); ok {
			return true
		}
	}
	return false
}

func hasCitedDescendant(gameID string, b *jomini.Block, cited map[string]bool) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Statements {
		a, ok := st.(*jomini.Assignment)
		if !ok {
			continue
		}
		name := game.KeyIdentity(gameID, a.Key.Text)
		if cited[name] {
			return true
		}
		if inner := jomini.BlockOf(a.Value); inner != nil &&
			hasCitedDescendant(gameID, inner, cited) {
			return true
		}
	}
	return false
}

// applyNestedShape harvests child defs from an already-parsed parent file.
func applyNestedShape(
	gameID, path, origin string, root *jomini.Root, li *jomini.LineIndex,
	shape NestedShape, cited map[string]bool,
) []Def {
	if root == nil {
		return nil
	}
	if shape.IsOption {
		// An option database knows its own rows by name, and its rows are not
		// all blocks — a cultural parameter is `name = yes`. It gets its own
		// walk rather than bending the citation rules below.
		return applyOptionShape(gameID, path, origin, root, li, shape)
	}
	var defs []Def
	add := func(a *jomini.Assignment) {
		name := game.KeyIdentity(gameID, a.Key.Text)
		if !defNameRe.MatchString(name) || stoplist[strings.ToLower(name)] {
			return
		}
		if !looksLikeObject(a.Value) {
			return
		}
		defs = append(defs, makeDef(shape.ChildKind, name, path, origin, a.Key.Range, li))
	}
	if shape.GroupKey != "" {
		var walk func([]jomini.Statement)
		walk = func(stmts []jomini.Statement) {
			for _, st := range stmts {
				a, ok := st.(*jomini.Assignment)
				if !ok {
					if vs, ok := st.(*jomini.ValueStmt); ok {
						if b := jomini.BlockOf(vs.Value); b != nil {
							walk(b.Statements)
						}
					}
					continue
				}
				if strings.EqualFold(a.Key.Text, shape.GroupKey) {
					if b := jomini.BlockOf(a.Value); b != nil {
						for _, inner := range b.Statements {
							ia, ok := inner.(*jomini.Assignment)
							if ok && !ia.Key.Quoted {
								add(ia)
							}
						}
					}
				}
				if b := jomini.BlockOf(a.Value); b != nil {
					walk(b.Statements)
				}
			}
		}
		walk(root.Statements)
		return defs
	}
	var walk func(stmts []jomini.Statement, depth int)
	walk = func(stmts []jomini.Statement, depth int) {
		for _, st := range stmts {
			a, ok := st.(*jomini.Assignment)
			if !ok {
				if vs, ok := st.(*jomini.ValueStmt); ok {
					if b := jomini.BlockOf(vs.Value); b != nil {
						walk(b.Statements, depth+1)
					}
				}
				continue
			}
			b := jomini.BlockOf(a.Value)
			if b != nil && depth > 0 {
				name := game.KeyIdentity(gameID, a.Key.Text)
				okKey := keyPrefixOK(name, shape.KeyPrefix)
				if shape.KeyPrefix != "" {
					if okKey {
						add(a)
					}
				} else if cited[name] || hasCitedDescendant(gameID, b, cited) {
					add(a)
				}
				walk(b.Statements, depth+1)
				continue
			}
			if b != nil {
				walk(b.Statements, depth+1)
			}
		}
	}
	walk(root.Statements, 0)
	return defs
}

func citedIDsForKind(cites []typedCite, prefixKinds map[string]string, childKind string) map[string]bool {
	out := map[string]bool{}
	ck := jomini.CanonicalKind(childKind)
	for _, c := range cites {
		k := ""
		if prefixKinds != nil {
			k = prefixKinds[c.prefix]
		}
		if k == "" {
			k = jomini.CanonicalKind(c.prefix)
		}
		if jomini.CanonicalKind(k) == ck {
			out[c.id] = true
		}
	}
	return out
}

// deriveWrappers marks top-level blocks whose inners are typed cites to existing
// defs (Vic3 COUNTRIES = { c:USA ?= { } }). Callers drop defs with these keys,
// so a false positive deletes a real object from the model.
//
// A `keyword name = { }` pair is never a wrapper, however its body is shaped.
// CK3 declares inline macros that way — `scripted_effect foo = { culture:x = {…} }`
// — and matching them on shape alone deleted 28 vanilla macros from the CK3
// model, breaking go-to-definition on every one. The keyword states what the
// block is, so the declaration wins over the shape.
func deriveWrappers(gameID, rel string, root *jomini.Root, defKeys map[string]bool) []string {
	if root == nil {
		return nil
	}
	// A folder that declares named script bodies has already said what its
	// top-level keys are, and those bodies are routinely a single typed cite
	// (`title:h_china.holder ?= { … }`) — the wrapper shape exactly. Running the
	// heuristic there deleted 24 CK3 and 20 EU5 vanilla macros from the model.
	if jomini.DeclaresNamedBodies(game.MatchExtract(gameID, rel).Kind) {
		return nil
	}
	var out []string
	for i, st := range root.Statements {
		a, ok := st.(*jomini.Assignment)
		if !ok || a.Key.Quoted {
			continue
		}
		if keywordDeclared(root.Statements, i) {
			continue
		}
		b := jomini.BlockOf(a.Value)
		if b == nil {
			continue
		}
		any, all := false, true
		for _, inner := range b.Statements {
			ia, ok := inner.(*jomini.Assignment)
			if !ok || ia.Key.Quoted {
				continue
			}
			any = true
			kind, id, ok := game.ParseTyped(gameID, ia.Key.Text)
			if !ok {
				if sp, sok := jomini.TypedSpanAt(ia.Key.Text, 0); sok {
					kind, id, ok = game.ParseTyped(gameID, sp.Prefix+":"+sp.ID)
				}
			}
			if !ok || kind == "" || !defKeys[id] {
				all = false
				break
			}
		}
		if any && all {
			out = append(out, a.Key.Text)
		}
	}
	return out
}

// keywordDeclared reports that stmts[i] is the named half of a `keyword name = { }`
// pair, the form extractKeywordName harvests as a def. This is the same pairing
// rule, read from the same statement list, so the two cannot disagree.
func keywordDeclared(stmts []jomini.Statement, i int) bool {
	if i == 0 {
		return false
	}
	mv, ok := stmts[i-1].(*jomini.ValueStmt)
	if !ok {
		return false
	}
	sc, ok := mv.Value.(*jomini.Scalar)
	if !ok || sc.Quoted {
		return false
	}
	return keywordNameKind(sc.Text) != ""
}

func defKeySet(defs []Def) map[string]bool {
	out := map[string]bool{}
	for _, d := range defs {
		if d.Key != "" && !jomini.IsEphemeral(d.Kind) && d.Kind != "loc_key" {
			out[d.Key] = true
		}
	}
	return out
}

func dropWrapperDefs(defs []Def, wrappers map[string]bool) []Def {
	if len(wrappers) == 0 {
		return defs
	}
	out := defs[:0]
	for _, d := range defs {
		if wrappers[d.Key] || wrappers[strings.ToLower(d.Key)] {
			continue
		}
		out = append(out, d)
	}
	return out
}

// deriveLocConventions picks id / id_desc / kind_id when a majority of defs of
// one kind have that loc key in the sidecar.
func deriveLocConventions(defs []Def, locKeys map[string]bool) map[string]string {
	if len(locKeys) == 0 {
		return nil
	}
	type counts struct{ n, id, desc, kindID int }
	byKind := map[string]*counts{}
	for _, d := range defs {
		k := jomini.CanonicalKind(d.Kind)
		if k == "" || jomini.IsEphemeral(k) || k == "loc_key" || k == "namespace" {
			continue
		}
		c := byKind[k]
		if c == nil {
			c = &counts{}
			byKind[k] = c
		}
		c.n++
		if locKeys[d.Key] {
			c.id++
		}
		if locKeys[d.Key+"_desc"] {
			c.desc++
		}
		stem := strings.TrimSuffix(k, "s")
		if locKeys[k+"_"+d.Key] || locKeys[stem+"_"+d.Key] {
			c.kindID++
		}
	}
	out := map[string]string{}
	for kind, c := range byKind {
		if c.n < 2 {
			continue
		}
		need := c.n/2 + 1
		switch {
		case c.kindID >= need && c.kindID >= c.id && c.kindID >= c.desc:
			out[kind] = "kind_id"
		case c.desc >= need && c.desc >= c.id:
			out[kind] = "id_desc"
		case c.id >= need:
			out[kind] = "id"
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func locKeySet(sites map[string]LocEntry) map[string]bool {
	out := map[string]bool{}
	for k := range sites {
		out[k] = true
	}
	return out
}

// harvestGetSet collects `GetX` / `SetX` data-function names used anywhere in a
// script file. It runs over the whole source of every file in the install —
// 180 MB on CK3 — so it scans for the two three-byte prefixes directly instead
// of running `\b((?:Get|Set)[A-Za-z0-9_]+)\b` over all of it, which cost 12% of
// the entire scan. Same matches, including two names in one dotted chain
// (`[GetPlayer.GetName]` yields both).
func harvestGetSet(src string) map[string]bool {
	var out map[string]bool
	for i := 0; i+3 < len(src); i++ {
		if c := src[i]; c != 'G' && c != 'S' {
			continue
		}
		if src[i+1] != 'e' || src[i+2] != 't' {
			continue
		}
		if i > 0 && isDataFnByte(src[i-1]) {
			continue // mid-word: the regex's leading \b
		}
		j := i + 3
		for j < len(src) && isDataFnByte(src[j]) {
			j++
		}
		if j == i+3 {
			continue // bare "Get"/"Set": the pattern needs at least one more byte
		}
		if out == nil {
			out = map[string]bool{}
		}
		out[src[i:j]] = true
		i = j - 1
	}
	return out
}

func isDataFnByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') ||
		(c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func leadingKindProse(text string) string {
	var lines []string
	for _, raw := range strings.Split(text, "\n") {
		body, hashed := peelDocLine(raw)
		if body == "" {
			if len(lines) > 0 {
				break
			}
			continue
		}
		if !hashed {
			break
		}
		if docKeyRe.MatchString(body) || attrBulletRe.MatchString(body) {
			break
		}
		lines = append(lines, body)
		if len(lines) >= 4 {
			break
		}
	}
	return prose(lines)
}
