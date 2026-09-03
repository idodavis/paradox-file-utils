// scriptname.go maps effect/trigger keys that define or use ephemeral script
// names (character flags, variables, saved scopes) and related RefFieldKind
// / prefix helpers. Game-specific key lists live here, not in catalog or lsp.

package game

import "strings"

// ScriptNameRule describes how an assignment key harvests an ephemeral name.
type ScriptNameRule struct {
	Kind     string // character_flag, variable, global_variable, …
	IsDef    bool   // definition site vs use/ref
	InnerKey string // "" = scalar RHS; else look for this field inside a block
}

// IsEphemeral reports kinds that must not win generic Resolve / outline / conflicts.
func IsEphemeral(kind string) bool {
	switch CanonicalKind(kind) {
	case "character_flag", "variable", "global_variable", "local_variable",
		"dead_character_variable", "saved_scope", "script_param":
		return true
	default:
		return false
	}
}

// PrefixKind maps scope:/var:/global_var:/local_var: to an ephemeral def kind.
func PrefixKind(prefix string) string {
	switch prefix {
	case ScopePrefix:
		return "saved_scope"
	case "var":
		return "variable"
	case "global_var":
		return "global_variable"
	case "local_var":
		return "local_variable"
	default:
		return ""
	}
}

// ScriptName returns the ephemeral-name rule for an assignment key, if any.
func ScriptName(gameID, key string) (ScriptNameRule, bool) {
	k := strings.ToLower(key)
	if r, ok := sharedScriptNames[k]; ok {
		return r, true
	}
	if gameID == "ck3" {
		if r, ok := ck3ScriptNames[k]; ok {
			return r, true
		}
	}
	return ScriptNameRule{}, false
}

// sharedScriptNames are Jomini names present on CK3 / Vic3 / EU5.
var sharedScriptNames = map[string]ScriptNameRule{
	"set_variable":        {Kind: "variable", IsDef: true, InnerKey: "name"},
	"has_variable":        {Kind: "variable"},
	"remove_variable":     {Kind: "variable"},
	"change_variable":     {Kind: "variable"},
	"set_global_variable": {Kind: "global_variable", IsDef: true, InnerKey: "name"},
	"has_global_variable": {Kind: "global_variable"},
	"remove_global_variable": {Kind: "global_variable"},
	"change_global_variable": {Kind: "global_variable"},
	"set_local_variable":     {Kind: "local_variable", IsDef: true, InnerKey: "name"},
	"has_local_variable":     {Kind: "local_variable"},
	"remove_local_variable":  {Kind: "local_variable"},
	"change_local_variable":  {Kind: "local_variable"},

	"save_scope_as":                 {Kind: "saved_scope", IsDef: true},
	"save_temporary_scope_as":       {Kind: "saved_scope", IsDef: true},
	"save_temporary_value_as":       {Kind: "saved_scope", IsDef: true},
	"save_scope_value_as":           {Kind: "saved_scope", IsDef: true, InnerKey: "name"},
	"save_temporary_scope_value_as": {Kind: "saved_scope", IsDef: true, InnerKey: "name"},
	"clear_saved_scope":             {Kind: "saved_scope"},
}

// ck3ScriptNames are CK3-only character / dead-character flags and variables.
var ck3ScriptNames = map[string]ScriptNameRule{
	"add_character_flag":          {Kind: "character_flag", IsDef: true, InnerKey: "flag"},
	"add_dead_character_flag":     {Kind: "character_flag", IsDef: true, InnerKey: "flag"},
	"has_character_flag":          {Kind: "character_flag"},
	"has_dead_character_flag":     {Kind: "character_flag"},
	"remove_character_flag":       {Kind: "character_flag"},
	"set_dead_character_variable": {Kind: "dead_character_variable", IsDef: true, InnerKey: "name"},
	"has_dead_character_variable": {Kind: "dead_character_variable"},
	"remove_dead_character_variable": {Kind: "dead_character_variable"},
}

// IsSaveScopeKey reports a scalar `save_*_scope_as = name` (or temporary value) site.
func IsSaveScopeKey(key string) bool {
	r, ok := ScriptName("", key)
	return ok && r.Kind == "saved_scope" && r.IsDef && r.InnerKey == ""
}

// IsSaveScopeValueKey reports a block `save_*_scope_value_as = { name = X }`.
func IsSaveScopeValueKey(key string) bool {
	r, ok := ScriptName("", key)
	return ok && r.Kind == "saved_scope" && r.IsDef && r.InnerKey == "name"
}
