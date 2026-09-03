// session.go is live per-workspace state: open buffers, session maps, and reindex.

package session

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
)

// buffer is one open file: its current text and parse result.
type buffer struct {
	text   string
	result jomini.Result
}

// Session is the live model of one workspace. All fields are guarded by mu.
type Session struct {
	WorkspaceID string
	GameID      string

	mu               sync.RWMutex
	cache            *catalog.VanillaCache
	vanillaLoc       *catalog.VanillaLoc
	defaultLang      string
	mods             []catalog.ModInput
	order            map[string]int
	locByLang        map[string]map[string]catalog.LocEntry
	defsByKey        map[string][]catalog.Def
	defsByFile       map[string][]catalog.Def
	refsByKey        map[string][]catalog.Ref
	refsByFile       map[string][]catalog.Ref
	locRefs          []catalog.Ref
	edgesByFrom      map[string][]catalog.Edge
	edgesByTo        map[string][]catalog.Edge
	edgesByFile      map[string][]catalog.Edge
	cacheByKey       map[string][]catalog.Def
	cacheRefsByKey   map[string][]catalog.Ref
	cacheEdgesByTo   map[string][]catalog.Edge
	cacheEdgesByFrom map[string][]catalog.Edge
	effectSet        map[string]bool
	fieldValueKinds  map[string]string
	modFieldKinds    map[string]string
	modFieldEnums    map[string]map[string][]string
	fieldEnumsByKind map[string]map[string][]string
	viaByTo          map[string][]catalog.Edge
	viaDirty         bool
	buffers          map[string]*buffer
	lastIndexed      map[string]string // path -> last-indexed text, for coalescing
	watcher          *Watcher
	onFS             func(path string, deleted bool)
}

// NewWithLoc is New with loc language and an optional vanilla loc sidecar.
func NewWithLoc(
	workspaceID, gameID, defaultLang string,
	cache *catalog.VanillaCache, vloc *catalog.VanillaLoc, mods []catalog.ModInput,
) *Session {
	if defaultLang == "" {
		defaultLang = "english"
	}
	s := &Session{
		WorkspaceID: workspaceID,
		GameID:      gameID,
		cache:       cache,
		vanillaLoc:  vloc,
		defaultLang: defaultLang,
		mods:        mods,
		buffers:     map[string]*buffer{},
		lastIndexed: map[string]string{},
		defsByKey:   map[string][]catalog.Def{},
		defsByFile:  map[string][]catalog.Def{},
		refsByKey:   map[string][]catalog.Ref{},
		refsByFile:  map[string][]catalog.Ref{},
		edgesByFrom: map[string][]catalog.Edge{},
		edgesByTo:   map[string][]catalog.Edge{},
		edgesByFile: map[string][]catalog.Edge{},
	}
	idx, err := catalog.BuildIndex(context.Background(), gameID, mods, cache)
	if err != nil {
		idx = catalog.Harvest{}
	}
	if cache != nil {
		catalog.PrepareCache(cache)
	}
	s.locByLang = idx.Loc
	if s.locByLang == nil {
		s.locByLang = map[string]map[string]catalog.LocEntry{}
	}
	s.order = catalog.OrderMap(idx.Order)
	s.addDefsLocked(idx.Defs)
	s.addRefsLocked(idx.Refs)
	s.addEdgesLocked(idx.Edges)
	s.modFieldKinds = idx.FieldValueKinds
	s.modFieldEnums = idx.FieldEnumsByKind
	s.rebuildCacheIndexLocked()
	s.rebuildFieldKindsLocked()
	s.rebuildEffectSetLocked()
	s.viaDirty = true
	return s
}

// ReplaceCache swaps the vanilla Cache after a Rescan without dropping the session.
func (s *Session) ReplaceCache(c *catalog.VanillaCache) {
	s.mu.Lock()
	if c != nil {
		catalog.PrepareCache(c)
	}
	s.cache = c
	s.rebuildCacheIndexLocked()
	s.rebuildFieldKindsLocked()
	s.rebuildEffectSetLocked()
	s.viaDirty = true
	s.mu.Unlock()
}

// ReplaceVanillaLoc swaps the vanilla loc sidecar after a loc-only harvest.
func (s *Session) ReplaceVanillaLoc(loc *catalog.VanillaLoc) {
	s.mu.Lock()
	s.vanillaLoc = loc
	s.mu.Unlock()
}

