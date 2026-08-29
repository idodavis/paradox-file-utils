// types.go declares the persisted semantic-model shapes and their format-version
// constants. Bumping a version invalidates on-disk files (persist.go rejects a
// mismatch and the caller rescans); there is no migration path by design.

package model

// CacheFormatVersion is the on-disk schema version of a vanilla Cache. Bump it
// whenever the Cache shape changes; LoadCache rejects any other version.
const CacheFormatVersion = 4

// IndexFormatVersion is the on-disk schema version of a workspace Index; bumping
// it invalidates saved indexes (LoadIndex rejects a mismatch, caller rebuilds).
const IndexFormatVersion = 5

// Def is one definition: a named object the game (or a mod) declares. Path is the
// absolute file it lives in; Line is 0-based. Type is the definition kind as
// classified by game.MatchExtract (e.g. "trait", "event", "scripted_effect").
// Origin is the mod id that declared it, or "" for vanilla.
type Def struct {
	Type   string `json:"type"`
	Key    string `json:"key"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Origin string `json:"origin,omitempty"`
}

// Ref is a use-site pointing at a definition key. Kind is "loc", "loc-broad",
// "event", or "on_action". Start/End are UTF-8 byte offsets of the referenced
// token.
type Ref struct {
	Key   string `json:"key"`
	Kind  string `json:"kind"`
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// Edge is a directed event-graph link, e.g. one event's `trigger_event` pointing
// at another event id. From is "" when the source is not itself an event.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Via  string `json:"via"`
	Path string `json:"path"`
	Line int    `json:"line"`
}

// ModInput is one mod in a workspace: a stable Origin id, its Root folder, and its
// load Order (ascending = loaded earlier; later mods win under LIOS).
type ModInput struct {
	Origin string
	Root   string
	Order  int
}

// Index is the workspace semantic model: every mod's defs, references, and event
// edges, the English loc map, and the origins in load order. Rebuilt on demand;
// persisted per workspace and discarded on a format mismatch.
type Index struct {
	FormatVersion int               `json:"formatVersion"`
	WorkspaceID   string            `json:"workspaceId"`
	BuiltAt       string            `json:"builtAt"`
	Defs          []Def             `json:"defs"`
	Refs          []Ref             `json:"refs"`
	Edges         []Edge            `json:"edges"`
	Loc           map[string]string `json:"loc"`
	Order         []string          `json:"order"`
}

// Cache is the vanilla semantic model for one workspace's install scan.
// All string slices are sorted, de-duplicated membership sets (no frequencies).
type Cache struct {
	FormatVersion int    `json:"formatVersion"`
	WorkspaceID   string `json:"workspaceId"`
	GameID        string `json:"gameId"`
	InstallPath   string `json:"installPath"`
	GameVersion   string `json:"gameVersion"`
	ScannedAt     string `json:"scannedAt"`

	// Defs are the vanilla object definitions (script kinds; not loc keys).
	Defs []Def `json:"defs"`
	// LocEnglish maps every english loc key to its (length-capped) value.
	LocEnglish map[string]string `json:"locEnglish"`
	// LocEnglishSites maps every english loc key to its defining file and line.
	LocEnglishSites map[string]LocSite `json:"locEnglishSites"`
	// FieldDocs maps a field/token key to human doc prose from shipped docs or
	// a script_docs dump (global first-wins fallback).
	FieldDocs map[string]string `json:"fieldDocs"`
	// FieldDocsByKind maps a definition kind to field-key prose from `_*.info`
	// files classified by game.MatchExtract (kind → field → prose).
	FieldDocsByKind map[string]map[string]string `json:"fieldDocsByKind"`
	// Structures maps a definition kind to the set of keys valid as its direct
	// children (union of documented keys and keys observed in the corpus).
	Structures map[string][]string `json:"structures"`

	// Vocabulary is the corpus baseline: every script key observed inside
	// definition bodies. Always populated, even with no script_docs dump.
	Vocabulary []string `json:"vocabulary"`
	// Effects/Triggers/Modifiers are classified by a script_docs dump when one is
	// present; otherwise empty and callers fall back to Vocabulary.
	Effects   []string `json:"effects"`
	Triggers  []string `json:"triggers"`
	Modifiers []string `json:"modifiers"`
	// Objects are the referable definition names (every Def.Key).
	Objects []string `json:"objects"`

	// GUITypes / GUIProps are the widget type names and property keys from .gui.
	GUITypes []string `json:"guiTypes"`
	GUIProps []string `json:"guiProps"`
	// MetaKeys are the top-level keys seen in .metadata/metadata.json files.
	MetaKeys []string `json:"metaKeys"`
}

// LocSite is one vanilla english localization definition (file + 0-based line).
type LocSite struct {
	File string `json:"file"`
	Line int    `json:"line"`
}
