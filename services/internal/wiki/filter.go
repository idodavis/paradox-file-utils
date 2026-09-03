package wiki

import (
	"regexp"
	"strings"
)

var (
	listOfRe  = regexp.MustCompile(`(?i)^List of\b`)
	listEndRe = regexp.MustCompile(`(?i)\blist$`)
	userRe    = regexp.MustCompile(`(?i)^User:`)
)

var denyExact = map[string]bool{
	"pdx deepl":                    true,
	"modding tools":                true,
	"exporters":                    true,
	"effects list":                 true,
	"triggers list":                true,
	"list of baronies":             true,
	"list of gui script functions": true,
}

func normTitle(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	return strings.ToLower(strings.TrimSpace(s))
}

// denied reports whether a ns-0 title should be skipped (tools, dumps, user pages).
func denied(title string) bool {
	n := normTitle(title)
	if denyExact[n] || userRe.MatchString(title) {
		return true
	}
	if listOfRe.MatchString(title) || listEndRe.MatchString(strings.TrimSpace(title)) {
		return true
	}
	return false
}

func keepMember(title string, ns int, mods map[string]bool) bool {
	if ns != 0 || denied(title) {
		return false
	}
	return !mods[normTitle(title)]
}