// SetDefaultLang sets the workspace default loc language (empty → english).
func (s *Session) SetDefaultLang(lang string) {
	if lang == "" {
		lang = "english"
	}
	s.mu.Lock()
	s.defaultLang = lang
	s.mu.Unlock()
}

// DefaultLang returns the workspace default loc language.
func (s *Session) DefaultLang() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.defaultLang == "" {
		return "english"
	}
	return s.defaultLang
}

// Result returns the parse result for an open file, or ok=false if not open.
func (s *Session) Result(path string) (jomini.Result, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if b := s.buffers[CanonPath(path)]; b != nil {
		return b.result, true
	}
	return jomini.Result{}, false
}

// DidOpen records a freshly opened file's text and indexes it.
func (s *Session) DidOpen(path, text string) { s.edit(path, text) }

// DidChange records an in-editor edit and reindexes the file.
func (s *Session) DidChange(path, text string) { s.edit(path, text) }

// DidClose drops the open buffer; the last-indexed on-disk state is kept.
func (s *Session) DidClose(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buffers, CanonPath(path))
}

// DidSave reindexes from the open buffer if present, else from disk.
func (s *Session) DidSave(path string) {
	path = CanonPath(path)
	s.mu.Lock()
	if b := s.buffers[path]; b != nil {
		s.reindexLocked(path, "", b.text, jomini.Result{})
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	_ = s.reindexFromDisk(path)
}

// ExternalChange reindexes a file touched outside the editor (watcher-driven).
func (s *Session) ExternalChange(path string) {
	if s.reindexFromDisk(path) {
		s.mu.Lock()
		s.dropPathLocked(CanonPath(path))
		s.mu.Unlock()
		s.emitFS(path, true)
		return
	}
	s.emitFS(path, false)
}

func (s *Session) reindexFromDisk(path string) (deleted bool) {
	path = CanonPath(path)
	raw, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	text, _ := jomini.Decode(raw)
	s.mu.Lock()
	s.reindexLocked(path, "", text, jomini.Result{})
	s.mu.Unlock()
	return false
}

// edit updates the buffer (parsing once) and reindexes the file.
func (s *Session) edit(path, text string) {
	path = CanonPath(path)
	text = jomini.Normalize(text)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, rel, ok := s.locate(path)
	if !ok {
		return
	}
	if game.MatchExtract(s.GameID, rel).Mode == game.ModeLocKey {
		s.buffers[path] = &buffer{text: text}
		s.reindexLocked(path, rel, text, jomini.Result{Src: text})
		return
	}
	result := jomini.Parse(text)
	s.buffers[path] = &buffer{text: text, result: result}
	s.reindexLocked(path, rel, text, result)
}

// reindexLocked patches indexes for one path. Identical bytes are a no-op.
func (s *Session) reindexLocked(path, rel, text string, parsed jomini.Result) {
	path = CanonPath(path)
	text = jomini.Normalize(text)
	if prev, ok := s.lastIndexed[path]; ok && prev == text {
		return
	}
	origin, locatedRel, ok := s.locate(path)
	if !ok {
		return
	}
	if rel == "" {
		rel = locatedRel
	}
	s.dropPathLocked(path)
	var ex catalog.FileExtract
	if parsed.Root != nil {
		ex = catalog.ExtractParsed(s.GameID, path, rel, origin, parsed, false)
	} else {
		ex = catalog.ExtractFile(s.GameID, path, rel, origin, text, false)
	}
	s.addDefsLocked(ex.Defs)
	s.addRefsLocked(ex.Refs)
	s.addEdgesLocked(ex.Edges)
	s.addEffectsLocked(ex.Defs)
	s.addEdgesLocked(catalog.ApplyCallEdges(ex.Cands, s.effectSet))
	s.locByLang = catalog.MergeLoc(s.locByLang, ex.Loc)
	s.lastIndexed[path] = text
	s.viaDirty = true
}

// mapKeysMatching returns map keys that SamePath-match path.
func mapKeysMatching[V any](m map[string]V, path string) []string {
	var keys []string
	for k := range m {
		if SamePath(k, path) {
			keys = append(keys, k)
		}
	}
	return keys
}

// dropPathLocked removes every index entry sourced from path.
// File maps are keyed by CanonPath; harvest and Monaco may still disagree on
// case until CanonPath folds them, so every SamePath alias is dropped.
func (s *Session) dropPathLocked(path string) {
	path = CanonPath(path)
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}
	for _, k := range mapKeysMatching(s.defsByFile, path) {
		add(k)
	}
	for _, k := range mapKeysMatching(s.refsByFile, path) {
		add(k)
	}
	for _, k := range mapKeysMatching(s.edgesByFile, path) {
		add(k)
	}
	add(path)
	for _, key := range keys {
		s.dropFileMapsLocked(key)
	}
	s.locRefs = removeRefsPath(s.locRefs, path)
	for _, k := range mapKeysMatching(s.lastIndexed, path) {
		delete(s.lastIndexed, k)
	}
	if s.locByLang == nil {
		return
	}
	for lang, m := range s.locByLang {
		for k, v := range m {
			if SamePath(v.Path, path) {
				delete(m, k)
			}
		}
		if len(m) == 0 {
			delete(s.locByLang, lang)
		}
	}
}

