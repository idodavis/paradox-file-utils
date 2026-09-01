// sessionservice.go is the Wails RPC shell for the session pool, install scan,
// and language health. Feature logic stays in session / catalog.

package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/session"
)

// SessionService owns the session pool and install-scan RPC.
type SessionService struct {
	Store *Store

	once sync.Once
	p    *session.Pool

	mu         sync.Mutex
	scanCancel context.CancelFunc
}

// ModelStatus reports whether a workspace session is live.
type ModelStatus struct {
	Live bool `json:"live"`
}

// LanguageHealth is the compact Library/IDE index status strip.
type LanguageHealth struct {
	InstallID   string `json:"installId"`
	InstallOk   bool   `json:"installOk"`
	GameVersion string `json:"gameVersion"`
	CacheStale  bool   `json:"cacheStale"`
	ScannedAt   string `json:"scannedAt"`
	DocsPresent bool   `json:"docsPresent"`
	IndexReady  bool   `json:"indexReady"`
	DefCount    int    `json:"defCount"`
	DumpHint    string `json:"dumpHint"`
}

func recoverErr(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("%v", r)
	}
}

func withSession[T any](s *SessionService, workspaceID string, fn func(*session.Session) T) (v T, err error) {
	defer recoverErr(&err)
	sess, e := s.sess(workspaceID)
	if e != nil {
		err = e
		return
	}
	return fn(sess), nil
}

func withSessionDo(s *SessionService, workspaceID string, fn func(*session.Session)) error {
	_, err := withSession(s, workspaceID, func(sess *session.Session) struct{} {
		fn(sess)
		return struct{}{}
	})
	return err
}

func (s *SessionService) pool() *session.Pool {
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

func (s *SessionService) buildSession(id string) (*session.Session, error) {
	var ws Workspace
	var inst *GameInstall
	var found bool
	s.Store.Read(func(c *Config) {
		if w := findWorkspace(c, id); w != nil {
			ws = cloneWorkspace(*w)
			found = true
			if w.InstallID != "" {
				if i := findInstall(c, w.InstallID); i != nil {
					cp := *i
					inst = &cp
				}
			}
		}
	})
	if !found {
		return nil, fmt.Errorf("workspace not found")
	}
	lang := ws.DefaultLocLang
	if lang == "" {
		lang = "english"
	}
	var cache *catalog.VanillaCache
	var vloc *catalog.VanillaLoc
	if inst != nil {
		ver := inst.Version
		if ver == "" {
			ver = "latest"
		}
		cache, _ = catalog.LoadCache(inst.ID, ver)
		loaded, locErr := catalog.LoadVanillaLoc(inst.ID, ver, lang)
		if locErr != nil && !os.IsNotExist(locErr) {
			return nil, locErr
		}
		vloc = loaded
	}
	inputs := make([]catalog.ModInput, 0, len(ws.Mods))
	mods := append([]WorkspaceMod(nil), ws.Mods...)
	sortWorkspaceMods(mods)
	for i, m := range mods {
		if m.IsBroken {
			continue
		}
		inputs = append(inputs, catalog.ModInput{
			Origin: m.ID, Root: m.Path, Order: i, Name: m.Name,
		})
	}
	sess := session.NewWithLoc(id, ws.GameID, lang, cache, vloc, inputs)
	sess.SetFileNotify(func(path string, deleted bool) {
		emitLang("fs:changed", map[string]any{"path": path, "deleted": deleted})
	})
	_ = sess.StartWatch()
	return sess, nil
}

func (s *SessionService) sess(id string) (*session.Session, error) {
	return s.pool().EnsureSession(id)
}

// EnsureSession builds or returns the live session for workspaceID.
func (s *SessionService) EnsureSession(workspaceID string) (err error) {
	defer recoverErr(&err)
	if workspaceID == "" {
		return fmt.Errorf("workspace id is required")
	}
	_, err = s.sess(workspaceID)
	return err
}

func (s *SessionService) cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scanCancel != nil {
		s.scanCancel()
		s.scanCancel = nil
	}
}

