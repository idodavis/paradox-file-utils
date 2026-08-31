// extract.go is the per-file catalog extractor (scan, BuildIndex, session reindex).

package catalog

import (
	"cmp"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"

	"github.com/samber/lo"
)

var (
	defNameRe = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)
	eventIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]+\.\d+$`)
	nameOKRe  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// stoplist are grammar/logic words never harvested as structure keys or vocabulary.
var stoplist = map[string]bool{
	"if": true, "else": true, "else_if": true, "limit": true, "and": true,
	"or": true, "not": true, "nor": true, "nand": true, "this": true,
	"root": true, "prev": true, "from": true, "yes": true, "no": true,
	"value": true, "add": true, "multiply": true, "divide": true,
	"subtract": true, "min": true, "max": true, "factor": true, "base": true,
	"first_valid": true, "triggered_desc": true, "random_list": true,
}

// EdgeKindCall marks a scripted-effect invocation (not drawn on the canvas).
const EdgeKindCall = "call"

// CallCandidate is a possible scripted-effect call; ApplyCallEdges filters it.
type CallCandidate struct {
	From, To, Path string
	Line           int
}

// FileExtract is one parsed file's catalog harvest.
type FileExtract struct {
	Defs       []Def
	Refs       []Ref
	Edges      []Edge
	Cands      []CallCandidate
	Loc        LocDelta
	StructKind string
	StructKeys map[string]bool
	Vocab      map[string]bool
	GUITypes   map[string]bool
	GUIProps   map[string]bool
}

// ExtractLoc turns a localization file into loc_key defs and per-language values.
func ExtractLoc(absPath, content, origin string) (defs []Def, locd LocDelta) {
	r := loc.Parse(content)
	lang := loc.LanguageOf(absPath, r.Language)
	vals := map[string]LocEntry{}
	for _, e := range r.Entries {
		defs = append(defs, Def{
			Type: "loc_key", Key: e.Key, Path: absPath, Line: e.Line,
			Start: e.KeyRange.Start, End: e.KeyRange.End, Origin: origin,
		})
		v := e.Value
		if len(v) > locValueLimit {
			v = v[:locValueLimit]
		}
		vals[e.Key] = LocEntry{Value: v, File: absPath, Line: e.Line}
	}
	return defs, LocDelta{Lang: lang, Vals: vals}
}

// ExtractParsed extracts defs, refs, edges, and call candidates from a parsed
// script or GUI file. Callers turn Cands into edges via ApplyCallEdges.
func ExtractParsed(
	gameID, absPath, rel, origin string,
	res jomini.Result,
	harvestBodies bool,
) FileExtract {
	absPath = filepath.Clean(absPath)
	rule := game.MatchExtract(gameID, rel)
	var ex FileExtract
	switch rule.Mode {
	case game.ModeGUIType:
		ex.Defs, ex.GUITypes, ex.GUIProps = extractGUI(res.Root, res.Lines(), absPath)
	case game.ModeTopLevelKey, game.ModeEventID:
		if rule.Mode == game.ModeEventID {
			ex.Defs = extractEvents(res.Root, res.Lines(), gameID, absPath, origin)
		} else {
			ex.Defs = extractTopLevel(res.Root, res.Lines(), gameID, rule.Kind, absPath, origin)
		}
		if harvestBodies {
			ex.StructKind = rule.Kind
			if rule.Mode == game.ModeEventID {
				ex.StructKind = "event"
			}
			ex.StructKeys, ex.Vocab = harvestStructVocab(res.Root)
		}
	default:
		return ex
	}
	ex.Refs, ex.Edges, ex.Cands = extractRefsAndEdges(res.Root, res.Lines(), absPath, ex.Defs)
	return ex
}

// ExtractFile extracts one decoded (CR-free) file, choosing the loc or script
// path from the game's extract rule; loc files skip the script parse.
func ExtractFile(gameID, absPath, rel, origin, text string, harvestBodies bool) FileExtract {
	text = jomini.Normalize(text)
	if game.MatchExtract(gameID, rel).Mode == game.ModeLocKey {
		var ex FileExtract
		ex.Defs, ex.Loc = ExtractLoc(filepath.Clean(absPath), text, origin)
		return ex
	}
	return ExtractParsed(gameID, absPath, rel, origin, jomini.Parse(text), harvestBodies)
}

func makeDef(kind, key, path, origin string, kr jomini.Range, li *jomini.LineIndex) Def {
	return Def{
		Type:   game.CanonicalKind(kind),
		Key:    key,
		Path:   path,
		Line:   li.PositionAt(kr.Start).Line,
		Start:  kr.Start,
		End:    kr.End,
		Origin: origin,
	}
}

func extractTopLevel(root *jomini.Root, li *jomini.LineIndex, gameID, kind, path, origin string) []Def {
	var defs []Def
	for _, st := range root.Statements {
		a, ok := st.(*jomini.Assignment)
		if !ok || a.Key.Quoted || (a.Op != "=" && a.Op != "?=") {
			continue
		}
		name := game.KeyIdentity(gameID, a.Key.Text)
		if !defNameRe.MatchString(name) || name == "namespace" {
			continue
		}
		defs = append(defs, makeDef(kind, name, path, origin, a.Key.Range, li))
	}
	return defs
}

func extractEvents(root *jomini.Root, li *jomini.LineIndex, gameID, path, origin string) []Def {
	var defs []Def
	var walk func(stmts []jomini.Statement, depth int)
	walk = func(stmts []jomini.Statement, depth int) {
		var marker string
		for _, st := range stmts {
			if vs, ok := st.(*jomini.ValueStmt); ok {
				if sc, ok := vs.Value.(*jomini.Scalar); ok && !sc.Quoted {
					if sc.Text == "scripted_trigger" || sc.Text == "scripted_effect" {
						marker = sc.Text
					} else {
						marker = ""
					}
				}
				if b := jomini.BlockOf(vs.Value); b != nil {
					walk(b.Statements, depth+1)
				}
				continue
			}
			a, ok := st.(*jomini.Assignment)
			if !ok {
				marker = ""
				continue
			}
			m := marker
			marker = ""
			if !a.Key.Quoted && (a.Op == "=" || a.Op == "?=") {
				if m != "" && defNameRe.MatchString(a.Key.Text) {
					defs = append(defs, makeDef(m, a.Key.Text, path, origin, a.Key.Range, li))
				} else if depth == 0 && eventIDRe.MatchString(a.Key.Text) {
					defs = append(defs, makeDef("event", a.Key.Text, path, origin, a.Key.Range, li))
				}
			}
			if b := jomini.BlockOf(a.Value); b != nil {
				walk(b.Statements, depth+1)
			}
		}
	}
	walk(root.Statements, 0)
	return defs
}

func extractGUI(root *jomini.Root, li *jomini.LineIndex, path string) ([]Def, map[string]bool, map[string]bool) {
	var defs []Def
	guiTypes := map[string]bool{}
	var scan func(stmts []jomini.Statement)
	scan = func(stmts []jomini.Statement) {
		for i := 0; i < len(stmts)-1; i++ {
			mv, ok := stmts[i].(*jomini.ValueStmt)
			if !ok {
				continue
			}
			sc, ok := mv.Value.(*jomini.Scalar)
			if !ok || sc.Quoted {
				continue
			}
			kw := strings.ToLower(sc.Text)
			if kw != "type" && kw != "template" && kw != "local_template" && kw != "types" {
				continue
			}
			named, ok := stmts[i+1].(*jomini.Assignment)
			if !ok || named.Key.Quoted {
				continue
			}
			if kw == "types" {
				if b := jomini.BlockOf(named.Value); b != nil {
					scan(b.Statements)
				}
			} else if defNameRe.MatchString(named.Key.Text) {
				defs = append(defs, makeDef("gui_type", named.Key.Text, path, "", named.Key.Range, li))
				guiTypes[named.Key.Text] = true
			}
		}
	}
	scan(root.Statements)
	guiProps := map[string]bool{}
	jomini.Walk(root, func(st jomini.Statement, _ int, _ *jomini.Block) bool {
		a, ok := st.(*jomini.Assignment)
		if !ok || a.Key.Quoted {
			return true
		}
		k := strings.ToLower(a.Key.Text)
		if nameOKRe.MatchString(k) && !stoplist[k] {
			guiProps[k] = true
		}
		return true
	})
	return defs, guiTypes, guiProps
}

func harvestStructVocab(root *jomini.Root) (structs, vocab map[string]bool) {
	structs, vocab = map[string]bool{}, map[string]bool{}
	if root == nil {
		return structs, vocab
	}
	for _, st := range root.Statements {
		a, ok := st.(*jomini.Assignment)
		if !ok {
			continue
		}
		body := jomini.BlockOf(a.Value)
		if body == nil {
			continue
		}
		for _, c := range body.Statements {
			ca, ok := c.(*jomini.Assignment)
			if !ok || ca.Key.Quoted {
				continue
			}
			k := strings.ToLower(ca.Key.Text)
			if !nameOKRe.MatchString(k) || stoplist[k] {
				continue
			}
			structs[k] = true
			vocab[k] = true
			if sub := jomini.BlockOf(ca.Value); sub != nil {
				for _, gc := range sub.Statements {
					if ga, ok := gc.(*jomini.Assignment); ok && !ga.Key.Quoted {
						gk := strings.ToLower(ga.Key.Text)
						if nameOKRe.MatchString(gk) && !stoplist[gk] {
							vocab[gk] = true
						}
					}
				}
			}
		}
	}
	return structs, vocab
}

type edgeFrame struct {
	key, nameKey string
}

type defAtLine struct {
	line int
	key  string
}

func defContainers(defs []Def) []defAtLine {
	var out []defAtLine
	for _, d := range defs {
		if d.Type != "loc_key" {
			out = append(out, defAtLine{d.Line, d.Key})
		}
	}
	slices.SortFunc(out, func(a, b defAtLine) int { return cmp.Compare(a.line, b.line) })
	return out
}

func containerAt(containers []defAtLine, line int) string {
	lo, hi, best := 0, len(containers)-1, ""
	for lo <= hi {
		mid := (lo + hi) / 2
		if containers[mid].line <= line {
			best = containers[mid].key
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return best
}

func extractRefsAndEdges(
	root *jomini.Root, li *jomini.LineIndex, path string, defs []Def,
) ([]Ref, []Edge, []CallCandidate) {
	var refs []Ref
	var edges []Edge
	var cands []CallCandidate
	if root == nil {
		return nil, nil, nil
	}
	containers := defContainers(defs)
	var stack []edgeFrame
	var walk func(stmts []jomini.Statement)
	walk = func(stmts []jomini.Statement) {
		for _, st := range stmts {
			a, ok := st.(*jomini.Assignment)
			if ok {
				if !a.Key.Quoted {
					key := a.Key.Text
					line := li.PositionAt(a.Key.Range.Start).Line
					from := containerAt(containers, line)
					prop := loc.Classify(key)
					if prop != loc.PropNone {
						if sc, ok := a.Value.(*jomini.Scalar); ok && sc.Text != "" && loc.LooksLikeKey(sc.Text) {
							kind := "loc-broad"
							if prop == loc.PropStrict {
								kind = "loc"
							}
							refs = append(refs, Ref{
								Key: sc.Text, Kind: kind, Path: path,
								Line:  li.PositionAt(sc.Range.Start).Line,
								Start: sc.Range.Start, End: sc.Range.End,
							})
						}
					}
					if fk := game.FireKind(key); fk != "" {
						kind, nameKey := fireSiteKind(stack, key)
						walkFireTargets(a.Value, func(to string, start, end int) {
							refs = append(refs, Ref{
								Key: to, Kind: fk, Path: path,
								Line: li.PositionAt(start).Line, Start: start, End: end,
							})
							edges = append(edges, Edge{
								From: from, To: to, Via: key, Path: path,
								Line: li.PositionAt(start).Line, Kind: kind, NameKey: nameKey,
							})
						})
					} else if from != "" && from != key {
						cands = append(cands, CallCandidate{
							From: from, To: key, Path: path, Line: line,
						})
					}
				}
				if b := jomini.BlockOf(a.Value); b != nil {
					fr := edgeFrame{key: a.Key.Text}
					if strings.EqualFold(a.Key.Text, "option") {
						fr.nameKey = optionNameKey(a.Value)
					}
					stack = append(stack, fr)
					walk(b.Statements)
					stack = stack[:len(stack)-1]
				}
				continue
			}
			if vs, ok := st.(*jomini.ValueStmt); ok {
				if b := jomini.BlockOf(vs.Value); b != nil {
					walk(b.Statements)
				}
			}
		}
	}
	walk(root.Statements)
	return refs, edges, cands
}

func fireSiteKind(stack []edgeFrame, fireKey string) (kind, nameKey string) {
	for i := len(stack) - 1; i >= 0; i-- {
		if k, ok := classifyFireSite(stack[i].key); ok {
			if k == "option" {
				return k, stack[i].nameKey
			}
			return k, ""
		}
	}
	if k, ok := classifyFireSite(fireKey); ok {
		return k, ""
	}
	return "effect", ""
}

func classifyFireSite(key string) (kind string, ok bool) {
	switch strings.ToLower(key) {
	case "option":
		return "option", true
	case "immediate", "after":
		return "immediate", true
	case "trigger":
		return "trigger", true
	case "on_action", "on_actions":
		return "on_action", true
	case "events", "random_events", "trigger_event", "first_valid", "fallback":
		return "events", true
	case "effect":
		return "effect", true
	default:
		return "", false
	}
}

func optionNameKey(v jomini.Value) string {
	b := jomini.BlockOf(v)
	if b == nil {
		return ""
	}
	for _, st := range b.Statements {
		a, ok := st.(*jomini.Assignment)
		if ok && strings.EqualFold(a.Key.Text, "name") {
			if sc, ok := a.Value.(*jomini.Scalar); ok {
				return sc.Text
			}
		}
	}
	return ""
}

func walkFireTargets(v jomini.Value, fn func(name string, start, end int)) {
	switch t := v.(type) {
	case *jomini.Scalar:
		if !t.Quoted && fireTargetOK(t.Text) {
			fn(t.Text, t.Range.Start, t.Range.End)
		}
	case *jomini.Block, *jomini.TaggedBlock:
		b := jomini.BlockOf(v)
		if b == nil {
			return
		}
		for _, st := range b.Statements {
			switch n := st.(type) {
			case *jomini.ValueStmt:
				if sc, ok := n.Value.(*jomini.Scalar); ok && !sc.Quoted && fireTargetOK(sc.Text) {
					fn(sc.Text, sc.Range.Start, sc.Range.End)
				} else if jomini.BlockOf(n.Value) != nil {
					walkFireTargets(n.Value, fn)
				}
			case *jomini.Assignment:
				key := strings.ToLower(n.Key.Text)
				own := game.FireKind(n.Key.Text)
				if sc, ok := n.Value.(*jomini.Scalar); ok && !sc.Quoted && fireTargetOK(sc.Text) &&
					(key == "id" || isDigitKey(key) || own != "") {
					fn(sc.Text, sc.Range.Start, sc.Range.End)
					continue
				}
				if own != "" || key == "id" || isDigitKey(key) {
					walkFireTargets(n.Value, fn)
				}
			}
		}
	}
}

func fireTargetOK(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	if c < 'A' || (c > 'Z' && c < 'a') || c > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c = s[i]
		if c != '_' && c != '.' && c != '-' &&
			(c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return true
}

func isDigitKey(s string) bool {
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil
}

// EffectSet is every scripted_effect key in defs plus cache.
func EffectSet(defs []Def, cache *VanillaCache) map[string]bool {
	out := map[string]bool{}
	addEffectDefs(out, defs)
	if cache != nil {
		addEffectDefs(out, cache.Defs)
	}
	return out
}

func addEffectDefs(set map[string]bool, defs []Def) {
	for _, d := range defs {
		if isScriptedEffect(d.Type) {
			set[d.Key] = true
		}
	}
}

func isScriptedEffect(t string) bool {
	return game.CanonicalKind(t) == "scripted_effect"
}

// ApplyCallEdges turns candidates into call edges using a complete effectSet.
func ApplyCallEdges(cands []CallCandidate, effectSet map[string]bool) []Edge {
	if len(cands) == 0 || len(effectSet) == 0 {
		return nil
	}
	var out []Edge
	for _, c := range cands {
		if effectSet[c.To] && c.From != c.To {
			out = append(out, Edge{
				From: c.From, To: c.To, Path: c.Path, Line: c.Line, Kind: EdgeKindCall,
			})
		}
	}
	return out
}

const maxViaHops = 3

// DeriveVia extends stored edges with indirect event hops through scripted effects.
func DeriveVia(stored []Edge, cache *VanillaCache, effectSet map[string]bool, mods []Def) []Edge {
	kinds := map[string]bool{}
	addKinds := func(defs []Def) {
		for _, d := range defs {
			switch game.CanonicalKind(d.Type) {
			case "event", "on_action", "decision":
				kinds[d.Key] = true
			}
		}
	}
	addKinds(mods)
	if cache != nil {
		stored = append(stored, cache.Edges...)
		addKinds(cache.Defs)
	}
	calls := map[string]map[string]bool{}
	fires := map[string][]Edge{}
	direct := map[string]bool{}
	for _, e := range stored {
		switch {
		case e.Kind == EdgeKindCall:
			if calls[e.From] == nil {
				calls[e.From] = map[string]bool{}
			}
			calls[e.From][e.To] = true
		case effectSet[e.From]:
			fires[e.From] = append(fires[e.From], e)
		default:
			direct[e.From+"→"+e.To] = true
		}
	}
	var via []Edge
	for from, hop := range calls {
		if !kinds[from] {
			continue
		}
		visited := map[string]bool{}
		type step struct {
			eff   string
			chain []string
		}
		var q []step
		for eff := range hop {
			visited[eff] = true
			q = append(q, step{eff, []string{eff}})
		}
		for n := 0; n < maxViaHops && len(q) > 0; n++ {
			var next []step
			for _, st := range q {
				for _, fired := range fires[st.eff] {
					if fired.To == from || direct[from+"→"+fired.To] {
						continue
					}
					direct[from+"→"+fired.To] = true
					via = append(via, Edge{
						From: from, To: fired.To, Via: fired.Via,
						Path: fired.Path, Line: fired.Line, Kind: "via",
						NameKey: strings.Join(st.chain, " → "),
					})
				}
				for deeper := range calls[st.eff] {
					if visited[deeper] {
						continue
					}
					visited[deeper] = true
					next = append(next, step{deeper, append(append([]string{}, st.chain...), deeper)})
				}
			}
			q = next
		}
	}
	return via
}

// Harvest is the in-memory result of indexing a workspace's mods.
type Harvest struct {
	Defs  []Def
	Refs  []Ref
	Edges []Edge
	Loc   map[string]map[string]LocEntry
	Order []string
}

func ingestFile(gameID string, f fileRef, harvestBodies bool) FileExtract {
	raw, err := os.ReadFile(f.abs)
	if err != nil {
		return FileExtract{}
	}
	text, _ := jomini.Decode(raw)
	return ExtractFile(gameID, f.abs, f.rel, f.origin, text, harvestBodies)
}

// BuildIndex walks every mod in load order and harvests its files.
func BuildIndex(gameID string, mods []ModInput, cache *VanillaCache) Harvest {
	ordered := append([]ModInput{}, mods...)
	slices.SortStableFunc(ordered, func(a, b ModInput) int { return cmp.Compare(a.Order, b.Order) })
	var files []fileRef
	var order []string
	for _, m := range ordered {
		order = append(order, m.Origin)
		for _, f := range modFiles(m.Root) {
			f.origin = m.Origin
			files = append(files, f)
		}
	}
	acc, _ := collectExtracts(context.Background(), gameID, files, false, cache)
	return Harvest{Defs: acc.defs, Refs: acc.refs, Edges: acc.edges, Loc: acc.loc, Order: lo.Uniq(order)}
}

// MergeLoc copies locd into dst, allocating maps as needed.
func MergeLoc(dst map[string]map[string]LocEntry, locd LocDelta) map[string]map[string]LocEntry {
	if locd.Lang == "" || len(locd.Vals) == 0 {
		return dst
	}
	if dst == nil {
		dst = map[string]map[string]LocEntry{}
	}
	m := dst[locd.Lang]
	if m == nil {
		m = map[string]LocEntry{}
		dst[locd.Lang] = m
	}
	for k, v := range locd.Vals {
		m[k] = v
	}
	return dst
}

func modFiles(root string) []fileRef {
	var out []fileRef
	walkClassified(root, func(kind string, f fileRef) {
		switch kind {
		case "mod", "gui", "loc", "script":
			out = append(out, f)
		}
	})
	return out
}

func enrichScriptDocs(dir, format string, c *VanillaCache) {
	if dir == "" {
		return
	}
	entries := parseScriptDocs(dir, format)
	if len(entries) == 0 {
		return
	}
	effects, triggers, vocab := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, v := range c.Vocabulary {
		vocab[v] = true
	}
	if c.FieldDocs == nil {
		c.FieldDocs = map[string]string{}
	}
	for _, e := range entries {
		vocab[e.name] = true
		if e.doc != "" && c.FieldDocs[e.name] == "" {
			c.FieldDocs[e.name] = e.doc
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

type docToken struct{ name, kind, doc string }

func parseScriptDocs(dir, format string) []docToken {
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
		raw, err := os.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			continue
		}
		kind := kindFromDocFilename(name)
		switch {
		case format == "markdown" && strings.HasSuffix(name, ".md"):
			out = append(out, parseMarkdownDocs(string(raw), kind)...)
		case format != "markdown" && strings.HasSuffix(name, ".log"):
			out = append(out, parseClassicDocs(string(raw), kind)...)
		}
	}
	return out
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

func parseMarkdownDocs(text, kind string) []docToken {
	var out []docToken
	var cur *docToken
	var body []string
	flush := func() {
		if cur == nil {
			return
		}
		cur.doc = prose(body)
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
		cur.doc = prose(parts)
		out = append(out, *cur)
		cur, parts = nil, nil
	}
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(trimmed, "----"):
			flush()
		case trimmed == "":
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