// dropFileMapsLocked removes defs/refs/edges stored under one file-map key.
func (s *Session) dropFileMapsLocked(path string) {
	defs := s.defsByFile[path]
	for _, d := range defs {
		s.defsByKey[d.Key] = removeDefsPath(s.defsByKey[d.Key], path)
		if len(s.defsByKey[d.Key]) == 0 {
			delete(s.defsByKey, d.Key)
		}
		if game.CanonicalKind(d.Kind) == "scripted_effect" {
			delete(s.effectSet, d.Key)
		}
	}
	delete(s.defsByFile, path)
	for _, r := range s.refsByFile[path] {
		s.refsByKey[r.Key] = removeRefsPath(s.refsByKey[r.Key], path)
		if len(s.refsByKey[r.Key]) == 0 {
			delete(s.refsByKey, r.Key)
		}
	}
	delete(s.refsByFile, path)
	for _, e := range s.edgesByFile[path] {
		s.edgesByFrom[e.From] = removeEdgesPath(s.edgesByFrom[e.From], path)
		if len(s.edgesByFrom[e.From]) == 0 {
			delete(s.edgesByFrom, e.From)
		}
		s.edgesByTo[e.To] = removeEdgesPath(s.edgesByTo[e.To], path)
		if len(s.edgesByTo[e.To]) == 0 {
			delete(s.edgesByTo, e.To)
		}
	}
	delete(s.edgesByFile, path)
}

func (s *Session) defsOf(key string) []catalog.Def {
	n := len(s.defsByKey[key]) + len(s.cacheByKey[key])
	if n == 0 {
		return nil
	}
	return append(append(make([]catalog.Def, 0, n), s.defsByKey[key]...), s.cacheByKey[key]...)
}

func cloneIf[T any](in []T) []T {
	if len(in) == 0 {
		return nil
	}
	return slices.Clone(in)
}

// Resolve returns the effective definition of key (winner across mods + vanilla).
func (s *Session) Resolve(key string) *catalog.Def {
	return s.ResolveMatching(key, nil)
}

// ResolveMatching is Resolve after keep filters defsOf(key). keep nil keeps all.
func (s *Session) ResolveMatching(key string, keep func(catalog.Def) bool) *catalog.Def {
	s.mu.RLock()
	defer s.mu.RUnlock()
	defs := s.defsOf(key)
	if keep != nil {
		n := 0
		for _, d := range defs {
			if keep(d) {
				defs[n] = d
				n++
			}
		}
		defs = defs[:n]
	}
	return catalog.Winner(s.GameID, defs, s.order)
}

// LocSite returns the winning localization definition site for key.
func (s *Session) LocSite(key string) (file string, line int, origin string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var locDefs []catalog.Def
	for _, d := range s.defsByKey[key] {
		if d.Kind == "loc_key" {
			locDefs = append(locDefs, d)
		}
	}
	if w := catalog.Winner(s.GameID, locDefs, s.order); w != nil {
		return w.Path, w.Line, w.Origin, true
	}
	if s.vanillaLoc != nil {
		if site, hit := s.vanillaLoc.Sites[key]; hit {
			return site.Path, site.Line, game.OriginVanilla, true
		}
	}
	return "", 0, "", false
}

