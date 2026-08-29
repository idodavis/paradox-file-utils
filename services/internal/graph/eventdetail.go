// eventdetail.go is the read-only inspector for one event: sections, options,
// loc text, capped script lines, targets, and refs. It does not expose editor
// insertion fields.

package graph

import (
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

const (
	maxSectionKeys = 12
	maxRefs        = 200
	maxBlockLines  = 60
	maxTargets     = 40
	maxFires       = 24
)

var (
	sectionKeys = map[string]bool{
		"trigger": true, "immediate": true, "after": true,
		"on_trigger_fail": true, "cancellation_trigger": true,
	}
	optionMetaKeys = map[string]bool{
		"name": true, "trigger": true, "ai_chance": true,
		"show_as_unavailable": true, "flag": true, "custom_tooltip": true,
		"default_option": true, "highlighted_option": true,
	}
	optionNonEffect = map[string]bool{
		"name": true, "trigger": true, "ai_chance": true, "ai_value": true,
	}
)

// Detail returns inspector content for one event id, or nil if it is not an event.
func Detail(s *session.Session, id string) *EventDetail {
	d := lookupDef(s, id)
	if d == nil || (graphKind(d.Type) != "event" && d.Type != "event") {
		return nil
	}
	res := parseOf(s, d.Path)
	if res.Root == nil {
		return nil
	}
	li := res.Lines()
	lineOf := func(off int) int { return li.PositionAt(off).Line }
	var stmt *parser.Assignment
	for _, st := range res.Root.Statements {
		a, ok := st.(*parser.Assignment)
		if ok && a.Key.Text == id && blockOf(a.Value) != nil {
			stmt = a
			break
		}
	}
	if stmt == nil {
		return nil
	}
	block := blockOf(stmt.Value)
	detail := &EventDetail{
		ID:     id,
		File:   d.Path,
		Rel:    relOf(s, d.Path),
		Origin: d.Origin,
		Line:   lineOf(stmt.Key.Range.Start),
		Fields: scalarFields(block, lineOf),
	}
	if block.CloseBrace >= 0 {
		detail.EndLine = lineOf(block.CloseBrace)
	} else {
		detail.EndLine = lineOf(block.Range.End)
	}
	for _, ch := range block.Statements {
		a, ok := ch.(*parser.Assignment)
		if !ok {
			continue
		}
		key := strings.ToLower(a.Key.Text)
		scalar, _ := a.Value.(*parser.Scalar)
		sub := blockOf(a.Value)
		switch {
		case key == "type" && scalar != nil:
			detail.Type = scalar.Text
		case key == "hidden" && scalar != nil:
			detail.Hidden = scalar.Text == "yes"
		case key == "theme" && scalar != nil:
			detail.Theme = scalar.Text
		case key == "title":
			detail.Title = locField(s, a)
		case key == "desc":
			detail.Desc = locField(s, a)
		case key == "flavor":
			detail.Flavor = locField(s, a)
		case sectionKeys[key] && sub != nil:
			detail.Sections = append(detail.Sections, sectionOf(s, a.Key.Text, sub, lineOf))
		case key == "option" && sub != nil:
			detail.Options = append(detail.Options, optionOf(s, sub, lineOf))
		}
	}
	detail.Refs = collectRefs(s, id, block, lineOf)
	return detail
}

func locField(s *session.Session, a *parser.Assignment) *EventLocField {
	sc, ok := a.Value.(*parser.Scalar)
	if !ok {
		return &EventLocField{Dynamic: true}
	}
	f := &EventLocField{Key: sc.Text, Text: locValue(s, sc.Text)}
	if hit := Lookup(s, sc.Text); hit != nil && hit.File != "" {
		f.File, f.Line = hit.File, hit.Line
	}
	return f
}

func scalarFields(block *parser.Block, lineOf func(int) int) []EventFieldInfo {
	byKey := map[string]EventFieldInfo{}
	for _, st := range block.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok || a.Key.Quoted {
			continue
		}
		sc, ok := a.Value.(*parser.Scalar)
		if !ok {
			continue
		}
		key := strings.ToLower(a.Key.Text)
		if key == "type" || key == "title" || key == "desc" || key == "flavor" ||
			key == "hidden" || key == "theme" {
			continue
		}
		byKey[a.Key.Text] = EventFieldInfo{
			Key: a.Key.Text, Value: sc.Text,
			Line: lineOf(a.Key.Range.Start), Quoted: sc.Quoted,
		}
	}
	out := make([]EventFieldInfo, 0, len(byKey))
	for _, f := range byKey {
		out = append(out, f)
	}
	// Stable by line.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Line < out[i].Line {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func sectionOf(s *session.Session, name string, block *parser.Block, lineOf func(int) int) EventSectionInfo {
	keys := []string{}
	seen := map[string]bool{}
	for _, st := range block.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok || seen[a.Key.Text] {
			continue
		}
		seen[a.Key.Text] = true
		if len(keys) < maxSectionKeys {
			keys = append(keys, a.Key.Text)
		}
	}
	rendered := renderBlock(block, lineOf, nil)
	tg := collectTargets(s, block, lineOf, true)
	return EventSectionInfo{
		Name: name, Line: lineOf(block.Range.Start), Keys: keys,
		Lines: rendered.lines, TotalLines: rendered.total,
		Targets: tg.targets, TargetsTotal: tg.total,
	}
}

