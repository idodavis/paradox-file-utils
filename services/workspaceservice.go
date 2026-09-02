// Package services provides backend services for the Paradox Modding Tools application.
package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"

	"github.com/google/uuid"
)

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

// WorkspaceService manages workspaces, game installs, and mods.
type WorkspaceService struct {
	Store   *Store
	Session *SessionService
}

// ListGames returns the supported-game table and vanilla origin token.
func (w *WorkspaceService) ListGames() game.GameList {
	return game.GameList{OriginVanilla: game.OriginVanilla, Games: game.All()}
}

// ListGameInstalls returns all installs for a game.
func (w *WorkspaceService) ListGameInstalls(gameID string) ([]GameInstall, error) {
	var out []GameInstall
	w.Store.Read(func(c *Config) {
		for _, inst := range c.Installs {
			if inst.GameID == gameID {
				out = append(out, inst)
			}
		}
	})
	return out, nil
}

// AddGameInstall adds a new game installation. version empty → detected or latest.
func (w *WorkspaceService) AddGameInstall(gameID, name, path, version string) (*GameInstall, error) {
	if game.Get(gameID) == nil {
		return nil, fmt.Errorf("unknown game %s", gameID)
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	detected := game.ReadGameVersion(path)
	if version == "" {
		version = detected
	}
	if version == "" {
		version = "latest"
	}
	inst := GameInstall{
		ID: uuid.New().String(), GameID: gameID, Name: name, Path: path,
		Version: version, VersionDetected: detected, CreatedAt: nowUTC(),
	}
	err := w.Store.Mutate(func(c *Config) error {
		c.Installs = append(c.Installs, inst)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("insert install: %w", err)
	}
	return &inst, nil
}

// MarkBrokenPaths checks all installs and mods, marks missing paths as broken.
func (w *WorkspaceService) MarkBrokenPaths() error {
	return w.Store.Mutate(func(c *Config) error {
		for i := range c.Installs {
			_, err := os.Stat(c.Installs[i].Path)
			c.Installs[i].IsBroken = err != nil
		}
		for i := range c.Workspaces {
			for j := range c.Workspaces[i].Mods {
				_, err := os.Stat(c.Workspaces[i].Mods[j].Path)
				c.Workspaces[i].Mods[j].IsBroken = err != nil
			}
		}
		return nil
	})
}

// ListWorkspaces returns workspaces, optionally filtered by game. Pass "" for all.
func (w *WorkspaceService) ListWorkspaces(gameID string) ([]Workspace, error) {
	var out []Workspace
	w.Store.Read(func(c *Config) {
		for _, ws := range c.Workspaces {
			if gameID == "" || ws.GameID == gameID {
				out = append(out, cloneWorkspace(ws))
			}
		}
	})
	return out, nil
}

// GetWorkspace returns a workspace by ID.
func (w *WorkspaceService) GetWorkspace(id string) (*Workspace, error) {
	if id == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	var ws *Workspace
	w.Store.Read(func(c *Config) {
		if found := findWorkspace(c, id); found != nil {
			cp := cloneWorkspace(*found)
			ws = &cp
		}
	})
	if ws == nil {
		return nil, fmt.Errorf("workspace not found")
	}
	return ws, nil
}

// CreateWorkspace creates a new workspace and default staging dir.
func (w *WorkspaceService) CreateWorkspace(
	gameID, name, installID string, tags []string,
) (*Workspace, error) {
	if game.Get(gameID) == nil {
		return nil, fmt.Errorf("unknown game %s", gameID)
	}
	id := uuid.New().String()
	stagingDir, err := w.defaultStagingDir(id)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, fmt.Errorf("create staging dir: %w", err)
	}
	ws := Workspace{
		ID: id, GameID: gameID, Name: name, InstallID: installID,
		StagingDir: stagingDir, Tags: tags, CreatedAt: nowUTC(),
	}
	err = w.Store.Mutate(func(c *Config) error {
		c.Workspaces = append(c.Workspaces, ws)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("insert workspace: %w", err)
	}
	return &ws, nil
}

// DeleteWorkspace removes a workspace from PMT. Mod folders on disk stay.
func (w *WorkspaceService) DeleteWorkspace(id string) error {
	if id == "" {
		return fmt.Errorf("workspace id is required")
	}
	err := w.Store.Mutate(func(c *Config) error {
		for i, ws := range c.Workspaces {
			if ws.ID == id {
				c.Workspaces = append(c.Workspaces[:i], c.Workspaces[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("workspace not found")
	})
	if err != nil {
		return err
	}
	if w.Session != nil {
		w.Session.pool().Drop(id)
	}
	return nil
}

// UpdateWorkspace updates workspace fields.
func (w *WorkspaceService) UpdateWorkspace(
	id, name, installID, stagingDir string, tags []string,
) error {
	var rebuild bool
	err := w.Store.Mutate(func(c *Config) error {
		ws := findWorkspace(c, id)
		if ws == nil {
			return fmt.Errorf("workspace not found")
		}
		rebuild = ws.InstallID != installID || ws.StagingDir != stagingDir
		ws.Name, ws.InstallID, ws.StagingDir, ws.Tags = name, installID, stagingDir, tags
		return nil
	})
	if err != nil {
		return err
	}
	if rebuild {
		return w.rebuildSession(id)
	}
	return nil
}

func (w *WorkspaceService) rebuildSession(id string) error {
	if w.Session == nil {
		return nil
	}
	w.Session.pool().Drop(id)
	return w.Session.EnsureSession(id)
}

// AddWorkspaceMod adds a mod to a workspace.
func (w *WorkspaceService) AddWorkspaceMod(
	workspaceID, name, path string, tags []string, thumbnail string,
) (*WorkspaceMod, error) {
	_, statErr := os.Stat(path)
	mod := WorkspaceMod{
		ID: uuid.New().String(), Name: name, Path: path, Tags: tags,
		Thumbnail: thumbnail, IsBroken: statErr != nil, CreatedAt: nowUTC(),
	}
	err := w.Store.Mutate(func(c *Config) error {
		ws := findWorkspace(c, workspaceID)
		if ws == nil {
			return fmt.Errorf("workspace not found")
		}
		mod.SortOrder = len(ws.Mods)
		ws.Mods = append(ws.Mods, mod)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("insert mod: %w", err)
	}
	if err := w.rebuildSession(workspaceID); err != nil {
		return &mod, err
	}
	return &mod, nil
}

// DefaultModParent is Documents/Paradox Interactive/<game>/mod, or "".
func (w *WorkspaceService) DefaultModParent(gameID string) string {
	return game.DefaultModParent(gameID)
}

// CreateMod writes a new mod skeleton under parentDir and returns its path.
func (w *WorkspaceService) CreateMod(
	gameID, parentDir, name, locLang, supportedVersion, description, thumbnailSrc string,
) (string, error) {
	if game.Get(gameID) == nil {
		return "", fmt.Errorf("unknown game %s", gameID)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if parentDir == "" {
		parentDir = game.DefaultModParent(gameID)
	}
	if parentDir == "" {
		return "", fmt.Errorf("parent folder is required")
	}
	root := filepath.Join(parentDir, game.ModSlug(name))
	if err := game.WriteNewMod(
		gameID, root, name, supportedVersion, locLang, description, thumbnailSrc,
	); err != nil {
		return "", err
	}
	return root, nil
}

// ListWorkspaceMods returns all mods for a workspace, sorted by SortOrder then Name.
func (w *WorkspaceService) ListWorkspaceMods(workspaceID string) ([]WorkspaceMod, error) {
	var mods []WorkspaceMod
	var ok bool
	w.Store.Read(func(c *Config) {
		if ws := findWorkspace(c, workspaceID); ws != nil {
			ok = true
			mods = append([]WorkspaceMod(nil), ws.Mods...)
			sortWorkspaceMods(mods)
		}
	})
	if !ok {
		return nil, fmt.Errorf("workspace not found")
	}
	return mods, nil
}

// RemoveWorkspaceMod detaches a mod from a workspace.
func (w *WorkspaceService) RemoveWorkspaceMod(workspaceID, modID string) error {
	err := w.Store.Mutate(func(c *Config) error {
		ws := findWorkspace(c, workspaceID)
		if ws == nil {
			return fmt.Errorf("workspace not found")
		}
		kept := ws.Mods[:0]
		found := false
		for _, m := range ws.Mods {
			if m.ID == modID {
				found = true
				continue
			}
			kept = append(kept, m)
		}
		if !found {
			return fmt.Errorf("mod not found")
		}
		ws.Mods = kept
		return nil
	})
	if err != nil {
		return err
	}
	return w.rebuildSession(workspaceID)
}

// ReorderWorkspaceMods writes SortOrder from the given id list.
func (w *WorkspaceService) ReorderWorkspaceMods(workspaceID string, modIDs []string) error {
	err := w.Store.Mutate(func(c *Config) error {
		ws := findWorkspace(c, workspaceID)
		if ws == nil {
			return fmt.Errorf("workspace not found")
		}
		if len(modIDs) != len(ws.Mods) {
			return fmt.Errorf("mod id set incomplete")
		}
		byID := make(map[string]*WorkspaceMod, len(ws.Mods))
		for i := range ws.Mods {
			byID[ws.Mods[i].ID] = &ws.Mods[i]
		}
		for i, id := range modIDs {
			m, ok := byID[id]
			if !ok {
				return fmt.Errorf("mod not found")
			}
			m.SortOrder = i
			delete(byID, id)
		}
		if len(byID) != 0 {
			return fmt.Errorf("mod id set incomplete")
		}
		return nil
	})
	if err != nil {
		return err
	}
	return w.rebuildSession(workspaceID)
}

// UpdateWorkspaceMod updates a mod's name, tags, and optional color override.
func (w *WorkspaceService) UpdateWorkspaceMod(
	workspaceID, modID, name string, tags []string, color, thumbnail string,
) error {
	return w.Store.Mutate(func(c *Config) error {
		ws := findWorkspace(c, workspaceID)
		if ws == nil {
			return fmt.Errorf("workspace not found")
		}
		for i := range ws.Mods {
			if ws.Mods[i].ID == modID {
				if name != "" {
					ws.Mods[i].Name = name
				}
				ws.Mods[i].Tags = tags
				ws.Mods[i].Color = color
				ws.Mods[i].Thumbnail = thumbnail
				return nil
			}
		}
		return fmt.Errorf("mod not found")
	})
}

// SaveIdeSession stores open editor paths for a workspace.
func (w *WorkspaceService) SaveIdeSession(workspaceID string, files []string, active string) error {
	return w.Store.Mutate(func(c *Config) error {
		ws := findWorkspace(c, workspaceID)
		if ws == nil {
			return fmt.Errorf("workspace not found")
		}
		ws.IdeOpenFiles = append([]string(nil), files...)
		ws.IdeActiveFile = active
		return nil
	})
}

var legalDefaultTools = map[string]bool{
	"":              true,
	"workspace-ide": true,
	"event-graph":   true,
	"conflicts":     true,
	"loc-coverage":  true,
	"patcher":       true,
}

// UpdateWorkspacePrefs sets IDE persist, default landing page, and origin colors.
func (w *WorkspaceService) UpdateWorkspacePrefs(
	workspaceID string, resetIdeOnOpen bool, defaultTool, gameColor, stagingColor string,
) error {
	if !legalDefaultTools[defaultTool] {
		return fmt.Errorf("invalid default tool")
	}
	return w.Store.Mutate(func(c *Config) error {
		ws := findWorkspace(c, workspaceID)
		if ws == nil {
			return fmt.Errorf("workspace not found")
		}
		ws.ResetIdeOnOpen = resetIdeOnOpen
		ws.DefaultTool = defaultTool
		ws.GameColor = gameColor
		ws.StagingColor = stagingColor
		return nil
	})
}

// CountWorkspacesUsingInstall returns names of workspaces on this install.
func (w *WorkspaceService) CountWorkspacesUsingInstall(installID string) []string {
	var names []string
	w.Store.Read(func(c *Config) {
		names = workspacesUsingInstall(c, installID)
	})
	return names
}

func workspacesUsingInstall(c *Config, installID string) []string {
	var names []string
	for _, ws := range c.Workspaces {
		if ws.InstallID == installID {
			names = append(names, ws.Name)
		}
	}
	return names
}

// DetectGameVersion reads version from launcher-settings.json.
func (w *WorkspaceService) DetectGameVersion(path string) string {
	return game.ReadGameVersion(path)
}

// FindGameInstalls returns Steam-detected installs for a game.
func (w *WorkspaceService) FindGameInstalls(gameID string) []game.DetectedInstall {
	return game.FindInstalls(gameID)
}

// UpdateGameInstall updates path and docs_path, then refreshes version.
func (w *WorkspaceService) UpdateGameInstall(id, path, docsPath string) (*GameInstall, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}
	}
	var out *GameInstall
	err := w.Store.Mutate(func(c *Config) error {
		inst := findInstall(c, id)
		if inst == nil {
			return fmt.Errorf("install not found")
		}
		if path != "" {
			inst.Path = path
		}
		inst.DocsPath = docsPath
		inst.VersionDetected = game.ReadGameVersion(inst.Path)
		cp := *inst
		out = &cp
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("update install: %w", err)
	}
	return out, nil
}

func (w *WorkspaceService) defaultStagingDir(workspaceID string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(configDir, appConfigDirName, "workspaces", workspaceID, "staging"), nil
}

// EnsureStagingDir creates the staging directory if it doesn't exist.
func (w *WorkspaceService) EnsureStagingDir(workspaceID string) (string, error) {
	ws, err := w.GetWorkspace(workspaceID)
	if err != nil {
		return "", err
	}
	dir, needPersist := ws.StagingDir, ws.StagingDir == ""
	if needPersist {
		if dir, err = w.defaultStagingDir(workspaceID); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}
	if needPersist {
		err = w.UpdateWorkspace(ws.ID, ws.Name, ws.InstallID, dir, ws.Tags)
		if err != nil {
			return "", fmt.Errorf("persist staging dir: %w", err)
		}
	}
	return dir, nil
}

// InstallCacheInfo is VanillaCache persist metadata for Settings/wizard cards.
type InstallCacheInfo struct {
	ScannedAt string `json:"scannedAt"`
}

// GetInstallCacheInfo returns scannedAt for the install's VanillaCache, or empty.
func (w *WorkspaceService) GetInstallCacheInfo(installID string) (*InstallCacheInfo, error) {
	var inst *GameInstall
	w.Store.Read(func(c *Config) {
		if found := findInstall(c, installID); found != nil {
			cp := *found
			inst = &cp
		}
	})
	if inst == nil {
		return nil, fmt.Errorf("install not found")
	}
	ver := inst.Version
	if ver == "" {
		ver = "latest"
	}
	return &InstallCacheInfo{ScannedAt: catalog.PeekCacheScannedAt(inst.ID, ver)}, nil
}

// DeleteGameInstall removes an install if no workspace uses it.
func (w *WorkspaceService) DeleteGameInstall(id string) (inUse []string, err error) {
	w.Store.Read(func(c *Config) {
		inUse = workspacesUsingInstall(c, id)
	})
	if len(inUse) > 0 {
		return inUse, fmt.Errorf("install in use by: %s", strings.Join(inUse, ", "))
	}
	if err := catalog.DropVanillaFiles(id); err != nil {
		return nil, err
	}
	return nil, w.Store.Mutate(func(c *Config) error {
		kept := c.Installs[:0]
		for _, inst := range c.Installs {
			if inst.ID != id {
				kept = append(kept, inst)
			}
		}
		c.Installs = kept
		return nil
	})
}

// SetInstallVersion pins the cache-key version (user override; empty → latest).
func (w *WorkspaceService) SetInstallVersion(id, version string) error {
	if version == "" {
		version = "latest"
	}
	return w.Store.Mutate(func(c *Config) error {
		inst := findInstall(c, id)
		if inst == nil {
			return fmt.Errorf("install not found")
		}
		inst.Version = version
		return nil
	})
}

// SetWorkspaceLocLang sets default loc language and loc-harvests if a session is live.
func (w *WorkspaceService) SetWorkspaceLocLang(id, lang string) error {
	var inst *GameInstall
	err := w.Store.Mutate(func(c *Config) error {
		ws := findWorkspace(c, id)
		if ws == nil {
			return fmt.Errorf("workspace not found")
		}
		ws.DefaultLocLang = lang
		if found := findInstall(c, ws.InstallID); found != nil {
			cp := *found
			inst = &cp
		}
		return nil
	})
	if err != nil || w.Session == nil {
		return err
	}
	live := w.Session.pool().Get(id)
	if live == nil {
		return nil
	}
	live.SetDefaultLang(lang)
	if inst == nil {
		return nil
	}
	vloc, err := loadVanillaLoc(
		context.Background(), inst.ID, inst.GameID, inst.Path, inst.Version, lang)
	if err != nil {
		return err
	}
	live.ReplaceVanillaLoc(vloc)
	return nil
}
