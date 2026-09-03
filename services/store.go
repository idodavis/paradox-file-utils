// store.go is the JSON app config (installs, workspaces, settings, patch runs).

package services

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/bytedance/sonic"

	"paradox-modding-tools/services/internal/game"
)

const (
	appConfigDirName    = "Paradox Modding Tools"
	configFileName      = "config.json"
	configFormatVersion = 1
	maxPatchRuns        = 20
	legacyDBName        = "pmt-workspace.db"
)

// Config is the on-disk app store.
type Config struct {
	FormatVersion int                          `json:"formatVersion"`
	Installs      []GameInstall                `json:"installs"`
	Workspaces    []Workspace                  `json:"workspaces"`
	Settings      map[string]map[string]string `json:"settings"`
	PatchRuns     []PatchRun                   `json:"patchRuns"`
}

// GameInstall is a user-configured game installation.
type GameInstall struct {
	ID              string `json:"id"`
	GameID          string `json:"gameId"`
	Name            string `json:"name"`
	Path            string `json:"path"`
	Version         string `json:"version"`
	VersionDetected string `json:"versionDetected"`
	IsBroken        bool   `json:"isBroken"`
	CreatedAt       string `json:"createdAt"`
	ScannedAt       string `json:"scannedAt,omitempty"`
}

// Workspace is a user-defined mod working environment tied to a game.
type Workspace struct {
	ID             string         `json:"id"`
	GameID         string         `json:"gameId"`
	Name           string         `json:"name"`
	InstallID      string         `json:"installId"`
	StagingDir     string         `json:"stagingDir"`
	DefaultLocLang string         `json:"defaultLocLang"`
	GameColor      string         `json:"gameColor,omitempty"`
	StagingColor   string         `json:"stagingColor,omitempty"`
	ResetIdeOnOpen bool           `json:"resetIdeOnOpen"`
	DefaultTool    string         `json:"defaultTool,omitempty"`
	IdeOpenFiles   []string       `json:"ideOpenFiles,omitempty"`
	IdeActiveFile  string         `json:"ideActiveFile,omitempty"`
	CreatedAt      string         `json:"createdAt"`
	Mods           []WorkspaceMod `json:"mods"`
}

// WorkspaceMod is a mod attached to a workspace.
type WorkspaceMod struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Path           string `json:"path"`
	SortOrder      int    `json:"sortOrder"`
	Color          string `json:"color,omitempty"`
	Thumbnail      string `json:"thumbnail,omitempty"`
	DescMdRel      string `json:"descMdRel,omitempty"`
	DescBbRel      string `json:"descBbRel,omitempty"`
	WorkshopIgnore string `json:"workshopIgnore,omitempty"`
	IsBroken       bool   `json:"isBroken"`
	CreatedAt      string `json:"createdAt"`
}

// PatchRun is one patch of a mod onto a target install.
type PatchRun struct {
	ID              string         `json:"id"`
	WorkspaceID     string         `json:"workspaceId"`
	ModID           string         `json:"modId"`
	TargetVersion   string         `json:"targetVersion"`
	TargetInstallID string         `json:"targetInstallId"`
	Status          string         `json:"status"`
	StagingDir      string         `json:"stagingDir"`
	CreatedAt       string         `json:"createdAt"`
	UpdatedAt       string         `json:"updatedAt"`
	Files           []PatchRunFile `json:"files"`
}

// PatchRunFile is one file in a patch run.
type PatchRunFile struct {
	ID          string            `json:"id"`
	RelPath     string            `json:"relPath"`
	Status      string            `json:"status"`
	Decision    string            `json:"decision"`
	PreviewPath string            `json:"previewPath"`
	ModPath     string            `json:"modPath,omitempty"`
	TargetPath  string            `json:"targetPath,omitempty"`
	Stats       PatchRunFileStats `json:"stats"`
}

// PatchRunFileStats is merge preview stats for one file.
type PatchRunFileStats struct {
	Changed   int    `json:"changed"`
	Added     int    `json:"added"`
	Conflicts int    `json:"conflicts"`
	Error     string `json:"error"`
}

// Store is the mutex-guarded JSON config.
type Store struct {
	mu   sync.Mutex
	path string
	cfg  Config
}

func appConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, appConfigDirName)
	return dir, os.MkdirAll(dir, 0o755)
}