func optionOf(s *session.Session, block *parser.Block, lineOf func(int) int) EventOptionInfo {
	rendered := renderBlock(block, lineOf, optionNonEffect)
	tg := collectTargets(s, block, lineOf, true)
	info := EventOptionInfo{
		Line: lineOf(block.Range.Start), Fields: scalarFields(block, lineOf),
		EffectKeys: []string{},
		Lines:      rendered.lines, TotalLines: rendered.total,
		Targets: tg.targets, TargetsTotal: tg.total,
	}
	seen := map[string]bool{}
	for _, st := range block.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok {
			continue
		}
		key := strings.ToLower(a.Key.Text)
		sub := blockOf(a.Value)
		if key == "name" {
			info.Name = locField(s, a)
			continue
		}
		if key == "trigger" {
			info.HasTrigger = true
			if sub != nil && info.Trigger == nil {
				r := renderBlock(sub, lineOf, nil)
				info.Trigger = &EventGateInfo{Line: lineOf(a.Key.Range.Start), Lines: r.lines, TotalLines: r.total}
			}
		}
		if key == "ai_chance" {
			info.HasAiChance = true
			if sub != nil && info.AiChance == nil {
				r := renderBlock(sub, lineOf, nil)
				info.AiChance = &EventGateInfo{Line: lineOf(a.Key.Range.Start), Lines: r.lines, TotalLines: r.total}
			}
		}
		if optionMetaKeys[key] || seen[a.Key.Text] {
			continue
		}
		seen[a.Key.Text] = true
		if len(info.EffectKeys) < maxSectionKeys {
			info.EffectKeys = append(info.EffectKeys, a.Key.Text)
		}
	}
	return info
}

type rendered struct {
	lines []EventScriptLine
	total int
}