// Mods returns the workspace mods in the order they were supplied.
func (s *Session) Mods() []catalog.ModInput {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mods
}

// Locate finds the origin and root-relative path for an absolute file path.
// Mods win; vanilla is the install script root (InstallPath + ScriptRoot).
func (s *Session) Locate(path string) (origin, rel string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.locate(path)
}

// FileText returns the buffer text if open, otherwise the on-disk decoded text.
func (s *Session) FileText(path string) string {
	s.mu.RLock()
	b := s.buffers[CanonPath(path)]
	s.mu.RUnlock()
	if b != nil {
		return b.text
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text, _ := jomini.Decode(raw)
	return text
}

// SetFileNotify registers a callback fired after each external file change.
func (s *Session) SetFileNotify(fn func(path string, deleted bool)) {
	s.mu.Lock()
	s.onFS = fn
	s.mu.Unlock()
}

func (s *Session) emitFS(path string, deleted bool) {
	s.mu.RLock()
	fn := s.onFS
	s.mu.RUnlock()
	if fn != nil {
		fn(path, deleted)
	}
}

// StartWatch begins watching mod roots for external changes.
func (s *Session) StartWatch() error {
	s.mu.Lock()
	roots := make([]string, 0, len(s.mods))
	for _, m := range s.mods {
		roots = append(roots, m.Root)
	}
	s.mu.Unlock()
	w, err := NewWatcher(roots, s.ExternalChange)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.watcher = w
	s.mu.Unlock()
	return nil
}

// Close stops the file watcher. Safe to call more than once.
func (s *Session) Close() {
	s.mu.Lock()
	w := s.watcher
	s.watcher = nil
	s.mu.Unlock()
	if w != nil {
		_ = w.Close()
	}
}

func (s *Session) locate(path string) (origin, rel string, ok bool) {
	path = CanonPath(path)
	for _, m := range s.mods {
		if r, inside := RelPath(m.Root, path); inside {
			return m.Origin, r, true
		}
	}
	for _, root := range s.vanillaRoots() {
		if r, inside := RelPath(root, path); inside {
			return game.OriginVanilla, r, true
		}
	}
	return "", "", false
}

func (s *Session) vanillaRoots() []string {
	if s.cache == nil || s.cache.InstallPath == "" {
		return nil
	}
	inst := CanonPath(s.cache.InstallPath)
	info := game.Get(s.GameID)
	if info != nil && info.ScriptRoot != "" {
		return []string{filepath.Join(inst, info.ScriptRoot), inst}
	}
	return []string{inst}
}

func removeByPath[T any](in []T, drop string, pathOf func(T) string) []T {
	out := in[:0]
	for _, x := range in {
		if !SamePath(pathOf(x), drop) {
			out = append(out, x)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func removeDefsPath(in []catalog.Def, drop string) []catalog.Def {
	return removeByPath(in, drop, func(d catalog.Def) string { return d.Path })
}
func removeRefsPath(in []catalog.Ref, drop string) []catalog.Ref {
	return removeByPath(in, drop, func(r catalog.Ref) string { return r.Path })
}
func removeEdgesPath(in []catalog.Edge, drop string) []catalog.Edge {
	return removeByPath(in, drop, func(e catalog.Edge) string { return e.Path })
}

func (s *Session) rebuildCacheIndexLocked() {
	s.cacheByKey = map[string][]catalog.Def{}
	s.cacheRefsByKey = map[string][]catalog.Ref{}
	s.cacheEdgesByTo = map[string][]catalog.Edge{}
	s.cacheEdgesByFrom = map[string][]catalog.Edge{}
	if s.cache == nil {
		return
	}
	for _, d := range s.cache.Defs {
		s.cacheByKey[d.Key] = append(s.cacheByKey[d.Key], d)
	}
	for _, r := range s.cache.LocRefs {
		s.cacheRefsByKey[r.Key] = append(s.cacheRefsByKey[r.Key], r)
	}
	for _, e := range s.cache.Edges {
		s.cacheEdgesByTo[e.To] = append(s.cacheEdgesByTo[e.To], e)
		s.cacheEdgesByFrom[e.From] = append(s.cacheEdgesByFrom[e.From], e)
	}
}

func (s *Session) rebuildFieldKindsLocked() {
	var vanilla map[string]string
	if s.cache != nil {
		vanilla = s.cache.FieldValueKinds
	}
	s.fieldValueKinds = catalog.MergeFieldValueKinds(vanilla, s.modFieldKinds)
	var vanEnums map[string]map[string][]string
	if s.cache != nil {
		vanEnums = s.cache.FieldEnumsByKind
	}
	s.fieldEnumsByKind = catalog.MergeFieldEnums(vanEnums, s.modFieldEnums)
}

func (s *Session) addDefsLocked(defs []catalog.Def) {
	for _, d := range defs {
		s.defsByKey[d.Key] = append(s.defsByKey[d.Key], d)
		p := CanonPath(d.Path)
		s.defsByFile[p] = append(s.defsByFile[p], d)
	}
}

func (s *Session) addRefsLocked(refs []catalog.Ref) {
	for _, r := range refs {
		s.refsByKey[r.Key] = append(s.refsByKey[r.Key], r)
		p := CanonPath(r.Path)
		s.refsByFile[p] = append(s.refsByFile[p], r)
		if r.Kind == "loc" || r.Kind == "loc-broad" || r.Kind == "loc-convention" {
			s.locRefs = append(s.locRefs, r)
		}
	}
}

func (s *Session) addEdgesLocked(edges []catalog.Edge) {
	for _, e := range edges {
		s.edgesByFrom[e.From] = append(s.edgesByFrom[e.From], e)
		s.edgesByTo[e.To] = append(s.edgesByTo[e.To], e)
		p := CanonPath(e.Path)
		s.edgesByFile[p] = append(s.edgesByFile[p], e)
	}
}

func (s *Session) addEffectsLocked(defs []catalog.Def) {
	for _, d := range defs {
		if game.CanonicalKind(d.Kind) == "scripted_effect" {
			s.effectSet[d.Key] = true
		}
	}
}

// DisplayRel is origin-relative, else install-relative, else the path with slashes.
func (s *Session) DisplayRel(path string) string {
	if path == "" {
		return ""
	}
	if _, rel, ok := s.Locate(path); ok {
		return filepath.ToSlash(rel)
	}
	if inst, _, _, _ := s.CacheInfo(); inst != "" {
		if rel, err := filepath.Rel(inst, path); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(path)
}

// OriginName is the mod display name, or the game title for vanilla.
func (s *Session) OriginName(origin string) string {
	if origin == "" || origin == game.OriginVanilla {
		if g := game.Get(s.GameID); g != nil && g.Name != "" {
			return g.Name
		}
		return "Game"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.mods {
		if m.Origin == origin && m.Name != "" {
			return m.Name
		}
	}
	return origin
}

// ModDefsOf returns workspace-only declarations of key.
func (s *Session) ModDefsOf(key string) []catalog.Def {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneIf(s.defsByKey[key])
}

// eachDef walks workspace defs, then vanilla if includeVanilla. fn false stops.
func (s *Session) eachDef(includeVanilla bool, fn func(catalog.Def) bool) {
	for _, defs := range s.defsByFile {
		for _, d := range defs {
			if !fn(d) {
				return
			}
		}
	}
	if includeVanilla && s.cache != nil {
		for _, d := range s.cache.Defs {
			if !fn(d) {
				return
			}
		}
	}
}

// FindDefs returns defs whose keys contain query (or, if byPrefix, start with it).
func (s *Session) FindDefs(query string, limit int, byPrefix, includeVanilla bool) []catalog.Def {
	if byPrefix && limit <= 0 {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	var out []catalog.Def
	s.eachDef(includeVanilla, func(d catalog.Def) bool {
		if d.Kind == "saved_scope" {
			return true
		}
		if q != "" {
			k := strings.ToLower(d.Key)
			hit := strings.Contains(k, q)
			if byPrefix {
				hit = strings.HasPrefix(k, q)
			}
			if !hit {
				return true
			}
		}
		out = append(out, d)
		return limit <= 0 || len(out) < limit
	})
	return out
}

// GraphCatalog returns event picker entries, node kinds, and scripted_effect keys.
func (s *Session) GraphCatalog(origins []string) (
	picker map[string]string, kinds map[string]string, effects map[string]bool,
) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	picker, kinds = map[string]string{}, map[string]string{}
	effects = make(map[string]bool, len(s.effectSet))
	for k, v := range s.effectSet {
		effects[k] = v
	}
	s.eachDef(true, func(d catalog.Def) bool {
		ck := game.CanonicalKind(d.Kind)
		if ck != "event" && ck != "on_action" && ck != "decision" {
			return true
		}
		kinds[d.Key] = ck
		want := d.Origin
		if want == "" {
			want = game.OriginVanilla
		}
		if len(origins) == 0 || slices.Contains(origins, want) {
			if prev, ok := picker[d.Key]; !ok || prev == "" || d.Origin != "" {
				picker[d.Key] = want
			}
		}
		return true
	})
	return picker, kinds, effects
}

// Contests returns contested override keys for the live workspace.
func (s *Session) Contests(order []string) []catalog.Contest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var mods []catalog.Def
	s.eachDef(false, func(d catalog.Def) bool {
		mods = append(mods, d)
		return true
	})
	var vanilla []catalog.Def
	if s.cache != nil {
		vanilla = s.cache.Defs
	}
	return catalog.Contests(s.GameID, mods, vanilla, order)
}

// LocFile returns a workspace loc file path for origin and lang, if indexed.
func (s *Session) LocFile(origin, lang string) (string, bool) {
	marker := "_l_" + strings.ToLower(lang)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var found string
	s.eachDef(false, func(d catalog.Def) bool {
		if d.Kind == "loc_key" && d.Origin == origin &&
			strings.Contains(strings.ToLower(d.Path), marker) {
			found = d.Path
			return false
		}
		return true
	})
	return found, found != ""
}

// MemberSets returns pre-built vanilla structure/effect/trigger maps for kind.
func (s *Session) MemberSets(kind string) (structKeys, effects, triggers map[string]bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return nil, nil, nil
	}
	return s.cache.MemberSets(kind)
}

// Structures returns vanilla structure keys for kind (frequency order).
func (s *Session) Structures(kind string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return nil
	}
	return s.cache.Structures[kind]
}

// StructureBlock reports whether key is usually a block field under kind.
func (s *Session) StructureBlock(kind, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return false
	}
	return s.cache.StructureBlock(kind, key)
}

// Vocab returns a vanilla membership slice: effect, trigger, vocabulary,
// gui_type, gui_prop, or meta.
func (s *Session) Vocab(kind string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return nil
	}
	switch kind {
	case "effect":
		return s.cache.Effects
	case "trigger":
		return s.cache.Triggers
	case "vocabulary":
		return s.cache.Vocabulary
	case "gui_type":
		return s.cache.GUITypes
	case "gui_prop":
		return s.cache.GUIProps
	case "meta":
		return s.cache.MetaKeys
	case "datafunction":
		return s.cache.DataFunctions
	default:
		return nil
	}
}

// RefsTo returns workspace refs whose key matches. Empty key returns none.
func (s *Session) RefsTo(key string) []catalog.Ref {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if key == "" {
		return nil
	}
	out := cloneIf(s.refsByKey[key])
	if extra := s.cacheRefsByKey[key]; len(extra) > 0 {
		out = append(out, extra...)
	}
	return out
}

// RefsInFile returns workspace refs sourced from path.
func (s *Session) RefsInFile(path string) []catalog.Ref {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneIf(s.refsByFile[CanonPath(path)])
}

// DefaultLoc returns the workspace-default-language localization string for key.
func (s *Session) DefaultLoc(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lang := s.defaultLang
	if lang == "" {
		lang = "english"
	}
	if m := s.locByLang[lang]; m != nil {
		if v, ok := m[key]; ok {
			return v.Value, true
		}
	}
	if s.vanillaLoc != nil {
		if site, ok := s.vanillaLoc.Sites[key]; ok && site.Value != "" {
			return site.Value, true
		}
	}
	return "", false
}

// DefsInFile returns workspace defs declared in path.
func (s *Session) DefsInFile(path string) []catalog.Def {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneIf(s.defsByFile[CanonPath(path)])
}

// FieldDoc returns kind-scoped field-info, then global field-info, then TokenDoc.
func (s *Session) FieldDoc(key, kind string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return ""
	}
	if v := s.fieldDocLocked(key, kind); v != "" {
		return v
	}
	if sib := portraitSibling(key); sib != "" {
		return s.fieldDocLocked(sib, kind)
	}
	return ""
}

