// scan.go harvests a game install into a vanilla Cache: it walks the script roots,
// parses the corpus in parallel, and derives definitions, per-kind structure keys,
// script vocabulary, English loc, shipped-doc prose, GUI types/props, and metadata
// keys. An optional script_docs dump only enriches effect/trigger classification
// and prose; every other field comes from the corpus and is always available.

package model

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"golang.org/x/sync/errgroup"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/loc"
	"paradox-modding-tools/services/internal/parser"
)

// locValueLimit caps loc values kept in the Cache; the edit flow re-reads the yml.
const locValueLimit = 200

// scanConcurrency bounds parser workers; parsing is CPU-bound and allocation-light.
const scanConcurrency = 8

var (
	defNameRe  = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)
	eventIDRe  = regexp.MustCompile(`^[A-Za-z0-9_-]+\.\d+$`)
	titleKeyRe = regexp.MustCompile(`^[ekdcb]_[A-Za-z0-9_-]+$`)
	nameOKRe   = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	docKeyRe   = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$`)
	tokenRe    = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)`)
)

// stoplist are grammar/logic words served by the context layer, never harvested as
// structure keys or vocabulary.
var stoplist = map[string]bool{
	"if": true, "else": true, "else_if": true, "limit": true, "and": true,
	"or": true, "not": true, "nor": true, "nand": true, "this": true,
	"root": true, "prev": true, "from": true, "yes": true, "no": true,
	"value": true, "add": true, "multiply": true, "divide": true,
	"subtract": true, "min": true, "max": true, "factor": true, "base": true,
	"first_valid": true, "triggered_desc": true, "random_list": true,
}

// Scan harvests installPath into a Cache for gameID. scriptDocsDir points at an
// optional in-game script_docs dump ("" to skip enrichment). onProgress may be nil.
func Scan(
	ctx context.Context,
	gameID, installPath, scriptDocsDir string,
	onProgress func(pct int, msg string),
) (*Cache, error) {
	info := game.Get(gameID)
	if info == nil {
		return nil, os.ErrInvalid
	}
	progress(onProgress, 5, "listing files")
	inv := gather(gameID, installPath, scriptRoots(info, installPath))

	progress(onProgress, 15, "parsing script")
	acc := newAccum()
	if err := parseCorpus(ctx, gameID, inv, acc); err != nil {
		return nil, err
	}

	progress(onProgress, 70, "reading localization")
	locEnglish := readEnglish(inv.loc)

	progress(onProgress, 80, "reading docs")
	fieldDocs, docStructs := harvestDocs(gameID, inv.docs)
	for kind, keys := range docStructs {
		for k := range keys {
			acc.structures[kind][k] = true
		}
	}

	c := &Cache{
		FormatVersion: CacheFormatVersion,
		GameID:        gameID,
		InstallPath:   installPath,
		GameVersion:   game.ReadGameVersion(installPath),
		ScannedAt:     time.Now().UTC().Format(time.RFC3339),
		Defs:          acc.defs,
		LocEnglish:    locEnglish,
		FieldDocs:     fieldDocs,
		Structures:    setsToLists(acc.structures),
		Vocabulary:    sortedKeys(acc.vocab),
		Objects:       sortedKeys(acc.objects),
		GUITypes:      sortedKeys(acc.guiTypes),
		GUIProps:      sortedKeys(acc.guiProps),
		MetaKeys:      readMetaKeys(installPath, inv.meta),
		RootScopes:    map[string]string{},
	}

	progress(onProgress, 90, "reading script_docs")
	enrichScriptDocs(scriptDocsDir, info.ScriptDocsFormat, c)

	progress(onProgress, 100, "done")
	return c, nil
}

// scriptRoots returns the absolute folders to walk for gameID.
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
type fileRef struct{ abs, rel string }

// inventory buckets the walked files by role.
type inventory struct {
	script, gui, docs []fileRef
	loc, meta         []string
}

// gather walks the roots once and classifies files by extension/name.
func gather(gameID string, installPath string, roots []string) inventory {
	var inv inventory
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := d.Name()
			lower := strings.ToLower(name)
			rel, _ := filepath.Rel(root, p)
			switch {
			case strings.HasPrefix(name, "_") && strings.HasSuffix(lower, ".info"):
				inv.docs = append(inv.docs, fileRef{p, rel})
			case strings.HasSuffix(lower, ".md"):
				inv.docs = append(inv.docs, fileRef{p, rel})
			case strings.HasSuffix(lower, ".gui"):
				inv.gui = append(inv.gui, fileRef{p, rel})
			case strings.HasSuffix(lower, ".yml"), strings.HasSuffix(lower, ".yaml"):
				if strings.Contains(strings.ReplaceAll(rel, "\\", "/"), "localization") {
					inv.loc = append(inv.loc, p)
				}
			case lower == "metadata.json":
				inv.meta = append(inv.meta, p)
			case strings.HasSuffix(lower, ".txt") && !strings.HasPrefix(name, "_"):
				inv.script = append(inv.script, fileRef{p, rel})
			}
			return nil
		})
	}
	return inv
}

