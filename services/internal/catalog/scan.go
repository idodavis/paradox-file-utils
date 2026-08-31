// scan.go harvests a game install into VanillaCache: script defs, structure keys,
// vocabulary, loc, shipped-doc prose, GUI types/props, metadata, optional script_docs.

package catalog

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"
)

const locValueLimit = 200
const scanConcurrency = 8

var (
	docKeyRe = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$`)
	tokenRe  = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)`)
)

// Scan harvests installPath into a VanillaCache. Empty docsPath skips script_docs.
func Scan(
	ctx context.Context,
	installID, gameID, installPath, version, docsPath, locLang string,
	onProgress func(pct int, msg string),
) (*VanillaCache, *VanillaLoc, error) {
	info := game.Get(gameID)
	if info == nil {
		return nil, nil, os.ErrInvalid
	}
	if locLang == "" {
		locLang = "english"
	}
	if version == "" {
		version = "latest"
	}
	progress(onProgress, 5, "listing files")
	inv := gather(scriptRoots(info, installPath))

	progress(onProgress, 15, "parsing script")
	files := append(append([]fileRef{}, inv.script...), inv.gui...)
	acc, err := collectExtracts(ctx, gameID, files, true, nil)
	if err != nil {
		return nil, nil, err
	}

	progress(onProgress, 70, "reading localization")
	vloc, err := harvestLoc(ctx, inv.loc, locLang)
	if err != nil {
		return nil, nil, err
	}

	progress(onProgress, 80, "reading docs")
	fieldDocs, fieldByKind, docStructs := harvestDocs(gameID, inv.docs)
	for kind, keys := range docStructs {
		set := acc.structures[kind]
		if set == nil {
			set = map[string]bool{}
			acc.structures[kind] = set
		}
		for k := range keys {
			set[k] = true
		}
	}

	c := &VanillaCache{
		FormatVersion:   CacheFormatVersion,
		InstallID:       installID,
		GameID:          gameID,
		InstallPath:     installPath,
		GameVersion:     version,
		ScannedAt:       time.Now().UTC().Format(time.RFC3339),
		Defs:            acc.defs,
		Edges:           acc.edges,
		FieldDocs:       fieldDocs,
		FieldDocsByKind: fieldByKind,
		Structures: lo.MapValues(acc.structures, func(set map[string]bool, _ string) []string {
			return sortedKeys(set)
		}),
		Vocabulary: sortedKeys(acc.vocab),
		GUITypes:   sortedKeys(acc.guiTypes),
		GUIProps:   sortedKeys(acc.guiProps),
		MetaKeys:   readMetaKeys(installPath, inv.meta),
	}

	progress(onProgress, 90, "reading script_docs")
	enrichScriptDocs(docsPath, info.ScriptDocsFormat, c)
	PrepareCache(c)

	progress(onProgress, 100, "done")
	return c, vloc, nil
}

func scriptRoots(info *game.GameInfo, installPath string) []string {
	if info.ScriptRoot != "" {
		return []string{filepath.Join(installPath, info.ScriptRoot)}
	}
	roots := make([]string, 0, len(info.StageRoots))
	for _, s := range info.StageRoots {
		roots = append(roots, filepath.Join(installPath, s))
	}
	return roots
}

// fileRef is a file plus its path relative to its script root (for MatchExtract).
type fileRef struct{ abs, rel, origin string }

// inventory buckets the walked files by role.
type inventory struct {
	script, gui, docs []fileRef
	loc, meta         []string
}

