// types.go holds LSP DTOs and shared cursor/path helpers.
package lsp

import (
	"os"
	"regexp"
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

// HoverResult is hover card markdown. Origin/rel/path/line describe the
// winning def site. Vanilla* is the install site this winner overlays.
type HoverResult struct {
	Contents          string `json:"contents"`
	Origin            string `json:"origin,omitempty"`
	OriginName        string `json:"originName,omitempty"`
	Rel               string `json:"rel,omitempty"`
	Path              string `json:"path,omitempty"`
	Line              int    `json:"line,omitempty"`
	Col               int    `json:"col,omitempty"`
	VanillaOriginName string `json:"vanillaOriginName,omitempty"`
	VanillaRel        string `json:"vanillaRel,omitempty"`
	VanillaPath       string `json:"vanillaPath,omitempty"`
	VanillaLine       int    `json:"vanillaLine,omitempty"`
	VanillaCol        int    `json:"vanillaCol,omitempty"`
}

// CompletionItem is one completion suggestion.
type CompletionItem struct {
	Label         string `json:"label"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	Kind          string `json:"kind,omitempty"` // property | value
	Range         Range  `json:"range,omitempty"`
	InsertText    string `json:"insertText,omitempty"`
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
	inKey           bool
	slotKey         string
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
			if !isLocalDefFile(s, path, at.kind) {
				if d := s.Resolve(a.Key.Text); d != nil && d.Kind != "" && d.Kind != "loc_key" {
					at.kind = d.Kind
				}
			}
		}
		fillCompleteSlot(&at, chain, off)
		if key := pendingAssignKey(at.src, off); key != "" {
			at.slotKey = key
			at.inKey = false
		}
		at.saveName, at.saveOK = saveScopeIn(chain, off)
	}
	return at, true
}

// fillCompleteSlot sets inKey/slotKey for completion, and assign when the
// cursor is on an assignment key (hover). chain is outermost-first.
func fillCompleteSlot(at *atPos, chain []jomini.Statement, off int) {
	var assigns []*jomini.Assignment
	for _, st := range chain {
		if a, ok := st.(*jomini.Assignment); ok && !a.Key.Quoted {
			assigns = append(assigns, a)
		}
	}
	if len(assigns) == 0 {
		return
	}
	inner := assigns[len(assigns)-1]
	if off >= inner.Key.Range.Start && off < inner.Key.Range.End {
		at.assign = inner
		at.inKey = true
		if len(assigns) >= 2 {
			at.slotKey = assigns[len(assigns)-2].Key.Text
		}
		return
	}
	innerSt := chain[len(chain)-1]
	if vs, ok := innerSt.(*jomini.ValueStmt); ok {
		at.slotKey = inner.Key.Text
		if sc, ok := vs.Value.(*jomini.Scalar); ok && !sc.Quoted {
			keyLine := at.res.Lines().PositionAt(inner.Key.Range.Start).Line
			valLine := at.res.Lines().PositionAt(sc.Range.Start).Line
			if valLine != keyLine {
				at.inKey = true
			}
		}
		return
	}
	if sc, ok := inner.Value.(*jomini.Scalar); ok &&
		off >= sc.Range.Start && off <= sc.Range.End {
		at.slotKey = inner.Key.Text
		return
	}
	if b := jomini.BlockOf(inner.Value); b != nil &&
		off >= b.Range.Start && off <= b.Range.End {
		at.inKey = true
		at.slotKey = inner.Key.Text
		return
	}
	if off >= inner.Key.Range.End {
		at.slotKey = inner.Key.Text
		end := min(off, len(at.src))
		from := inner.Key.Range.End
		if from < end && !strings.Contains(at.src[from:end], "=") {
			at.inKey = true
		}
	}
}

var pendingAssignRe = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*=\s*$`)

func pendingAssignKey(src string, off int) string {
	if off < 0 {
		return ""
	}
	if off > len(src) {
		off = len(src)
	}
	lineStart := strings.LastIndex(src[:off], "\n") + 1
	m := pendingAssignRe.FindStringSubmatch(src[lineStart:off])
	if m == nil {
		return ""
	}
	return m[1]
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

// isLocalDefFile reports mod descriptor / metadata files whose keys are
// defined locally in each mod, not resolved across the workspace.
func isLocalDefFile(s *session.Session, path string, extractKind string) bool {
	if extractKind == "mod_descriptor" {
		return true
	}
	return isMetaFile(s, path)
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
		return d.Kind != "loc_key" && d.Kind != "saved_scope"
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
