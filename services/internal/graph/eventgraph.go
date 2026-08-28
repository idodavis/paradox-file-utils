// eventgraph.go builds a defs-first event/on_action/decision graph: orphans
// stay visible, scripted-effect chains become `via` hops (≤3), BFS is capped
// at 400, and nodes carry no coordinates.

package graph

import (
	"sort"
	"strconv"
	"strings"

	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

const (
	defaultMaxNodes  = 400
	maxEffectHops    = 3
	maxSuggestions   = 2000
	maxCardSteps     = 8
	triggerKeysShown = 2
)

type rawEdge struct {
	from, to, via, file, label string
	line, char                 int
}

type site struct {
	file       string
	line, char int
}

type fileDef struct {
	name, kind string
	line       int
}

// Graph returns the event graph for the live session. Layout is not computed.
func Graph(s *session.Session, params EventGraphParams) EventGraph {
	idx := s.Index()
	if idx == nil {
		return EventGraph{}
	}
	maxNodes := params.MaxNodes
	if maxNodes <= 0 {
		maxNodes = defaultMaxNodes
	}

	defsByFile := map[string][]fileDef{}
	vocabulary := map[string]bool{}
	effectSet := map[string]bool{}
	addDef := func(d model.Def) {
		if isEffectKind(d.Type) {
			effectSet[d.Key] = true
		}
		gk := graphKind(d.Type)
		if gk == "" && !isEffectKind(d.Type) {
			return
		}
		if d.Origin == "" || !inFocus(s, d.Path, params.ModRoot) {
			return
		}
		key := strings.ToLower(d.Path)
		defsByFile[key] = append(defsByFile[key], fileDef{d.Key, d.Type, d.Line})
		if gk != "" {
			vocabulary[d.Key] = true
		}
	}
	for _, d := range idx.Defs {
		addDef(d)
	}
	if c := s.Cache(); c != nil {
		for _, d := range c.Defs {
			if isEffectKind(d.Type) {
				effectSet[d.Key] = true
			}
		}
	}
	for _, list := range defsByFile {
		sort.Slice(list, func(i, j int) bool { return list[i].line < list[j].line })
	}
	// Re-sort after possible duplicate appends of effect defs.
	for k, list := range defsByFile {
		seen := map[string]bool{}
		out := list[:0]
		for _, d := range list {
			id := d.name + "\x00" + strconv.Itoa(d.line)
			if seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, d)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].line < out[j].line })
		defsByFile[k] = out
	}

	containerOf := func(path string, line int) *fileDef {
		list := defsByFile[strings.ToLower(path)]
		var best *fileDef
		for i := range list {
			if list[i].line <= line {
				best = &list[i]
			} else {
				break
			}
		}
		return best
	}

	var edges []rawEdge
	direct := map[string]bool{}
	effectFires := map[string][]rawEdge{}
	effectCalls := map[string]map[string]site{}
	addCall := func(from, effect string, st site) {
		m := effectCalls[from]
		if m == nil {
			m = map[string]site{}
			effectCalls[from] = m
		}
		if _, ok := m[effect]; !ok {
			m[effect] = st
		}
	}

	parsed := map[string]parser.Result{}
	for path := range defsByFile {
		// defsByFile keys are lowercased; recover a real path from the first def.
		real := ""
		for _, d := range idx.Defs {
			if strings.EqualFold(d.Path, path) {
				real = d.Path
				break
			}
		}
		if real == "" {
			continue
		}
		res := parseOf(s, real)
		parsed[strings.ToLower(real)] = res
		if res.Root == nil {
			continue
		}
		li := res.Lines()
		parser.WalkStatements(res.Root, func(st parser.Statement) bool {
			a, ok := st.(*parser.Assignment)
			if !ok || a.Key.Quoted {
				return true
			}
			line := li.PositionAt(a.Key.Range.Start).Line
			char := li.PositionAt(a.Key.Range.Start).Character
			from := containerOf(real, line)
			if from == nil {
				return true
			}
			key := a.Key.Text
			if effectSet[key] && from.name != key {
				addCall(from.name, key, site{real, line, char})
			}
			if fk := fireKind(key); fk != "" {
				eachTarget(a.Value, fk, func(to string, start int) {
					pos := li.PositionAt(start)
					e := rawEdge{
						from: from.name, to: to, via: key,
						file: real, line: pos.Line, char: pos.Character,
					}
					if isEffectKind(from.kind) {
						effectFires[from.name] = append(effectFires[from.name], e)
					} else {
						edges = append(edges, e)
						direct[e.from+"→"+e.to] = true
					}
				})
			}
			return true
		})
	}

	for from, firstHop := range effectCalls {
		if !vocabulary[from] {
			continue
		}
		visited := map[string]bool{}
		type hop struct {
			effect string
			chain  []string
			site   site
		}
		var frontier []hop
		for effect, st := range firstHop {
			visited[effect] = true
			frontier = append(frontier, hop{effect, []string{effect}, st})
		}
		for n := 0; n < maxEffectHops && len(frontier) > 0; n++ {
			var next []hop
			for _, step := range frontier {
				for _, fired := range effectFires[step.effect] {
					if fired.to == from || direct[from+"→"+fired.to] {
						continue
					}
					direct[from+"→"+fired.to] = true
					edges = append(edges, rawEdge{
						from: from, to: fired.to, via: fired.via,
						file: step.site.file, line: step.site.line, char: step.site.char,
						label: "via " + strings.Join(step.chain, " → "),
					})
				}
				for deeper := range effectCalls[step.effect] {
					if visited[deeper] {
						continue
					}
					visited[deeper] = true
					chain := append(append([]string{}, step.chain...), deeper)
					next = append(next, hop{deeper, chain, step.site})
				}
			}
			frontier = next
		}
	}

	adj := map[string][]rawEdge{}
	for _, e := range edges {
		adj[e.from] = append(adj[e.from], e)
		adj[e.to] = append(adj[e.to], e)
	}

	selected := map[string]bool{}
	truncated := false
	if params.Root != "" {
		q := []string{params.Root}
		selected[params.Root] = true
		for len(q) > 0 && len(selected) < maxNodes {
			id := q[0]
			q = q[1:]
			for _, e := range adj[id] {
				for _, next := range []string{e.from, e.to} {
					if selected[next] {
						continue
					}
					if len(selected) >= maxNodes {
						truncated = true
						break
					}
					selected[next] = true
					q = append(q, next)
				}
			}
		}
	} else {
		inScope := func(id string) bool {
			if params.Namespace == "" {
				return true
			}
			return strings.HasPrefix(id, params.Namespace+".")
		}
		ids := map[string]bool{}
		for id := range vocabulary {
			if inScope(id) {
				ids[id] = true
			}
		}
		for _, e := range edges {
			if !inScope(e.from) && !inScope(e.to) {
				continue
			}
			ids[e.from] = true
			ids[e.to] = true
		}
		sorted := make([]string, 0, len(ids))
		for id := range ids {
			sorted = append(sorted, id)
		}
		sort.Strings(sorted)
		for _, id := range sorted {
			if len(selected) >= maxNodes {
				truncated = true
				break
			}
			selected[id] = true
		}
	}

	outEdges := make([]EventGraphEdge, 0)
	rawOf := make([]rawEdge, 0)
	seen := map[string]bool{}
	for _, e := range edges {
		if !selected[e.from] || !selected[e.to] {
			continue
		}
		key := e.from + "→" + e.to + ":" + e.via + ":" + e.label + ":" +
			strconv.Itoa(e.line) + ":" + strconv.Itoa(e.char)
		if seen[key] {
			continue
		}
		seen[key] = true
		outEdges = append(outEdges, EventGraphEdge{From: e.from, To: e.to, Via: e.via, Label: e.label})
		rawOf = append(rawOf, e)
	}
	labelEdges(s, parsed, outEdges, rawOf)

	firesCount := map[string]int{}
	for _, e := range outEdges {
		firesCount[e.From]++
	}

	factsCache := map[string]map[string]defFacts{}
	nodes := make([]EventGraphNode, 0, len(selected))
	for id := range selected {
		d := lookupDef(s, id)
		n := EventGraphNode{ID: id, Kind: "unknown", Source: "vanilla"}
		if d != nil {
			n.Kind = d.Type
			if k := graphKind(d.Type); k != "" {
				n.Kind = k
			}
			n.Source = sourceOf(d.Origin)
			n.File = d.Path
			n.Line = d.Line
		}
		n.Title = titleOf(s, id)
		if d != nil && d.Origin != "" && inFocus(s, d.Path, params.ModRoot) {
			fm := factsCache[d.Path]
			if fm == nil {
				fm = fileFacts(s, d.Path)
				factsCache[d.Path] = fm
			}
			if f, ok := fm[id]; ok {
				if n.Kind == "event" {
					n.Options = f.options
					n.Steps = resolveSteps(s, f.steps)
				}
				n.TriggerSummary = f.trigger
				if params.Themes {
					if f.background != "" {
						n.Theme = f.background
					} else {
						n.Theme = f.theme
					}
				}
			}
		}
		if c := firesCount[id]; c > 0 {
			n.Fires = c
		}
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	g := EventGraph{
		Nodes: nodes, Edges: outEdges, Truncated: truncated,
		Suggestions: suggestionsOf(vocabulary),
	}
	if len(nodes) == 0 {
		g.EmptyReason = emptyReason(s, params)
	}
	return g
}

