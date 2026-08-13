// Package scanner builds versioned semantic caches from game installs.
package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"paradox-modding-tools/services/internal/loc"
	parser "paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/semantics"
	"paradox-modding-tools/services/internal/semantics/schemaeval"
)

// Cache is the generated install semantic model (JSON in user-data).
type Cache struct {
	GameID      string                       `json:"gameId"`
	InstallPath string                       `json:"installPath"`
	ScannedAt   string                       `json:"scannedAt"`
	Schemas     map[string]schemaeval.Schema `json:"schemas"`
	DefKeys     map[string][]string          `json:"defKeys"`
	GuiTypes    []string                     `json:"guiTypes"`
	GuiProps    []string                     `json:"guiProps"`
	LocKeyCount map[string]int               `json:"locKeyCount"`
	Effects     []string                     `json:"effects,omitempty"`
	Triggers    []string                     `json:"triggers,omitempty"`
}

// ProgressFunc reports scan progress (done/total for current phase).
type ProgressFunc func(phase string, done, total int, message string)

// ScanInstall walks an install using bootstrap roots and returns a Cache.
func ScanInstall(gameID, installPath string) (*Cache, error) {
	return ScanInstallCtx(context.Background(), gameID, installPath, nil)
}

// ScanInstallCtx scans with cancellation and optional progress reporting.
func ScanInstallCtx(ctx context.Context, gameID, installPath string, onProgress ProgressFunc) (*Cache, error) {
	boot := semantics.BootstrapFor(gameID)
	if boot == nil {
		return nil, fmt.Errorf("unknown game %s", gameID)
	}
	scriptRoot := filepath.Join(installPath, filepath.FromSlash(boot.ScriptRoot))

	schemas := map[string]schemaeval.Schema{}
	for k, v := range boot.Schemas {
		cp := v
		cp.Attributes = append([]string{}, v.Attributes...)
		schemas[k] = cp
	}

	attrHist := map[string]map[string]int{}
	defKeys := map[string][]string{}
	pack := schemaeval.MustLoad(mustMarshalSchemas(schemas), "scan")

	var paths []string
	_ = filepath.WalkDir(scriptRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".txt" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	type fileHit struct {
		typeName string
		keys     []string
		attrs    map[string]int
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
				f, err := parser.ParseFile(path)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					case hits <- fileHit{}:
					}
					continue
				}
				types := pack.ApplicableTypesForPath(path)
				if len(types) == 0 {
					select {
					case <-ctx.Done():
						return
					case hits <- fileHit{}:
					}
					continue
				}
				typeName := types[0]
				h := fileHit{typeName: typeName, attrs: map[string]int{}}
				for _, entry := range f.Entries {
					if entry.Expression == nil || entry.Expression.Key == "" {
						continue
					}
					h.keys = append(h.keys, entry.Expression.Key)
					if entry.Expression.Object == nil {
						continue
					}
					for _, e := range entry.Expression.Object.Entries {
						if e.Expression != nil && e.Expression.Key != "" {
							h.attrs[e.Expression.Key]++
						}
					}
				}
				select {
				case <-ctx.Done():
					return
				case hits <- h:
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

	total := len(paths)
	done := 0
	report := func(phase string, d, t int, msg string) {
		if onProgress != nil {
			onProgress(phase, d, t, msg)
		}
	}
	report("scripts", 0, total, "Scanning scripts…")

	for h := range hits {
		done++
		if h.typeName != "" {
			if attrHist[h.typeName] == nil {
				attrHist[h.typeName] = map[string]int{}
			}
			for _, k := range h.keys {
				defKeys[h.typeName] = appendUnique(defKeys[h.typeName], k)
			}
			for k, n := range h.attrs {
				attrHist[h.typeName][k] += n
			}
		}
		if done == total || done%50 == 0 {
			report("scripts", done, total, "Scanning scripts…")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	for typeName, hist := range attrHist {
		s := schemas[typeName]
		attrs := topAttrs(hist, 40)
		s.Attributes = mergeUnique(s.Attributes, attrs)
		schemas[typeName] = s
	}

	report("gui", 0, 1, "Scanning GUI…")
	guiTypes, guiProps := scanGUICtx(ctx, installPath, boot.GuiRoots, func(done, total int) {
		report("gui", done, total, "Scanning GUI…")
	})
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	report("loc", 0, 1, "Collecting localization…")
	locRoots := make([]string, 0, len(boot.LocRoots))
	for _, lr := range boot.LocRoots {
		locRoots = append(locRoots, filepath.Join(installPath, filepath.FromSlash(lr)))
	}
	locKeys, err := loc.CollectKeysCtx(ctx, locRoots, func(done, total int, path string) {
		report("loc", done, total, "Collecting localization… "+filepath.Base(path))
	})
	if err != nil {
		return nil, err
	}
	locCount := map[string]int{}
	for lang, keys := range locKeys {
		locCount[lang] = len(keys)
	}

	effects, triggers := harvestScriptDocs(gameID, boot)
	report("done", 1, 1, "Scan complete")

	return &Cache{
		GameID:      gameID,
		InstallPath: installPath,
		ScannedAt:   time.Now().UTC().Format(time.RFC3339),
		Schemas:     schemas,
		DefKeys:     defKeys,
		GuiTypes:    guiTypes,
		GuiProps:    guiProps,
		LocKeyCount: locCount,
		Effects:     effects,
		Triggers:    triggers,
	}, nil
}

func mustMarshalSchemas(schemas map[string]schemaeval.Schema) []byte {
	b, err := json.Marshal(struct {
		Schemas map[string]schemaeval.Schema `json:"schemas"`
	}{Schemas: schemas})
	if err != nil {
		panic(err)
	}
	return b
}

func topAttrs(hist map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	var list []kv
	for k, v := range hist {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v > list[j].v })
	if len(list) > n {
		list = list[:n]
	}
	out := make([]string, len(list))
	for i, x := range list {
		out[i] = x.k
	}
	return out
}

func mergeUnique(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range a {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	for _, x := range b {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func scanGUI(installPath string, guiRoots []string) (types, props []string) {
	return scanGUICtx(context.Background(), installPath, guiRoots, nil)
}

// scanGUICtx walks GUI roots concurrently with optional progress.
func scanGUICtx(
	ctx context.Context, installPath string, guiRoots []string,
	onProgress func(done, total int),
) (types, props []string) {
	var paths []string
	for _, gr := range guiRoots {
		root := filepath.Join(installPath, filepath.FromSlash(gr))
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".gui" {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
	}
	total := len(paths)
	if total == 0 {
		return nil, nil
	}

	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}
	jobs := make(chan string, workers*2)
	type hit struct {
		types []string
		props []string
	}
	hits := make(chan hit, workers*2)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if ctx.Err() != nil {
					return
				}
				f, err := parser.ParseFile(path)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					case hits <- hit{}:
					}
					continue
				}
				typeSet := map[string]bool{}
				propSet := map[string]bool{}
				collectGUI(f, typeSet, propSet)
				h := hit{}
				for t := range typeSet {
					h.types = append(h.types, t)
				}
				for p := range propSet {
					h.props = append(h.props, p)
				}
				select {
				case <-ctx.Done():
					return
				case hits <- h:
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

	typeSet := map[string]bool{}
	propSet := map[string]bool{}
	done := 0
	for h := range hits {
		done++
		for _, t := range h.types {
			typeSet[t] = true
		}
		for _, p := range h.props {
			propSet[p] = true
		}
		if onProgress != nil && (done == total || done%20 == 0) {
			onProgress(done, total)
		}
	}
	for t := range typeSet {
		types = append(types, t)
	}
	for p := range propSet {
		props = append(props, p)
	}
	sort.Strings(types)
	sort.Strings(props)
	if len(types) > 200 {
		types = types[:200]
	}
	if len(props) > 200 {
		props = props[:200]
	}
	return types, props
}

func collectGUI(f *parser.ParadoxFile, typeSet, propSet map[string]bool) {
	var walkObj func(obj *parser.Object)
	walkObj = func(obj *parser.Object) {
		if obj == nil {
			return
		}
		for _, e := range obj.Entries {
			if e.Expression == nil {
				continue
			}
			k := e.Expression.Key
			if k != "" {
				propSet[k] = true
			}
			if e.Expression.Object != nil {
				walkObj(e.Expression.Object)
			}
		}
	}
	for _, e := range f.Entries {
		if e.Expression == nil {
			continue
		}
		if e.Expression.Key == "types" || e.Expression.Key == "template" {
			if e.Expression.Object != nil {
				for _, te := range e.Expression.Object.Entries {
					if te.Expression != nil && te.Expression.Key != "" {
						typeSet[te.Expression.Key] = true
					}
				}
			}
		}
		if e.Expression.Object != nil {
			walkObj(e.Expression.Object)
		}
	}
}

func harvestScriptDocs(gameID string, boot *semantics.Bootstrap) (effects, triggers []string) {
	home, err := os.UserHomeDir()
	if err != nil || boot.DocumentsFolder == "" {
		return nil, nil
	}
	docs := filepath.Join(home, "Documents", "Paradox Interactive", boot.DocumentsFolder, boot.ScriptDocsRel)
	effects = readNameList(filepath.Join(docs, "effects.log"))
	if len(effects) == 0 {
		effects = readNameList(filepath.Join(docs, "effects.txt"))
	}
	triggers = readNameList(filepath.Join(docs, "triggers.log"))
	if len(triggers) == 0 {
		triggers = readNameList(filepath.Join(docs, "triggers.txt"))
	}
	_ = gameID
	return effects, triggers
}

func readNameList(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// effects.log lines often start with the effect name
		name := strings.Fields(line)
		if len(name) == 0 {
			continue
		}
		n := name[0]
		if strings.ContainsAny(n, "()=<>") {
			continue
		}
		out = appendUnique(out, n)
		if len(out) >= 500 {
			break
		}
	}
	return out
}

// CachePath returns the user-data path for an install cache file.
func CachePath(installID string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "Paradox Modding Tools", "semantics")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, installID+".json"), nil
}

// SaveCache writes cache JSON to user-data.
func SaveCache(installID string, c *Cache) error {
	path, err := CachePath(installID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadCache reads a previously saved cache.
func LoadCache(installID string) (*Cache, error) {
	path, err := CachePath(installID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// ApplyCache merges cache schemas into the in-memory pack for the game.
func ApplyCache(gameID string, c *Cache) error {
	if c == nil {
		return fmt.Errorf("nil cache")
	}
	raw, err := json.Marshal(struct {
		Schemas map[string]schemaeval.Schema `json:"schemas"`
	}{Schemas: c.Schemas})
	if err != nil {
		return err
	}
	p, err := schemaeval.Load(raw)
	if err != nil {
		return err
	}
	semantics.SetPackForGame(gameID, p)
	return nil
}