// accum collects harvest results from all workers behind one mutex.
type accum struct {
	mu         sync.Mutex
	defs       []Def
	structures map[string]map[string]bool
	vocab      map[string]bool
	objects    map[string]bool
	guiTypes   map[string]bool
	guiProps   map[string]bool
}

func newAccum() *accum {
	return &accum{
		structures: map[string]map[string]bool{},
		vocab:      map[string]bool{},
		objects:    map[string]bool{},
		guiTypes:   map[string]bool{},
		guiProps:   map[string]bool{},
	}
}

// fileResult is one file's harvest, merged into the accumulator under lock.
type fileResult struct {
	defs               []Def
	structKind         string
	structKeys, vocab  []string
	guiTypes, guiProps []string
}

func (a *accum) merge(r fileResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.defs = append(a.defs, r.defs...)
	for _, d := range r.defs {
		a.objects[d.Key] = true
	}
	if r.structKind != "" {
		set := a.structures[r.structKind]
		if set == nil {
			set = map[string]bool{}
			a.structures[r.structKind] = set
		}
		for _, k := range r.structKeys {
			set[k] = true
		}
	}
	for _, v := range r.vocab {
		a.vocab[v] = true
	}
	for _, t := range r.guiTypes {
		a.guiTypes[t] = true
	}
	for _, p := range r.guiProps {
		a.guiProps[p] = true
	}
}

// parseCorpus parses script + gui files in parallel and merges their harvest.
func parseCorpus(ctx context.Context, gameID string, inv inventory, acc *accum) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(scanConcurrency)
	files := append(append([]fileRef{}, inv.script...), inv.gui...)
	for _, f := range files {
		f := f
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			acc.merge(processFile(gameID, f))
			return nil
		})
	}
	return g.Wait()
}

// processFile parses one script/gui file and extracts its defs + harvest.
func processFile(gameID string, f fileRef) fileResult {
	res, err := parser.ParseFile(f.abs)
	if err != nil {
		return fileResult{}
	}
	rule := game.MatchExtract(gameID, f.rel)
	li := res.Lines()
	switch rule.Mode {
	case game.ModeGUIType:
		return extractGUI(res.Root, li, f.abs)
	case game.ModeEventID:
		fr := fileResult{structKind: "event"}
		fr.defs = extractEvents(res.Root, li, gameID, f.abs)
		harvestBodies(res.Root, &fr)
		return fr
	case game.ModeTopLevelKey:
		fr := fileResult{structKind: rule.Kind}
		fr.defs = extractTopLevel(res.Root, li, gameID, rule.Kind, f.abs)
		harvestBodies(res.Root, &fr)
		return fr
	case game.ModeLocKey:
		return fileResult{}
	default:
		return fileResult{}
	}
}

// extractTopLevel reads root `key = { ... }` definitions (common/ folders).
func extractTopLevel(root *parser.Root, li *parser.LineIndex, gameID, kind, path string) []Def {
	var defs []Def
	for _, st := range root.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok || a.Key.Quoted || (a.Op != "=" && a.Op != "?=") {
			continue
		}
		name := game.KeyIdentity(gameID, a.Key.Text)
		if !defNameRe.MatchString(name) || name == "namespace" {
			continue
		}
		defs = append(defs, Def{Type: kind, Key: name, Path: path, Line: li.PositionAt(a.Key.Range.Start).Line})
	}
	return defs
}

// extractEvents reads `namespace.N = { ... }` event ids (top-level only) plus any
// inline `scripted_trigger/effect NAME = { ... }` declared at any depth.
func extractEvents(root *parser.Root, li *parser.LineIndex, gameID, path string) []Def {
	var defs []Def
	var walk func(stmts []parser.Statement, depth int)
	walk = func(stmts []parser.Statement, depth int) {
		var marker string
		for _, st := range stmts {
			if vs, ok := st.(*parser.ValueStmt); ok {
				if sc, ok := vs.Value.(*parser.Scalar); ok && !sc.Quoted {
					if sc.Text == "scripted_trigger" || sc.Text == "scripted_effect" {
						marker = sc.Text
					} else {
						marker = ""
					}
				}
				if b := blockOf(vs.Value); b != nil {
					walk(b.Statements, depth+1)
				}
				continue
			}
			a, ok := st.(*parser.Assignment)
			if !ok {
				marker = ""
				continue
			}
			m := marker
			marker = ""
			if !a.Key.Quoted && (a.Op == "=" || a.Op == "?=") {
				if m != "" && defNameRe.MatchString(a.Key.Text) {
					defs = append(defs, Def{Type: m, Key: a.Key.Text, Path: path, Line: li.PositionAt(a.Key.Range.Start).Line})
				} else if depth == 0 && eventIDRe.MatchString(a.Key.Text) {
					defs = append(defs, Def{Type: "event", Key: a.Key.Text, Path: path, Line: li.PositionAt(a.Key.Range.Start).Line})
				}
			}
			if b := blockOf(a.Value); b != nil {
				walk(b.Statements, depth+1)
			}
		}
	}
	walk(root.Statements, 0)
	return defs
}

