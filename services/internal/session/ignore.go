// ignore.go reads inline diagnostic suppressions from comments: `# pmt:ignore`
// suppresses its own line and `# pmt:ignore-next-line` the following line. A bare
// marker suppresses every code on the target line; codes after it limit it to
// those codes. A trailing `-- rationale` is allowed and ignored. Fail-soft: a
// malformed comment is skipped, never fatal.

package session

import (
	"regexp"
	"strings"
)

// ignoreRe captures the optional `-next-line` suffix and the trailing code list.
var ignoreRe = regexp.MustCompile(`(?i)#\s*pmt:ignore(-next-line)?\b(.*)`)

// lineSuppression is what is suppressed on one line: all codes, or a specific set.
type lineSuppression struct {
	all   bool
	codes map[string]bool
}

// Suppressions maps a 0-based line to what diagnostics are suppressed on it.
type Suppressions map[int]lineSuppression

// ScanSuppressions builds the suppression map from source text (script or loc).
func ScanSuppressions(text string) Suppressions {
	sup := Suppressions{}
	if !strings.Contains(text, "pmt:ignore") {
		return sup
	}
	line, start := 0, 0
	for start <= len(text) {
		end := start
		for end < len(text) && text[end] != '\n' && text[end] != '\r' {
			end++
		}
		sup.scanLine(text[start:end], line)
		if end < len(text) && text[end] == '\r' && end+1 < len(text) && text[end+1] == '\n' {
			end++
		}
		start = end + 1
		line++
		if start > len(text) {
			break
		}
	}
	return sup
}

// scanLine records a pmt:ignore directive on one line.
func (s Suppressions) scanLine(lineText string, line int) {
	hash := strings.IndexByte(lineText, '#')
	if hash < 0 {
		return
	}
	m := ignoreRe.FindStringSubmatch(lineText[hash:])
	if m == nil {
		return
	}
	target := line
	if m[1] != "" {
		target = line + 1
	}
	s.merge(target, parseCodes(m[2]))
}

// Covers reports whether line suppresses code (an empty code matches only all-mode).
func (s Suppressions) Covers(line int, code string) bool {
	ls, ok := s[line]
	if !ok {
		return false
	}
	return ls.all || (code != "" && ls.codes[code])
}

// merge folds a directive into the map; all-mode wins over specific codes.
func (s Suppressions) merge(line int, codes []string) {
	ls := s[line]
	if len(codes) == 0 {
		ls.all = true
	} else if !ls.all {
		if ls.codes == nil {
			ls.codes = map[string]bool{}
		}
		for _, c := range codes {
			ls.codes[c] = true
		}
	}
	s[line] = ls
}

// parseCodes reads the whitespace-separated codes after the marker, stopping at a
// `--` rationale. No codes means suppress everything on the line.
func parseCodes(rest string) []string {
	var out []string
	for _, tok := range strings.Fields(rest) {
		if strings.HasPrefix(tok, "-") {
			break
		}
		out = append(out, tok)
	}
	return out
}
