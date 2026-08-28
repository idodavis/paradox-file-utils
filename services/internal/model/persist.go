// persist.go saves and loads the semantic model as JSON in user-data. Loads reject
// a file whose formatVersion does not match the current constant — there is no
// migration; the caller rescans. sonic is used for (de)serialization.

package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
)

// CacheDir returns the user-data directory holding cached vanilla models,
// creating it if needed.
func CacheDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Paradox Modding Tools", "cache")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// cachePath is the on-disk path for one (game, version) Cache.
func cachePath(gameID, version string) (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	if version == "" {
		version = "unknown"
	}
	return filepath.Join(dir, gameID+"-"+version+".json"), nil
}

// SaveCache writes c to its (game, version) path in user-data.
func SaveCache(c *Cache) error {
	path, err := cachePath(c.GameID, c.GameVersion)
	if err != nil {
		return err
	}
	return SaveCacheFile(path, c)
}

// LoadCache reads the Cache for (gameID, version), or an error if it is absent or
// its formatVersion does not match CacheFormatVersion.
func LoadCache(gameID, version string) (*Cache, error) {
	path, err := cachePath(gameID, version)
	if err != nil {
		return nil, err
	}
	return LoadCacheFile(path)
}

// SaveCacheFile marshals c to an explicit path (used by SaveCache and tests).
func SaveCacheFile(path string, c *Cache) error {
	raw, err := sonic.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// LoadCacheFile reads a Cache from an explicit path and rejects a version mismatch.
func LoadCacheFile(path string) (*Cache, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Cache
	if err := sonic.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if c.FormatVersion != CacheFormatVersion {
		return nil, fmt.Errorf("cache format %d != %d (rescan required)", c.FormatVersion, CacheFormatVersion)
	}
	return &c, nil
}

// indexPath is the on-disk path for one workspace Index, alongside the caches.
func indexPath(workspaceID string) (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "index-"+sanitize(workspaceID)+".json"), nil
}

// sanitize replaces path-hostile characters so a workspace id is a safe filename.
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// SaveIndex writes idx to its workspace path in user-data.
func SaveIndex(idx *Index) error {
	path, err := indexPath(idx.WorkspaceID)
	if err != nil {
		return err
	}
	return SaveIndexFile(path, idx)
}

// LoadIndex reads the Index for workspaceID, rejecting a format-version mismatch.
func LoadIndex(workspaceID string) (*Index, error) {
	path, err := indexPath(workspaceID)
	if err != nil {
		return nil, err
	}
	return LoadIndexFile(path)
}

// SaveIndexFile marshals idx to an explicit path (used by SaveIndex and tests).
func SaveIndexFile(path string, idx *Index) error {
	raw, err := sonic.Marshal(idx)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// LoadIndexFile reads an Index from an explicit path and rejects a version mismatch.
func LoadIndexFile(path string) (*Index, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := sonic.Unmarshal(raw, &idx); err != nil {
		return nil, err
	}
	if idx.FormatVersion != IndexFormatVersion {
		return nil, fmt.Errorf("index format %d != %d (rebuild required)", idx.FormatVersion, IndexFormatVersion)
	}
	return &idx, nil
}