// extractGUI reads `type/template/local_template NAME` (and `types Group { ... }`)
// definitions and harvests GUI property keys.
func extractGUI(root *parser.Root, li *parser.LineIndex, path string) fileResult {
	fr := fileResult{}
	var scan func(stmts []parser.Statement)
	scan = func(stmts []parser.Statement) {
		for i := 0; i < len(stmts)-1; i++ {
			mv, ok := stmts[i].(*parser.ValueStmt)
			if !ok {
				continue
			}
			sc, ok := mv.Value.(*parser.Scalar)
			if !ok || sc.Quoted {
				continue
			}
			kw := strings.ToLower(sc.Text)
			if kw != "type" && kw != "template" && kw != "local_template" && kw != "types" {
				continue
			}
			named, ok := stmts[i+1].(*parser.Assignment)
			if !ok || named.Key.Quoted {
				continue
			}
			if kw == "types" {
				if b := blockOf(named.Value); b != nil {
					scan(b.Statements)
				}
			} else if defNameRe.MatchString(named.Key.Text) {
				fr.defs = append(fr.defs, Def{Type: "gui_type", Key: named.Key.Text, Path: path, Line: li.PositionAt(named.Key.Range.Start).Line})
				fr.guiTypes = append(fr.guiTypes, named.Key.Text)
			}
		}
	}
	scan(root.Statements)
	props := map[string]bool{}
	parser.WalkStatements(root, func(st parser.Statement) bool {
		if a, ok := st.(*parser.Assignment); ok && !a.Key.Quoted {
			k := strings.ToLower(a.Key.Text)
			if nameOKRe.MatchString(k) && !stoplist[k] {
				props[k] = true
			}
		}
		return true
	})
	for k := range props {
		fr.guiProps = append(fr.guiProps, k)
	}
	return fr
}

// harvestBodies collects depth-1 child keys (structure keys) and depth-2 keys
// (vocabulary) of every top-level definition body in the file.
func harvestBodies(root *parser.Root, fr *fileResult) {
	structs := map[string]bool{}
	vocab := map[string]bool{}
	for _, st := range root.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok {
			continue
		}
		body := blockOf(a.Value)
		if body == nil {
			continue
		}
		for _, c := range body.Statements {
			ca, ok := c.(*parser.Assignment)
			if !ok || ca.Key.Quoted {
				continue
			}
			k := strings.ToLower(ca.Key.Text)
			if !nameOKRe.MatchString(k) || stoplist[k] {
				continue
			}
			structs[k] = true
			vocab[k] = true
			if sub := blockOf(ca.Value); sub != nil {
				for _, gc := range sub.Statements {
					if ga, ok := gc.(*parser.Assignment); ok && !ga.Key.Quoted {
						gk := strings.ToLower(ga.Key.Text)
						if nameOKRe.MatchString(gk) && !stoplist[gk] {
							vocab[gk] = true
						}
					}
				}
			}
		}
	}
	for k := range structs {
		fr.structKeys = append(fr.structKeys, k)
	}
	for k := range vocab {
		fr.vocab = append(fr.vocab, k)
	}
}

// blockOf returns the Block a value owns (block or tagged-block), or nil.
func blockOf(v parser.Value) *parser.Block {
	switch b := v.(type) {
	case *parser.Block:
		return b
	case *parser.TaggedBlock:
		return &b.Block
	default:
		return nil
	}
}

// readEnglish parses english loc files into a key->value map (values capped).
func readEnglish(files []string) map[string]string {
	out := map[string]string{}
	for _, f := range files {
		if loc.LanguageFromFilename(f) != "english" {
			continue
		}
		res, err := loc.ParseFile(f)
		if err != nil {
			continue
		}
		for _, e := range res.Entries {
			v := e.Value
			if len(v) > locValueLimit {
				v = v[:locValueLimit]
			}
			out[e.Key] = v
		}
	}
	return out
}

