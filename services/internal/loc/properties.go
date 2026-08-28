// properties.go lists script properties whose right-hand side is a localization
// key, split into STRICT (unresolved => "missing loc") and BROAD (resolved hint
// only). These are engine property names, not game data.

package loc

import "strings"

// strictProps virtually always hold a loc key.
var strictProps = map[string]bool{
	"title":         true,
	"desc":          true,
	"flavor":        true,
	"custom_tooltip": true,
	"confirm_text":  true,
	"confirm_title": true,
	"prompt":        true,
	"failure_desc":  true,
	"success_desc":  true,
}

// broadProps often hold a loc key (superset of strict). These also hold non-loc
// values, so only a resolved key produces a hint.
var broadProps = map[string]bool{
	"name": true, "text": true, "tooltip": true, "first_valid": true,
	"reason": true, "format": true, "header": true, "opinion_text": true,
	"what": true, "who": true,
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

// Classify returns whether prop holds a loc key (case-insensitive). Strict wins
// over broad; broadProps holds only the additional (non-strict) names.
func Classify(prop string) Property {
	p := strings.ToLower(prop)
	if strictProps[p] {
		return PropStrict
	}
	if broadProps[p] {
		return PropBroad
	}
	return PropNone
}
