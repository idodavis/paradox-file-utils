// persist.go saves and loads VanillaCache and vanilla loc sidecars as JSON.

package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
)

// CacheDir is the user-data directory for cached vanilla models.
func CacheDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Paradox Modding Tools", "cache")
	return dir, os.MkdirAll(dir, 0o755)
}

func vanillaJSON(kind string, parts ...string) (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	name := kind
	for _, p := range parts {
		name += "-" + sanitize(p)
	}
	return filepath.Join(dir, name+".json"), nil
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// saveJSON marshals v to path via a temp file then rename.
func saveJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := sonic.Marshal(v)
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

// loadJSON reads path and rejects a missing or mismatched formatVersion.
func loadJSON[T any](path string, wantVersion int) (*T, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	n, err := sonic.Get(raw, "formatVersion")
	got, nerr := n.Int64()
	if err != nil || nerr != nil {
		return nil, fmt.Errorf("missing formatVersion")
	}
	if int(got) != wantVersion {
		return nil, fmt.Errorf("format %d != %d", got, wantVersion)
	}
	var dst T
	if err := sonic.Unmarshal(raw, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

func dropLegacyWorkspaceCaches(dir string) {
	matches, _ := filepath.Glob(filepath.Join(dir, "cache-*.json"))
	for _, p := range matches {
		_ = os.Remove(p)
	}
}

// SaveCache writes c keyed by installId + version and drops leftover cache-*.json.
func SaveCache(c *VanillaCache) error {
	path, err := vanillaJSON("vanilla", c.InstallID, c.GameVersion)
	if err != nil {
		return err
	}
	if err := saveCacheFile(path, c); err != nil {
		return err
	}
	dropLegacyWorkspaceCaches(filepath.Dir(path))
	return nil
}

// LoadCache reads the VanillaCache for installID + version.
func LoadCache(installID, version string) (*VanillaCache, error) {
	path, err := vanillaJSON("vanilla", installID, version)
	if err != nil {
		return nil, err
	}
	return loadCacheFile(path)
}

// PeekCacheScannedAt returns scannedAt from the vanilla JSON without loading defs.
func PeekCacheScannedAt(installID, version string) string {
	path, err := vanillaJSON("vanilla", installID, version)
	if err != nil {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	n, err := sonic.Get(raw, "scannedAt")
	if err != nil {
		return ""
	}
	s, _ := n.String()
	return s
}

// saveCacheFile marshals c to an explicit path, interning repeated file paths.
func saveCacheFile(path string, c *VanillaCache) error {
	PrepareCache(c)
	return saveJSON(path, internPaths(c))
}

// loadCacheFile reads a VanillaCache from an explicit path.
func loadCacheFile(path string) (*VanillaCache, error) {
	c, err := loadJSON[VanillaCache](path, CacheFormatVersion)
	if err != nil {
		return nil, err
	}
	expandPaths(c)
	PrepareCache(c)
	return c, nil
}

// SaveVanillaLoc writes the loc sidecar for install + version + lang.
func SaveVanillaLoc(installID, version, lang string, loc *VanillaLoc) error {
	if loc == nil {
		return nil
	}
	path, err := vanillaJSON("vanilla-loc", installID, version, lang)
	if err != nil {
		return err
	}
	loc.FormatVersion = LocFormatVersion
	return saveJSON(path, loc)
}

// LoadVanillaLoc reads the loc sidecar, or an error if absent / version mismatch.
func LoadVanillaLoc(installID, version, lang string) (*VanillaLoc, error) {
	path, err := vanillaJSON("vanilla-loc", installID, version, lang)
	if err != nil {
		return nil, err
	}
	loc, err := loadJSON[VanillaLoc](path, LocFormatVersion)
	if err != nil {
		return nil, err
	}
	if loc.Sites == nil {
		loc.Sites = map[string]LocEntry{}
	}
	return loc, nil
}

// DropVanillaFiles removes vanilla-* files for installID (script + loc sidecars)
// and the archived script_docs copies. Dropping the archive here is deliberate:
// this is the "forget this install" path, and a stale archive would otherwise
// outlive the install it describes.
func DropVanillaFiles(installID string) error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}
	sid := sanitize(installID)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		name := e.Name()
		switch {
		case strings.HasPrefix(name, "vanilla-"+sid),
			strings.HasPrefix(name, "vanilla-loc-"+sid):
			_ = os.Remove(filepath.Join(dir, name))
		case strings.HasPrefix(name, "script-docs-"+sid):
			_ = os.RemoveAll(filepath.Join(dir, name))
		}
	}
	return nil
}
