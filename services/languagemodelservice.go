// languagemodelservice.go is the Wails RPC shell for the language engine.
// Methods recover panics and delegate to session/lsp/graph; they hold no feature logic.

package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/wailsapp/wails/v3/pkg/application"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/graph"
	"paradox-modding-tools/services/internal/lsp"
	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/repos"
	"paradox-modding-tools/services/internal/session"
)

// LanguageModelService exposes the language engine to the frontend over Wails.
type LanguageModelService struct {
	DB *sqlx.DB

	once sync.Once
	p    *session.Pool

	mu         sync.Mutex
	scanCancel context.CancelFunc
}

// ModelStatus reports whether a workspace session is live and how large it is.
type ModelStatus struct {
	Live        bool   `json:"live"`
	Present     bool   `json:"present"`
	WorkspaceID string `json:"workspaceId"`
	GameID      string `json:"gameId"`
	DefCount    int    `json:"defCount"`
	CacheAge    string `json:"cacheAge"`
}

// LanguageHealth is the compact Library/IDE index status strip.
type LanguageHealth struct {
	InstallOk    bool   `json:"installOk"`
	InstallPath  string `json:"installPath"`
	GameVersion  string `json:"gameVersion"`
	CacheVersion string `json:"cacheVersion"`
	CacheStale   bool   `json:"cacheStale"`
	ScannedAt    string `json:"scannedAt"`
	DocsPresent  bool   `json:"docsPresent"`
	DocsPath     string `json:"docsPath"`
	IndexReady   bool   `json:"indexReady"`
	DefCount     int    `json:"defCount"`
	DumpHint     string `json:"dumpHint"`
}

func recoverErr(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("%v", r)
	}
}

func (s *LanguageModelService) pool() *session.Pool {
	s.once.Do(func() {
		s.p = session.NewPool(s.buildSession, emitLang)
	})
	return s.p
}

func emitLang(event string, data any) {
	if app := application.Get(); app != nil {
		app.Event.Emit(event, data)
	}
}

func (s *LanguageModelService) repo() *repos.WorkspaceRepository {
	return repos.NewWorkspaceRepository(s.DB)
}

func (s *LanguageModelService) buildSession(id string) (*session.Session, error) {
	r := s.repo()
	ws, err := r.GetWorkspace(id)
	if err != nil {
		return nil, err
	}
	var cache *model.Cache
	if ws.InstallID != "" {
		cache, _ = model.LoadCache(id)
	}
	mods, err := r.ListMods(id)
	if err != nil {
		return nil, err
	}
	inputs := make([]model.ModInput, 0, len(mods))
	for i, m := range mods {
		if m.IsBroken {
			continue
		}
		inputs = append(inputs, model.ModInput{Origin: m.ID, Root: m.Path, Order: i})
	}
	sess := session.New(id, ws.GameID, cache, inputs)
	if docs, err := LoadCachedDocs(ws.GameID); err == nil {
		sess.SetWiki(docs)
	}
	_ = sess.StartWatch()
	_ = model.SaveIndex(sess.Index())
	return sess, nil
}

func (s *LanguageModelService) sess(id string) (*session.Session, error) {
	return s.pool().EnsureSession(id)
}

// EnsureSession builds or returns the live session for workspaceID.
func (s *LanguageModelService) EnsureSession(workspaceID string) (err error) {
	defer recoverErr(&err)
	_, err = s.sess(workspaceID)
	return err
}

// Cancel cancels an in-flight install scan.
func (s *LanguageModelService) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scanCancel != nil {
		s.scanCancel()
		s.scanCancel = nil
	}
}

// RebuildWorkspaceSemantics scans the workspace install into a vanilla Cache.
func (s *LanguageModelService) RebuildWorkspaceSemantics(workspaceID string) (_ *model.Cache, err error) {
	defer recoverErr(&err)
	r := s.repo()
	ws, err := r.GetWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	if ws.InstallID == "" {
		return nil, fmt.Errorf("workspace has no game install")
	}
	inst, err := r.GetInstall(ws.InstallID)
	if err != nil {
		return nil, err
	}
	s.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.scanCancel = cancel
	s.mu.Unlock()
	defer cancel()

	c, err := model.Scan(ctx, workspaceID, ws.GameID, inst.Path, scriptDocsDir(ws.GameID),
		func(pct int, msg string) {
			emitLang("lang:scan-progress", map[string]any{
				"workspaceId": workspaceID,
				"pct":         pct,
				"msg":         msg,
			})
		})
	if err != nil {
		return nil, err
	}
	if err := model.SaveCache(c); err != nil {
		return nil, err
	}
	if live := s.pool().Get(workspaceID); live != nil {
		live.ReplaceCache(c)
	}
	return c, nil
}