func (s *Session) fieldDocLocked(key, kind string) string {
	lk := strings.ToLower(key)
	if kind != "" {
		if m := s.cache.FieldInfoByKind[kind]; m != nil && m[lk] != "" {
			return m[lk]
		}
	}
	if v := s.cache.FieldInfo[lk]; v != "" {
		return v
	}
	return s.cache.TokenDoc[lk]
}

// portraitSibling maps left_foo ↔ right_foo for script_docs pair fallback.
func portraitSibling(key string) string {
	k := strings.ToLower(key)
	switch {
	case strings.HasPrefix(k, "left_"):
		return "right_" + k[len("left_"):]
	case strings.HasPrefix(k, "right_"):
		return "left_" + k[len("right_"):]
	default:
		return ""
	}
}

// TokenUsage returns script_docs usage text for an engine token.
func (s *Session) TokenUsage(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil || s.cache.TokenUsage == nil {
		return ""
	}
	lk := strings.ToLower(key)
	if u := s.cache.TokenUsage[lk]; u != "" {
		return u
	}
	if s.fieldDocLocked(key, "") != "" {
		return ""
	}
	sib := portraitSibling(key)
	if sib == "" || s.fieldDocLocked(sib, "") != "" {
		return ""
	}
	return s.cache.TokenUsage[sib]
}

