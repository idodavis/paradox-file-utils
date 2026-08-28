// eventdetail.go is the read-only inspector for one event: sections, options,
// loc text, capped script lines, targets, refs, and simulation order. It does
// not expose editor insertion fields.

package graph

import (
	"strconv"
	"strings"

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
	detail.SimSteps = simSteps(detail)
	return detail
}

func locField(s *session.Session, a *parser.Assignment) *EventLocField {
	sc, ok := a.Value.(*parser.Scalar)
	if !ok {
		return &EventLocField{Dynamic: true}
	}
	f := &EventLocField{Key: sc.Text, Text: locValue(s, sc.Text)}
	if d := lookupDef(s, sc.Text); d != nil && d.Type == "loc_key" && d.Origin != "" {
		f.File = d.Path
		f.Line = d.Line
	} else if idx := s.Index(); idx != nil {
		for _, d := range idx.Defs {
			if d.Type == "loc_key" && d.Key == sc.Text {
				f.File, f.Line = d.Path, d.Line
				break
			}
		}
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
						add(inherited, sc.Text, sc.Range.Start, fireKind(inherited))
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
			own := fireKind(a.Key.Text)
			if own != "" {
				if sc, ok := a.Value.(*parser.Scalar); ok && !sc.Quoted {
					add(key, sc.Text, sc.Range.Start, own)
					continue
				}
			}
			if inherited != "" {
				if sc, ok := a.Value.(*parser.Scalar); ok && !sc.Quoted &&
					(isDigits(key) || key == "id") {
					add(inherited, sc.Text, sc.Range.Start, fireKind(inherited))
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
		if d := lookupDef(s, name); d != nil {
			ref.DefFile, ref.DefLine = d.Path, d.Line
		}
		refs[mk] = ref
	}
	scan := func(text string, off int, isKey bool) {
		if m := scopePrefix.FindStringSubmatch(text); m != nil {
			bare := strings.Split(m[2], ".")[0]
			if m[1] == "scope" {
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

func simSteps(detail *EventDetail) []SimStep {
	steps := []SimStep{}
	named := func(name string) *EventSectionInfo {
		for i := range detail.Sections {
			if strings.EqualFold(detail.Sections[i].Name, name) {
				return &detail.Sections[i]
			}
		}
		return nil
	}
	push := func(name, kind, title, absent string) {
		sec := named(name)
		if sec == nil {
			if absent != "" {
				steps = append(steps, SimStep{
					Kind: kind, Title: title, Line: detail.Line, Note: absent,
				})
			}
			return
		}
		note := ""
		if sec.TotalLines == 0 {
			note = "(empty block)"
		}
		steps = append(steps, SimStep{
			Kind: kind, Title: title, Line: sec.Line, Note: note,
			Lines: sec.Lines, Hidden: max(0, sec.TotalLines-len(sec.Lines)),
			Targets: sec.Targets, HiddenTargets: max(0, sec.TargetsTotal-len(sec.Targets)),
		})
	}
	push("trigger", "trigger", "TRIGGER", "(no trigger: fires whenever it is called)")
	push("cancellation_trigger", "cancellation_trigger", "CANCELLATION TRIGGER", "")
	push("on_trigger_fail", "on_trigger_fail", "ON TRIGGER FAIL", "")
	push("immediate", "immediate", "IMMEDIATE", "")
	for i, opt := range detail.Options {
		label := string(rune('A' + i))
		if i >= 26 {
			label = "#" + strconv.Itoa(i+1)
		}
		sub := "(unnamed option)"
		if opt.Name != nil {
			switch {
			case opt.Name.Dynamic:
				sub = "(dynamic name, resolved in game)"
			case opt.Name.Text != "":
				sub = opt.Name.Text
			case opt.Name.Key != "":
				sub = opt.Name.Key + " (no localization)"
			}
		}
		note := ""
		if opt.TotalLines == 0 {
			note = "(no effects: the option only closes the event)"
		}
		steps = append(steps, SimStep{
			Kind: "option", Title: "OPTION " + label, Subtitle: sub,
			Line: opt.Line, Note: note, Lines: opt.Lines,
			Hidden:  max(0, opt.TotalLines-len(opt.Lines)),
			Targets: opt.Targets, HiddenTargets: max(0, opt.TargetsTotal-len(opt.Targets)),
		})
	}
	push("after", "after", "AFTER", "")
	known := map[string]bool{
		"trigger": true, "cancellation_trigger": true, "on_trigger_fail": true,
		"immediate": true, "after": true,
	}
	for _, sec := range detail.Sections {
		if known[strings.ToLower(sec.Name)] {
			continue
		}
		note := ""
		if sec.TotalLines == 0 {
			note = "(empty block)"
		}
		steps = append(steps, SimStep{
			Kind: "other", Title: strings.ToUpper(sec.Name), Line: sec.Line, Note: note,
			Lines: sec.Lines, Hidden: max(0, sec.TotalLines-len(sec.Lines)),
			Targets: sec.Targets, HiddenTargets: max(0, sec.TargetsTotal-len(sec.Targets)),
		})
	}
	return steps
}
