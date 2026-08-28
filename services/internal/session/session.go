// session.go is the live per-workspace state: open buffers and their parse results,
// the workspace Index, and the single reindex path used by both editor saves and
// the external-change watcher. Override winners are resolved via model.Winner only.

package session

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/parser"
)

// buffer is one open file: its current text and parse result.
type buffer struct {
	text   string
	result parser.Result
}

// Session is the live model of one workspace. All fields are guarded by mu.
type Session struct {
	WorkspaceID string
	GameID      string

	mu          sync.RWMutex
	cache       *model.Cache
	mods        []model.ModInput
	index       *model.Index
	order       map[string]int
	buffers     map[string]*buffer
	lastIndexed map[string]string // path -> last-indexed text, for coalescing
	wiki        map[string]string
	watcher     *Watcher
	reindexN    int
}

// New builds a Session for a workspace: it indexes the mods on disk and prepares
// the override order. cache may be nil (queries then see only workspace defs).
func New(workspaceID, gameID string, cache *model.Cache, mods []model.ModInput) *Session {
	s := &Session{
		WorkspaceID: workspaceID,
		GameID:      gameID,
		cache:       cache,
		mods:        mods,
		buffers:     map[string]*buffer{},
		lastIndexed: map[string]string{},
	}
	s.index = model.BuildIndex(workspaceID, gameID, mods, cache)
	s.order = orderMap(s.index.Order)
	return s
}

// Index returns the current workspace index (read-only; callers must not mutate).
func (s *Session) Index() *model.Index {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index
}

// Cache returns the vanilla model backing this session (may be nil).
func (s *Session) Cache() *model.Cache {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache
}

// ReindexCount reports how many single-file reindexes have run (test/telemetry).
func (s *Session) ReindexCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reindexN
}

// Result returns the parse result for an open file, or ok=false if not open.
func (s *Session) Result(path string) (parser.Result, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if b := s.buffers[path]; b != nil {
		return b.result, true
	}
	return parser.Result{}, false
}

// DidOpen records a freshly opened file's text and indexes it.
func (s *Session) DidOpen(path, text string) { s.edit(path, text) }

// DidChange records an in-editor edit and reindexes the file.
func (s *Session) DidChange(path, text string) { s.edit(path, text) }

// DidClose drops the open buffer; the last-indexed on-disk state is kept.
func (s *Session) DidClose(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buffers, path)
}

// DidSave reindexes from the open buffer if present, else from disk.
func (s *Session) DidSave(path string) {
	s.mu.Lock()
	text, ok := "", false
	if b := s.buffers[path]; b != nil {
		text, ok = b.text, true
	}
	s.mu.Unlock()
	if !ok {
		raw, err := os.ReadFile(path)
		if err != nil {
			return
		}
		text = string(raw)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reindexFileLocked(path, text)
}

// ExternalChange reindexes a file touched outside the editor (watcher-driven).
func (s *Session) ExternalChange(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		s.mu.Lock()
		s.dropPathLocked(path)
		s.mu.Unlock()
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reindexFileLocked(path, string(raw))
}

// edit updates the buffer (parsing once) and reindexes the file.
func (s *Session) edit(path, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffers[path] = &buffer{text: text, result: parser.Parse(text)}
	s.reindexFileLocked(path, text)
}

// reindexFileLocked patches the index for one path only. Identical bytes are a
// no-op so editor-save + watcher-of-the-same-write does not double-index.
func (s *Session) reindexFileLocked(path, text string) {
	if prev, ok := s.lastIndexed[path]; ok && prev == text {
		return
	}
	origin, rel, ok := s.locate(path)
	if !ok {
		return
	}
	s.dropPathLocked(path)
	defs, refs, edges, locEng := model.IndexFile(s.GameID, path, rel, text, origin)
	s.index.Defs = append(s.index.Defs, defs...)
	s.index.Refs = append(s.index.Refs, refs...)
	s.index.Edges = append(s.index.Edges, edges...)
	for k, v := range locEng {
		s.index.Loc[k] = v
	}
	s.lastIndexed[path] = text
	s.reindexN++
}

// dropPathLocked removes every index entry sourced from path.
func (s *Session) dropPathLocked(path string) {
	s.index.Defs = keepDefs(s.index.Defs, path)
	s.index.Refs = keepRefs(s.index.Refs, path)
	s.index.Edges = keepEdges(s.index.Edges, path)
}

// Resolve returns the effective definition of key (winner across mods + vanilla).
func (s *Session) Resolve(key string) *model.Def {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return model.Winner(s.index.LookupAll(key, s.cache), s.order)
}

// Mods returns the workspace mods in the order they were supplied.
func (s *Session) Mods() []model.ModInput {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mods
}

// Locate finds the mod origin and root-relative path for an absolute file path.
func (s *Session) Locate(path string) (origin, rel string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.locate(path)
}

// Text returns the open buffer text, or ok=false if the file is not open.
func (s *Session) Text(path string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if b := s.buffers[path]; b != nil {
		return b.text, true
	}
	return "", false
}

// FileText returns the buffer text if open, otherwise the on-disk decoded text.
func (s *Session) FileText(path string) string {
	if t, ok := s.Text(path); ok {
		return t
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text, _ := parser.Decode(raw)
	return text
}

// SetWiki stores the merged wiki-doc map (lowercase keys) for hover fallback.
func (s *Session) SetWiki(docs map[string]string) {
	s.mu.Lock()
	s.wiki = docs
	s.mu.Unlock()
}

// Wiki returns cached wiki prose for key, or "".
func (s *Session) Wiki(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.wiki == nil {
		return ""
	}
	return s.wiki[strings.ToLower(key)]
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

// locate finds the mod origin and root-relative path for an absolute file path.
func (s *Session) locate(path string) (origin, rel string, ok bool) {
	for _, m := range s.mods {
		if r, err := filepath.Rel(m.Root, path); err == nil && !strings.HasPrefix(r, "..") {
			return m.Origin, r, true
		}
	}
	return "", "", false
}

func keepDefs(in []model.Def, drop string) []model.Def {
	out := in[:0]
	for _, d := range in {
		if d.Path != drop {
			out = append(out, d)
		}
	}
	return out
}

func keepRefs(in []model.Ref, drop string) []model.Ref {
	out := in[:0]
	for _, r := range in {
		if r.Path != drop {
			out = append(out, r)
		}
	}
	return out
}

func keepEdges(in []model.Edge, drop string) []model.Edge {
	out := in[:0]
	for _, e := range in {
		if e.Path != drop {
			out = append(out, e)
		}
	}
	return out
}

// orderMap turns a load-order slice into origin -> rank.
func orderMap(order []string) map[string]int {
	m := make(map[string]int, len(order))
	for i, o := range order {
		m[o] = i
	}
	return m
}
