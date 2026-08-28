// types.go holds LSP DTOs (UTF-8 line/byte-column throughout) and the small
// shared helpers every feature uses to read a session file and the word at a
// position.

package lsp

import (
	"os"
	"path/filepath"
	"strings"

	"paradox-modding-tools/services/internal/loc"
	"paradox-modding-tools/services/internal/parser"
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
	Source   string `json:"source,omitempty"`
	Code     string `json:"code,omitempty"`
}

// HoverResult is hover card text.
type HoverResult struct {
	Contents string `json:"contents"`
	Range    *Range `json:"range,omitempty"`
}

// CompletionItem is one completion suggestion. Kind uses LSP CompletionItemKind.
type CompletionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind,omitempty"`
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

// SemanticSpan is one highlighted token in UTF-8 columns.
type SemanticSpan struct {
	Line     int    `json:"line"`
	StartCol int    `json:"startCol"`
	Length   int    `json:"length"`
	Type     string `json:"type"`
}

// SignatureHelp is a call-signature tooltip.
type SignatureHelp struct {
	Label         string `json:"label"`
	Documentation string `json:"documentation,omitempty"`
}

// InlayHint is a loc-value preview or scope label.
type InlayHint struct {
	Line      int    `json:"line"`
	Character int    `json:"character"`
	Label     string `json:"label"`
	Kind      int    `json:"kind,omitempty"`
}

// CodeAction is a quick-fix.
type CodeAction struct {
	Title string         `json:"title"`
	Kind  string         `json:"kind,omitempty"`
	Edit  *WorkspaceEdit `json:"edit,omitempty"`
}

// SymbolInformation describes a document or workspace symbol.
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          int      `json:"kind"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

const (
	sevError   = 1
	sevWarning = 2
	srcScript  = "pmt-script"
	srcLoc     = "pmt-loc"
	srcGUI     = "pmt-gui"
	srcMod     = "pmt-mod"
)

// byteRange converts a UTF-8 byte span in src into an LSP range.
func byteRange(src string, start, end int) Range {
	li := parser.NewLineIndex(src)
	a := li.PositionAt(start)
	b := li.PositionAt(end)
	return Range{
		Start: Position{Line: a.Line, Character: a.Character},
		End:   Position{Line: b.Line, Character: b.Character},
	}
}

// offsetOf maps a (line, UTF-8 column) to a byte offset in src.
func offsetOf(src string, line, col int) int {
	return parser.NewLineIndex(src).OffsetAt(line, col)
}

// wordAt returns the identifier covering offset, or ("", 0, 0).
func wordAt(src string, offset int) (word string, start, end int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	start, end = offset, offset
	for start > 0 && isIdentByte(src[start-1]) {
		start--
	}
	for end < len(src) && isIdentByte(src[end]) {
		end++
	}
	if start == end {
		return "", 0, 0
	}
	return src[start:end], start, end
}

func isIdentByte(c byte) bool {
	return c == '_' || c == '.' || c == '-' ||
		(c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

// fileText is the open buffer, or the decoded on-disk file.
func fileText(s *session.Session, path string) string {
	return s.FileText(path)
}

// parseOf returns the buffered parse result, or parses src on the fly.
func parseOf(s *session.Session, path, src string) parser.Result {
	if r, ok := s.Result(path); ok && r.Src == src {
		return r
	}
	return parser.Parse(src)
}

func isLocPath(path string) bool {
	l := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.HasSuffix(l, ".yml") || strings.HasSuffix(l, ".yaml")
}

func isGUIPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".gui")
}

func isModPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".mod")
}

func isMetaJSON(path string) bool {
	l := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.Contains(l, "/.metadata/") && strings.HasSuffix(l, "/metadata.json")
}

func sourceFor(path string) string {
	switch {
	case isLocPath(path):
		return srcLoc
	case isGUIPath(path):
		return srcGUI
	case isModPath(path):
		return srcMod
	default:
		return srcScript
	}
}

// locValue looks up an english loc string in the workspace, then vanilla.
func locValue(s *session.Session, key string) (string, bool) {
	idx := s.Index()
	if idx != nil {
		if v, ok := idx.Loc[key]; ok {
			return v, true
		}
	}
	if c := s.Cache(); c != nil && c.LocEnglish != nil {
		if v, ok := c.LocEnglish[key]; ok {
			return v, true
		}
	}
	return "", false
}

func locDefined(s *session.Session, key string) bool {
	_, ok := locValue(s, key)
	if ok {
		return true
	}
	idx := s.Index()
	if idx == nil {
		return false
	}
	for _, d := range idx.Defs {
		if d.Type == "loc_key" && d.Key == key {
			return true
		}
	}
	return false
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

func descriptorPath(gameID, root string) string {
	switch gameID {
	case "vic3", "eu5":
		return filepath.Join(root, ".metadata", "metadata.json")
	default:
		return filepath.Join(root, "descriptor.mod")
	}
}

func lowerPrefix(s, prefix string) bool {
	if prefix == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

func parseLoc(src string) loc.Result { return loc.Parse(src) }
