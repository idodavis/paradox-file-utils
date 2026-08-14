// Package lsp provides shared Paradox language-feature handlers for Wails/LSP clients.
package lsp

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"paradox-modding-tools/services/internal/langmodel"
	parser "paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/parser/walk"
)

// Position is a 0-based line/character offset (LSP style).
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is an LSP-style text range.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location points at a definition or reference.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic is a parse/semantic issue for a file.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"` // 1=error 2=warning
	Message  string `json:"message"`
	Source   string `json:"source,omitempty"`
}

// HoverResult is markdown/plain hover text.
type HoverResult struct {
	Contents string `json:"contents"`
	Range    *Range `json:"range,omitempty"`
}

// CompletionItem is a single completion suggestion.
type CompletionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// SymbolInformation describes a document or workspace symbol.
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          int      `json:"kind"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

const (
	severityError = 1
	kindVariable  = 13
	kindObject    = 19
	completionRef = 6
)

var identRe = regexp.MustCompile(`[\p{L}\p{N}_.$:'&|%/\\-]+`)

// Diagnose parses path and returns parse errors as diagnostics.
func Diagnose(path string) []Diagnostic {
	_, err := parser.ParseFile(path)
	if err == nil {
		return nil
	}
	line := 0
	if pe, ok := err.(interface{ Line() int }); ok {
		line = pe.Line()
	}
	if line < 1 {
		line = 1
	}
	return []Diagnostic{{
		Range: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line - 1, Character: 1},
		},
		Severity: severityError,
		Message:  err.Error(),
		Source:   "pmt-parser",
	}}
}

// Hover returns info for the identifier at pos using the workspace model.
func Hover(model *langmodel.Model, path string, line, character int) *HoverResult {
	word := wordAt(path, line, character)
	if word == "" {
		return nil
	}
	defs := findDefs(model, word, "")
	if len(defs) == 0 {
		return &HoverResult{Contents: word}
	}
	d := defs[0]
	var b strings.Builder
	b.WriteString("**")
	b.WriteString(d.Key)
	b.WriteString("** (`")
	b.WriteString(d.Type)
	b.WriteString("`)\n\n")
	if d.Summary != "" {
		b.WriteString(d.Summary)
		b.WriteString("\n\n")
	}
	b.WriteString(filepath.Base(d.FilePath))
	b.WriteString(":")
	b.WriteString(strconv.Itoa(d.Line))
	return &HoverResult{Contents: b.String()}
}

// Complete suggests definition keys matching the prefix at pos.
func Complete(model *langmodel.Model, path string, line, character int) []CompletionItem {
	prefix := wordAt(path, line, character)
	if model == nil {
		return nil
	}
	var out []CompletionItem
	seen := map[string]bool{}
	for _, d := range model.Definitions {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(d.Key), strings.ToLower(prefix)) {
			continue
		}
		if seen[d.Key] {
			continue
		}
		seen[d.Key] = true
		out = append(out, CompletionItem{
			Label: d.Key, Kind: completionRef, Detail: d.Type,
		})
		if len(out) >= 50 {
			break
		}
	}
	return out
}

// Definition returns locations for the symbol at pos.
func Definition(model *langmodel.Model, path string, line, character int) []Location {
	word := wordAt(path, line, character)
	if word == "" {
		return nil
	}
	return defsToLocations(findDefs(model, word, ""))
}

// References returns locations that reference the symbol at pos.
func References(model *langmodel.Model, path string, line, character int) []Location {
	word := wordAt(path, line, character)
	if word == "" || model == nil {
		return nil
	}
	out := defsToLocations(findDefs(model, word, ""))
	for _, e := range model.Edges {
		if e.ToKey != word && e.FromKey != word {
			continue
		}
		for _, d := range findDefs(model, e.FromKey, "") {
			out = append(out, defLocation(d))
		}
	}
	return dedupeLocations(out)
}

// DocumentSymbols returns top-level keys in a file.
func DocumentSymbols(path string) []SymbolInformation {
	f, err := parser.ParseFile(path)
	if err != nil {
		return nil
	}
	var out []SymbolInformation
	for _, entry := range f.Entries {
		expr := entry.Expression
		if expr == nil || expr.Key == "" {
			continue
		}
		endLine := walk.LineEnd(expr.Pos.Line, expr.GetRawText())
		kind := kindVariable
		if expr.Object != nil {
			kind = kindObject
		}
		out = append(out, SymbolInformation{
			Name: expr.Key,
			Kind: kind,
			Location: Location{
				URI: pathToURI(path),
				Range: Range{
					Start: Position{Line: max(0, expr.Pos.Line-1), Character: max(0, expr.Pos.Column-1)},
					End:   Position{Line: max(0, endLine-1), Character: 0},
				},
			},
		})
	}
	return out
}

// WorkspaceSymbols searches the model for keys matching query.
func WorkspaceSymbols(model *langmodel.Model, query string) []SymbolInformation {
	if model == nil {
		return nil
	}
	q := strings.ToLower(query)
	var out []SymbolInformation
	for _, d := range model.Definitions {
		if q != "" && !strings.Contains(strings.ToLower(d.Key), q) &&
			!strings.Contains(strings.ToLower(d.Type), q) {
			continue
		}
		out = append(out, SymbolInformation{
			Name:          d.Key,
			Kind:          kindObject,
			Location:      defLocation(d),
			ContainerName: d.Type,
		})
		if len(out) >= 100 {
			break
		}
	}
	return out
}

func findDefs(model *langmodel.Model, key, typeFilter string) []langmodel.Def {
	if model == nil || key == "" {
		return nil
	}
	var out []langmodel.Def
	for _, d := range model.Definitions {
		if d.Key != key {
			continue
		}
		if typeFilter != "" && !strings.EqualFold(d.Type, typeFilter) {
			continue
		}
		out = append(out, d)
	}
	return out
}

func defsToLocations(defs []langmodel.Def) []Location {
	out := make([]Location, 0, len(defs))
	for _, d := range defs {
		out = append(out, defLocation(d))
	}
	return out
}

func defLocation(d langmodel.Def) Location {
	end := d.EndLine
	if end < d.Line {
		end = d.Line
	}
	return Location{
		URI: pathToURI(d.FilePath),
		Range: Range{
			Start: Position{Line: max(0, d.Line-1), Character: max(0, d.Col-1)},
			End:   Position{Line: max(0, end-1), Character: 0},
		},
	}
}

func dedupeLocations(in []Location) []Location {
	seen := map[string]bool{}
	var out []Location
	for _, loc := range in {
		k := loc.URI + ":" + strconv.Itoa(loc.Range.Start.Line)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, loc)
	}
	return out
}

func pathToURI(path string) string {
	return "file:///" + filepath.ToSlash(path)
}

func wordAt(path string, line, character int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	row := lines[line]
	if strings.HasSuffix(row, "\r") {
		row = strings.TrimSuffix(row, "\r")
	}
	if character < 0 {
		character = 0
	}
	if character > len(row) {
		character = len(row)
	}
	start, end := character, character
	for start > 0 && isIdentByte(row[start-1]) {
		start--
	}
	for end < len(row) && isIdentByte(row[end]) {
		end++
	}
	if start >= end {
		return ""
	}
	w := row[start:end]
	if !identRe.MatchString(w) {
		return ""
	}
	return w
}

func isIdentByte(b byte) bool {
	r := rune(b)
	return unicode.IsLetter(r) || unicode.IsDigit(r) ||
		b == '_' || b == '.' || b == '$' || b == ':' || b == '-'
}