// TokenScopes returns script_docs supported-scopes text for an engine token.
func (s *Session) TokenScopes(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil || s.cache.TokenScopes == nil {
		return ""
	}
	return s.cache.TokenScopes[strings.ToLower(key)]
}

// FieldValueKind returns the harvested unique def type for an assignment field.
func (s *Session) FieldValueKind(field string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v := s.fieldValueKinds[field]; v != "" {
		return v
	}
	return s.fieldValueKinds[strings.ToLower(field)]
}

// FieldEnums returns unique harvested scalar values for a kind+field, or nil.
func (s *Session) FieldEnums(kind, field string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lk := strings.ToLower(field)
	try := func(k string) []string {
		if k == "" || s.fieldEnumsByKind == nil {
			return nil
		}
		if m := s.fieldEnumsByKind[k]; m != nil {
			return m[lk]
		}
		return nil
	}
	if v := try(kind); len(v) > 0 {
		return v
	}
	return try(game.CanonicalKind(kind))
}

// CacheInfo returns vanilla install path, scan timestamp, game version, and install id.
func (s *Session) CacheInfo() (installPath, scannedAt, gameVersion, installID string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache == nil {
		return "", "", "", ""
	}
	return s.cache.InstallPath, s.cache.ScannedAt, s.cache.GameVersion, s.cache.InstallID
}

