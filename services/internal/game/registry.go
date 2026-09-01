// registry.go holds the GameInfo identity table for the supported games and Get.

package game

// OriginVanilla is the session/IDE origin id for game files (not a cache filename).
const OriginVanilla = "vanilla"

// GameInfo is the static identity and convention set for one game. It carries no
// derived data — the model package scans the install for objects, vocabulary, etc.
type GameInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`

	// WikiAPI is the MediaWiki api.php endpoint for this game's wiki.
	WikiAPI string `json:"wikiApi,omitempty"`

	// ScriptRoot is the install-relative folder holding script (e.g. "game").
	// Empty for EU5, whose content is split across StageRoots.
	ScriptRoot string   `json:"scriptRoot,omitempty"`
	StageRoots []string `json:"stageRoots,omitempty"`

	// Descriptor is how a mod root is identified: "mod" (descriptor.mod) or
	// "metadata" (.metadata/metadata.json).
	Descriptor string `json:"descriptor,omitempty"`

	// DocsFolderName is the per-game Documents subfolder (e.g. "Crusader Kings III").
	DocsFolderName string `json:"docsFolderName,omitempty"`
	// ScriptDocsSubdir is where in-game console dumps land ("logs" or "docs").
	ScriptDocsSubdir string `json:"scriptDocsSubdir,omitempty"`
	// ScriptDocsFormat is "classic" (CK3 `_*.info`) or "markdown" (Vic3/EU5 `*.md`).
	ScriptDocsFormat string `json:"scriptDocsFormat,omitempty"`

	// EntryModes are EU5 database entry-mode prefixes (INJECT:, REPLACE:, ...).
	EntryModes []string `json:"entryModes,omitempty"`

	SteamAppID int `json:"steamAppId,omitempty"`
}

// GameList is the Wails payload for ListGames (consts are not exported).
type GameList struct {
	OriginVanilla string      `json:"originVanilla"`
	Games         []*GameInfo `json:"games"`
}

// registry is the supported-game table, keyed by id.
var registry = map[string]*GameInfo{
	"ck3": {
		ID:               "ck3",
		Name:             "Crusader Kings III",
		ShortName:        "CK3",
		WikiAPI:          "https://ck3.paradoxwikis.com/api.php",
		ScriptRoot:       "game",
		Descriptor:       "mod",
		DocsFolderName:   "Crusader Kings III",
		ScriptDocsSubdir: "logs",
		ScriptDocsFormat: "classic",
		SteamAppID:       1158310,
	},
	"vic3": {
		ID:               "vic3",
		Name:             "Victoria 3",
		ShortName:        "Vic3",
		WikiAPI:          "https://vic3.paradoxwikis.com/api.php",
		ScriptRoot:       "game",
		Descriptor:       "metadata",
		DocsFolderName:   "Victoria 3",
		ScriptDocsSubdir: "docs",
		ScriptDocsFormat: "markdown",
		SteamAppID:       529340,
	},
	"eu5": {
		ID:               "eu5",
		Name:             "Europa Universalis V",
		ShortName:        "EU5",
		WikiAPI:          "https://eu5.paradoxwikis.com/api.php",
		ScriptRoot:       "",
		StageRoots:       []string{"in_game", "main_menu", "loading_screen"},
		Descriptor:       "metadata",
		DocsFolderName:   "Europa Universalis V",
		ScriptDocsSubdir: "docs",
		ScriptDocsFormat: "markdown",
		EntryModes:       []string{"INJECT", "REPLACE", "TRY_INJECT", "TRY_REPLACE", "INJECT_OR_CREATE", "REPLACE_OR_CREATE"},
		SteamAppID:       3450310,
	},
}

// Get returns the GameInfo for id, or nil if the game is not supported.
func Get(id string) *GameInfo {
	return registry[id]
}

// All returns supported games in UI order (ck3, eu5, vic3).
func All() []*GameInfo {
	return []*GameInfo{registry["ck3"], registry["eu5"], registry["vic3"]}
}
