// format.go is a conservative formatter: script files get one tab per brace
// depth (lexer-based); loc files get a header at column 0 and one-space entries.

package lsp

import (
	"strings"

	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

// FormatDocument returns leading-whitespace edits for path, or nil if already
// formatted / empty.
func FormatDocument(s *session.Session, path string) []TextEdit {
	src := fileText(s, path)
	if src == "" || isMetaJSON(path) {
		return nil
	}
	if isLocPath(path) {
		return formatLoc(src)
	}
	return formatScript(src)
}

func formatScript(src string) []TextEdit {
	indents := parser.LineIndents(src)
	lines := splitKeep(src)
	var edits []TextEdit
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cur := leadingWS(line)
		want := strings.Repeat("\t", indents[i])
		if cur == want {
			continue
		}
		edits = append(edits, TextEdit{
			Range: Range{
				Start: Position{Line: i, Character: 0},
				End:   Position{Line: i, Character: len(cur)},
			},
			NewText: want,
		})
	}
	return edits
}

func formatLoc(src string) []TextEdit {
	lines := splitKeep(src)
	var edits []TextEdit
	headerDone := false
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		cur := leadingWS(line)
		want := ""
		if headerDone {
			want = " "
		} else {
			headerDone = true
		}
		if cur == want {
			continue
		}
		edits = append(edits, TextEdit{
			Range: Range{
				Start: Position{Line: i, Character: 0},
				End:   Position{Line: i, Character: len(cur)},
			},
			NewText: want,
		})
	}
	return edits
}

func leadingWS(line string) string {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return line[:i]
}

func splitKeep(src string) []string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	return strings.Split(src, "\n")
}