func lookupDef(s *session.Session, key string) *model.Def {
	if d := s.Resolve(key); d != nil {
		return d
	}
	return model.LookupVanillaDef(s.Cache(), key)
}

func eachTarget(v parser.Value, kind string, fn func(name string, start int)) {
	switch t := v.(type) {
	case *parser.Scalar:
		if !t.Quoted && targetNameRe.MatchString(t.Text) {
			fn(t.Text, t.Range.Start)
		}
	case *parser.Block, *parser.TaggedBlock:
		b := blockOf(v)
		if b == nil {
			return
		}
		for _, st := range b.Statements {
			switch n := st.(type) {
			case *parser.ValueStmt:
				if sc, ok := n.Value.(*parser.Scalar); ok && !sc.Quoted &&
					targetNameRe.MatchString(sc.Text) {
					fn(sc.Text, sc.Range.Start)
				} else if inner := blockOf(n.Value); inner != nil {
					eachTarget(n.Value, kind, fn)
				}
			case *parser.Assignment:
				key := strings.ToLower(n.Key.Text)
				if sc, ok := n.Value.(*parser.Scalar); ok && !sc.Quoted &&
					targetNameRe.MatchString(sc.Text) &&
					(key == "id" || isDigits(key) || fireKind(n.Key.Text) != "") {
					fn(sc.Text, sc.Range.Start)
					continue
				}
				if fireKind(n.Key.Text) != "" || key == "id" || isDigits(key) {
					eachTarget(n.Value, kind, fn)
				}
			}
		}
	}
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			if s[i] == '.' {
				continue
			}
			return false
		}
	}
	return true
}

