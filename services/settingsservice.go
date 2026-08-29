package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/repos"

	"github.com/jmoiron/sqlx"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ############
// SettingsService
// ############

// SettingsService provides persisted app settings.
type SettingsService struct {
	DB      *sqlx.DB
	repo    *repos.SettingsRepository
	Version string
}

func (s *SettingsService) GetVersion() string {
	return s.Version
}

func (s *SettingsService) CheckForUpdates() {
	app := application.Get()
	if app == nil {
		return
	}
	if err := app.Updater.CheckAndInstall(context.Background()); err != nil {
		app.Logger.Error("update", "error", err)
	}
}

func (s *SettingsService) getRepo() *repos.SettingsRepository {
	if s.repo == nil {
		s.repo = repos.NewSettingsRepository(s.DB)
	}
	return s.repo
}

// GetSettings loads settings as a map "game.key" -> value.
func (s *SettingsService) GetSettings() (map[string]string, error) {
	settings, err := s.getRepo().GetAllSettings()
	if err != nil {
		return nil, fmt.Errorf("read settings: %w", err)
	}

	out := make(map[string]string)
	for _, setting := range settings {
		out[setting.Game+"."+setting.Key] = setting.Value
	}
	return out, nil
}

// SaveSettings writes user settings to app_settings table.
func (s *SettingsService) SaveSettings(settings map[string]string) error {
	repo := s.getRepo()
	for k, v := range settings {
		parts := strings.SplitN(k, ".", 2)
		if len(parts) == 2 {
			if err := repo.UpsertSetting(parts[0], parts[1], v); err != nil {
				return err
			}
		}
	}
	return nil
}

// ResetData wipes workspaces, mods, installs, patch runs, and the semantic cache.
func (s *SettingsService) ResetData() error {
	if s.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	tables := []string{
		"patch_run_files", "patch_runs", "workspace_mods", "workspaces",
		"game_installs",
	}
	for _, t := range tables {
		if _, err := s.DB.Exec(`DELETE FROM ` + t); err != nil {
			return fmt.Errorf("delete %s: %w", t, err)
		}
	}
	dir, err := model.CacheDir()
	if err != nil {
		return fmt.Errorf("cache dir: %w", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("wipe cache: %w", err)
	}
	return nil
}
