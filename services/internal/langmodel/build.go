// Build walks script roots and extracts definitions and reference edges.
package langmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	parser "paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/parser/walk"
)

// Build walks .txt under roots, parses top-level keys as definitions, and
// records simple identifier references to known keys as edges.
// installCachePath is optional; when set, DefKeys from the semantics cache
// seed type hints for matching keys.
func Build(
	ctx context.Context, workspaceID, installCachePath string, roots []string,
) (*Model, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	keyType := map[string]string{}
	if installCachePath != "" {
		loadCacheKeyTypes(installCachePath, keyType)
	}

	var paths []string
	seenPath := map[string]bool{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.ToLower(filepath.Ext(path)) != ".txt" {
				return nil
			}
			if seenPath[path] {
				return nil
			}
			seenPath[path] = true
			paths = append(paths, path)
			return nil
		})
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	defs := make([]Def, 0, 256)
	defKeys := map[string]bool{}

	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fileDefs, err := extractDefs(path, keyType)
		if err != nil {
			continue // skip unparseable files
		}
		for _, d := range fileDefs {
			defs = append(defs, d)
			defKeys[d.Key] = true
		}
	}

	// Second pass: edges from identifiers that match known definition keys.
	edges := make([]Edge, 0, 128)
	edgeSeen := map[string]bool{}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fileEdges, err := extractEdges(path, defKeys)
		if err != nil {
			continue
		}
		for _, e := range fileEdges {
			id := e.FromKey + "\x00" + e.ToKey + "\x00" + e.EdgeType
			if edgeSeen[id] {
				continue
			}
			edgeSeen[id] = true
			edges = append(edges, e)
		}
	}

	return &Model{
		WorkspaceID: workspaceID,
		BuiltAt:     time.Now().UTC().Format(time.RFC3339),
		Definitions: defs,
		Edges:       edges,
	}, nil
}

type cacheShape struct {
	DefKeys map[string][]string `json:"defKeys"`
}

func loadCacheKeyTypes(path string, out map[string]string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var c cacheShape
	if json.Unmarshal(data, &c) != nil || c.DefKeys == nil {
		return
	}
	for typ, keys := range c.DefKeys {
		for _, k := range keys {
			if k != "" {
				out[k] = typ
			}
		}
	}
}

func extractDefs(path string, keyType map[string]string) ([]Def, error) {
	f, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}
	var out []Def
	for _, entry := range f.Entries {
		expr := entry.Expression
		if expr == nil || expr.Key == "" {
			continue
		}
		typ := keyType[expr.Key]
		if typ == "" {
			if expr.Object != nil {
				typ = "object"
			} else {
				typ = "value"
			}
		}
		summary := ""
		if expr.Object != nil {
			summary = summaryFromObject(expr.Object)
		}
		endLine := walk.LineEnd(expr.Pos.Line, expr.GetRawText())
		out = append(out, Def{
			Type:     typ,
			Key:      expr.Key,
			FilePath: path,
			Line:     expr.Pos.Line,
			Col:      expr.Pos.Column,
			EndLine:  endLine,
			Summary:  summary,
		})
	}
	return out, nil
}

func extractEdges(path string, known map[string]bool) ([]Edge, error) {
	f, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}
	var out []Edge
	for _, entry := range f.Entries {
		expr := entry.Expression
		if expr == nil || expr.Key == "" {
			continue
		}
		from := expr.Key
		for _, id := range walk.CollectIdentifiers(expr, from) {
			if id == from || !known[id] {
				continue
			}
			out = append(out, Edge{
				FromKey: from, ToKey: id, EdgeType: "ref",
			})
		}
	}
	return out, nil
}

func summaryFromObject(obj *parser.Object) string {
	var parts []string
	for _, e := range obj.Entries {
		if e.Expression == nil {
			continue
		}
		k := e.Expression.Key
		if k == "" || strings.HasPrefix(k, "#") {
			continue
		}
		parts = append(parts, k)
		if len(parts) >= 3 {
			break
		}
	}
	return strings.Join(parts, ", ")
}