// ClassifyRel buckets a file by name/rel for the install walk and mod indexer.
func ClassifyRel(rel, name string) string {
	lower := strings.ToLower(name)
	slash := strings.ReplaceAll(rel, "\\", "/")
	switch {
	case strings.HasPrefix(name, "_") && strings.HasSuffix(lower, ".info"):
		return "docs"
	case strings.HasSuffix(lower, ".md"):
		return "docs"
	case strings.HasSuffix(lower, ".gui"):
		return "gui"
	case strings.HasSuffix(lower, ".mod"):
		return "mod"
	case strings.HasSuffix(lower, ".yml"), strings.HasSuffix(lower, ".yaml"):
		if strings.Contains(slash, "localization") {
			return "loc"
		}
	case lower == "metadata.json":
		return "meta"
	case strings.HasSuffix(lower, ".txt") && !strings.HasPrefix(name, "_"):
		return "script"
	}
	return ""
}

// walkClassified walks root and hands each ClassifyRel-recognized file to fn.
func walkClassified(root string, fn func(kind string, f fileRef)) {
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if kind := ClassifyRel(rel, d.Name()); kind != "" {
			fn(kind, fileRef{abs: filepath.Clean(p), rel: rel})
		}
		return nil
	})
}

func gather(roots []string) inventory {
	var inv inventory
	for _, root := range roots {
		walkClassified(root, func(kind string, f fileRef) {
			switch kind {
			case "docs":
				inv.docs = append(inv.docs, f)
			case "gui":
				inv.gui = append(inv.gui, f)
			case "loc":
				inv.loc = append(inv.loc, f.abs)
			case "meta":
				inv.meta = append(inv.meta, f.abs)
			case "script":
				inv.script = append(inv.script, f)
			}
		})
	}
	return inv
}

// accum collects FileExtract results from all workers behind one mutex. It is
// the single merge sink for both Scan and BuildIndex.
type accum struct {
	mu         sync.Mutex
	defs       []Def
	refs       []Ref
	edges      []Edge
	cands      []CallCandidate
	loc        map[string]map[string]LocEntry
	structures map[string]map[string]bool
	vocab      map[string]bool
	guiTypes   map[string]bool
	guiProps   map[string]bool
}

func newAccum() *accum {
	return &accum{
		loc:        map[string]map[string]LocEntry{},
		structures: map[string]map[string]bool{},
		vocab:      map[string]bool{},
		guiTypes:   map[string]bool{},
		guiProps:   map[string]bool{},
	}
}

func (a *accum) merge(ex FileExtract) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.defs = append(a.defs, ex.Defs...)
	a.refs = append(a.refs, ex.Refs...)
	a.edges = append(a.edges, ex.Edges...)
	a.loc = MergeLoc(a.loc, ex.Loc)
	if ex.StructKind != "" {
		set := a.structures[ex.StructKind]
		if set == nil {
			set = map[string]bool{}
			a.structures[ex.StructKind] = set
		}
		maps.Copy(set, ex.StructKeys)
	}
	maps.Copy(a.vocab, ex.Vocab)
	maps.Copy(a.guiTypes, ex.GUITypes)
	maps.Copy(a.guiProps, ex.GUIProps)
	a.cands = append(a.cands, ex.Cands...)
}

func walkFiles(ctx context.Context, files []fileRef, fn func(fileRef) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(scanConcurrency)
	for _, f := range files {
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return fn(f)
		})
	}
	return g.Wait()
}

func collectExtracts(
	ctx context.Context, gameID string, files []fileRef, bodies bool, cache *VanillaCache,
) (*accum, error) {
	acc := newAccum()
	err := walkFiles(ctx, files, func(f fileRef) error {
		acc.merge(ingestFile(gameID, f, bodies))
		return nil
	})
	acc.edges = append(acc.edges, ApplyCallEdges(acc.cands, EffectSet(acc.defs, cache))...)
	return acc, err
}

func harvestLoc(ctx context.Context, files []string, locLang string) (*VanillaLoc, error) {
	out := &VanillaLoc{FormatVersion: LocFormatVersion, Sites: map[string]LocEntry{}}
	var refs []fileRef
	for _, f := range files {
		if loc.LanguageFromRel(f) == locLang {
			refs = append(refs, fileRef{abs: f})
		}
	}
	if len(refs) == 0 {
		return out, nil
	}
	var mu sync.Mutex
	err := walkFiles(ctx, refs, func(f fileRef) error {
		raw, err := os.ReadFile(f.abs)
		if err != nil {
			return nil
		}
		text, _ := jomini.Decode(raw)
		_, locd := ExtractLoc(f.abs, text, "")
		mu.Lock()
		for k, v := range locd.Vals {
			out.Sites[k] = LocEntry{File: v.File, Line: v.Line, Value: v.Value}
		}
		mu.Unlock()
		return nil
	})
	return out, err
}

