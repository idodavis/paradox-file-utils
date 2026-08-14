// Package services provides backend services for the Paradox Modding Tools application.
package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const (
	appConfigDirName = "Paradox Modding Tools"
	dbFileName       = "pmt-workspace.db"
)

// DbService manages the SQLite database.
type DbService struct {
	DB *sqlx.DB
}

// ServiceShutdown closes the database connection. Called by Wails when the app exits.
func (d *DbService) ServiceShutdown() error {
	_, _ = d.DB.Exec("VACUUM")
	return d.DB.Close()
}

// ServiceStartup opens the database and initializes the schema. Call from main before creating services that need DB.
func (d *DbService) ServiceStartup() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("user config dir: %w", err)
	}
	appDir := filepath.Join(configDir, appConfigDirName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	dbPath := filepath.Join(appDir, dbFileName)
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	d.DB = db
	// Single writer connection avoids SQLITE_BUSY under worker-pool index jobs.
	d.DB.SetMaxOpenConns(1)
	d.DB.SetMaxIdleConns(1)

	if _, err := d.DB.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		return fmt.Errorf("set wal mode: %w", err)
	}
	if _, err := d.DB.Exec(`PRAGMA busy_timeout = 8000`); err != nil {
		return fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := d.DB.Exec(`PRAGMA synchronous = NORMAL`); err != nil {
		return fmt.Errorf("set synchronous normal: %w", err)
	}
	if _, err := d.DB.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	return d.initSchema()
}

// initSchema creates tables and seeds initial data. Old tables (inventories, patchnotes, doc_files) are dropped.
func (d *DbService) initSchema() error {
	drops := []string{
		`DROP TABLE IF EXISTS inventory_items`,
		`DROP TABLE IF EXISTS inventories`,
		`DROP TABLE IF EXISTS patchnotes`,
		`DROP TABLE IF EXISTS doc_files`,
	}
	for _, q := range drops {
		if _, err := d.DB.Exec(q); err != nil {
			return fmt.Errorf("drop old table: %w", err)
		}
	}

	migrations := []string{
		// Core game registry (seeded)
		`CREATE TABLE IF NOT EXISTS games (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			wiki_api TEXT NOT NULL,
			script_root TEXT NOT NULL,
			steam_app_id INTEGER NOT NULL
		)`,

		// User-defined game installs (multiple versions)
		`CREATE TABLE IF NOT EXISTS game_installs (
			id TEXT PRIMARY KEY,
			game_id TEXT NOT NULL REFERENCES games(id),
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			version TEXT,
			is_broken INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_game_installs_game ON game_installs(game_id)`,

		// Workspaces group mods and settings per game
		`CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			game_id TEXT NOT NULL REFERENCES games(id),
			name TEXT NOT NULL,
			install_id TEXT REFERENCES game_installs(id),
			staging_dir TEXT,
			tags TEXT DEFAULT '[]',
			thumbnail_path TEXT DEFAULT '',
			is_active INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_workspaces_game ON workspaces(game_id)`,

		// Mods attached to a workspace
		`CREATE TABLE IF NOT EXISTS workspace_mods (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			is_broken INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_workspace_mods_ws ON workspace_mods(workspace_id)`,

		// Wiki patch cache (MediaWiki API)
		`CREATE TABLE IF NOT EXISTS wiki_patches (
			game_id TEXT NOT NULL REFERENCES games(id),
			version TEXT NOT NULL,
			fetched_at TEXT NOT NULL,
			source_url TEXT NOT NULL,
			html_content TEXT NOT NULL,
			modding_html TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (game_id, version)
		)`,

		// Script log imports (error.log analysis)
		`CREATE TABLE IF NOT EXISTS script_log_imports (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			path TEXT NOT NULL,
			imported_at TEXT NOT NULL,
			summary TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_script_log_ws ON script_log_imports(workspace_id)`,

		// Patch runs (mod version update campaigns)
		`CREATE TABLE IF NOT EXISTS patch_runs (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			mod_id TEXT NOT NULL REFERENCES workspace_mods(id),
			baseline_version TEXT NOT NULL,
			target_version TEXT NOT NULL,
			baseline_install_id TEXT REFERENCES game_installs(id),
			target_install_id TEXT REFERENCES game_installs(id),
			status TEXT NOT NULL DEFAULT 'pending',
			staging_dir TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_patch_runs_ws ON patch_runs(workspace_id)`,

		// Per-file decisions within a patch run
		`CREATE TABLE IF NOT EXISTS patch_run_files (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL REFERENCES patch_runs(id) ON DELETE CASCADE,
			rel_path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			decision TEXT,
			preview_path TEXT,
			stats TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_patch_run_files_run ON patch_run_files(run_id)`,

		// App settings (unchanged)
		`CREATE TABLE IF NOT EXISTS app_settings (
			game TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY (game, key)
		)`,
	}
	for _, m := range migrations {
		if _, err := d.DB.Exec(m); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	if err := d.ensureWikiPatchColumns(); err != nil {
		return err
	}
	return d.seedGames()
}

// ensureWikiPatchColumns adds modding_html on older DBs and clears stale caches once.
func (d *DbService) ensureWikiPatchColumns() error {
	var n int
	if err := d.DB.Get(&n,
		`SELECT COUNT(*) FROM pragma_table_info('wiki_patches') WHERE name = 'modding_html'`,
	); err != nil {
		return fmt.Errorf("check wiki_patches columns: %w", err)
	}
	if n > 0 {
		return nil
	}
	if _, err := d.DB.Exec(
		`ALTER TABLE wiki_patches ADD COLUMN modding_html TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("add wiki_patches.modding_html: %w", err)
	}
	// Old rows stored modding-only HTML in html_content; force refetch.
	if _, err := d.DB.Exec(`DELETE FROM wiki_patches`); err != nil {
		return fmt.Errorf("clear stale wiki_patches: %w", err)
	}
	return nil
}

// seedGames inserts the core game definitions (ck3, eu5, vic3).
func (d *DbService) seedGames() error {
	games := []struct {
		id, name, wikiAPI, scriptRoot string
		steamAppID                    int
	}{
		{"ck3", "Crusader Kings III", "https://ck3.paradoxwikis.com/api.php", "game", 1158310},
		{"eu5", "Europa Universalis V", "https://eu5.paradoxwikis.com/api.php", "game/in_game", 3450310},
		{"vic3", "Victoria 3", "https://vic3.paradoxwikis.com/api.php", "game", 529340},
	}
	for _, g := range games {
		_, err := d.DB.Exec(
			`INSERT OR IGNORE INTO games (id, name, wiki_api, script_root, steam_app_id) VALUES (?, ?, ?, ?, ?)`,
			g.id, g.name, g.wikiAPI, g.scriptRoot, g.steamAppID,
		)
		if err != nil {
			return fmt.Errorf("seed game %s: %w", g.id, err)
		}
	}
	return nil
}

// ResetData wipes user data (workspaces, mods, installs, runs) but keeps games and app_settings.
func (d *DbService) ResetData() error {
	if d.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	tables := []string{
		"patch_run_files", "patch_runs", "workspace_mods", "workspaces",
		"game_installs", "script_log_imports", "wiki_patches",
	}
	for _, t := range tables {
		if _, err := d.DB.Exec(`DELETE FROM ` + t); err != nil {
			return fmt.Errorf("delete %s: %w", t, err)
		}
	}
	return nil
}