// GetModelStatus reports live session (or on-disk index) readiness.
func (s *LanguageModelService) GetModelStatus(workspaceID string) (_ *ModelStatus, err error) {
	defer recoverErr(&err)
	st := &ModelStatus{WorkspaceID: workspaceID}
	if live := s.pool().Get(workspaceID); live != nil {
		st.Live, st.Present = true, true
		st.GameID = live.GameID
		if idx := live.Index(); idx != nil {
			st.DefCount = len(idx.Defs)
		}
		if c := live.Cache(); c != nil {
			st.CacheAge = c.ScannedAt
		}
		return st, nil
	}
	if idx, e := model.LoadIndex(workspaceID); e == nil && idx != nil {
		st.Present = true
		st.DefCount = len(idx.Defs)
	}
	if ws, e := s.repo().GetWorkspace(workspaceID); e == nil {
		st.GameID = ws.GameID
	}
	return st, nil
}

// GetLanguageHealth returns install/cache/index status for the workspace.
func (s *LanguageModelService) GetLanguageHealth(workspaceID string) (_ *LanguageHealth, err error) {
	defer recoverErr(&err)
	h := &LanguageHealth{}
	ws, err := s.repo().GetWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	docs := scriptDocsDir(ws.GameID)
	h.DocsPath = docs
	h.DocsPresent = docs != ""
	if !h.DocsPresent {
		h.DumpHint = "Open the game, run script_docs in the console, then Rescan."
	}
	if ws.InstallID != "" {
		if inst, e := s.repo().GetInstall(ws.InstallID); e == nil {
			h.InstallPath = inst.Path
			h.GameVersion = inst.Version
			if fi, e := os.Stat(inst.Path); e == nil && fi.IsDir() {
				h.InstallOk = true
			}
		}
	}
	if c, e := model.LoadCache(workspaceID); e == nil {
		applyCacheHealth(h, c)
	}
	if live := s.pool().Get(workspaceID); live != nil {
		h.IndexReady = true
		if idx := live.Index(); idx != nil {
			h.DefCount = len(idx.Defs)
		}
		applyCacheHealth(h, live.Cache())
	}
	return h, nil
}

func applyCacheHealth(h *LanguageHealth, c *model.Cache) {
	if c == nil {
		return
	}
	h.CacheVersion = c.GameVersion
	h.ScannedAt = c.ScannedAt
	h.CacheStale = c.GameVersion != h.GameVersion && h.GameVersion != ""
}

// LaunchGameDebug starts the game with debug flags (best-effort).
func (s *LanguageModelService) LaunchGameDebug(workspaceID string) (err error) {
	defer recoverErr(&err)
	ws, err := s.repo().GetWorkspace(workspaceID)
	if err != nil {
		return err
	}
	if ws.InstallID == "" {
		return fmt.Errorf("no install")
	}
	inst, err := s.repo().GetInstall(ws.InstallID)
	if err != nil {
		return err
	}
	exe := findGameExe(ws.GameID, inst.Path)
	if exe == "" {
		return fmt.Errorf("game executable not found in %s", inst.Path)
	}
	cmd := exec.Command(exe, "-debug_mode")
	cmd.Dir = filepath.Dir(exe)
	return cmd.Start()
}

func findGameExe(gameID, install string) string {
	names := map[string][]string{
		"ck3":  {"ck3.exe", "ck3"},
		"vic3": {"victoria3.exe", "victoria3"},
		"eu5":  {"eu5.exe", "eu5"},
	}
	for _, n := range names[gameID] {
		for _, p := range []string{filepath.Join(install, n), filepath.Join(install, "binaries", n), filepath.Join(install, "bin", n)} {
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return p
			}
		}
	}
	return ""
}

// DidOpen syncs an editor buffer into the session.
func (s *LanguageModelService) DidOpen(workspaceID, path, text, languageID string, version int) (err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return err
	}
	sess.DidOpen(path, text)
	return nil
}

// DidChange syncs buffer text into the session.
func (s *LanguageModelService) DidChange(workspaceID, path, text, languageID string, version int) (err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return err
	}
	sess.DidChange(path, text)
	return nil
}

// DidClose drops an editor buffer overlay.
func (s *LanguageModelService) DidClose(workspaceID, path string) (err error) {
	defer recoverErr(&err)
	if live := s.pool().Get(workspaceID); live != nil {
		live.DidClose(path)
	}
	return nil
}

