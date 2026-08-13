// Package services provides backend services for the Paradox Modding Tools application.
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/loc"
	parser "paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/repos"
	"paradox-modding-tools/services/internal/semantics"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// IndexerService indexes game and mod script objects for search and cascade analysis.
type IndexerService struct {
	DB   *sqlx.DB
	repo *repos.IndexRepository

	mu     sync.Mutex
	cancel context.CancelFunc
}

// CascadeNode represents a node in cascade simulation results.
type CascadeNode struct {
	Key      string   `json:"key"`
	Depth    int      `json:"depth"`
	EdgeType string   `json:"edgeType,omitempty"`
	Children []string `json:"children,omitempty"`
}

func (i *IndexerService) getRepo() *repos.IndexRepository {
	if i.repo == nil {
		i.repo = repos.NewIndexRepository(i.DB)
	}
	return i.repo
}

func (i *IndexerService) beginJob() context.Context {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.cancel != nil {
		i.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	i.cancel = cancel
	return ctx
}

func (i *IndexerService) clearJob() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.cancel = nil
}

// CancelIndex cancels an in-flight ReindexWorkspace job.
func (i *IndexerService) CancelIndex() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.cancel != nil {
		i.cancel()
		i.cancel = nil
	}
}

// ReindexWorkspace walks game+mod roots, parses scripts and loc, stores index_objects.
func (i *IndexerService) ReindexWorkspace(workspaceID string) (int, error) {
	if workspaceID == "" {
		return 0, fmt.Errorf("workspace id is required")
	}
	ctx := i.beginJob()
	defer i.clearJob()

	repo := i.getRepo()

	gameID, installID, err := repo.GetWorkspaceInfo(workspaceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("workspace not found")
		}
		return 0, fmt.Errorf("get workspace: %w", err)
	}

	emitProgress(eventIndexProgress, ProgressEvent{
		Job: "index", Phase: "clear", WorkspaceID: workspaceID, Message: "Clearing index…",
	})

	if err := repo.ClearObjects(workspaceID); err != nil {
		return 0, fmt.Errorf("clear index: %w", err)
	}
	if err := repo.ClearEdges(workspaceID); err != nil {
		return 0, fmt.Errorf("clear edges: %w", err)
	}
	if err := ctx.Err(); err != nil {
		emitCancelled(workspaceID)
		return 0, nil
	}

	var scriptRoots []string
	var locRoots []string

	if installID != "" {
		instPath, instGameID, err := repo.GetInstallPath(installID)
		if err == nil {
			info := game.Get(instGameID)
			if info != nil {
				scriptRoots = append(scriptRoots, filepath.Join(instPath, info.ScriptRoot))
				for _, lr := range info.LocRoots {
					locRoots = append(locRoots, filepath.Join(instPath, lr))
				}
			}
			if gameID == "" {
				gameID = instGameID
			}
		}
	}

	modPaths, err := repo.ListModPaths(workspaceID)
	if err == nil {
		scriptRoots = append(scriptRoots, modPaths...)
		for _, mp := range modPaths {
			locRoots = append(locRoots, filepath.Join(mp, "localization"))
		}
	}

	count := 0
	for _, root := range scriptRoots {
		n, err := i.indexScriptRoot(ctx, workspaceID, gameID, root)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				emitCancelled(workspaceID)
				return count, nil
			}
			return count, fmt.Errorf("index %s: %w", root, err)
		}
		count += n
	}
	emitProgress(eventIndexProgress, ProgressEvent{
		Job: "index", Phase: "loc", WorkspaceID: workspaceID, Message: "Indexing localization…",
	})
	n, err := i.indexLocRoots(ctx, workspaceID, locRoots)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			emitCancelled(workspaceID)
			return count, nil
		}
		return count, fmt.Errorf("index loc: %w", err)
	}
	count += n

	if err := ctx.Err(); err != nil {
		emitCancelled(workspaceID)
		return count, nil
	}
	emitProgress(eventIndexProgress, ProgressEvent{
		Job: "index", Phase: "edges", WorkspaceID: workspaceID,
		Message: "Building event edges…",
	})
	if _, err := i.buildEventEdgesCtx(ctx, workspaceID); err != nil {
		if errors.Is(err, context.Canceled) {
			emitCancelled(workspaceID)
			return count, nil
		}
		return count, fmt.Errorf("build edges: %w", err)
	}

	emitProgress(eventIndexProgress, ProgressEvent{
		Job: "index", Phase: "done", Done: count, Total: count, WorkspaceID: workspaceID,
		Message: fmt.Sprintf("Indexed %d objects", count),
	})
	return count, nil
}

