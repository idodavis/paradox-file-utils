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
	docKeyRe     = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$`)
	attrBulletRe = regexp.MustCompile(`^-\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?::|=)\s*(.*)$`)
	tokenRe      = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)`)
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
	vloc, locFileRefs, err := harvestLoc(ctx, inv.loc, locLang, installID, version)
	if err != nil {
		return nil, nil, err
	}

	progress(onProgress, 80, "reading docs")
	fieldDocs, fieldByKind, docStructs := harvestDocs(gameID, inv.docs)
	for kind, keys := range docStructs {
		set := acc.structures[kind]
		if set == nil {
			set = map[string]int{}
			acc.structures[kind] = set
		}
		for k := range keys {
			if set[k] == 0 {
				set[k] = 1
			}
		}
	}

	kinds := VoteFieldValueKinds(acc.fieldRHS, acc.defs)
	c := &VanillaCache{
		FormatVersion:    CacheFormatVersion,
		InstallID:        installID,
		GameID:           gameID,
		InstallPath:      installPath,
		GameVersion:      version,
		ScannedAt:        time.Now().UTC().Format(time.RFC3339),
		Defs:             acc.defs,
		Edges:            acc.edges,
		LocRefs:          append(locKindRefs(acc.refs), locKindRefs(locFileRefs)...),
		FieldValueKinds:  kinds,
		FieldEnumsByKind: VoteFieldEnums(acc.fieldRHSByKind, kinds),
		FieldDocs:        fieldDocs,
		FieldDocsByKind:  fieldByKind,
		Structures:       keysByCount(acc.structures),
		StructureBlocks:  blockKeys(acc.structures, acc.structBlocks),
		Vocabulary:       sortedKeys(acc.vocab),
		GUITypes:         sortedKeys(acc.guiTypes),
		GUIProps:         sortedKeys(acc.guiProps),
		MetaKeys:         readMetaKeys(installPath, inv.meta),
	}

	progress(onProgress, 90, "reading script_docs")
	enrichScriptDocs(docsPath, info.ScriptDocsFormat, c)
	enrichDataTypes(docsPath, c)
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
	case isShippedDocName(name):
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

// isShippedDocName reports CK3 `_*.info` / `*.md` and EU5 readme/info text files.
// Underscored data (`_default.txt`, `_hardcoded.txt`) stays unclassified.
func isShippedDocName(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".md") {
		return true
	}
	if strings.HasPrefix(name, "_") && strings.HasSuffix(lower, ".info") {
		return true
	}
	stem := strings.Trim(strings.TrimSuffix(lower, filepath.Ext(lower)), "_")
	return stem == "readme" || stem == "info"
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
	mu             sync.Mutex
	defs           []Def
	refs           []Ref
	edges          []Edge
	cands          []CallCandidate
	loc            map[string]map[string]LocEntry
	structures     map[string]map[string]int
	structBlocks   map[string]map[string]int
	vocab          map[string]bool
	guiTypes       map[string]bool
	guiProps       map[string]bool
	fieldRHS       map[string]map[string]bool
	fieldRHSByKind map[string]map[string]map[string]bool
}