// harvestDocs reads shipped `_*.info`/`*.md` docs into field prose and per-kind
// documented structure keys.
func harvestDocs(gameID string, docs []fileRef) (fieldDocs map[string]string, structs map[string]map[string]bool) {
	fieldDocs = map[string]string{}
	structs = map[string]map[string]bool{}
	for _, f := range docs {
		raw, err := os.ReadFile(f.abs)
		if err != nil {
			continue
		}
		kind := game.MatchExtract(gameID, f.rel).Kind
		for key, doc := range harvestDocFile(string(raw)) {
			if doc != "" {
				if _, seen := fieldDocs[key]; !seen {
					fieldDocs[key] = doc
				}
			}
			if kind != "" {
				if structs[kind] == nil {
					structs[kind] = map[string]bool{}
				}
				structs[kind][key] = true
			}
		}
	}
	return fieldDocs, structs
}

// harvestDocFile extracts `key = value  # doc` candidates from one doc file, with
// any preceding `#` comment lines as prose. Returns key -> doc (may be "").
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
				doc := strings.Join(append(pending, inlineDoc), " ")
				doc = strings.TrimSpace(strings.Join(strings.Fields(doc), " "))
				if _, seen := out[key]; !seen || (out[key] == "" && doc != "") {
					out[key] = doc
				}
			}
		}
		pending = pending[:0]
	}
	return out
}

// readMetaKeys collects top-level keys from every .metadata/metadata.json plus the
// install's own metadata.json, if present.
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

// enrichScriptDocs folds an optional in-game script_docs dump into c: it classifies
// vocabulary into effects/triggers/modifiers and adds field-doc prose. Missing dir
// or format leaves the corpus baseline untouched.
func enrichScriptDocs(dir, format string, c *Cache) {
	if dir == "" {
		return
	}
	entries := parseScriptDocs(dir, format)
	if len(entries) == 0 {
		return
	}
	effects, triggers, modifiers := map[string]bool{}, map[string]bool{}, map[string]bool{}
	vocab := map[string]bool{}
	for _, v := range c.Vocabulary {
		vocab[v] = true
	}
	for _, e := range entries {
		vocab[e.name] = true
		if e.doc != "" {
			if _, seen := c.FieldDocs[e.name]; !seen {
				c.FieldDocs[e.name] = e.doc
			}
		}
		switch e.kind {
		case "effect":
			effects[e.name] = true
		case "trigger":
			triggers[e.name] = true
		case "modifier":
			modifiers[e.name] = true
		}
	}
	c.Vocabulary = sortedKeys(vocab)
	c.Effects = sortedKeys(effects)
	c.Triggers = sortedKeys(triggers)
	c.Modifiers = sortedKeys(modifiers)
}

// docToken is one entry parsed from a script_docs dump.
type docToken struct{ name, kind, doc string }

// parseScriptDocs reads a script_docs dump directory. "markdown" reads `## name`
// sections from *.md; "classic" reads blank-line-separated blocks from *.log.
func parseScriptDocs(dir, format string) []docToken {
	var out []docToken
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		name := strings.ToLower(de.Name())
		kind := kindFromDocFilename(name)
		raw, err := os.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			continue
		}
		if format == "markdown" && strings.HasSuffix(name, ".md") {
			out = append(out, parseMarkdownDocs(string(raw), kind)...)
		} else if format != "markdown" && strings.HasSuffix(name, ".log") {
			out = append(out, parseClassicDocs(string(raw), kind)...)
		}
	}
	return out
}

// kindFromDocFilename maps a dump filename to a token kind by keyword.
func kindFromDocFilename(name string) string {
	switch {
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

// parseMarkdownDocs reads `## name` sections; the body until the next header is doc.
func parseMarkdownDocs(text, kind string) []docToken {
	var out []docToken
	var cur *docToken
	var body []string
	flush := func() {
		if cur != nil {
			cur.doc = strings.TrimSpace(strings.Join(strings.Fields(strings.Join(body, " ")), " "))
			out = append(out, *cur)
		}
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

// parseClassicDocs reads blank-line-separated blocks; the first token names the
// entry and the remaining lines are its doc.
func parseClassicDocs(text, kind string) []docToken {
	var out []docToken
	for _, block := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) == 0 || lines[0] == "" {
			continue
		}
		name := tokenRe.FindString(strings.TrimSpace(lines[0]))
		if name == "" {
			continue
		}
		doc := strings.TrimSpace(strings.Join(strings.Fields(strings.Join(lines[1:], " ")), " "))
		out = append(out, docToken{name: name, kind: kind, doc: doc})
	}
	return out
}

// setsToLists sorts each membership set into a stable slice.
func setsToLists(m map[string]map[string]bool) map[string][]string {
	out := make(map[string][]string, len(m))
	for k, set := range m {
		out[k] = sortedKeys(set)
	}
	return out
}

// sortedKeys returns the set's keys sorted ascending.
func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// progress invokes onProgress if non-nil.
func progress(fn func(pct int, msg string), pct int, msg string) {
	if fn != nil {
		fn(pct, msg)
	}
}
