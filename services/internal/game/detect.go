// detect.go finds game installs (via Steam library folders), reads the installed
// version from launcher-settings.json, and recognizes mod roots. It avoids the
// Windows registry so it cross-compiles; it scans well-known Steam locations.

package game

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// DetectedInstall is a game install found on disk.
type DetectedInstall struct {
	GameID  string `json:"gameId"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version"`
	AppID   int    `json:"appId"`
}

// rawVersionRe / versionRe pull the version out of launcher-settings.json without
// a full JSON decode (the file has both "rawVersion" and "version").
var (
	rawVersionRe = regexp.MustCompile(`"rawVersion"\s*:\s*"([^"]+)"`)
	versionRe    = regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
	vdfPathRe    = regexp.MustCompile(`"path"\s*"([^"]+)"`)
	installDirRe = regexp.MustCompile(`"installdir"\s*"([^"]+)"`)
)

// ReadGameVersion reads the installed version from launcher-settings.json in the
// install path (checking the game/ subfolder too). Returns "" if unreadable.
func ReadGameVersion(installPath string) string {
	for _, rel := range []string{"launcher-settings.json", filepath.Join("game", "launcher-settings.json")} {
		data, err := os.ReadFile(filepath.Join(installPath, rel))
		if err != nil {
			continue
		}
		if m := rawVersionRe.FindSubmatch(data); m != nil {
			return string(m[1])
		}
		if m := versionRe.FindSubmatch(data); m != nil {
			return string(m[1])
		}
	}
	return ""
}

// IsModRoot reports whether path is a valid mod root for gameID.
func IsModRoot(gameID, path string) bool {
	info := Get(gameID)
	if info == nil {
		return false
	}
	switch info.Descriptor {
	case "mod":
		return fileExists(filepath.Join(path, "descriptor.mod"))
	case "metadata":
		if fileExists(filepath.Join(path, ".metadata", "metadata.json")) {
			return true
		}
		for _, stage := range info.StageRoots {
			if dirExists(filepath.Join(path, stage)) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// FindInstalls returns Steam-detected installs for a game.
func FindInstalls(gameID string) []DetectedInstall {
	info := Get(gameID)
	if info == nil {
		return nil
	}
	var out []DetectedInstall
	for _, lib := range steamLibraries() {
		manifest := filepath.Join(lib, "steamapps", "appmanifest_"+strconv.Itoa(info.SteamAppID)+".acf")
		data, err := os.ReadFile(manifest)
		if err != nil {
			continue
		}
		m := installDirRe.FindSubmatch(data)
		if m == nil {
			continue
		}
		path := filepath.Join(lib, "steamapps", "common", string(m[1]))
		if !dirExists(path) {
			continue
		}
		out = append(out, DetectedInstall{
			GameID:  gameID,
			Name:    info.Name,
			Path:    path,
			Version: ReadGameVersion(path),
			AppID:   info.SteamAppID,
		})
	}
	return out
}

// steamLibraries returns candidate Steam library roots by reading
// libraryfolders.vdf from each default Steam install location.
func steamLibraries() []string {
	seen := map[string]bool{}
	var libs []string
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		libs = append(libs, p)
	}
	for _, root := range defaultSteamRoots() {
		add(root)
		vdf := filepath.Join(root, "steamapps", "libraryfolders.vdf")
		data, err := os.ReadFile(vdf)
		if err != nil {
			continue
		}
		for _, m := range vdfPathRe.FindAllSubmatch(data, -1) {
			add(filepath.FromSlash(strings.ReplaceAll(string(m[1]), `\\`, `/`)))
		}
	}
	return libs
}

// defaultSteamRoots returns the platform's typical Steam install directories.
func defaultSteamRoots() []string {
	switch runtime.GOOS {
	case "windows":
		var roots []string
		for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
			if base := os.Getenv(env); base != "" {
				roots = append(roots, filepath.Join(base, "Steam"))
			}
		}
		return roots
	case "darwin":
		home, _ := os.UserHomeDir()
		return []string{filepath.Join(home, "Library", "Application Support", "Steam")}
	default:
		home, _ := os.UserHomeDir()
		return []string{
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".local", "share", "Steam"),
		}
	}
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
