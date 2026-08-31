// settingsservice.go persists app settings via the JSON store.

package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	"paradox-modding-tools/services/internal/catalog"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SettingsService provides persisted app settings.
type SettingsService struct {
	Store   *Store
	Version string
}

// GetVersion returns the app version string.
func (s *SettingsService) GetVersion() string {
	return s.Version
}

// CheckForUpdates triggers the Wails updater.
func (s *SettingsService) CheckForUpdates() {
	app := application.Get()
	if app == nil {
		return
	}
	if err := app.Updater.CheckAndInstall(context.Background()); err != nil {
		app.Logger.Error("update", "error", err)
	}
}

// GetSettings loads settings as a map "game.key" -> value.
func (s *SettingsService) GetSettings() (map[string]string, error) {
	out := map[string]string{}
	s.Store.Read(func(c *Config) {
		for game, kv := range c.Settings {
			for k, v := range kv {
				out[game+"."+k] = v
			}
		}
	})
	return out, nil
}

// SaveSettings writes user settings keyed as "game.key".
func (s *SettingsService) SaveSettings(settings map[string]string) error {
	return s.Store.Mutate(func(c *Config) error {
		for k, v := range settings {
			parts := strings.SplitN(k, ".", 2)
			if len(parts) != 2 {
				continue
			}
			if c.Settings[parts[0]] == nil {
				c.Settings[parts[0]] = map[string]string{}
			}
			c.Settings[parts[0]][parts[1]] = v
		}
		return nil
	})
}

// ResetData wipes workspaces, mods, installs, patch runs, and the semantic cache.
func (s *SettingsService) ResetData() error {
	if err := s.Store.Mutate(func(c *Config) error {
		c.Installs, c.Workspaces, c.PatchRuns = nil, nil, nil
		return nil
	}); err != nil {
		return err
	}
	dir, err := catalog.CacheDir()
	if err != nil {
		return fmt.Errorf("cache dir: %w", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("wipe cache: %w", err)
	}
	return nil
}
