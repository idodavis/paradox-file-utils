// rules.go is the only place that hardcodes engine grammar: how a folder maps to
// a definition kind/extract mode, which kinds are "calls" (scripted_effect etc.),
// which kinds use FIOS override order, and how EU5 MODE: key prefixes are stripped
// to a canonical identity. This is engine mechanics (a few small tables), never
// derived game data.

package game

import (
	"slices"
	"strings"
)

// ExtractMode names how definition keys are read out of a file.
type ExtractMode string

const (
	ModeTopLevelKey ExtractMode = "top-level-key" // root `key = { ... }`
	ModeEventID     ExtractMode = "event-id"      // `namespace.1 = { ... }`
	ModeGUIType     ExtractMode = "gui-type"      // GUI `type X = ...` / `template X { }`
	ModeLocKey      ExtractMode = "loc-key"       // localization `key: "..."`
)

// ExtractRule says what kind of definitions a file yields and how to read them.
type ExtractRule struct {
	Kind string
	Mode ExtractMode
}

// MatchExtract maps a file path (relative to the game/mod script root) to its
// extract rule. The default is "parent folder name => kind, top-level-key";
// events/gui/localization override that. EU5 stage-root prefixes are ignored.
func MatchExtract(gameID, relPath string) ExtractRule {
	segs := pathSegments(relPath)
	segs = stripStageRoot(gameID, segs)
	if slices.Contains(segs, "localization") {
		return ExtractRule{Kind: "loc_key", Mode: ModeLocKey}
	}
	if isModFile(relPath) {
		return ExtractRule{Kind: "mod_descriptor", Mode: ModeTopLevelKey}
	}
	parent := parentFolder(segs)
	switch {
	case parent == "events" || slices.Contains(segs, "events"):
		return ExtractRule{Kind: "event", Mode: ModeEventID}
	case parent == "gui" || strings.HasSuffix(strings.ToLower(relPath), ".gui"):
		return ExtractRule{Kind: "gui_type", Mode: ModeGUIType}
	default:
		return ExtractRule{Kind: CanonicalKind(parent), Mode: ModeTopLevelKey}
	}
}

// CanonicalKind maps folder plurals to singular def types used everywhere else.
func CanonicalKind(kind string) string {
	switch kind {
	case "scripted_effects":
		return "scripted_effect"
	case "scripted_triggers":
		return "scripted_trigger"
	case "scripted_modifiers":
		return "scripted_modifier"
	case "on_actions":
		return "on_action"
	case "decisions":
		return "decision"
	default:
		return kind
	}
}

// DescriptorModKeys is the static .mod / descriptor.mod key list (CK3-style).
var DescriptorModKeys = []string{
	"name", "version", "tags", "supported_version", "path",
	"remote_file_id", "picture", "dependencies",
}

// isModFile reports whether relPath is a Paradox .mod descriptor.
func isModFile(relPath string) bool {
	p := strings.ReplaceAll(relPath, "\\", "/")
	base := p
	if i := strings.LastIndex(p, "/"); i >= 0 {
		base = p[i+1:]
	}
	return strings.HasSuffix(strings.ToLower(base), ".mod")
}

// callKinds are definition kinds whose key IS a callable name (invoked by writing
// the key as an effect/trigger).
var callKinds = map[string]bool{
	"scripted_effect":   true,
	"scripted_trigger":  true,
	"scripted_modifier": true,
}

// IsCallKind reports whether kind's keys are called directly as effects/triggers.
func IsCallKind(kind string) bool { return callKinds[CanonicalKind(kind)] }

// IsFIOS reports whether kind resolves first-in-order for this game.
// Empty/unknown gameID keeps the historical gui_type-only default.
func IsFIOS(gameID, kind string) bool {
	switch gameID {
	case "vic3":
		return kind == "gui_type" || kind == "event"
	case "eu5":
		return kind == "event"
	default:
		return kind == "gui_type"
	}
}

// ScopePrefix is the saved-scope qualifier (`scope:target`).
const ScopePrefix = "scope"

var saveScopeKeys = map[string]bool{
	"save_scope_as":           true,
	"save_temporary_scope_as": true,
	"save_temporary_value_as": true,
}

var saveScopeValueKeys = map[string]bool{
	"save_scope_value_as":           true,
	"save_temporary_scope_value_as": true,
}

// IsSaveScopeKey reports a scalar `save_*_scope_as = name` (or value_as) site.
func IsSaveScopeKey(key string) bool { return saveScopeKeys[key] }

// IsSaveScopeValueKey reports a block `save_*_scope_value_as = { name = X }`.
func IsSaveScopeValueKey(key string) bool { return saveScopeValueKeys[key] }

