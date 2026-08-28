// registry.go holds the GameInfo identity table for the supported games and the
// Get/All accessors. See doc.go for the package overview.

package game

// GameInfo is the static identity and convention set for one game. It carries no
// derived data — the model package scans the install for objects, vocabulary, etc.
type GameInfo struct {
	ID        string
	Name      string
	ShortName string

	// WikiAPI is the MediaWiki api.php endpoint; WikiDocPages are the pages the
	// wiki hover fallback caches.
	WikiAPI      string
	WikiDocPages []string

	// ScriptRoot is the install-relative folder holding script (e.g. "game").
	// Empty for EU5, whose content is split across StageRoots.
	ScriptRoot string
	StageRoots []string

	// Descriptor is how a mod root is identified: "mod" (descriptor.mod) or
	// "metadata" (.metadata/metadata.json).
	Descriptor string

	// DocsFolderName is the per-game Documents subfolder (e.g. "Crusader Kings III").
	DocsFolderName string
	// ScriptDocsSubdir is where in-game console dumps land ("logs" or "docs").
	ScriptDocsSubdir string
	// ScriptDocsFormat is "classic" (CK3 `_*.info`) or "markdown" (Vic3/EU5 `*.md`).
	ScriptDocsFormat string

	// EntryModes are EU5 database entry-mode prefixes (INJECT:, REPLACE:, ...).
	EntryModes []string

	SteamAppID      int
	EventNamespaces bool
}

// registry is the supported-game table, keyed by id.
var registry = map[string]*GameInfo{
	"ck3": {
		ID:               "ck3",
		Name:             "Crusader Kings III",
		ShortName:        "CK3",
		WikiAPI:          "https://ck3.paradoxwikis.com/api.php",
		WikiDocPages:     []string{"Effects", "Triggers", "Modifiers", "Scopes"},
		ScriptRoot:       "game",
		Descriptor:       "mod",
		DocsFolderName:   "Crusader Kings III",
		ScriptDocsSubdir: "logs",
		ScriptDocsFormat: "classic",
		SteamAppID:       1158310,
		EventNamespaces:  true,
	},
	"vic3": {
		ID:               "vic3",
		Name:             "Victoria 3",
		ShortName:        "Vic3",
		WikiAPI:          "https://vic3.paradoxwikis.com/api.php",
		WikiDocPages:     []string{"Effects", "Triggers", "Modifiers", "Scopes"},
		ScriptRoot:       "game",
		Descriptor:       "metadata",
		DocsFolderName:   "Victoria 3",
		ScriptDocsSubdir: "docs",
		ScriptDocsFormat: "markdown",
		SteamAppID:       529340,
		EventNamespaces:  true,
	},
	"eu5": {
		ID:               "eu5",
		Name:             "Europa Universalis V",
		ShortName:        "EU5",
		WikiAPI:          "https://eu5.paradoxwikis.com/api.php",
		WikiDocPages:     []string{"Effects", "Triggers", "Modifiers", "Scopes"},
		ScriptRoot:       "",
		StageRoots:       []string{"in_game", "main_menu", "loading_screen"},
		Descriptor:       "metadata",
		DocsFolderName:   "Europa Universalis V",
		ScriptDocsSubdir: "docs",
		ScriptDocsFormat: "markdown",
		EntryModes:       []string{"INJECT", "REPLACE", "TRY_INJECT", "TRY_REPLACE", "INJECT_OR_CREATE", "REPLACE_OR_CREATE"},
		SteamAppID:       3450310,
		EventNamespaces:  true,
	},
}

// Get returns the GameInfo for id, or nil if the game is not supported.
func Get(id string) *GameInfo {
	return registry[id]
}

// All returns the supported games in a stable-enough map (callers sort if needed).
func All() map[string]*GameInfo {
	return registry
}
