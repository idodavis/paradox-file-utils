// properties.go lists script properties whose right-hand side is a localization
// key, split into STRICT (unresolved => "missing loc") and BROAD (resolved hint
// only). These are engine property names, not game data.

package loc

import "strings"

// strictProps virtually always hold a loc key.
var strictProps = map[string]bool{
	"title":             true,
	"desc":              true,
	"flavor":            true,
	"custom_tooltip":    true,
	"selection_tooltip": true,
	"confirm_text":      true,
	"confirm_title":     true,
	"prompt":            true,
	"failure_desc":      true,
	"success_desc":      true,
	"localization_key":  true,
	"war_name":          true,
	"cb_name":           true,
	"notification_text": true,
	// trigger_localization / effect_localization person+tense slots
	"first": true, "third": true, "global": true, "none": true,
	"first_not": true, "third_not": true, "global_not": true, "none_not": true,
	"first_past": true, "third_past": true, "global_past": true,
	"first_neg": true, "third_neg": true, "global_neg": true,
	"first_past_neg": true, "third_past_neg": true, "global_past_neg": true,
}

// broadProps often hold a loc key (superset of strict). These also hold non-loc
// values, so only a resolved key produces a hint.
var broadProps = map[string]bool{
	"name": true, "text": true, "tooltip": true, "first_valid": true,
	"reason": true, "format": true, "header": true, "opinion_text": true,
	"what": true, "who": true, "notification": true,
}

// Property classifies a script property name as a loc-key holder.
type Property string

const (
	// PropNone is a property that does not hold a loc key.
	PropNone Property = ""
	// PropStrict flags an unresolved value as a missing-loc diagnostic.
	PropStrict Property = "strict"
	// PropBroad shows a resolved-loc hint only, silent when unresolved.
	PropBroad Property = "broad"
)

// Classify returns whether prop holds a loc key. Names match the listed
// lowercase engine fields (`title`, `desc`); ALL_CAPS trigger arguments
// (`TITLE = primary_title`) are not loc properties.
func Classify(prop string) Property {
	if strictProps[prop] {
		return PropStrict
	}
	if broadProps[prop] {
		return PropBroad
	}
	return PropNone
}

// LooksLikeKey reports whether s is a plausible loc key (not yes/no/none,
// not a `scope:` / `character:` prefix, not a built-in scope).
func LooksLikeKey(s string) bool {
	if s == "" || strings.Contains(s, ":") {
		return false
	}
	switch strings.ToLower(s) {
	case "yes", "no", "none", "root", "prev", "this", "from":
		return false
	}
	return true
}