// eventGateKeys are event-body blocks that gate whether the event runs.
var eventGateKeys = map[string]bool{
	"trigger":              true,
	"cancellation_trigger": true,
	"on_trigger_fail":      true,
}

// EventSectionRole is "gate" for run-conditions, else "effect".
func EventSectionRole(name string) string {
	if eventGateKeys[strings.ToLower(name)] {
		return "gate"
	}
	return "effect"
}

var triggerSlotKeys = map[string]bool{
	"limit": true, "potential": true,
	"and": true, "or": true, "not": true, "nor": true, "nand": true,
}

var effectSlotKeys = map[string]bool{
	"immediate": true, "option": true, "after": true, "else": true, "effect": true,
}

// ScriptSlot is the completion bag for a block named key: "trigger", "effect",
// or "" (object-body structure keys). Event gates map to trigger; EventSectionRole's
// default "effect" is not used (title/type are not effect slots).
func ScriptSlot(key string) string {
	k := strings.ToLower(key)
	if EventSectionRole(k) == "gate" {
		return "trigger"
	}
	if triggerSlotKeys[k] {
		return "trigger"
	}
	if effectSlotKeys[k] {
		return "effect"
	}
	switch {
	case strings.HasPrefix(k, "any_"):
		return "trigger"
	case strings.HasPrefix(k, "every_"),
		strings.HasPrefix(k, "random_"),
		strings.HasPrefix(k, "ordered_"):
		return "effect"
	}
	return ""
}

// RefFieldKind is a thin field→def-kind map used when FieldValueKind is empty.
func RefFieldKind(field string) string {
	switch strings.ToLower(field) {
	case "culture":
		return "culture"
	case "faith":
		return "faith"
	case "title":
		return "title"
	case "define":
		return "define"
	default:
		return ""
	}
}

// RequiredLocKeys are convention loc keys for a def kind+id (empty if none).
func RequiredLocKeys(kind, id string) []string {
	if id == "" {
		return nil
	}
	switch CanonicalKind(kind) {
	case "trait", "traits":
		return []string{"trait_" + id}
	case "game_rules", "game_rule":
		return []string{"rule_" + id}
	default:
		return nil
	}
}

// FireKind reports whether key is an event or on_action fire site.
func FireKind(key string) string {
	switch strings.ToLower(key) {
	case "trigger_event", "events", "random_events", "first_valid", "fallback":
		return "event"
	case "on_action", "on_actions":
		return "on_action"
	default:
		return ""
	}
}

// KeyIdentity canonicalizes a raw key for matching/merge: trims whitespace and
// surrounding quotes, and strips an EU5 entry-mode prefix (INJECT:foo -> foo).
// gameID "" (e.g. merge, which is game-agnostic) strips any known EU5 mode.
func KeyIdentity(gameID, key string) string {
	key = trimQuotes(strings.TrimSpace(key))
	if i := strings.IndexByte(key, ':'); i > 0 && isEntryMode(gameID, key[:i]) {
		return key[i+1:]
	}
	return key
}

// eu5EntryModes is the union of EU5 entry-mode prefixes, used when the game is
// unknown (merge passes gameID "").
var eu5EntryModes = map[string]bool{
	"INJECT": true, "REPLACE": true, "TRY_INJECT": true,
	"TRY_REPLACE": true, "INJECT_OR_CREATE": true, "REPLACE_OR_CREATE": true,
}

func isEntryMode(gameID, tok string) bool {
	if g := Get(gameID); g != nil {
		for _, m := range g.EntryModes {
			if m == tok {
				return true
			}
		}
		return false
	}
	return eu5EntryModes[tok]
}

// trimQuotes removes a single pair of surrounding single or double quotes.
func trimQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// pathSegments splits a slash- or backslash-separated path into non-empty segments.
func pathSegments(p string) []string {
	p = strings.ReplaceAll(p, "\\", "/")
	raw := strings.Split(p, "/")
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		if s != "" && s != "." {
			out = append(out, s)
		}
	}
	return out
}

// parentFolder returns the directory segment immediately containing the file.
func parentFolder(segs []string) string {
	if len(segs) >= 2 {
		return segs[len(segs)-2]
	}
	return ""
}

// stripStageRoot drops a leading EU5 stage-root segment (in_game, etc.) so extract
// rules see the same folder layout across games.
func stripStageRoot(gameID string, segs []string) []string {
	g := Get(gameID)
	if g == nil || len(segs) == 0 {
		return segs
	}
	for _, stage := range g.StageRoots {
		if segs[0] == stage {
			return segs[1:]
		}
	}
	return segs
}
