// detect.go finds game installs (via Steam library folders) and reads the
// installed version from launcher-settings.json. It avoids the Windows registry
// so it cross-compiles; it scans well-known Steam locations.

package game

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

// DetectedInstall is a game install found on disk.
type DetectedInstall struct {
	GameID  string `json:"gameId"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version"`
	AppID   int    `json:"appId"`
}

var (
	vdfPathRe    = regexp.MustCompile(`"path"\s*"([^"]+)"`)
	installDirRe = regexp.MustCompile(`"installdir"\s*"([^"]+)"`)
)

type launcherSettings struct {
	RawVersion   string `json:"rawVersion"`
	Version      string `json:"version"`
	GameDataPath string `json:"gameDataPath"`
}

// ReadGameVersion reads rawVersion (then version, stripping a parenthetical
// codename) from launcher/launcher-settings.json. Missing file → "".
func ReadGameVersion(installPath string) string {
	data, err := os.ReadFile(filepath.Join(installPath, "launcher", "launcher-settings.json"))
	if err != nil {
		return ""
	}
	var ls launcherSettings
	if err := sonic.Unmarshal(data, &ls); err != nil {
		return ""
	}
	if ls.RawVersion != "" {
		return ls.RawVersion
	}
	v := ls.Version
	if i := strings.Index(v, " ("); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

// UserDataDir is the OS user-data folder for gameID (script_docs, default mod/).
// First existing candidate wins; if none exist, the preferred conventional path.
func UserDataDir(gameID, installPath string) string {
	cands := userDataCandidates(gameID, installPath)
	for _, p := range cands {
		if dirExists(p) {
			return p
		}
	}
	if len(cands) > 0 {
		return cands[0]
	}
	return ""
}

func userDataCandidates(gameID, installPath string) []string {
	info := Get(gameID)
	if info == nil {
		return nil
	}
	name := info.DocsFolderName
	var cands []string
	add := func(p string) {
		if p != "" {
			cands = append(cands, filepath.Clean(p))
		}
	}
	add(launcherGameDataPath(installPath))
	home, homeErr := os.UserHomeDir()
	nativeDocs := ""
	if homeErr == nil {
		nativeDocs = filepath.Join(home, "Documents", "Paradox Interactive", name)
	}
	switch runtime.GOOS {
	case "windows", "darwin":
		add(nativeDocs)
	default:
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg == "" && homeErr == nil {
			xdg = filepath.Join(home, ".local", "share")
		}
		if xdg != "" {
			add(filepath.Join(xdg, "Paradox Interactive", name))
		}
		add(nativeDocs)
		appID := strconv.Itoa(info.SteamAppID)
		for _, steam := range steamLibraries() {
			add(filepath.Join(
				steam, "steamapps", "compatdata", appID,
				"pfx", "drive_c", "users", "steamuser",
				"Documents", "Paradox Interactive", name,
			))
		}
	}
	return cands
}

func launcherGameDataPath(installPath string) string {
	if installPath == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(installPath, "launcher", "launcher-settings.json"))
	if err != nil {
		return ""
	}
	var ls launcherSettings
	if sonic.Unmarshal(data, &ls) != nil {
		return ""
	}
	return expandGameDataPath(ls.GameDataPath)
}

func expandGameDataPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if strings.Contains(p, "%USER_DOCUMENTS%") {
		home, err := os.UserHomeDir()
		docs := ""
		if err == nil {
			docs = filepath.Join(home, "Documents")
		}
		p = strings.ReplaceAll(p, "%USER_DOCUMENTS%", docs)
	}
	return filepath.Clean(filepath.FromSlash(p))
}

// DescriptorPath is the expected descriptor file for gameID under a mod root.
func DescriptorPath(gameID, root string) string {
	info := Get(gameID)
	if info != nil && info.Descriptor == "metadata" {
		return filepath.Join(root, ".metadata", "metadata.json")
	}
	return filepath.Join(root, "descriptor.mod")
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
			filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
		}
	}
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
