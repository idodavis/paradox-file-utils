// Package repos provides database repositories for the application.
package repos

import "github.com/jmoiron/sqlx"

// Game represents a supported Paradox game.
type Game struct {
	ID         string `json:"id" db:"id"`
	Name       string `json:"name" db:"name"`
	WikiAPI    string `json:"wikiApi" db:"wiki_api"`
	ScriptRoot string `json:"scriptRoot" db:"script_root"`
	SteamAppID int    `json:"steamAppId" db:"steam_app_id"`
}

// GameInstall represents a user-configured game installation.
type GameInstall struct {
	ID        string `json:"id" db:"id"`
	GameID    string `json:"gameId" db:"game_id"`
	Name      string `json:"name" db:"name"`
	Path      string `json:"path" db:"path"`
	Version   string `json:"version" db:"version"`
	DocsPath  string `json:"docsPath" db:"docs_path"`
	IsBroken  bool   `json:"isBroken" db:"is_broken"`
	CreatedAt string `json:"createdAt" db:"created_at"`
}

// Workspace is a user-defined mod working environment tied to a game.
type Workspace struct {
	ID            string `json:"id" db:"id"`
	GameID        string `json:"gameId" db:"game_id"`
	Name          string `json:"name" db:"name"`
	InstallID     string `json:"installId" db:"install_id"`
	StagingDir    string `json:"stagingDir" db:"staging_dir"`
	Tags          string `json:"tags" db:"tags"`
	ThumbnailPath string `json:"thumbnailPath" db:"thumbnail_path"`
	IsActive      bool   `json:"isActive" db:"is_active"`
	CreatedAt     string `json:"createdAt" db:"created_at"`
}

// WorkspaceMod is a mod attached to a workspace.
type WorkspaceMod struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspaceId" db:"workspace_id"`
	Name        string `json:"name" db:"name"`
	Path        string `json:"path" db:"path"`
	IsBroken    bool   `json:"isBroken" db:"is_broken"`
	CreatedAt   string `json:"createdAt" db:"created_at"`
}

// WorkspaceRepository handles workspace-related database operations.
type WorkspaceRepository struct {
	db *sqlx.DB
}