func labelEdges(s *session.Session, parsed map[string]parser.Result, edges []EventGraphEdge, rawOf []rawEdge) {
	for i := range edges {
		e := &edges[i]
		site := rawOf[i]
		res, ok := parsed[strings.ToLower(site.file)]
		if !ok || res.Root == nil {
			res = parseOf(s, site.file)
		}
		if res.Root == nil {
			continue
		}
		li := res.Lines()
		off := li.OffsetAt(site.line, site.char)
		hit := parser.NodeAtOffset(res.Root, off+1)
		inRandom := false
		for _, st := range hit {
			a, ok := st.(*parser.Assignment)
			if !ok {
				continue
			}
			key := strings.ToLower(a.Key.Text)
			isStep := key == "option" || key == "immediate" || key == "after"
			if isStep && e.Phase == "" {
				e.Phase = key
				ln := li.PositionAt(a.Key.Range.Start).Line
				e.FromLine = &ln
			}
			switch key {
			case "option":
				label := "option"
				if b := blockOf(a.Value); b != nil {
					for _, ch := range b.Statements {
						c, ok := ch.(*parser.Assignment)
						if !ok || !strings.EqualFold(c.Key.Text, "name") {
							continue
						}
						if sc, ok := c.Value.(*parser.Scalar); ok {
							if t := locValue(s, sc.Text); t != "" {
								label = "option: " + clip(t, 28)
							}
						}
					}
				}
				if e.Label == "" {
					e.Label = label
				}
			case "immediate", "after", "on_actions", "trigger", "effect",
				"events", "random_events", "first_valid":
				if e.Label == "" {
					e.Label = key
				}
				if e.Phase == "" && !isStep {
					e.Phase = key
				}
				if key == "random_events" {
					inRandom = true
				}
			case "trigger_event":
				if b := blockOf(a.Value); b != nil {
					e.Delay = delayOf(b)
				}
			default:
				if inRandom && isDigits(key) {
					if w, err := strconv.Atoi(strings.Split(key, ".")[0]); err == nil {
						e.Weight = &w
					}
				}
			}
		}
	}
}