// Parsed returns the buffer Result if open, else parses FileText.
func (s *Session) Parsed(path string) jomini.Result {
	src := s.FileText(path)
	if src == "" {
		return jomini.Result{}
	}
	if r, ok := s.Result(path); ok && r.Src == src {
		return r
	}
	if _, rel, ok := s.Locate(path); ok && game.MatchExtract(s.GameID, rel).Mode == game.ModeLocKey {
		return jomini.Result{Src: src}
	}
	return jomini.Parse(src)
}

// DefCount is the number of workspace definitions.
func (s *Session) DefCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	s.eachDef(false, func(d catalog.Def) bool {
		n++
		return true
	})
	return n
}

// LocRefs returns workspace loc and loc-broad refs.
func (s *Session) LocRefs() []catalog.Ref {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneIf(s.locRefs)
}

func addLocKeys(seen map[string]bool, out *[]string, k string) {
	if k == "" || seen[k] {
		return
	}
	seen[k] = true
	*out = append(*out, k)
}

// LocKeys returns workspace plus vanilla loc keys for completions.
func (s *Session) LocKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]bool{}
	var out []string
	for _, m := range s.locByLang {
		for k := range m {
			addLocKeys(seen, &out, k)
		}
	}
	if s.vanillaLoc != nil {
		for k := range s.vanillaLoc.Sites {
			addLocKeys(seen, &out, k)
		}
	}
	s.eachDef(false, func(d catalog.Def) bool {
		if d.Kind != "loc_key" {
			return true
		}
		addLocKeys(seen, &out, d.Key)
		return true
	})
	return out
}