// NewWorkspaceRepository creates a new WorkspaceRepository.
func NewWorkspaceRepository(db *sqlx.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// ListGames returns all supported games.
func (r *WorkspaceRepository) ListGames() ([]Game, error) {
	var out []Game
	err := r.db.Select(&out, `SELECT id, name, wiki_api, script_root, steam_app_id FROM games ORDER BY name`)
	return out, err
}

// ListInstalls returns all installs for a game.
func (r *WorkspaceRepository) ListInstalls(gameID string) ([]GameInstall, error) {
	var out []GameInstall
	err := r.db.Select(&out, `SELECT id, game_id, name, path, version, COALESCE(docs_path, '') as docs_path, is_broken, created_at FROM game_installs WHERE game_id = ? ORDER BY name`, gameID)
	return out, err
}

// InsertInstall adds a new game install.
func (r *WorkspaceRepository) InsertInstall(inst *GameInstall) error {
	_, err := r.db.Exec(
		`INSERT INTO game_installs (id, game_id, name, path, version, docs_path, is_broken, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		inst.ID, inst.GameID, inst.Name, inst.Path, inst.Version, inst.DocsPath, inst.IsBroken, inst.CreatedAt,
	)
	return err
}

// GetInstall returns a game install by ID.
func (r *WorkspaceRepository) GetInstall(id string) (*GameInstall, error) {
	var inst GameInstall
	err := r.db.Get(&inst, `SELECT id, game_id, name, path, version, COALESCE(docs_path, '') as docs_path, is_broken, created_at FROM game_installs WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// AllInstalls returns all game installs (for broken path marking).
func (r *WorkspaceRepository) AllInstalls() ([]GameInstall, error) {
	var out []GameInstall
	err := r.db.Select(&out, `SELECT id, path, COALESCE(docs_path, '') as docs_path FROM game_installs`)
	return out, err
}

// SetInstallBroken updates the is_broken flag for an install.
func (r *WorkspaceRepository) SetInstallBroken(id string, broken bool) error {
	_, err := r.db.Exec(`UPDATE game_installs SET is_broken = ? WHERE id = ?`, broken, id)
	return err
}

// UpdateInstallVersion writes a refreshed launcher version onto the install row.
func (r *WorkspaceRepository) UpdateInstallVersion(id, version string) error {
	_, err := r.db.Exec(`UPDATE game_installs SET version = ? WHERE id = ?`, version, id)
	return err
}

// UpdateInstallPath updates path and optional docs_path for an install.
func (r *WorkspaceRepository) UpdateInstallPath(id, path, docsPath string) error {
	_, err := r.db.Exec(
		`UPDATE game_installs SET path = ?, docs_path = ? WHERE id = ?`,
		path, docsPath, id,
	)
	return err
}

// ListWorkspaces returns workspaces, optionally filtered by game.
func (r *WorkspaceRepository) ListWorkspaces(gameID string) ([]Workspace, error) {
	var out []Workspace
	query := `SELECT id, game_id, name, COALESCE(install_id, '') as install_id, COALESCE(staging_dir, '') as staging_dir, COALESCE(tags, '[]') as tags, COALESCE(thumbnail_path, '') as thumbnail_path, is_active, created_at FROM workspaces`
	if gameID != "" {
		err := r.db.Select(&out, query+` WHERE game_id = ? ORDER BY name`, gameID)
		return out, err
	}
	err := r.db.Select(&out, query+` ORDER BY name`)
	return out, err
}

// GetWorkspace returns a workspace by ID.
func (r *WorkspaceRepository) GetWorkspace(id string) (*Workspace, error) {
	var ws Workspace
	err := r.db.Get(&ws, `SELECT id, game_id, name, COALESCE(install_id, '') as install_id, COALESCE(staging_dir, '') as staging_dir, COALESCE(tags, '[]') as tags, COALESCE(thumbnail_path, '') as thumbnail_path, is_active, created_at FROM workspaces WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

// InsertWorkspace creates a new workspace.
func (r *WorkspaceRepository) InsertWorkspace(ws *Workspace) error {
	_, err := r.db.Exec(
		`INSERT INTO workspaces (id, game_id, name, install_id, staging_dir, tags, thumbnail_path, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ws.ID, ws.GameID, ws.Name, nullIfEmpty(ws.InstallID), ws.StagingDir, ws.Tags, ws.ThumbnailPath, ws.IsActive, ws.CreatedAt,
	)
	return err
}

// UpdateWorkspace updates workspace fields.
func (r *WorkspaceRepository) UpdateWorkspace(id, name, installID, stagingDir, tagsJSON, thumbnailPath string) error {
	_, err := r.db.Exec(
		`UPDATE workspaces SET name = ?, install_id = ?, staging_dir = ?, tags = ?, thumbnail_path = ? WHERE id = ?`,
		name, nullIfEmpty(installID), stagingDir, tagsJSON, thumbnailPath, id,
	)
	return err
}

// DeleteWorkspace removes a workspace by ID.
func (r *WorkspaceRepository) DeleteWorkspace(id string) error {
	_, err := r.db.Exec(`DELETE FROM workspaces WHERE id = ?`, id)
	return err
}

// GetWorkspaceGameID returns the game_id for a workspace.
func (r *WorkspaceRepository) GetWorkspaceGameID(id string) (string, error) {
	var gameID string
	err := r.db.Get(&gameID, `SELECT game_id FROM workspaces WHERE id = ?`, id)
	return gameID, err
}

// ClearActiveForGame clears is_active for all workspaces of a game.
func (r *WorkspaceRepository) ClearActiveForGame(gameID string) error {
	_, err := r.db.Exec(`UPDATE workspaces SET is_active = 0 WHERE game_id = ?`, gameID)
	return err
}

// SetWorkspaceActive sets a workspace as active.
func (r *WorkspaceRepository) SetWorkspaceActive(id string) error {
	_, err := r.db.Exec(`UPDATE workspaces SET is_active = 1 WHERE id = ?`, id)
	return err
}

// ListMods returns all mods for a workspace.
func (r *WorkspaceRepository) ListMods(workspaceID string) ([]WorkspaceMod, error) {
	var out []WorkspaceMod
	err := r.db.Select(&out, `SELECT id, workspace_id, name, path, is_broken, created_at FROM workspace_mods WHERE workspace_id = ? ORDER BY name`, workspaceID)
	return out, err
}

// AllMods returns all mods (for broken path marking).
func (r *WorkspaceRepository) AllMods() ([]WorkspaceMod, error) {
	var out []WorkspaceMod
	err := r.db.Select(&out, `SELECT id, path FROM workspace_mods`)
	return out, err
}

// InsertMod adds a mod to a workspace.
func (r *WorkspaceRepository) InsertMod(mod *WorkspaceMod) error {
	_, err := r.db.Exec(
		`INSERT INTO workspace_mods (id, workspace_id, name, path, is_broken, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		mod.ID, mod.WorkspaceID, mod.Name, mod.Path, mod.IsBroken, mod.CreatedAt,
	)
	return err
}

// DeleteMod removes a mod by ID.
func (r *WorkspaceRepository) DeleteMod(id string) error {
	_, err := r.db.Exec(`DELETE FROM workspace_mods WHERE id = ?`, id)
	return err
}

// SetModBroken updates the is_broken flag for a mod.
func (r *WorkspaceRepository) SetModBroken(id string, broken bool) error {
	_, err := r.db.Exec(`UPDATE workspace_mods SET is_broken = ? WHERE id = ?`, broken, id)
	return err
}

// GetGame returns a game by ID.
func (r *WorkspaceRepository) GetGame(id string) (*Game, error) {
	var g Game
	err := r.db.Get(&g, `SELECT id, name, wiki_api, script_root, steam_app_id FROM games WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// GetWorkspaceInfo returns game_id and install_id for a workspace.
func (r *WorkspaceRepository) GetWorkspaceInfo(workspaceID string) (gameID, installID string, err error) {
	var ws struct {
		GameID    string `db:"game_id"`
		InstallID string `db:"install_id"`
	}
	err = r.db.Get(&ws,
		`SELECT game_id, COALESCE(install_id, '') as install_id FROM workspaces WHERE id = ?`,
		workspaceID,
	)
	return ws.GameID, ws.InstallID, err
}

// GetInstallPath returns the path, game_id, and docs_path for an install.
func (r *WorkspaceRepository) GetInstallPath(installID string) (path, gameID, docsPath string, err error) {
	var inst struct {
		Path     string `db:"path"`
		GameID   string `db:"game_id"`
		DocsPath string `db:"docs_path"`
	}
	err = r.db.Get(&inst, `SELECT path, game_id, COALESCE(docs_path, '') as docs_path FROM game_installs WHERE id = ?`, installID)
	return inst.Path, inst.GameID, inst.DocsPath, err
}

// ListModPaths returns all mod paths for a workspace.
func (r *WorkspaceRepository) ListModPaths(workspaceID string) ([]string, error) {
	var paths []string
	err := r.db.Select(&paths,
		`SELECT path FROM workspace_mods WHERE workspace_id = ?`, workspaceID)
	return paths, err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
