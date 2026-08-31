// types.go holds LSP DTOs and shared cursor/path helpers.
package lsp

import (
	"os"
	"strings"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/session"
)

// Position is a 0-based line and UTF-8 byte column.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is a half-open LSP-style span in UTF-8 byte columns.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Diagnostic is a parse or semantic issue for a file.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"` // 1 error, 2 warning
	Message  string `json:"message"`
	Code     string `json:"code,omitempty"`
}

// HoverResult is hover card text. Origin/rel/line describe the def site.
type HoverResult struct {
	Contents string `json:"contents"`
	Origin   string `json:"origin,omitempty"`
	Rel      string `json:"rel,omitempty"`
	Line     int    `json:"line,omitempty"`
}

// CompletionItem is one completion suggestion.
type CompletionItem struct {
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

// Location points at a definition or reference.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextEdit is an LSP-style replacement.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// WorkspaceEdit maps document URIs (absolute paths) to edits. Create lists
// files that should be created before the edits are applied.
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
	Create  []string              `json:"create,omitempty"`
}

// FoldingRange is a foldable line span (0-based, inclusive).
type FoldingRange struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

// CodeAction is a quick-fix.
type CodeAction struct {
	Title string         `json:"title"`
	Edit  *WorkspaceEdit `json:"edit,omitempty"`
}

// SymbolInformation describes a document or workspace symbol.
type SymbolInformation struct {
	Name          string   `json:"name"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

const (
	sevError   = 1
	sevWarning = 2
)

// atPos is the parsed file and one NodeAtOffset probe at a cursor.
type atPos struct {
	src, word       string
	off, start, end int
	res             jomini.Result
	kind            string
	assign          *jomini.Assignment
	saveName        string
	saveOK          bool
}

// resolveAt parses path and returns the identifier covering (line, col).
func resolveAt(s *session.Session, path string, line, col int) (atPos, bool) {
	src := s.FileText(path)
	if src == "" {
		return atPos{}, false
	}
	res := s.Parsed(path)
	off := res.Lines().OffsetAt(line, col)
	if inComment(res, off) {
		return atPos{}, false
	}
	word, start, end := res.TokenAt(off)
	at := atPos{src: src, off: off, res: res, word: word, start: start, end: end}
	rel := path
	if _, r, ok := s.Locate(path); ok {
		rel = r
	}
	at.kind = game.MatchExtract(s.GameID, rel).Kind
	if res.Root == nil || s.KindFor(path) == "loc" {
		return at, true
	}
	chain := jomini.NodeAtOffset(res.Root, off)
	if len(chain) > 0 {
		if a, ok := chain[0].(*jomini.Assignment); ok && !a.Key.Quoted {
			if d := s.Resolve(a.Key.Text); d != nil && d.Type != "" && d.Type != "loc_key" {
				at.kind = d.Type
			}
		}
		for i := len(chain) - 1; i >= 0; i-- {
			a, ok := chain[i].(*jomini.Assignment)
			if !ok {
				continue
			}
			if off >= a.Key.Range.Start && off < a.Key.Range.End {
				at.assign = a
			}
			break
		}
		at.saveName, at.saveOK = saveScopeIn(chain, off)
	}
	return at, true
}

func saveScopeIn(chain []jomini.Statement, off int) (name string, ok bool) {
	for _, n := range chain {
		a, isA := n.(*jomini.Assignment)
		if !isA || a.Key.Quoted {
			continue
		}
		if game.IsSaveScopeKey(a.Key.Text) {
			sc, isS := a.Value.(*jomini.Scalar)
			if isS && !sc.Quoted && off >= sc.Range.Start && off <= sc.Range.End {
				return sc.Text, true
			}
			return "", false
		}
		if !game.IsSaveScopeValueKey(a.Key.Text) {
			continue
		}
		b := jomini.BlockOf(a.Value)
		if b == nil {
			continue
		}
		for _, st := range b.Statements {
			ca, isC := st.(*jomini.Assignment)
			if !isC || ca.Key.Text != "name" {
				continue
			}
			sc, isS := ca.Value.(*jomini.Scalar)
			if isS && !sc.Quoted && off >= sc.Range.Start && off <= sc.Range.End {
				return sc.Text, true
			}
		}
	}
	return "", false
}

// byteRange converts a UTF-8 byte span using a cached LineIndex.
func byteRange(li *jomini.LineIndex, start, end int) Range {
	a := li.PositionAt(start)
	b := li.PositionAt(end)
	return Range{
		Start: Position{Line: a.Line, Character: a.Character},
		End:   Position{Line: b.Line, Character: b.Character},
	}
}

func isMetaFile(s *session.Session, path string) bool {
	if s.KindFor(path) != "meta" {
		return false
	}
	l := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.Contains(l, "/.metadata/") && strings.HasSuffix(l, "/metadata.json")
}

func locDefined(s *session.Session, key string) bool {
	if _, ok := s.DefaultLoc(key); ok {
		return true
	}
	_, _, _, ok := s.LocSite(key)
	return ok
}

func resolveNonLoc(s *session.Session, word string) *catalog.Def {
	return s.ResolveMatching(word, func(d catalog.Def) bool {
		return d.Type != "loc_key" && d.Type != "saved_scope"
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func modRootOf(s *session.Session, path string) string {
	origin, _, ok := s.Locate(path)
	if !ok {
		return ""
	}
	for _, m := range s.Mods() {
		if m.Origin == origin {
			return m.Root
		}
	}
	return ""
}

func lowerPrefix(s, prefix string) bool {
	return prefix == "" || strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

// inComment reports whether offset falls inside a `#` comment span.
func inComment(res jomini.Result, off int) bool {
	for _, c := range res.Comments {
		if off >= c.Range.Start && off < c.Range.End {
			return true
		}
	}
	return false
}