// emitCancelled reports a soft cancel (not an error) to the frontend.
func emitCancelled(workspaceID string) {
	emitProgress(eventIndexProgress, ProgressEvent{
		Job: "index", Phase: "cancelled", WorkspaceID: workspaceID, Message: "Cancelled",
	})
}

func (i *IndexerService) indexScriptRoot(ctx context.Context, workspaceID, gameID, root string) (int, error) {
	var paths []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".txt" && ext != ".gui" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})

	total := len(paths)
	if total == 0 {
		return 0, nil
	}

	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}

	type parsed struct {
		objs []repos.IndexObject
	}
	jobs := make(chan string, workers*2)
	results := make(chan parsed, workers*2)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if ctx.Err() != nil {
					return
				}
				objs, err := i.parseFileObjects(gameID, path)
				if err != nil || len(objs) == 0 {
					select {
					case <-ctx.Done():
						return
					case results <- parsed{}:
					}
					continue
				}
				select {
				case <-ctx.Done():
					return
				case results <- parsed{objs: objs}:
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
		close(results)
	}()

	repo := i.getRepo()
	count := 0
	done := 0
	batch := make([]repos.IndexObject, 0, 64)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := repo.UpsertObjects(batch); err != nil {
			return err
		}
		count += len(batch)
		batch = batch[:0]
		return nil
	}
	for r := range results {
		done++
		if ctx.Err() != nil {
			_ = flush()
			return count, context.Canceled
		}
		for _, obj := range r.objs {
			obj.ID = uuid.New().String()
			obj.WorkspaceID = workspaceID
			batch = append(batch, obj)
			if len(batch) >= 64 {
				if err := flush(); err != nil {
					return count, fmt.Errorf("insert object: %w", err)
				}
			}
		}
		if done == total || done%25 == 0 {
			emitProgress(eventIndexProgress, ProgressEvent{
				Job: "index", Phase: "scripts", Done: done, Total: total,
				WorkspaceID: workspaceID,
				Message:     filepath.Base(root),
			})
		}
	}
	if err := flush(); err != nil {
		return count, fmt.Errorf("insert object: %w", err)
	}
	if ctx.Err() != nil {
		return count, context.Canceled
	}
	return count, nil
}

func (i *IndexerService) indexLocRoots(ctx context.Context, workspaceID string, roots []string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	repo := i.getRepo()
	byLang, err := loc.CollectKeysCtx(ctx, roots, func(done, total int, path string) {
		emitProgress(eventIndexProgress, ProgressEvent{
			Job: "index", Phase: "loc", Done: done, Total: total,
			WorkspaceID: workspaceID,
			Message:     "Indexing localization… " + filepath.Base(path),
		})
	})
	if err != nil {
		return 0, err
	}
	count := 0
	batch := make([]repos.IndexObject, 0, 128)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := repo.UpsertObjects(batch); err != nil {
			return err
		}
		count += len(batch)
		batch = batch[:0]
		return nil
	}
	for lang, keys := range byLang {
		for key, e := range keys {
			if ctx.Err() != nil {
				_ = flush()
				return count, context.Canceled
			}
			batch = append(batch, repos.IndexObject{
				ID:          uuid.New().String(),
				WorkspaceID: workspaceID,
				ObjType:     "loc_key",
				ObjKey:      lang + ":" + key,
				FilePath:    e.FilePath,
				Line:        e.Line,
				Summary:     e.Value,
			})
			if len(batch) >= 128 {
				if err := flush(); err != nil {
					return count, err
				}
			}
		}
	}
	if err := flush(); err != nil {
		return count, err
	}
	return count, nil
}

