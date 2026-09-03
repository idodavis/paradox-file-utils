package wiki

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	backtickRe = regexp.MustCompile("`([^`]+)`")
	quotedRe   = regexp.MustCompile(`'([A-Za-z0-9_./-]+)'`)
	snakeRe    = regexp.MustCompile(`\b[a-z][a-z0-9]*(?:_[a-z0-9]+)+\b`)
	pathRe     = regexp.MustCompile(`\b(?:common|events|gui|localization)/[A-Za-z0-9_./-]+`)
	renameRe   = regexp.MustCompile(`(?i)renamed\s+` + "`" + `?([A-Za-z0-9_]+)` + "`" + `?\s+to\s+` + "`" + `?([A-Za-z0-9_]+)` + "`" + `?`)
)

// Token is one identifier extracted from a Modding bullet.
type Token struct {
	Value  string
	Kind   string // ident, path, rename
	To     string // rename target
	Bullet string
}

func splitBullets(html string) []string {
	raw := html
	raw = strings.ReplaceAll(raw, "</li>", "\n")
	raw = strings.ReplaceAll(raw, "</p>", "\n")
	raw = strings.ReplaceAll(raw, "<br/>", "\n")
	raw = strings.ReplaceAll(raw, "<br>", "\n")
	var b strings.Builder
	inTag := false
	for _, r := range raw {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	var out []string
	for _, line := range strings.Split(b.String(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func keepIdent(s string) bool {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			n++
		}
	}
	return n >= 4
}

func addIdent(seen map[string]bool, out *[]Token, bullet, v, kind, to string) {
	v = strings.TrimSpace(v)
	if v == "" || seen[kind+"\x00"+v+"\x00"+to] {
		return
	}
	if kind == "ident" && !keepIdent(v) {
		return
	}
	seen[kind+"\x00"+v+"\x00"+to] = true
	*out = append(*out, Token{Value: v, Kind: kind, To: to, Bullet: bullet})
}

// Tokenize extracts backtick / quoted / snake_case / path / renamed-X-to-Y tokens.
func Tokenize(html string) []Token {
	var out []Token
	seen := map[string]bool{}
	for _, bullet := range splitBullets(html) {
		for _, m := range renameRe.FindAllStringSubmatch(bullet, -1) {
			addIdent(seen, &out, bullet, m[1], "rename", m[2])
		}
		for _, m := range backtickRe.FindAllStringSubmatch(bullet, -1) {
			v := m[1]
			if strings.Contains(v, "/") {
				addIdent(seen, &out, bullet, v, "path", "")
			} else {
				addIdent(seen, &out, bullet, v, "ident", "")
			}
		}
		for _, m := range quotedRe.FindAllStringSubmatch(bullet, -1) {
			addIdent(seen, &out, bullet, m[1], "ident", "")
		}
		for _, m := range pathRe.FindAllString(bullet, -1) {
			addIdent(seen, &out, bullet, m, "path", "")
		}
		for _, m := range snakeRe.FindAllString(bullet, -1) {
			if keepIdent(m) {
				addIdent(seen, &out, bullet, m, "ident", "")
			}
		}
	}
	return out
}
