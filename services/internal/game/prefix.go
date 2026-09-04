// prefix.go parses `scope:name` (and var: prefixes the graph still labels)
// plus first-class `type:id` object cites (`culture:english`).

package game

import (
	"regexp"
	"strings"
	"unicode"
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
	return p.Prefix == ScopePrefix
}

// SkipObjectRHS reports interpolation (`$COA$`) or a dotted link/scope chain
// (`capital_province.culture`) — not an object id.
func SkipObjectRHS(text string) bool {
	return text == "" || strings.ContainsAny(text, "$.")
}

// SkipFieldRHS reports a field value that is grammar, not an object id.
// `titles = target_titles` is a CB target selector, not a landed title.
func SkipFieldRHS(kind, text string) bool {
	if SkipObjectRHS(text) {
		return true
	}
	switch CanonicalKind(kind) {
	case "title":
		switch strings.ToLower(text) {
		case "none", "target_titles", "all":
			return true
		}
	}
	return false
}

// ParseTyped splits `culture:english` into a harvested kind and id.
// Ephemeral prefixes (scope:/var:) and EU5 entry modes are not typed.
func ParseTyped(gameID, text string) (kind, id string, ok bool) {
	i := strings.IndexByte(text, ':')
	if i <= 0 || i >= len(text)-1 {
		return "", "", false
	}
	prefix, rest := text[:i], text[i+1:]
	if PrefixKind(prefix) != "" || isEntryMode(gameID, prefix) {
		return "", "", false
	}
	if !typedIdent(rest) || strings.ContainsAny(rest, ":.$") {
		return "", "", false
	}
	kind = typedKind(prefix)
	if kind == "" {
		return "", "", false
	}
	return kind, rest, true
}

func typedIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func typedKind(prefix string) string {
	p := strings.ToLower(prefix)
	if PrefixKind(p) != "" {
		return ""
	}
	if k := RefFieldKind("", p); k != "" && !IsEphemeral(k) {
		return k
	}
	ck := CanonicalKind(p)
	if typedKindOK(ck) {
		return ck
	}
	if typedKindOK(p + "s") {
		return p + "s"
	}
	return ""
}

func typedKindOK(k string) bool {
	if k == "" || IsEphemeral(k) {
		return false
	}
	switch k {
	case "effect", "trigger", "scope", "field", "loc_key", "loc_value",
		"data_function", "mod_descriptor", "animation":
		return false
	}
	_, ok := kindMeta[k]
	return ok
}
