// lang.go derives the localization language id (e.g. "english") from a loc file
// name or header token.

package loc

import (
	"path/filepath"
	"regexp"
	"strings"
)

// filenameLangRe matches the `l_<language>` token that precedes the extension in
// a loc filename, e.g. `events_l_english.yml` -> "english".
var filenameLangRe = regexp.MustCompile(`(?i)l_([a-z]+)\.(yml|yaml)$`)

// LanguageFromFilename returns the language id encoded in a loc filename, or ""
// if the name does not follow the `..._l_<language>.yml` convention.
func LanguageFromFilename(path string) string {
	base := filepath.Base(path)
	if m := filenameLangRe.FindStringSubmatch(base); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// LanguageFromRel returns the loc language from the filename, else from a
// `localization/<lang>/` path segment, or "".
func LanguageFromRel(path string) string {
	if l := LanguageFromFilename(path); l != "" {
		return l
	}
	slash := strings.ReplaceAll(path, "\\", "/")
	lower := strings.ToLower(slash)
	const marker = "/localization/"
	i := strings.Index(lower, marker)
	if i < 0 {
		return ""
	}
	rest := slash[i+len(marker):]
	lang, _, _ := strings.Cut(rest, "/")
	return strings.ToLower(lang)
}

// LanguageOf returns the loc language from the filename, else the parsed header.
func LanguageOf(path, parsedLang string) string {
	if l := LanguageFromFilename(path); l != "" {
		return l
	}
	return strings.ToLower(parsedLang)
}