func renderBlock(block *parser.Block, lineOf func(int) int, skip map[string]bool) rendered {
	var lines []EventScriptLine
	total := 0
	push := func(depth int, text string, off int) {
		total++
		if len(lines) < maxBlockLines {
			lines = append(lines, EventScriptLine{Depth: depth, Text: text, Line: lineOf(off)})
		}
	}
	var flatten func(b *parser.Block, depth int)
	flatten = func(b *parser.Block, depth int) {
		for _, st := range b.Statements {
			if vs, ok := st.(*parser.ValueStmt); ok {
				if sc, ok := vs.Value.(*parser.Scalar); ok {
					push(depth, sc.Text, sc.Range.Start)
					continue
				}
				if inner := blockOf(vs.Value); inner != nil {
					push(depth, "{", inner.Range.Start)
					flatten(inner, depth+1)
					end := inner.Range.End
					if inner.CloseBrace >= 0 {
						end = inner.CloseBrace
					}
					push(depth, "}", end)
				}
				continue
			}
			a, ok := st.(*parser.Assignment)
			if !ok {
				continue
			}
			if depth == 0 && skip[strings.ToLower(a.Key.Text)] {
				continue
			}
			head := a.Key.Text
			if a.Op != "" {
				head = a.Key.Text + " " + a.Op
			}
			if a.Value == nil {
				push(depth, head, a.Key.Range.Start)
				continue
			}
			if sc, ok := a.Value.(*parser.Scalar); ok {
				rhs := sc.Text
				if sc.Quoted {
					rhs = `"` + sc.Text + `"`
				}
				push(depth, head+" "+rhs, a.Key.Range.Start)
				continue
			}
			inner := blockOf(a.Value)
			if inner == nil {
				continue
			}
			tag := ""
			if tb, ok := a.Value.(*parser.TaggedBlock); ok {
				tag = tb.Tag.Text + " "
			}
			push(depth, head+" "+tag+"{", a.Key.Range.Start)
			flatten(inner, depth+1)
			end := inner.Range.End
			if inner.CloseBrace >= 0 {
				end = inner.CloseBrace
			}
			push(depth, "}", end)
		}
	}
	flatten(block, 0)
	return rendered{lines, total}
}

type targetList struct {
	targets []EventStepTarget
	total   int
}

func collectTargets(s *session.Session, block *parser.Block, lineOf func(int) int, resolveOA bool) targetList {
	var out []EventStepTarget
	seen := map[string]bool{}
	total := 0
	add := func(via, name string, off int, wanted string) {
		if !targetNameRe.MatchString(name) {
			return
		}
		key := via + ":" + name
		if seen[key] {
			return
		}
		seen[key] = true
		total++
		if len(out) >= maxTargets {
			return
		}
		t := EventStepTarget{Via: via, Name: name, Kind: "unknown", Line: lineOf(off)}
		if d := lookupByKind(s, name, wanted); d != nil {
			t.Kind = wanted
			t.File = d.Path
			t.DefLine = d.Line
			if resolveOA && wanted == "on_action" {
				if fired := resolveOnAction(s, d.Path, name); fired != nil {
					t.FiresTotal = fired.total
					if len(fired.targets) > maxFires {
						t.Fires = fired.targets[:maxFires]
					} else {
						t.Fires = fired.targets
					}
				}
			}
		}
		out = append(out, t)
	}
	var walk func(b *parser.Block, inherited string)
	walk = func(b *parser.Block, inherited string) {
		for _, st := range b.Statements {
			if vs, ok := st.(*parser.ValueStmt); ok {
				if inherited != "" {
					if sc, ok := vs.Value.(*parser.Scalar); ok && !sc.Quoted {
						add(inherited, sc.Text, sc.Range.Start, game.FireKind(inherited))
					}
				}
				if inner := blockOf(vs.Value); inner != nil {
					walk(inner, "")
				}
				continue
			}
			a, ok := st.(*parser.Assignment)
			if !ok {
				continue
			}
			key := ""
			if !a.Key.Quoted {
				key = strings.ToLower(a.Key.Text)
			}
			own := game.FireKind(a.Key.Text)
			if own != "" {
				if sc, ok := a.Value.(*parser.Scalar); ok && !sc.Quoted {
					add(key, sc.Text, sc.Range.Start, own)
					continue
				}
			}
			if inherited != "" {
				if sc, ok := a.Value.(*parser.Scalar); ok && !sc.Quoted &&
					(isDigits(key) || key == "id") {
					add(inherited, sc.Text, sc.Range.Start, game.FireKind(inherited))
				}
			}
			if sub := blockOf(a.Value); sub != nil {
				next := inherited
				if own != "" {
					next = key
				}
				walk(sub, next)
			}
		}
	}
	walk(block, "")
	return targetList{out, total}
}