func delayOf(b *parser.Block) string {
	suffix := map[string]string{"days": "d", "months": "mo", "years": "y"}
	for _, st := range b.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok {
			continue
		}
		unit := suffix[strings.ToLower(a.Key.Text)]
		if unit == "" {
			continue
		}
		if sc, ok := a.Value.(*parser.Scalar); ok {
			return sc.Text + unit
		}
		if rng := blockOf(a.Value); rng != nil {
			var ends []string
			for _, s := range rng.Statements {
				vs, ok := s.(*parser.ValueStmt)
				if !ok {
					continue
				}
				if sc, ok := vs.Value.(*parser.Scalar); ok {
					ends = append(ends, sc.Text)
				}
			}
			if len(ends) == 2 {
				return ends[0] + "–" + ends[1] + unit
			}
		}
	}
	return ""
}

type defStep struct {
	phase, nameKey string
	index, line    int
}

type defFacts struct {
	options                    int
	trigger, theme, background string
	steps                      []defStep
}

func fileFacts(s *session.Session, path string) map[string]defFacts {
	out := map[string]defFacts{}
	res := parseOf(s, path)
	if res.Root == nil {
		return out
	}
	li := res.Lines()
	for _, st := range res.Root.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok {
			continue
		}
		b := blockOf(a.Value)
		if b == nil {
			continue
		}
		f := defFacts{}
		var immediate, after *defStep
		var options []defStep
		for _, ch := range b.Statements {
			c, ok := ch.(*parser.Assignment)
			if !ok {
				continue
			}
			key := strings.ToLower(c.Key.Text)
			line := li.PositionAt(c.Key.Range.Start).Line
			switch key {
			case "option":
				ds := defStep{phase: "option", index: f.options, line: line}
				if ob := blockOf(c.Value); ob != nil {
					for _, o := range ob.Statements {
						oa, ok := o.(*parser.Assignment)
						if ok && strings.EqualFold(oa.Key.Text, "name") {
							if sc, ok := oa.Value.(*parser.Scalar); ok {
								ds.nameKey = sc.Text
							}
						}
					}
				}
				options = append(options, ds)
				f.options++
			case "immediate":
				if immediate == nil {
					immediate = &defStep{phase: "immediate", line: line}
				}
			case "after":
				if after == nil {
					after = &defStep{phase: "after", line: line}
				}
			case "theme":
				if sc, ok := c.Value.(*parser.Scalar); ok {
					f.theme = sc.Text
				}
			case "override_background":
				f.background = backgroundRef(c.Value)
			case "trigger":
				if f.trigger == "" {
					f.trigger = triggerKeys(c.Value)
				}
			}
		}
		steps := make([]defStep, 0, 2+len(options))
		if immediate != nil {
			steps = append(steps, *immediate)
		}
		steps = append(steps, options...)
		if after != nil {
			steps = append(steps, *after)
		}
		if len(steps) > maxCardSteps {
			steps = steps[:maxCardSteps]
		}
		f.steps = steps
		out[a.Key.Text] = f
	}
	return out
}