// InheritedLocKeys returns vanilla loc keys for lang (sidecar + vanilla loc_key defs).
func (s *Session) InheritedLocKeys(lang string) []string {
	s.mu.RLock()
	defLang := s.defaultLang
	var sites map[string]catalog.LocEntry
	if s.vanillaLoc != nil && (lang == "" || lang == defLang) {
		sites = s.vanillaLoc.Sites
	}
	installID, version := "", ""
	if s.cache != nil {
		installID, version = s.cache.InstallID, s.cache.GameVersion
	}
	s.mu.RUnlock()

	seen := map[string]bool{}
	var out []string
	if sites != nil {
		for k := range sites {
			addLocKeys(seen, &out, k)
		}
	} else if lang != "" && lang != defLang && installID != "" {
		if version == "" {
			version = "latest"
		}
		if vl, err := catalog.LoadVanillaLoc(installID, version, lang); err == nil && vl != nil {
			for k := range vl.Sites {
				addLocKeys(seen, &out, k)
			}
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.eachDef(true, func(d catalog.Def) bool {
		if d.Kind != "loc_key" || d.Origin != "" {
			return true
		}
		addLocKeys(seen, &out, d.Key)
		return true
	})
	return out
}

// UsedLocKeys is loc / loc-broad / loc-convention keys from workspace and vanilla.
func (s *Session) UsedLocKeys() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]bool{}
	add := func(refs []catalog.Ref) {
		for _, r := range refs {
			switch r.Kind {
			case "loc", "loc-broad", "loc-convention":
				out[r.Key] = true
			}
		}
	}
	add(s.locRefs)
	if s.cache != nil {
		add(s.cache.LocRefs)
	}
	return out
}

// LocByLang returns a copy of workspace loc entries keyed by language then key.
func (s *Session) LocByLang() map[string]map[string]catalog.LocEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.locByLang == nil {
		return nil
	}
	out := make(map[string]map[string]catalog.LocEntry, len(s.locByLang))
	for lang, m := range s.locByLang {
		out[lang] = maps.Clone(m)
	}
	return out
}

// EdgesFrom returns stored call+fire edges leaving id. Empty id returns every edge.
func (s *Session) EdgesFrom(id string) []catalog.Edge {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return s.allEdgesLocked()
	}
	out := append(append([]catalog.Edge{}, s.edgesByFrom[id]...), s.cacheEdgesByFrom[id]...)
	if len(out) == 0 {
		return nil
	}
	return out
}

// EdgesTo returns stored call+fire/via edges arriving at id.
func (s *Session) EdgesTo(id string) []catalog.Edge {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil
	}
	s.ensureViaLocked()
	out := append(append(append([]catalog.Edge{}, s.edgesByTo[id]...),
		s.cacheEdgesByTo[id]...), s.viaByTo[id]...)
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Session) rebuildEffectSetLocked() {
	s.effectSet = map[string]bool{}
	s.eachDef(true, func(d catalog.Def) bool {
		if game.CanonicalKind(d.Kind) == "scripted_effect" {
			s.effectSet[d.Key] = true
		}
		return true
	})
	if s.cache == nil {
		return
	}
	_, effects, _ := s.cache.MemberSets("")
	for k := range effects {
		s.effectSet[k] = true
	}
}

func (s *Session) ensureViaLocked() {
	if !s.viaDirty {
		return
	}
	var stored []catalog.Edge
	for _, edges := range s.edgesByFile {
		stored = append(stored, edges...)
	}
	s.viaByTo = map[string][]catalog.Edge{}
	var mods []catalog.Def
	s.eachDef(false, func(d catalog.Def) bool {
		mods = append(mods, d)
		return true
	})
	for _, e := range catalog.DeriveVia(stored, s.cache, s.effectSet, mods) {
		s.viaByTo[e.To] = append(s.viaByTo[e.To], e)
	}
	s.viaDirty = false
}

func (s *Session) allEdgesLocked() []catalog.Edge {
	s.ensureViaLocked()
	var out []catalog.Edge
	for _, edges := range s.edgesByFrom {
		out = append(out, edges...)
	}
	if s.cache != nil {
		out = append(out, s.cache.Edges...)
	}
	for _, edges := range s.viaByTo {
		out = append(out, edges...)
	}
	return out
}