// DidSave reindexes path from its buffer or disk.
func (s *LanguageModelService) DidSave(workspaceID, path string) (err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return err
	}
	sess.DidSave(path)
	return nil
}

// Diagnose returns diagnostics for a file.
func (s *LanguageModelService) Diagnose(workspaceID, path string) (_ []lsp.Diagnostic, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.Diagnose(sess, path), nil
}

// Hover returns hover info at a 0-based UTF-8 position.
func (s *LanguageModelService) Hover(workspaceID, path string, line, character int) (_ *lsp.HoverResult, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.Hover(sess, path, line, character), nil
}

// Complete returns completion items at a 0-based UTF-8 position.
func (s *LanguageModelService) Complete(workspaceID, path string, line, character int) (_ []lsp.CompletionItem, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.Complete(sess, path, line, character), nil
}

// Definition returns go-to-definition locations.
func (s *LanguageModelService) Definition(workspaceID, path string, line, character int) (_ []lsp.Location, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.Definition(sess, path, line, character), nil
}

// References returns reference locations.
func (s *LanguageModelService) References(workspaceID, path string, line, character int) (_ []lsp.Location, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.References(sess, path, line, character), nil
}

// Rename returns a workspace edit for the identifier at pos.
func (s *LanguageModelService) Rename(workspaceID, path string, line, character int, newName string) (_ *lsp.WorkspaceEdit, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.Rename(sess, path, line, character, newName), nil
}

// FormatDocument returns conservative format edits.
func (s *LanguageModelService) FormatDocument(workspaceID, path string) (_ []lsp.TextEdit, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.FormatDocument(sess, path), nil
}

// FoldingRanges returns foldable block ranges.
func (s *LanguageModelService) FoldingRanges(workspaceID, path string) (_ []lsp.FoldingRange, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.FoldingRanges(sess, path), nil
}

// SignatureHelp returns a signature tooltip.
func (s *LanguageModelService) SignatureHelp(workspaceID, path string, line, character int) (_ *lsp.SignatureHelp, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.SignatureAt(sess, path, line, character), nil
}

// CodeActions returns quick-fixes for the file.
func (s *LanguageModelService) CodeActions(workspaceID, path string) (_ []lsp.CodeAction, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.CodeActions(sess, path), nil
}

// DocumentSymbols returns top-level symbols in a file.
func (s *LanguageModelService) DocumentSymbols(workspaceID, path string) (_ []lsp.SymbolInformation, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.DocumentSymbols(sess, path), nil
}

// WorkspaceSymbols searches definitions in the session.
func (s *LanguageModelService) WorkspaceSymbols(workspaceID, query string) (_ []lsp.SymbolInformation, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.WorkspaceSymbols(sess, query), nil
}

// GetEventGraph returns the defs-first event graph (no coordinates).
func (s *LanguageModelService) GetEventGraph(workspaceID string, params graph.EventGraphParams) (_ graph.EventGraph, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return graph.EventGraph{}, err
	}
	return graph.Graph(sess, params), nil
}

// GetEventDetail returns inspector content for one event id.
func (s *LanguageModelService) GetEventDetail(workspaceID, eventID string) (_ *graph.EventDetail, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return graph.Detail(sess, eventID), nil
}

// GetOverrides returns FIOS/LIOS override rows for the conflict monitor.
func (s *LanguageModelService) GetOverrides(workspaceID string) (_ []model.OverrideRow, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return graph.OverrideRows(sess), nil
}

// GetLocCoverage returns per-language localization health for workspace mods.
func (s *LanguageModelService) GetLocCoverage(workspaceID string) (_ []graph.LocCoverage, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return graph.Coverage(sess), nil
}

// LookupLoc returns the english loc text and site for key.
func (s *LanguageModelService) LookupLoc(workspaceID, key string) (_ *graph.LocLookup, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return nil, err
	}
	return graph.Lookup(sess, key), nil
}

// GetDependencies returns dependents and inner references for one definition.
func (s *LanguageModelService) GetDependencies(workspaceID, name, kind string) (_ graph.Dependencies, err error) {
	defer recoverErr(&err)
	sess, err := s.sess(workspaceID)
	if err != nil {
		return graph.Dependencies{}, err
	}
	return graph.Deps(sess, name, kind), nil
}

// scriptDocsDir resolves the per-game in-game script_docs dump folder, or "".
func scriptDocsDir(gameID string) string {
	info := game.Get(gameID)
	if info == nil {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, "Documents", "Paradox Interactive", info.DocsFolderName, info.ScriptDocsSubdir)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return ""
	}
	return dir
}
