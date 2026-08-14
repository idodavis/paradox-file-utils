// Build walks script roots and extracts definitions and reference edges.
package langmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	parser "paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/parser/walk"
)

// ProgressFunc reports build progress (done/total for the current phase).
type ProgressFunc func(phase string, done, total int, message string)

// Directories that never hold gameplay .txt scripts; skip to keep walks cancellable.
var skipDirNames = map[string]bool{
	"gfx": true, "sound": true, "fonts": true, "map_data": true,
	"pdx_launcher": true, "launcher": true, "tweakergui": true,
	"benchmarks": true, "crash_reporter": true,
}

// Build walks .txt under roots, parses top-level keys as definitions, and
// records simple identifier references to known keys as edges.
// installCachePath is optional; when set, DefKeys from the semantics cache
// seed type hints for matching keys.
func Build(
	ctx context.Context,
	workspaceID, installCachePath string,
	roots []string,
	onProgress ProgressFunc,
) (*Model, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	report := func(phase string, done, total int, message string) {
		if onProgress != nil {
			onProgress(phase, done, total, message)
		}
	}

	keyType := map[string]string{}
	if installCachePath != "" {
		report("cache", 0, 0, "Loading semantics cache…")
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		loadCacheKeyTypes(installCachePath, keyType)
	}

	report("walk", 0, 0, "Collecting script files…")
	var paths []string
	seenPath := map[string]bool{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if skipDirNames[strings.ToLower(d.Name())] {
					return filepath.SkipDir
				}
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
			if len(paths)%500 == 0 {
				report("walk", len(paths), 0, fmt.Sprintf("Found %d script files…", len(paths)))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	total := len(paths)
	report("parse", 0, total, fmt.Sprintf("Parsing %d files…", total))

	type fileHit struct {
		defs []Def
		refs []pendingRef
	}
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}
	jobs := make(chan string, workers*2)
	hits := make(chan fileHit, workers*2)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if ctx.Err() != nil {
					return
				}
				defs, refs, err := extractFile(path, keyType)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					case hits <- fileHit{}:
					}
					continue
				}
				select {
				case <-ctx.Done():
					return
				case hits <- fileHit{defs: defs, refs: refs}:
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, p := range paths {
			select {
			case <-ctx.Done():
				return
			case jobs <- p:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(hits)
	}()

	defs := make([]Def, 0, 256)
	defKeys := map[string]bool{}
	var allRefs []pendingRef
	done := 0
	for h := range hits {
		done++
		for _, d := range h.defs {
			defs = append(defs, d)
			defKeys[d.Key] = true
		}
		allRefs = append(allRefs, h.refs...)
		if done == total || done%50 == 0 {
			report("parse", done, total, fmt.Sprintf("Parsed %d / %d files…", done, total))
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	report("edges", 0, len(allRefs), "Resolving reference edges…")
	edges := make([]Edge, 0, 128)
	edgeSeen := map[string]bool{}
	for i, r := range allRefs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, id := range r.ids {
			if id == r.from || !defKeys[id] {
				continue
			}
			eid := r.from + "\x00" + id + "\x00" + "ref"
			if edgeSeen[eid] {
				continue
			}
			edgeSeen[eid] = true
			edges = append(edges, Edge{FromKey: r.from, ToKey: id, EdgeType: "ref"})
		}
		if i > 0 && i%2000 == 0 {
			report("edges", i, len(allRefs), "Resolving reference edges…")
		}
	}

	report("done", total, total, fmt.Sprintf("Built %d definitions", len(defs)))
	return &Model{
		WorkspaceID: workspaceID,
		BuiltAt:     time.Now().UTC().Format(time.RFC3339),
		Definitions: defs,
		Edges:       edges,
	}, nil
}

// pendingRef holds identifiers collected under a definition key before edge resolve.
type pendingRef struct {
	from string
	ids  []string
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

func extractFile(path string, keyType map[string]string) ([]Def, []pendingRef, error) {
	f, err := parser.ParseFile(path)
	if err != nil {
		return nil, nil, err
	}
	var out []Def
	var refs []pendingRef
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
		ids := walk.CollectIdentifiers(expr, expr.Key)
		if len(ids) > 0 {
			refs = append(refs, pendingRef{from: expr.Key, ids: ids})
		}
	}
	return out, refs, nil
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
