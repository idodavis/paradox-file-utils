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
	schemaGen        = 1
)

// DbService manages the SQLite database.
type DbService struct {
	DB *sqlx.DB
}

// ServiceShutdown closes the database connection. Wired via app.OnShutdown in main.
func (d *DbService) ServiceShutdown() error {
	if d.DB == nil {
		return nil
	}
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
	_, statErr := os.Stat(dbPath)
	if err := d.open(dbPath); err != nil {
		return err
	}
	if statErr == nil {
		var ver int
		if err := d.DB.QueryRow(`PRAGMA user_version`).Scan(&ver); err != nil {
			return fmt.Errorf("read schema version: %w", err)
		}
		if ver < schemaGen {
			if err := d.DB.Close(); err != nil {
				return fmt.Errorf("close stale db: %w", err)
			}
			for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
				_ = os.Remove(p)
			}
			if err := d.open(dbPath); err != nil {
				return err
			}
		}
	}
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
	if err := d.initSchema(); err != nil {
		return err
	}
	if _, err := d.DB.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaGen)); err != nil {
		return fmt.Errorf("set schema version: %w", err)
	}
	return nil
}

// open connects to dbPath and installs the single-writer pool settings.
func (d *DbService) open(dbPath string) error {
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return fmt.Errorf("ping db: %w", err)
	}
	d.DB = db
	d.DB.SetMaxOpenConns(1)
	d.DB.SetMaxIdleConns(1)
	return nil
}

// initSchema creates the current tables. game_id is validated in Go via game.Get.
func (d *DbService) initSchema() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS game_installs (
			id TEXT PRIMARY KEY,
			game_id TEXT NOT NULL,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			version TEXT,
			docs_path TEXT NOT NULL DEFAULT '',
			is_broken INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_game_installs_game ON game_installs(game_id)`,

		`CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			game_id TEXT NOT NULL,
			name TEXT NOT NULL,
			install_id TEXT REFERENCES game_installs(id),
			staging_dir TEXT,
			tags TEXT DEFAULT '[]',
			thumbnail_path TEXT DEFAULT '',
			is_active INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_workspaces_game ON workspaces(game_id)`,

		`CREATE TABLE IF NOT EXISTS workspace_mods (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			is_broken INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_workspace_mods_ws ON workspace_mods(workspace_id)`,

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
	return nil
}