func (i *IndexerService) parseFileObjects(gameID, path string) ([]repos.IndexObject, error) {
	f, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}

	var objs []repos.IndexObject
	for _, entry := range f.Entries {
		if entry.Expression == nil || entry.Expression.Key == "" {
			continue
		}
		expr := entry.Expression
		objType := i.classifyType(gameID, path, expr)
		if objType == "" {
			continue
		}
		summary := ""
		if expr.Object != nil && len(expr.Object.Entries) > 0 {
			summary = i.extractSummary(expr.Object)
		}
		key := expr.Key
		if pack := semantics.ForGame(gameID); pack != nil {
			applicable := pack.ApplicableTypesForPath(path)
			attrs := attrSet(expr)
			if t, display, ok := pack.ClassifyKey(expr.Key, expr.Object != nil, attrs, applicable, false); ok {
				objType = t
				key = display
			}
		}
		objs = append(objs, repos.IndexObject{
			ObjType:  objType,
			ObjKey:   key,
			FilePath: path,
			Line:     expr.Pos.Line,
			Summary:  summary,
		})
	}
	return objs, nil
}

func attrSet(expr *parser.Expression) map[string]bool {
	out := map[string]bool{}
	if expr.Object == nil {
		return out
	}
	for _, e := range expr.Object.Entries {
		if e.Expression != nil && e.Expression.Key != "" {
			out[e.Expression.Key] = true
		}
	}
	return out
}

func (i *IndexerService) classifyType(gameID, path string, expr *parser.Expression) string {
	if strings.EqualFold(filepath.Ext(path), ".gui") {
		return "gui"
	}
	if pack := semantics.ForGame(gameID); pack != nil {
		applicable := pack.ApplicableTypesForPath(path)
		if len(applicable) > 0 {
			attrs := attrSet(expr)
			if t, _, ok := pack.ClassifyKey(expr.Key, expr.Object != nil, attrs, applicable, false); ok {
				return t
			}
			return applicable[0]
		}
		if t := pack.TypeForPath(path); t != "" {
			return t
		}
	}
	if expr.Object != nil {
		return "object"
	}
	return ""
}

func (i *IndexerService) extractSummary(obj *parser.Object) string {
	var parts []string
	for _, e := range obj.Entries {
		if e.Expression != nil && len(parts) < 3 {
			key := e.Expression.Key
			if key != "" && !strings.HasPrefix(key, "#") {
				parts = append(parts, key)
			}
		}
	}
	return strings.Join(parts, ", ")
}

// SearchIndex searches indexed objects by key pattern.
func (i *IndexerService) SearchIndex(workspaceID, query string) ([]repos.IndexObject, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	out, err := i.getRepo().Search(workspaceID, query)
	if err != nil {
		return nil, fmt.Errorf("search index: %w", err)
	}
	return out, nil
}

// CountIndexObjects returns how many objects are indexed for a workspace.
func (i *IndexerService) CountIndexObjects(workspaceID string) (int, error) {
	if workspaceID == "" {
		return 0, fmt.Errorf("workspace id is required")
	}
	return i.getRepo().CountObjects(workspaceID)
}

// ListIndexObjects returns all objects for a workspace, optionally filtered by type.
func (i *IndexerService) ListIndexObjects(workspaceID, typeFilter string) ([]repos.IndexObject, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	out, err := i.getRepo().ListObjects(workspaceID, typeFilter)
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}
	return out, nil
}

// BuildEventEdges creates edges for event triggers and immediates.
func (i *IndexerService) BuildEventEdges(workspaceID string) (int, error) {
	if workspaceID == "" {
		return 0, fmt.Errorf("workspace id is required")
	}
	ctx := i.beginJob()
	defer i.clearJob()
	n, err := i.buildEventEdgesCtx(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			emitCancelled(workspaceID)
			return n, nil
		}
		return n, err
	}
	return n, nil
}