func dropLegacyDB(dir string) {
	p := filepath.Join(dir, legacyDBName)
	for _, s := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(p + s)
	}
}

func emptyConfig() Config {
	return Config{FormatVersion: configFormatVersion, Settings: map[string]map[string]string{}}
}

func newStore(path string) *Store {
	s := &Store{path: path, cfg: emptyConfig()}
	if c, err := loadConfig(path); err == nil {
		s.cfg = c
		if s.cfg.Settings == nil {
			s.cfg.Settings = map[string]map[string]string{}
		}
	}
	return s
}

func loadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	n, err := sonic.Get(raw, "formatVersion")
	got, nerr := n.Int64()
	if err != nil || nerr != nil {
		return Config{}, fmt.Errorf("missing formatVersion")
	}
	if int(got) != configFormatVersion {
		return Config{}, fmt.Errorf("format %d != %d", got, configFormatVersion)
	}
	var dst Config
	if err := sonic.Unmarshal(raw, &dst); err != nil {
		return Config{}, err
	}
	return dst, nil
}

func saveConfig(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := sonic.Marshal(cfg)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}

// OpenStore loads config.json (missing or version mismatch → empty; drops SQLite).
func OpenStore() (*Store, error) {
	dir, err := appConfigDir()
	if err != nil {
		return nil, err
	}
	dropLegacyDB(dir)
	return newStore(filepath.Join(dir, configFileName)), nil
}

// Read runs fn with the live config. Do not retain pointers after return.
func (s *Store) Read(fn func(*Config)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.cfg)
}

// Mutate runs fn then saves. A fn error skips save.
func (s *Store) Mutate(fn func(*Config) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.cfg); err != nil {
		return err
	}
	if n := len(s.cfg.PatchRuns); n > maxPatchRuns {
		s.cfg.PatchRuns = append([]PatchRun(nil), s.cfg.PatchRuns[n-maxPatchRuns:]...)
	}
	s.cfg.FormatVersion = configFormatVersion
	if s.cfg.Settings == nil {
		s.cfg.Settings = map[string]map[string]string{}
	}
	return saveConfig(s.path, &s.cfg)
}

func cloneWorkspace(w Workspace) Workspace {
	w.IdeOpenFiles = append([]string(nil), w.IdeOpenFiles...)
	mods := make([]WorkspaceMod, len(w.Mods))
	copy(mods, w.Mods)
	sortWorkspaceMods(mods)
	w.Mods = mods
	return w
}

func sortWorkspaceMods(mods []WorkspaceMod) {
	slices.SortFunc(mods, func(a, b WorkspaceMod) int {
		if a.SortOrder != b.SortOrder {
			return a.SortOrder - b.SortOrder
		}
		return strings.Compare(a.Name, b.Name)
	})
}

func findInstall(c *Config, id string) *GameInstall {
	for i := range c.Installs {
		if c.Installs[i].ID == id {
			return &c.Installs[i]
		}
	}
	return nil
}

func installScriptRoot(c *Config, installID string) (string, error) {
	inst := findInstall(c, installID)
	if inst == nil {
		return "", fmt.Errorf("install not found")
	}
	info := game.Get(inst.GameID)
	if info == nil {
		return "", fmt.Errorf("unknown game: %s", inst.GameID)
	}
	return filepath.Join(inst.Path, info.ScriptRoot), nil
}
func findWorkspace(c *Config, id string) *Workspace {
	for i := range c.Workspaces {
		if c.Workspaces[i].ID == id {
			return &c.Workspaces[i]
		}
	}
	return nil
}
func findMod(c *Config, id string) *WorkspaceMod {
	for i := range c.Workspaces {
		for j := range c.Workspaces[i].Mods {
			if c.Workspaces[i].Mods[j].ID == id {
				return &c.Workspaces[i].Mods[j]
			}
		}
	}
	return nil
}
func findRun(c *Config, id string) *PatchRun {
	for i := range c.PatchRuns {
		if c.PatchRuns[i].ID == id {
			return &c.PatchRuns[i]
		}
	}
	return nil
}
func findRunFile(c *Config, fileID string) *PatchRunFile {
	for i := range c.PatchRuns {
		for j := range c.PatchRuns[i].Files {
			if c.PatchRuns[i].Files[j].ID == fileID {
				return &c.PatchRuns[i].Files[j]
			}
		}
	}
	return nil
}
