// scan.go harvests a game install into VanillaCache: script defs, structure keys,
// vocabulary, loc, shipped game-info prose, GUI types/props, metadata, script_docs.

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

// ScanRequest is the install harvest input. script_docs come from UserDataDir.
type ScanRequest struct {
	InstallID, GameID, InstallPath, Version, LocLang string
	OnProgress                                       func(pct int, msg string)
}

// Scan harvests installPath into a VanillaCache.
func Scan(ctx context.Context, req ScanRequest) (*VanillaCache, *VanillaLoc, error) {
	info := game.Get(req.GameID)
	if info == nil {
		return nil, nil, os.ErrInvalid
	}
	locLang := req.LocLang
	if locLang == "" {
		locLang = "english"
	}
	version := req.Version
	if version == "" {
		version = "latest"
	}
	onProgress := req.OnProgress
	progress(onProgress, 5, "listing files")
	inv := gather(scriptRoots(info, req.InstallPath))

	progress(onProgress, 15, "parsing script")
	files := append(append([]fileRef{}, inv.script...), inv.gui...)
	acc, err := collectExtracts(ctx, req.GameID, files, true, nil)
	if err != nil {
		return nil, nil, err
	}

	progress(onProgress, 70, "reading localization")
	vloc, locFileRefs, err := harvestLoc(ctx, inv.loc, locLang, req.InstallID, version)
	if err != nil {
		return nil, nil, err
	}

	progress(onProgress, 80, "reading game info")
	fieldInfo, fieldByKind, docStructs := harvestGameInfo(req.GameID, inv.info)
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
		InstallID:        req.InstallID,
		GameID:           req.GameID,
		InstallPath:      req.InstallPath,
		GameVersion:      version,
		ScannedAt:        time.Now().UTC().Format(time.RFC3339),
		Defs:             acc.defs,
		Edges:            acc.edges,
		LocRefs:          append(locKindRefs(acc.refs), locKindRefs(locFileRefs)...),
		FieldValueKinds:  kinds,
		FieldEnumsByKind: VoteFieldEnums(acc.fieldRHSByKind, kinds),
		FieldInfo:        fieldInfo,
		FieldInfoByKind:  fieldByKind,
		Structures:       keysByCount(acc.structures),
		StructureBlocks:  blockKeys(acc.structures, acc.structBlocks),
		Vocabulary:       sortedKeys(acc.vocab),
		GUITypes:         sortedKeys(acc.guiTypes),
		GUIProps:         sortedKeys(acc.guiProps),
		MetaKeys:         readMetaKeys(req.InstallPath, inv.meta),
	}

	progress(onProgress, 90, "reading script_docs")
	docsDir := scriptDocsDir(req.GameID, req.InstallPath)
	enrichScriptDocs(docsDir, c)
	enrichDataTypes(docsDir, c)
	PrepareCache(c)

	progress(onProgress, 100, "done")
	return c, vloc, nil
}

func scriptDocsDir(gameID, installPath string) string {
	info := game.Get(gameID)
	if info == nil {
		return ""
	}
	ud := game.UserDataDir(gameID, installPath)
	if ud == "" {
		return ""
	}
	return filepath.Join(ud, info.ScriptDocsSubdir)
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
	script, gui, info []fileRef
	loc, meta         []string
}

