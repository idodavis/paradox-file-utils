// prefix.go parses `scope:name` (and var: prefixes the graph still labels).

package model

import (
	"regexp"

	"paradox-modding-tools/services/internal/game"
)

var prefixedRe = regexp.MustCompile(
	`^(scope|var|local_var|global_var):([A-Za-z][A-Za-z0-9_]*)`,
)

// Prefixed is one `prefix:name` token. NameOff is the byte offset of Name in Text.
type Prefixed struct {
	Prefix  string
	Name    string
	NameOff int
	Text    string
}

// ParsePrefixed splits a bare scalar/key. The segment after the first `.` is ignored.
func ParsePrefixed(text string) (Prefixed, bool) {
	m := prefixedRe.FindStringSubmatch(text)
	if m == nil {
		return Prefixed{}, false
	}
	return Prefixed{
		Prefix:  m[1],
		Name:    m[2],
		NameOff: len(m[1]) + 1,
		Text:    text,
	}, true
}

// IsSavedScopePrefix reports whether p is a `scope:` reference (not var:).
func IsSavedScopePrefix(p Prefixed) bool {
	return p.Prefix == game.ScopePrefix
}