// HarvestLoc gathers install loc files for locLang without opening other languages.
func HarvestLoc(ctx context.Context, gameID, installPath, locLang string) (*VanillaLoc, error) {
	info := game.Get(gameID)
	if info == nil {
		return nil, os.ErrInvalid
	}
	if locLang == "" {
		locLang = "english"
	}
	inv := gather(scriptRoots(info, installPath))
	return harvestLoc(ctx, inv.loc, locLang)
}

// harvestDocs reads shipped `_*.info`/`*.md` docs into field prose and struct keys.
func harvestDocs(gameID string, docs []fileRef) (
	fieldDocs map[string]string,
	byKind map[string]map[string]string,
	structs map[string]map[string]bool,
) {
	fieldDocs, byKind, structs = map[string]string{}, map[string]map[string]string{}, map[string]map[string]bool{}
	for _, f := range docs {
		raw, err := os.ReadFile(f.abs)
		if err != nil {
			continue
		}
		kind := game.MatchExtract(gameID, f.rel).Kind
		for key, doc := range harvestDocFile(string(raw)) {
			if doc != "" && fieldDocs[key] == "" {
				fieldDocs[key] = doc
			}
			if kind == "" {
				continue
			}
			if structs[kind] == nil {
				structs[kind] = map[string]bool{}
			}
			structs[kind][key] = true
			if doc == "" {
				continue
			}
			if byKind[kind] == nil {
				byKind[kind] = map[string]string{}
			}
			if byKind[kind][key] == "" {
				byKind[kind][key] = doc
			}
		}
	}
	return fieldDocs, byKind, structs
}

// harvestDocFile extracts `key = value  # doc` candidates (preceding `#` as prose).
func harvestDocFile(text string) map[string]string {
	out := map[string]string{}
	var pending []string
	for _, raw := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if strings.HasPrefix(trimmed, "#") {
			pending = append(pending, strings.TrimSpace(strings.TrimLeft(trimmed, "# ")))
			continue
		}
		if trimmed == "" {
			pending = pending[:0]
			continue
		}
		if m := docKeyRe.FindStringSubmatch(strings.TrimSuffix(raw, "\r")); m != nil {
			key := strings.ToLower(m[2])
			if nameOKRe.MatchString(key) && !stoplist[key] {
				rhs := m[3]
				inlineDoc := ""
				if h := strings.IndexByte(rhs, '#'); h >= 0 {
					inlineDoc = strings.TrimSpace(strings.TrimLeft(rhs[h+1:], "# "))
				}
				doc := prose(append(pending, inlineDoc))
				if _, seen := out[key]; !seen || (out[key] == "" && doc != "") {
					out[key] = doc
				}
			}
		}
		pending = pending[:0]
	}
	return out
}

// readMetaKeys collects top-level keys from .metadata/metadata.json files.
func readMetaKeys(installPath string, metas []string) []string {
	set := map[string]bool{}
	paths := append([]string{filepath.Join(installPath, ".metadata", "metadata.json")}, metas...)
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var obj map[string]any
		if sonic.Unmarshal(raw, &obj) != nil {
			continue
		}
		for k := range obj {
			set[k] = true
		}
	}
	return sortedKeys(set)
}

func sortedKeys(set map[string]bool) []string {
	out := lo.Keys(set)
	slices.Sort(out)
	return out
}

func progress(fn func(pct int, msg string), pct int, msg string) {
	if fn != nil {
		fn(pct, msg)
	}
}
