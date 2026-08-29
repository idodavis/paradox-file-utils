// Package services provides backend services for the Paradox Modding Tools application.
package services

import (
	"cmp"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/repos"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// WorkspaceService manages workspaces, game installs, and mods.
type WorkspaceService struct {
	DB   *sqlx.DB
	repo *repos.WorkspaceRepository
}

func (w *WorkspaceService) getRepo() *repos.WorkspaceRepository {
	if w.repo == nil {
		w.repo = repos.NewWorkspaceRepository(w.DB)
	}
	return w.repo
}

// ListGames returns supported games from the Go registry.
func (w *WorkspaceService) ListGames() ([]repos.Game, error) {
	all := game.All()
	out := make([]repos.Game, 0, len(all))
	for _, g := range all {
		out = append(out, repos.Game{
			ID:         g.ID,
			Name:       g.Name,
			WikiAPI:    g.WikiAPI,
			ScriptRoot: g.ScriptRoot,
			SteamAppID: g.SteamAppID,
		})
	}
	slices.SortFunc(out, func(a, b repos.Game) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return out, nil
}

// ListGameInstalls returns all installs for a game.
func (w *WorkspaceService) ListGameInstalls(gameID string) ([]repos.GameInstall, error) {
	out, err := w.getRepo().ListInstalls(gameID)
	if err != nil {
		return nil, fmt.Errorf("list installs: %w", err)
	}
	return out, nil
}

// AddGameInstall adds a new game installation record. Detects version if possible.
func (w *WorkspaceService) AddGameInstall(gameID, name, path string) (*repos.GameInstall, error) {
	if game.Get(gameID) == nil {
		return nil, fmt.Errorf("unknown game %s", gameID)
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	version := game.ReadGameVersion(path)
	inst := &repos.GameInstall{
		ID:        uuid.New().String(),
		GameID:    gameID,
		Name:      name,
		Path:      path,
		Version:   version,
		DocsPath:  "",
		IsBroken:  false,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := w.getRepo().InsertInstall(inst); err != nil {
		return nil, fmt.Errorf("insert install: %w", err)
	}
	return inst, nil
}

// MarkBrokenPaths checks all installs and mods, marks missing paths as broken.
func (w *WorkspaceService) MarkBrokenPaths() error {
	repo := w.getRepo()

	installs, err := repo.AllInstalls()
	if err != nil {
		return fmt.Errorf("select installs: %w", err)
	}
	for _, inst := range installs {
		broken := false
		if _, err := os.Stat(inst.Path); err != nil {
			broken = true
		}
		if err := repo.SetInstallBroken(inst.ID, broken); err != nil {
			return fmt.Errorf("update install %s: %w", inst.ID, err)
		}
	}

	mods, err := repo.AllMods()
	if err != nil {
		return fmt.Errorf("select mods: %w", err)
	}
	for _, mod := range mods {
		broken := false
		if _, err := os.Stat(mod.Path); err != nil {
			broken = true
		}
		if err := repo.SetModBroken(mod.ID, broken); err != nil {
			return fmt.Errorf("update mod %s: %w", mod.ID, err)
		}
	}
	return nil
}

// ListWorkspaces returns workspaces, optionally filtered by game. Pass "" for all.
func (w *WorkspaceService) ListWorkspaces(gameID string) ([]repos.Workspace, error) {
	out, err := w.getRepo().ListWorkspaces(gameID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	return out, nil
}

// GetWorkspace returns a workspace by ID.
func (w *WorkspaceService) GetWorkspace(id string) (*repos.Workspace, error) {
	if id == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	ws, err := w.getRepo().GetWorkspace(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("workspace not found")
		}
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	return ws, nil
}

// CreateWorkspace creates a new workspace. Sets default staging dir if not provided.
func (w *WorkspaceService) CreateWorkspace(gameID, name, installID, tagsJSON, thumbnailPath string) (*repos.Workspace, error) {
	if game.Get(gameID) == nil {
		return nil, fmt.Errorf("unknown game %s", gameID)
	}
	id := uuid.New().String()
	stagingDir, err := w.DefaultStagingDir(id)
	if err != nil {
		return nil, err
	}
	if tagsJSON == "" {
		tagsJSON = "[]"
	}
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, fmt.Errorf("create staging dir: %w", err)
	}
	ws := &repos.Workspace{
		ID:            id,
		GameID:        gameID,
		Name:          name,
		InstallID:     installID,
		StagingDir:    stagingDir,
		Tags:          tagsJSON,
		ThumbnailPath: thumbnailPath,
		IsActive:      false,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	if err := w.getRepo().InsertWorkspace(ws); err != nil {
		return nil, fmt.Errorf("insert workspace: %w", err)
	}
	return ws, nil
}

// UpdateWorkspace updates workspace fields.
func (w *WorkspaceService) UpdateWorkspace(id, name, installID, stagingDir, tagsJSON, thumbnailPath string) error {
	if tagsJSON == "" {
		tagsJSON = "[]"
	}
	if err := w.getRepo().UpdateWorkspace(id, name, installID, stagingDir, tagsJSON, thumbnailPath); err != nil {
		return fmt.Errorf("update workspace: %w", err)
	}
	return nil
}

// DeleteWorkspace removes a workspace and cascades deletes to mods and runs.
func (w *WorkspaceService) DeleteWorkspace(id string) error {
	if err := w.getRepo().DeleteWorkspace(id); err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	return nil
}

// AddWorkspaceMod adds a mod to a workspace.
func (w *WorkspaceService) AddWorkspaceMod(workspaceID, name, path string) (*repos.WorkspaceMod, error) {
	broken := false
	if _, err := os.Stat(path); err != nil {
		broken = true
	}
	mod := &repos.WorkspaceMod{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Name:        name,
		Path:        path,
		IsBroken:    broken,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := w.getRepo().InsertMod(mod); err != nil {
		return nil, fmt.Errorf("insert mod: %w", err)
	}
	return mod, nil
}

// RemoveWorkspaceMod deletes a mod from a workspace.
func (w *WorkspaceService) RemoveWorkspaceMod(modID string) error {
	if err := w.getRepo().DeleteMod(modID); err != nil {
		return fmt.Errorf("delete mod: %w", err)
	}
	return nil
}

// ListWorkspaceMods returns all mods for a workspace.
func (w *WorkspaceService) ListWorkspaceMods(workspaceID string) ([]repos.WorkspaceMod, error) {
	out, err := w.getRepo().ListMods(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list mods: %w", err)
	}
	return out, nil
}

// SetActiveWorkspace sets a workspace as active (clears others for that game).
func (w *WorkspaceService) SetActiveWorkspace(workspaceID string) error {
	repo := w.getRepo()
	gameID, err := repo.GetWorkspaceGameID(workspaceID)
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}
	if err := repo.ClearActiveForGame(gameID); err != nil {
		return fmt.Errorf("clear active: %w", err)
	}
	if err := repo.SetWorkspaceActive(workspaceID); err != nil {
		return fmt.Errorf("set active: %w", err)
	}
	return nil
}

// DetectGameVersion attempts to read version from launcher-settings.json.
func (w *WorkspaceService) DetectGameVersion(path string) string {
	return game.ReadGameVersion(path)
}

// FindGameInstalls returns Steam-detected installs for a game.
func (w *WorkspaceService) FindGameInstalls(gameID string) []game.DetectedInstall {
	return game.FindInstalls(gameID)
}

// UpdateGameInstall updates path and docs_path, then refreshes version.
func (w *WorkspaceService) UpdateGameInstall(id, path, docsPath string) (*repos.GameInstall, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}
	}
	inst, err := w.getRepo().GetInstall(id)
	if err != nil {
		return nil, fmt.Errorf("install: %w", err)
	}
	if path == "" {
		path = inst.Path
	}
	if err := w.getRepo().UpdateInstallPath(id, path, docsPath); err != nil {
		return nil, fmt.Errorf("update install: %w", err)
	}
	ver := game.ReadGameVersion(path)
	if ver != "" {
		_ = w.getRepo().UpdateInstallVersion(id, ver)
	}
	return w.getRepo().GetInstall(id)
}

// DetectModRoot reports whether path is a valid mod root for gameID.
func (w *WorkspaceService) DetectModRoot(gameID, path string) bool {
	return game.IsModRoot(gameID, path)
}

// DefaultStagingDir returns the default staging directory path for a workspace.
func (w *WorkspaceService) DefaultStagingDir(workspaceID string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(configDir, appConfigDirName, "workspaces", workspaceID, "staging"), nil
}

// EnsureStagingDir creates the staging directory if it doesn't exist.
// When staging_dir is empty, assigns the default path and persists it.
func (w *WorkspaceService) EnsureStagingDir(workspaceID string) (string, error) {
	ws, err := w.GetWorkspace(workspaceID)
	if err != nil {
		return "", err
	}
	dir := ws.StagingDir
	needPersist := dir == ""
	if needPersist {
		dir, err = w.DefaultStagingDir(workspaceID)
		if err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}
	if needPersist {
		if err := w.getRepo().UpdateWorkspace(
			ws.ID, ws.Name, ws.InstallID, dir, ws.Tags, ws.ThumbnailPath,
		); err != nil {
			return "", fmt.Errorf("persist staging dir: %w", err)
		}
	}
	return dir, nil
}

// GetScriptRoot returns the script root path for a game install.
func (w *WorkspaceService) GetScriptRoot(installID string) (string, error) {
	repo := w.getRepo()
	inst, err := repo.GetInstall(installID)
	if err != nil {
		return "", fmt.Errorf("get install: %w", err)
	}
	info := game.Get(inst.GameID)
	if info == nil {
		return "", fmt.Errorf("unknown game: %s", inst.GameID)
	}
	return filepath.Join(inst.Path, info.ScriptRoot), nil
}