func backgroundRef(v parser.Value) string {
	if sc, ok := v.(*parser.Scalar); ok {
		return sc.Text
	}
	b := blockOf(v)
	if b == nil {
		return ""
	}
	ref := ""
	for _, st := range b.Statements {
		a, ok := st.(*parser.Assignment)
		if ok && strings.EqualFold(a.Key.Text, "reference") {
			if sc, ok := a.Value.(*parser.Scalar); ok {
				ref = sc.Text
			}
		}
	}
	return ref
}

func triggerKeys(v parser.Value) string {
	b := blockOf(v)
	if b == nil {
		return ""
	}
	var keys []string
	more := false
	for _, st := range b.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok {
			continue
		}
		if len(keys) < triggerKeysShown {
			keys = append(keys, a.Key.Text)
		} else {
			more = true
			break
		}
	}
	if len(keys) == 0 {
		return ""
	}
	s := strings.Join(keys, ", ")
	if more {
		s += "…"
	}
	return s
}

func resolveSteps(s *session.Session, steps []defStep) []EventGraphStep {
	out := make([]EventGraphStep, 0, len(steps))
	for _, st := range steps {
		row := EventGraphStep{Phase: st.phase, Line: st.line}
		if st.phase == "option" {
			i := st.index
			row.Index = &i
			if t := locValue(s, st.nameKey); t != "" {
				row.Text = clip(t, 40)
			}
		}
		out = append(out, row)
	}
	return out
}

func suggestionsOf(vocab map[string]bool) EventGraphSuggestions {
	ids := make([]string, 0, len(vocab))
	for id := range vocab {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	ns := map[string]bool{}
	for _, id := range ids {
		if i := strings.IndexByte(id, '.'); i > 0 {
			ns[id[:i]] = true
		}
	}
	namespaces := make([]string, 0, len(ns))
	for n := range ns {
		namespaces = append(namespaces, n)
	}
	sort.Strings(namespaces)
	if len(ids) > maxSuggestions {
		ids = ids[:maxSuggestions]
	}
	return EventGraphSuggestions{IDs: ids, Namespaces: namespaces}
}

func emptyReason(s *session.Session, params EventGraphParams) string {
	if params.Root == "" && params.Namespace == "" {
		return ""
	}
	match := func(name string) bool {
		if params.Root != "" {
			return name == params.Root
		}
		return strings.HasPrefix(name, params.Namespace+".")
	}
	what := params.Root
	if what == "" {
		what = "namespace " + params.Namespace
	}
	idx := s.Index()
	if idx != nil {
		for _, d := range idx.Defs {
			if graphKind(d.Type) == "" || !match(d.Key) {
				continue
			}
			if d.Origin != "" && !inFocus(s, d.Path, params.ModRoot) {
				return what + " exists in another workspace mod, outside the current focus."
			}
		}
	}
	if v := model.LookupVanillaDef(s.Cache(), params.Root); v != nil && graphKind(v.Type) != "" {
		return what + " is vanilla content. The graph shows workspace mods only."
	}
	return ""
}
