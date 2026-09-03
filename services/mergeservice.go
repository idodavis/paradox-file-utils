package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"paradox-modding-tools/services/internal/game"
	jomini "paradox-modding-tools/services/internal/parser/jomini"
)

const (
	additionalEntriesHdr = "\n############# Additional Entries From B (PDX-Merge-Tools) #############\n"
)

var precedenceRe = regexp.MustCompile(`(?i)(?:PREFER|USE|MERGE)\s*:\s*([AB])|(?i)(?:PROTECT|KEEP)`)

// MergeService merges matching script files from two path sets into an output directory using core merge logic.
type MergeService struct {
	FileService *FileService
}

// MergerOptions configures how files are merged.
type MergerOptions struct {
	AddAdditionalEntries bool `json:"addAdditionalEntries"`
}

// PreviewItem is a single file match for the merge preview.
type PreviewItem struct {
	RelPath        string `json:"relPath"`
	PathA          string `json:"pathA"`
	PathB          string `json:"pathB"`
	OutputPath     string `json:"outputPath"`
	WouldOverwrite bool   `json:"wouldOverwrite"`
}

// FileMergeResult is the result of merging one file.
type FileMergeResult struct {
	FilePath          string             `json:"filePath"`
	FileAPath         string             `json:"fileAPath"`
	FileBPath         string             `json:"fileBPath"`
	OutputPath        string             `json:"outputPath"`
	Changed           int                `json:"changed"`
	Added             int                `json:"added"`
	EntriesChanged    []string           `json:"entriesChanged,omitempty"`
	EntriesAdded      []string           `json:"entriesAdded,omitempty"`
	ResolvedConflicts []ResolvedConflict `json:"resolvedConflicts,omitempty"`
	Error             string             `json:"error,omitempty"`
}

// ResolvedConflict records a conflict that was auto-resolved (for report/audit).
type ResolvedConflict struct {
	Key      string `json:"key"`
	UsedSide string `json:"usedSide"` // "A" or "B"
	Reason   string `json:"reason"`   // "directive", "keyList", "default"
}

// mergeConflictChunk is a unit of content used while iterating merge conflicts.
type mergeConflictChunk struct {
	Type       string        `json:"type"` // "unchanged", "added", or "conflict"
	TextA      string        `json:"textA"`
	TextB      string        `json:"textB"`
	StartLineA int           `json:"startLineA"`
	StartLineB int           `json:"startLineB"`
	EndLineA   int           `json:"endLineA"`
	EndLineB   int           `json:"endLineB"`
	ObjA       *scriptObject `json:"-"`
	ObjB       *scriptObject `json:"-"`
}

// scriptObject represents a parsed top-level entry (assignment or object) with its comments.
type scriptObject struct {
	Key        string
	RawText    string   // Full text including comments
	ValueText  string   // Just the value part (for normalization)
	Comments   []string // Preceding comments
	PreferSide string   // "A" or "B" parsed from directives
	StartLine  int
	EndLine    int
}

// mergeResult holds the result of merging one file (internal use).
type mergeResult struct {
	Content           string
	EntriesAdded      []string
	EntriesChanged    []string
	ResolvedConflicts []ResolvedConflict
}

// MergePreview collects matching files from pathA/pathB and returns preview items with output paths.
func (m *MergeService) MergePreview(ctx context.Context, pathA, pathB, outputDir string) ([]PreviewItem, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	matches, err := m.FileService.collectAndMatchPaths(pathA, pathB, []string{".txt"})
	if err != nil {
		return nil, err
	}
	var items []PreviewItem
	for relPath, match := range matches {
		outPath := filepath.Join(outputDir, relPath)
		_, overwrite := os.Stat(outPath)
		items = append(items, PreviewItem{
			RelPath:        relPath,
			PathA:          match.PathA,
			PathB:          match.PathB,
			OutputPath:     outPath,
			WouldOverwrite: overwrite == nil,
		})
	}
	return items, nil
}

// Merge performs the merge for each task (PreviewItem) and writes results. Single entry point for merge operations.
func (m *MergeService) Merge(ctx context.Context, tasks []PreviewItem, opts MergerOptions) ([]FileMergeResult, error) {
	var results []FileMergeResult
	for _, task := range tasks {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		results = append(results, m.mergeAndWrite(task.PathA, task.PathB, task.OutputPath, task.RelPath, opts))
	}
	return results, nil
}

func parsePrecedenceFromComment(comment string) string {
	comment = strings.TrimSpace(strings.TrimPrefix(comment, "#"))
	if m := precedenceRe.FindStringSubmatch(comment); len(m) > 0 {
		if len(m) > 1 && m[1] != "" {
			return strings.ToUpper(m[1])
		}
		return "A" // PROTECT/KEEP -> keep from A
	}
	return ""
}

// normalizeMergeKey strips quotes and EU5 INJECT:/REPLACE: prefixes so keys match.
// Merge is game-agnostic, so it passes gameID "" (KeyIdentity then strips any
// known EU5 mode prefix).
func normalizeMergeKey(key string) string {
	return game.KeyIdentity("", key)
}