// ClassifyRel buckets a file by name/rel for the install walk and mod indexer.
func ClassifyRel(rel, name string) string {
	lower := strings.ToLower(name)
	slash := strings.ReplaceAll(rel, "\\", "/")
	switch {
	case isGameInfoName(name):
		return "info"
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

// isGameInfoName reports CK3 `_*.info` / `*.md` and EU5 readme/info text files.
// Underscored data (`_default.txt`, `_hardcoded.txt`) stays unclassified.
func isGameInfoName(name string) bool {
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
			case "info":
				inv.info = append(inv.info, f)
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
			out.Sites[k] = LocEntry{Path: v.Path, Line: v.Line, Value: v.Value}
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

// harvestGameInfo reads shipped `_*.info` / `*.md` / readme files into field prose.
func harvestGameInfo(gameID string, docs []fileRef) (
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
	out := slices.Collect(maps.Keys(set))
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
		keys := slices.Collect(maps.Keys(counts))
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

func enrichScriptDocs(dir string, c *VanillaCache) {
	if dir == "" {
		return
	}
	entries := parseScriptDocs(dir)
	if len(entries) == 0 {
		return
	}
	effects, triggers, vocab := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, v := range c.Vocabulary {
		vocab[v] = true
	}
	if c.TokenDoc == nil {
		c.TokenDoc = map[string]string{}
	}
	if c.TokenUsage == nil {
		c.TokenUsage = map[string]string{}
	}
	if c.TokenScopes == nil {
		c.TokenScopes = map[string]string{}
	}
	for _, e := range entries {
		vocab[e.name] = true
		lk := strings.ToLower(e.name)
		if e.doc != "" && c.TokenDoc[lk] == "" {
			c.TokenDoc[lk] = e.doc
		}
		if e.usage != "" {
			c.TokenUsage[lk] = e.usage
		}
		if e.scopes != "" {
			c.TokenScopes[lk] = e.scopes
		}
		switch e.kind {
		case "effect":
			effects[e.name] = true
		case "trigger":
			triggers[e.name] = true
		}
	}
	c.Vocabulary, c.Effects, c.Triggers = sortedKeys(vocab), sortedKeys(effects), sortedKeys(triggers)
}

type docToken struct{ name, kind, doc, usage, scopes string }

func parseScriptDocs(dir string) []docToken {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []docToken
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		name := strings.ToLower(de.Name())
		if !strings.HasSuffix(name, ".log") && !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			continue
		}
		text := string(raw)
		kind := kindFromDocFilename(name)
		if looksMarkdownDocs(text) {
			out = append(out, parseMarkdownDocs(text, kind)...)
		} else {
			out = append(out, parseClassicDocs(text, kind)...)
		}
	}
	return out
}

func looksMarkdownDocs(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## ") {
			return true
		}
		if strings.HasPrefix(t, "----") {
			return false
		}
	}
	return false
}

func kindFromDocFilename(name string) string {
	switch {
	case strings.Contains(name, "event_target"):
		return "event_target"
	case strings.Contains(name, "event_scope"):
		return "scope_type"
	case strings.Contains(name, "effect"):
		return "effect"
	case strings.Contains(name, "trigger"):
		return "trigger"
	case strings.Contains(name, "modif"):
		return "modifier"
	default:
		return ""
	}
}

func prose(parts []string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(strings.Join(parts, " ")), " "))
}

func stripDocMarkup(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "*", "")
	return strings.TrimSpace(s)
}

func splitDocMeta(parts []string) (doc, usage, scopes string) {
	var body []string
	var targets string
	for _, p := range parts {
		t := stripDocMarkup(strings.TrimSpace(p))
		low := strings.ToLower(t)
		switch {
		case strings.HasPrefix(low, "usage:"):
			usage = strings.TrimSpace(t[len("usage:"):])
		case strings.HasPrefix(low, "supported scopes:"):
			scopes = strings.TrimSpace(t[len("supported scopes:"):])
		case strings.HasPrefix(low, "supported targets:"):
			targets = strings.TrimSpace(t[len("supported targets:"):])
		default:
			body = append(body, p)
		}
	}
	if targets != "" {
		if scopes != "" {
			scopes = scopes + "; targets: " + targets
		} else {
			scopes = targets
		}
	}
	return prose(body), usage, scopes
}

func parseMarkdownDocs(text, kind string) []docToken {
	var out []docToken
	var cur *docToken
	var body []string
	flush := func() {
		if cur == nil {
			return
		}
		cur.doc, cur.usage, cur.scopes = splitDocMeta(body)
		out = append(out, *cur)
		cur, body = nil, body[:0]
	}
	for _, line := range strings.Split(text, "\n") {
		if h := strings.TrimSpace(line); strings.HasPrefix(h, "## ") {
			flush()
			if m := tokenRe.FindString(strings.TrimSpace(strings.TrimLeft(h, "# "))); m != "" {
				cur = &docToken{name: m, kind: kind}
			}
			continue
		}
		if cur != nil {
			body = append(body, line)
		}
	}
	flush()
	return out
}

func parseClassicDocs(text, kind string) []docToken {
	var out []docToken
	var cur *docToken
	var parts []string
	flush := func() {
		if cur == nil {
			return
		}
		cur.doc, cur.usage, cur.scopes = splitDocMeta(parts)
		out = append(out, *cur)
		cur, parts = nil, nil
	}
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(trimmed, "----"):
			flush()
		case trimmed == "":
		case strings.Contains(strings.ToLower(trimmed), "documentation"):
			continue
		case cur == nil:
			name := tokenRe.FindString(trimmed)
			if name == "" {
				continue
			}
			cur = &docToken{name: name, kind: kind}
			if i := strings.Index(trimmed, " - "); i >= 0 {
				parts = append(parts, trimmed[i+3:])
			}
		default:
			parts = append(parts, trimmed)
		}
	}
	flush()
	return out
}