func lookupByKind(s *session.Session, name, kind string) *struct {
	Path string
	Line int
} {
	idx := s.Index()
	if idx == nil {
		return nil
	}
	for _, d := range idx.Defs {
		if d.Key == name && (graphKind(d.Type) == kind || d.Type == kind) {
			return &struct {
				Path string
				Line int
			}{d.Path, d.Line}
		}
	}
	if c := s.Cache(); c != nil {
		for _, d := range c.Defs {
			if d.Key == name && (graphKind(d.Type) == kind || d.Type == kind) {
				return &struct {
					Path string
					Line int
				}{d.Path, d.Line}
			}
		}
	}
	return nil
}

func resolveOnAction(s *session.Session, file, name string) *targetList {
	res := parseOf(s, file)
	if res.Root == nil {
		return nil
	}
	li := res.Lines()
	for _, st := range res.Root.Statements {
		a, ok := st.(*parser.Assignment)
		if !ok || a.Key.Text != name {
			continue
		}
		if b := blockOf(a.Value); b != nil {
			tl := collectTargets(s, b, func(o int) int { return li.PositionAt(o).Line }, false)
			return &tl
		}
	}
	return nil
}

func collectRefs(s *session.Session, selfID string, block *parser.Block, lineOf func(int) int) []EventRefInfo {
	refs := map[string]EventRefInfo{}
	add := func(kind, name string, off int) {
		mk := kind + ":" + name
		if _, ok := refs[mk]; ok || len(refs) >= maxRefs {
			return
		}
		ref := EventRefInfo{Name: name, Kind: kind, Line: lineOf(off)}
		if kind != "saved_scope" {
			if d := lookupDef(s, name); d != nil {
				ref.DefFile, ref.DefLine = d.Path, d.Line
			}
		}
		refs[mk] = ref
	}
	scan := func(text string, off int, isKey bool) {
		if p, ok := model.ParsePrefixed(text); ok {
			bare := p.Name
			if model.IsSavedScopePrefix(p) {
				add("saved_scope", bare, off)
			} else {
				add("variable", bare, off)
			}
			return
		}
		if !isKey && eventIDRe.MatchString(text) && text != selfID {
			if d := lookupDef(s, text); d != nil && graphKind(d.Type) == "event" {
				add("event", text, off)
			}
			return
		}
		d := lookupDef(s, text)
		if d == nil {
			return
		}
		if isKey && isCallKind(d.Type) {
			kind := "scripted_effect"
			if strings.Contains(d.Type, "trigger") {
				kind = "scripted_trigger"
			}
			add(kind, text, off)
		} else if !isKey && (d.Type == "script_value" || d.Type == "script_values") {
			add("script_value", text, off)
		}
	}
	var walk func(b *parser.Block)
	walk = func(b *parser.Block) {
		for _, st := range b.Statements {
			if a, ok := st.(*parser.Assignment); ok {
				if !a.Key.Quoted {
					scan(a.Key.Text, a.Key.Range.Start, true)
				}
				if sc, ok := a.Value.(*parser.Scalar); ok && !sc.Quoted {
					scan(sc.Text, sc.Range.Start, false)
				}
				if sub := blockOf(a.Value); sub != nil {
					walk(sub)
				}
			} else if vs, ok := st.(*parser.ValueStmt); ok {
				if sc, ok := vs.Value.(*parser.Scalar); ok && !sc.Quoted {
					scan(sc.Text, sc.Range.Start, false)
				} else if inner := blockOf(vs.Value); inner != nil {
					walk(inner)
				}
			}
		}
	}
	walk(block)
	out := make([]EventRefInfo, 0, len(refs))
	for _, r := range refs {
		out = append(out, r)
	}
	return out
}