// parseFileObjects uses the shared Paradox parser for top-level object keys.
// Semantic typing (install cache / session) can later refine matching when keys
// collide across types; for now matching is by normalized top-level key only.
func parseFileObjects(path string) ([]scriptObject, error) {
	t, err := jomini.ParseFile(path)
	if err != nil {
		return nil, err
	}
	src := t.Src
	var objects []scriptObject
	prev := 0
	for _, a := range jomini.TopAssignments(t) {
		raw := string(src[prev:a.EndByte])
		val := string(src[a.StartByte:a.EndByte])
		comments := commentLines(raw)
		prefer := ""
		for _, c := range comments {
			if p := parsePrecedenceFromComment(c); p != "" {
				prefer = p
				break
			}
		}
		objects = append(objects, scriptObject{
			Key:        normalizeMergeKey(a.Key),
			RawText:    raw,
			ValueText:  val,
			Comments:   comments,
			PreferSide: prefer,
			StartLine:  a.StartLine,
			EndLine:    a.EndLine,
		})
		prev = a.EndByte
	}
	if prev < len(src) {
		tail := string(src[prev:])
		if strings.TrimSpace(tail) != "" {
			objects = append(objects, scriptObject{
				RawText:   tail,
				StartLine: 1,
				EndLine:   1 + strings.Count(tail, "\n"),
			})
		} else if len(objects) > 0 {
			objects[len(objects)-1].RawText += tail
		}
	}
	return objects, nil
}

func commentLines(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "#") {
			out = append(out, trim)
		}
	}
	return out
}

func determinePrecedence(a, b scriptObject) (string, string) {
	for _, obj := range []scriptObject{a, b} {
		if obj.PreferSide != "" {
			return obj.PreferSide, "directive"
		}
	}
	return "A", "default"
}

// mergeFileItems builds conflict chunks by comparing parsed objects from fileA and fileB.
func (m *MergeService) mergeFileItems(fileAPath, fileBPath string, opts MergerOptions) ([]mergeConflictChunk, error) {
	objectsA, err := parseFileObjects(fileAPath)
	if err != nil {
		return nil, fmt.Errorf("parsing file A: %w", err)
	}
	objectsB, err := parseFileObjects(fileBPath)
	if err != nil {
		return nil, fmt.Errorf("parsing file B: %w", err)
	}
	mapB := make(map[string]scriptObject, len(objectsB))
	for _, e := range objectsB {
		if e.Key != "" {
			mapB[e.Key] = e
		}
	}
	keysInA := make(map[string]bool, len(objectsA))
	var items []mergeConflictChunk

	for _, entA := range objectsA {
		a := entA
		chunk := mergeConflictChunk{
			Type: "unchanged", TextA: a.RawText,
			StartLineA: a.StartLine, EndLineA: a.EndLine, ObjA: &a,
		}
		if a.Key != "" {
			keysInA[a.Key] = true
			if entB, ok := mapB[a.Key]; ok {
				b := entB
				chunk.StartLineB, chunk.EndLineB, chunk.ObjB = b.StartLine, b.EndLine, &b
				if !scriptValuesEqual(a.ValueText, b.ValueText) {
					chunk.Type, chunk.TextB = "conflict", b.RawText
				}
			}
		}
		items = append(items, chunk)
	}
	if opts.AddAdditionalEntries {
		for _, entB := range objectsB {
			if entB.Key != "" && !keysInA[entB.Key] {
				b := entB
				items = append(items, mergeConflictChunk{
					Type: "added", TextB: b.RawText,
					StartLineB: b.StartLine, EndLineB: b.EndLine, ObjB: &b,
				})
			}
		}
	}
	return items, nil
}

// performMerge resolves conflicts using precedence rules and produces final content.
func (m *MergeService) performMerge(fileAPath, fileBPath string, opts MergerOptions) (*mergeResult, error) {
	items, err := m.mergeFileItems(fileAPath, fileBPath, opts)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString(utf8BOM)
	r := &mergeResult{}
	addHeader := false

	for _, it := range items {
		switch it.Type {
		case "unchanged":
			out.WriteString(it.TextA)
			continue
		case "added":
			if !addHeader {
				out.WriteString(additionalEntriesHdr)
				addHeader = true
			}
			out.WriteString(it.TextB)
			r.EntriesAdded = append(r.EntriesAdded, it.ObjB.Key)
			continue
		}
		// conflict
		decision, reason := determinePrecedence(*it.ObjA, *it.ObjB)
		if decision == "B" {
			out.WriteString(it.TextB)
			r.EntriesChanged = append(r.EntriesChanged, it.ObjA.Key)
		} else {
			out.WriteString(it.TextA)
		}
		r.ResolvedConflicts = append(r.ResolvedConflicts, ResolvedConflict{
			Key: it.ObjA.Key, UsedSide: decision, Reason: reason,
		})
	}
	r.Content = out.String()
	return r, nil
}

// mergeAndWrite performs merge and writes the result to outputPath.
func (m *MergeService) mergeAndWrite(pathA, pathB, outputPath, filePath string, opts MergerOptions) FileMergeResult {
	mr, err := m.performMerge(pathA, pathB, opts)
	if err != nil {
		return FileMergeResult{FilePath: filePath, Error: err.Error()}
	}
	if err := m.FileService.writeWithBOM(outputPath, mr.Content); err != nil {
		return FileMergeResult{FilePath: filePath, Error: err.Error()}
	}
	return FileMergeResult{
		FilePath: filePath, FileAPath: pathA, FileBPath: pathB, OutputPath: outputPath,
		Changed: len(mr.EntriesChanged), Added: len(mr.EntriesAdded),
		EntriesChanged: mr.EntriesChanged, EntriesAdded: mr.EntriesAdded, ResolvedConflicts: mr.ResolvedConflicts,
	}
}

func canonicalScriptValue(s string) string {
	var parts []string
	for _, line := range strings.Split(s, "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		parts = append(parts, strings.Fields(line)...)
	}
	return strings.Join(parts, "")
}

func scriptValuesEqual(a, b string) bool {
	return a == b || canonicalScriptValue(a) == canonicalScriptValue(b)
}