func newAccum() *accum {
	return &accum{
		loc:            map[string]map[string]LocEntry{},
		structures:     map[string]map[string]int{},
		structBlocks:   map[string]map[string]int{},
		vocab:          map[string]bool{},
		guiTypes:       map[string]bool{},
		guiProps:       map[string]bool{},
		fieldRHS:       map[string]map[string]bool{},
		fieldRHSByKind: map[string]map[string]map[string]bool{},
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
		addCounts(&a.structures, ex.StructKind, ex.StructCounts)
		addCounts(&a.structBlocks, ex.StructKind, ex.StructBlocks)
	}
	maps.Copy(a.vocab, ex.Vocab)
	maps.Copy(a.guiTypes, ex.GUITypes)
	maps.Copy(a.guiProps, ex.GUIProps)
	for field, vals := range ex.FieldRHS {
		m := a.fieldRHS[field]
		if m == nil {
			m = map[string]bool{}
			a.fieldRHS[field] = m
		}
		maps.Copy(m, vals)
		if ex.StructKind == "" {
			continue
		}
		kindM := a.fieldRHSByKind[ex.StructKind]
		if kindM == nil {
			kindM = map[string]map[string]bool{}
			a.fieldRHSByKind[ex.StructKind] = kindM
		}
		km := kindM[field]
		if km == nil {
			km = map[string]bool{}
			kindM[field] = km
		}
		maps.Copy(km, vals)
	}
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

func harvestLoc(
	ctx context.Context, files []string, locLang, installID, version string,
) (*VanillaLoc, []Ref, error) {
	byLang := map[string]*VanillaLoc{}
	refs := make([]fileRef, 0, len(files))
	for _, f := range files {
		if loc.LanguageFromRel(f) == "" {
			continue
		}
		refs = append(refs, fileRef{abs: f})
	}
	var mu sync.Mutex
	var locRefs []Ref
	err := walkFiles(ctx, refs, func(f fileRef) error {
		lang := loc.LanguageFromRel(f.abs)
		if lang == "" {
			return nil
		}
		raw, err := os.ReadFile(f.abs)
		if err != nil {
			return nil
		}
		text, _ := jomini.Decode(raw)
		_, locd, fileRefs := ExtractLoc(f.abs, text, "")
		mu.Lock()
		out := byLang[lang]
		if out == nil {
			out = &VanillaLoc{FormatVersion: LocFormatVersion, Sites: map[string]LocEntry{}}
			byLang[lang] = out
		}
		for k, v := range locd.Vals {
			out.Sites[k] = LocEntry{File: v.File, Line: v.Line, Value: v.Value}
		}
		if lang == locLang {
			locRefs = append(locRefs, fileRefs...)
		}
		mu.Unlock()
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if installID != "" {
		for lang, vl := range byLang {
			if err := SaveVanillaLoc(installID, version, lang, vl); err != nil {
				return nil, nil, err
			}
		}
	}
	if out := byLang[locLang]; out != nil {
		return out, locRefs, nil
	}
	return &VanillaLoc{FormatVersion: LocFormatVersion, Sites: map[string]LocEntry{}}, locRefs, nil
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
	var one []string
	for _, f := range inv.loc {
		if loc.LanguageFromRel(f) == locLang {
			one = append(one, f)
		}
	}
	v, _, err := harvestLoc(ctx, one, locLang, "", "")
	return v, err
}

// harvestDocs reads shipped `_*.info` / `*.md` / readme docs into field prose.
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

// harvestDocFile extracts field prose from `_*.info`, markdown, and commented
// READMEs. Comments above a key, on the same line, or immediately before a
// closing `}` (EU5 `key = { # desc }` / below-the-brace) all attach. Fully
// hashed lines are peeled so templates still yield `key = value` pairs.
// `# - key: desc` attribute lists are harvested.
func harvestDocFile(text string) map[string]string {
	out := map[string]string{}
	var pending []string
	lastKey := ""
	for _, raw := range strings.Split(text, "\n") {
		body, hashed := peelDocLine(raw)
		if body == "" {
			pending = pending[:0]
			continue
		}
		if m := attrBulletRe.FindStringSubmatch(body); m != nil {
			putFieldDoc(out, m[1], attrProse(m[2]))
			pending = pending[:0]
			lastKey = ""
			continue
		}
		if m := docKeyRe.FindStringSubmatch(body); m != nil {
			key := strings.ToLower(m[2])
			inline := inlineHashDoc(m[3])
			putFieldDoc(out, key, prose(append(pending, inline)))
			if nameOKRe.MatchString(key) && !stoplist[key] {
				lastKey = key
			} else {
				lastKey = ""
			}
			pending = pending[:0]
			continue
		}
		if body == "}" || strings.HasPrefix(body, "}") {
			if lastKey != "" {
				appendFieldDoc(out, lastKey, prose(pending))
			}
			pending = pending[:0]
			continue
		}
		if hashed {
			pending = append(pending, body)
			continue
		}
		pending = pending[:0]
	}
	return out
}

// peelDocLine strips leading `#` markers so commented READMEs parse as script.
func peelDocLine(raw string) (body string, hashed bool) {
	t := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
	if t == "" {
		return "", false
	}
	hashed = strings.HasPrefix(t, "#")
	for strings.HasPrefix(t, "#") {
		t = strings.TrimSpace(t[1:])
	}
	return t, hashed
}

func inlineHashDoc(rhs string) string {
	h := strings.IndexByte(rhs, '#')
	if h < 0 {
		return ""
	}
	return strings.TrimSpace(strings.TrimLeft(rhs[h+1:], "# "))
}

// attrProse is the description after `# - key: <type> …` / `= { … }: …`.
func attrProse(rhs string) string {
	rhs = strings.TrimSpace(rhs)
	if i := strings.Index(rhs, "}:"); i >= 0 {
		return strings.TrimSpace(rhs[i+2:])
	}
	if h := strings.IndexByte(rhs, '#'); h >= 0 {
		return strings.TrimSpace(rhs[h+1:])
	}
	if strings.HasPrefix(rhs, "{") {
		return ""
	}
	return rhs
}

func putFieldDoc(out map[string]string, key, doc string) {
	key = strings.ToLower(key)
	if !nameOKRe.MatchString(key) || stoplist[key] {
		return
	}
	if _, seen := out[key]; !seen || (out[key] == "" && doc != "") {
		out[key] = doc
	}
}

func appendFieldDoc(out map[string]string, key, extra string) {
	if extra == "" || key == "" {
		return
	}
	if out[key] == "" {
		out[key] = extra
		return
	}
	out[key] = prose([]string{out[key], extra})
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

var dataFnRe = regexp.MustCompile(`\b((?:Get|Set)[A-Za-z0-9_]+)\b`)

func enrichDataTypes(dir string, c *VanillaCache) {
	if dir == "" || c == nil {
		return
	}
	seen := map[string]bool{}
	walk := dir
	for i := 0; i < 2; i++ {
		_ = filepath.WalkDir(walk, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := strings.ToLower(d.Name())
			if !strings.Contains(name, "data_type") && !strings.Contains(name, "datatypes") {
				return nil
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			for _, m := range dataFnRe.FindAllSubmatch(raw, -1) {
				seen[string(m[1])] = true
			}
			return nil
		})
		parent := filepath.Dir(walk)
		if parent == walk {
			break
		}
		walk = parent
	}
	c.DataFunctions = sortedKeys(seen)
}

func sortedKeys(set map[string]bool) []string {
	out := lo.Keys(set)
	slices.Sort(out)
	return out
}

func addCounts(dst *map[string]map[string]int, kind string, src map[string]int) {
	if kind == "" || len(src) == 0 {
		return
	}
	set := (*dst)[kind]
	if set == nil {
		set = map[string]int{}
		(*dst)[kind] = set
	}
	for k, n := range src {
		set[k] += n
	}
}

func keysByCount(m map[string]map[string]int) map[string][]string {
	out := make(map[string][]string, len(m))
	for kind, counts := range m {
		keys := lo.Keys(counts)
		slices.SortFunc(keys, func(a, b string) int {
			if c := counts[b] - counts[a]; c != 0 {
				return c
			}
			return strings.Compare(a, b)
		})
		out[kind] = keys
	}
	return out
}

func blockKeys(counts, blocks map[string]map[string]int) map[string][]string {
	out := map[string][]string{}
	for kind, c := range counts {
		b := blocks[kind]
		var keys []string
		for k, n := range c {
			if b[k]*2 > n {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 {
			continue
		}
		slices.Sort(keys)
		out[kind] = keys
	}
	return out
}

func progress(fn func(pct int, msg string), pct int, msg string) {
	if fn != nil {
		fn(pct, msg)
	}
}
