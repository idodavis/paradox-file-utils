// Package game provides game registry constants and metadata for supported Paradox titles.
package game

// GameInfo holds static metadata for a supported Paradox game.
type GameInfo struct {
	ID              string
	Name            string
	WikiAPI         string
	ScriptRoot      string
	GuiRoots        []string
	LocRoots        []string
	DocumentsFolder string
	ScriptDocsRel   string
	SteamAppID      int
}

// Registry contains metadata for all supported games.
var Registry = map[string]GameInfo{
	"ck3": {
		ID:              "ck3",
		Name:            "Crusader Kings III",
		WikiAPI:         "https://ck3.paradoxwikis.com/api.php",
		ScriptRoot:      "game",
		GuiRoots:        []string{"game/gui"},
		LocRoots:        []string{"game/localization"},
		DocumentsFolder: "Crusader Kings III",
		ScriptDocsRel:   "logs",
		SteamAppID:      1158310,
	},
	"eu5": {
		ID:              "eu5",
		Name:            "Europa Universalis V",
		WikiAPI:         "https://eu5.paradoxwikis.com/api.php",
		ScriptRoot:      "game/in_game",
		GuiRoots:        []string{"game/in_game/gui", "game/main_menu/gui"},
		LocRoots:        []string{"game/main_menu/localization", "game/in_game/localization", "game/loading_screen/localization"},
		DocumentsFolder: "Europa Universalis V",
		ScriptDocsRel:   "docs",
		SteamAppID:      3450310,
	},
	"vic3": {
		ID:              "vic3",
		Name:            "Victoria 3",
		WikiAPI:         "https://vic3.paradoxwikis.com/api.php",
		ScriptRoot:      "game",
		GuiRoots:        []string{"game/gui"},
		LocRoots:        []string{"game/localization"},
		DocumentsFolder: "Victoria 3",
		ScriptDocsRel:   "logs",
		SteamAppID:      529340,
	},
}

// Get returns game info by ID, or nil if not found.
func Get(gameID string) *GameInfo {
	if info, ok := Registry[gameID]; ok {
		return &info
	}
	return nil
}

// IDs returns all supported game IDs.
func IDs() []string {
	return []string{"ck3", "eu5", "vic3"}
}