// RebuildInstallSemantics scans an install into VanillaCache + loc sidecar.
func (s *SessionService) RebuildInstallSemantics(installID string) (_ *catalog.VanillaCache, err error) {
	defer recoverErr(&err)
	var inst GameInstall
	var ok bool
	s.Store.Read(func(c *Config) {
		if found := findInstall(c, installID); found != nil {
			inst, ok = *found, true
		}
	})
	if !ok {
		return nil, fmt.Errorf("install not found")
	}
	ver := inst.Version
	if ver == "" {
		ver = "latest"
	}
	docs := inst.DocsPath
	if docs == "" {
		docs = scriptDocsDir(inst.GameID)
	}
	s.cancel()
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.scanCancel = cancel
	s.mu.Unlock()
	defer cancel()

	c, vloc, err := catalog.Scan(ctx, inst.ID, inst.GameID, inst.Path, ver, docs, "english",
		func(pct int, msg string) {
			emitLang("lang:scan-progress", map[string]any{
				"installId": installID,
				"pct":       pct,
				"msg":       msg,
			})
		})
	if err != nil {
		return nil, err
	}
	if err := catalog.SaveCache(c); err != nil {
		return nil, err
	}
	if err := catalog.SaveVanillaLoc(inst.ID, ver, "english", vloc); err != nil {
		return nil, err
	}

	var workspaceIDs []string
	s.Store.Read(func(cfg *Config) {
		for _, ws := range cfg.Workspaces {
			if ws.InstallID == installID {
				workspaceIDs = append(workspaceIDs, ws.ID)
			}
		}
	})
	for _, wsID := range workspaceIDs {
		live := s.pool().Get(wsID)
		if live == nil {
			continue
		}
		live.ReplaceCache(c)
		lang := live.DefaultLang()
		sloc := vloc
		if lang != "english" {
			sloc, _ = loadVanillaLoc(ctx, inst.ID, inst.GameID, inst.Path, ver, lang)
		}
		live.ReplaceVanillaLoc(sloc)
	}
	emitLang("lang:cache-updated", map[string]any{"installId": installID})
	return c, nil
}

// GetModelStatus reports whether a live session exists for the workspace.
func (s *SessionService) GetModelStatus(workspaceID string) (_ *ModelStatus, err error) {
	defer recoverErr(&err)
	st := &ModelStatus{}
	if live := s.pool().Get(workspaceID); live != nil {
		st.Live = true
	}
	return st, nil
}

// GetLanguageHealth returns install/cache/session status for the workspace.
func (s *SessionService) GetLanguageHealth(workspaceID string) (_ *LanguageHealth, err error) {
	defer recoverErr(&err)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	h := &LanguageHealth{}
	var ws Workspace
	var inst *GameInstall
	var found bool
	s.Store.Read(func(c *Config) {
		if w := findWorkspace(c, workspaceID); w != nil {
			ws = cloneWorkspace(*w)
			found = true
			if w.InstallID != "" {
				if i := findInstall(c, w.InstallID); i != nil {
					cp := *i
					inst = &cp
				}
			}
		}
	})
	if !found {
		return nil, fmt.Errorf("workspace not found")
	}
	docs := scriptDocsDir(ws.GameID)
	h.DocsPresent = docs != ""
	if !h.DocsPresent {
		h.DumpHint = "Open the game, run script_docs in the console, then Rescan."
	}
	if inst != nil {
		h.InstallID = inst.ID
		h.GameVersion = inst.Version
		if fi, e := os.Stat(inst.Path); e == nil && fi.IsDir() {
			h.InstallOk = true
		}
	}
	if live := s.pool().Get(workspaceID); live != nil {
		h.IndexReady = true
		h.DefCount = live.DefCount()
		_, scanned, ver, _ := live.CacheInfo()
		if scanned != "" {
			h.ScannedAt = scanned
		}
		if ver != "" && h.GameVersion != "" {
			h.CacheStale = ver != h.GameVersion
		}
	} else if inst != nil {
		ver := h.GameVersion
		if ver == "" {
			ver = "latest"
		}
		if c, e := catalog.LoadCache(inst.ID, ver); e == nil && c != nil {
			h.ScannedAt = c.ScannedAt
			h.CacheStale = c.GameVersion != h.GameVersion && h.GameVersion != ""
		}
	}
	return h, nil
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

// loadVanillaLoc returns the loc sidecar, harvesting and saving if it is missing.
func loadVanillaLoc(
	ctx context.Context, installID, gameID, installPath, version, lang string,
) (*catalog.VanillaLoc, error) {
	if version == "" {
		version = "latest"
	}
	if lang == "" {
		lang = "english"
	}
	if v, e := catalog.LoadVanillaLoc(installID, version, lang); e == nil {
		return v, nil
	}
	v, err := catalog.HarvestLoc(ctx, gameID, installPath, lang)
	if err != nil {
		return nil, err
	}
	_ = catalog.SaveVanillaLoc(installID, version, lang, v)
	return v, nil
}