func (i *IndexerService) buildEventEdgesCtx(ctx context.Context, workspaceID string) (int, error) {
	repo := i.getRepo()

	if err := repo.ClearEdges(workspaceID); err != nil {
		return 0, fmt.Errorf("clear edges: %w", err)
	}

	objs, err := repo.ListEventObjects(workspaceID)
	if err != nil {
		return 0, fmt.Errorf("get events: %w", err)
	}

	triggerRe := regexp.MustCompile(`trigger_event\s*=\s*\{\s*id\s*=\s*([a-zA-Z0-9_.]+)`)
	fireRe := regexp.MustCompile(`fire_(?:on_action|scripted_effect)\s*=\s*([a-zA-Z0-9_.]+)`)

	total := len(objs)
	if total == 0 {
		return 0, nil
	}

	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}

	type edgeBatch struct {
		edges []repos.IndexEdge
	}
	jobs := make(chan repos.IndexObject, workers*2)
	results := make(chan edgeBatch, workers*2)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for obj := range jobs {
				if ctx.Err() != nil {
					return
				}
				content, err := os.ReadFile(obj.FilePath)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					case results <- edgeBatch{}:
					}
					continue
				}
				contentStr := string(content)
				var edges []repos.IndexEdge
				for _, m := range triggerRe.FindAllStringSubmatch(contentStr, -1) {
					if len(m) > 1 {
						edges = append(edges, repos.IndexEdge{
							ID: uuid.New().String(), WorkspaceID: workspaceID,
							FromKey: obj.ObjKey, ToKey: m[1], EdgeType: "triggers",
						})
					}
				}
				for _, m := range fireRe.FindAllStringSubmatch(contentStr, -1) {
					if len(m) > 1 {
						edges = append(edges, repos.IndexEdge{
							ID: uuid.New().String(), WorkspaceID: workspaceID,
							FromKey: obj.ObjKey, ToKey: m[1], EdgeType: "fires",
						})
					}
				}
				select {
				case <-ctx.Done():
					return
				case results <- edgeBatch{edges: edges}:
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, obj := range objs {
			select {
			case <-ctx.Done():
				return
			case jobs <- obj:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	count := 0
	done := 0
	batch := make([]repos.IndexEdge, 0, 128)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := repo.InsertEdges(batch); err != nil {
			return err
		}
		count += len(batch)
		batch = batch[:0]
		return nil
	}

	for r := range results {
		if err := ctx.Err(); err != nil {
			_ = flush()
			return count, err
		}
		batch = append(batch, r.edges...)
		if len(batch) >= 128 {
			if err := flush(); err != nil {
				return count, err
			}
		}
		done++
		if done == total || done%50 == 0 {
			emitProgress(eventIndexProgress, ProgressEvent{
				Job: "index", Phase: "edges", Done: done, Total: total,
				WorkspaceID: workspaceID, Message: "Building edges…",
			})
		}
	}
	if err := flush(); err != nil {
		return count, err
	}
	return count, nil
}

// SimulateCascade walks edges BFS from startKey up to maxDepth.
func (i *IndexerService) SimulateCascade(workspaceID, startKey string, maxDepth int) ([]CascadeNode, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	if maxDepth <= 0 {
		maxDepth = 5
	}

	edges, err := i.getRepo().ListEdges(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get edges: %w", err)
	}

	graph := make(map[string][]struct {
		to       string
		edgeType string
	})
	for _, e := range edges {
		graph[e.FromKey] = append(graph[e.FromKey], struct {
			to       string
			edgeType string
		}{e.ToKey, e.EdgeType})
	}

	visited := make(map[string]bool)
	var result []CascadeNode

	queue := []struct {
		key   string
		depth int
	}{{startKey, 0}}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if visited[curr.key] || curr.depth > maxDepth {
			continue
		}
		visited[curr.key] = true

		node := CascadeNode{Key: curr.key, Depth: curr.depth}
		for _, edge := range graph[curr.key] {
			node.Children = append(node.Children, edge.to)
			if !visited[edge.to] {
				queue = append(queue, struct {
					key   string
					depth int
				}{edge.to, curr.depth + 1})
			}
		}
		if len(graph[curr.key]) > 0 {
			node.EdgeType = graph[curr.key][0].edgeType
		}
		result = append(result, node)
	}

	return result, nil
}
