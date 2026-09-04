// extract_names.go harvests ephemeral names, script params, game-rule settings,
// and convention loc refs — kept out of extract.go.

package catalog

import (
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
)

// extractScriptNames walks assignments for ScriptName defs/refs and scope:/var: refs.
func extractScriptNames(
	gameID string, root *jomini.Root, li *jomini.LineIndex, path, origin string,
) (defs []Def, refs []Ref) {
	if root == nil {
		return nil, nil
	}
	var walk func(stmts []jomini.Statement)
	walk = func(stmts []jomini.Statement) {
		for _, st := range stmts {
			a, ok := st.(*jomini.Assignment)
			if !ok {
				if vs, ok := st.(*jomini.ValueStmt); ok {
					if b := jomini.BlockOf(vs.Value); b != nil {
						walk(b.Statements)
					}
				}
				continue
			}
			if !a.Key.Quoted {
				if r, ok := game.ScriptName(gameID, a.Key.Text); ok {
					name, start, end, found := scriptNameSpan(a, r.InnerKey)
					if found && name != "" {
						if r.IsDef {
							d := makeDef(r.Kind, name, path, origin,
								jomini.Range{Start: start, End: end}, li)
							d.Value = scriptNameValue(a, r)
							defs = append(defs, d)
						} else {
							refs = append(refs, Ref{
								Key: name, Kind: r.Kind, Path: path,
								Line:  li.PositionAt(start).Line,
								Start: start, End: end,
							})
						}
					}
				}
				if sc, ok := a.Value.(*jomini.Scalar); ok && !sc.Quoted {
					if p, ok := game.ParsePrefixed(sc.Text); ok {
						if kind := game.PrefixKind(p.Prefix); kind != "" {
							refs = append(refs, Ref{
								Key: p.Name, Kind: kind, Path: path,
								Line:  li.PositionAt(sc.Range.Start).Line,
								Start: sc.Range.Start + p.NameOff,
								End:   sc.Range.Start + p.NameOff + len(p.Name),
							})
						}
					}
				}
			}
			if b := jomini.BlockOf(a.Value); b != nil {
				walk(b.Statements)
			}
		}
	}
	walk(root.Statements)
	return defs, refs
}

// extractScriptParams harvests $NAME$ macro defs/refs and call-site TARGET = refs.
func extractScriptParams(
	_ string, root *jomini.Root, li *jomini.LineIndex, path, origin string,
	existing []Def,
) (defs []Def, refs []Ref) {
	if root == nil {
		return nil, nil
	}
	owners := scriptedMacroKeys(existing)
	ownerParams := map[string]map[string]bool{}
	for _, st := range root.Statements {
		a, ok := st.(*jomini.Assignment)
		if !ok || a.Key.Quoted || !owners[a.Key.Text] {
			continue
		}
		b := jomini.BlockOf(a.Value)
		if b == nil {
			continue
		}
		pdefs, prefs, names := harvestDollarParams(b, li, path, origin, a.Key.Text)
		defs = append(defs, pdefs...)
		refs = append(refs, prefs...)
		ownerParams[a.Key.Text] = names
	}
	var walk func(stmts []jomini.Statement, callKey string)
	walk = func(stmts []jomini.Statement, callKey string) {
		for _, st := range stmts {
			a, ok := st.(*jomini.Assignment)
			if !ok {
				if vs, ok := st.(*jomini.ValueStmt); ok {
					if b := jomini.BlockOf(vs.Value); b != nil {
						walk(b.Statements, callKey)
					}
				}
				continue
			}
			if !a.Key.Quoted {
				if callKey != "" && ownerParams[callKey][a.Key.Text] {
					refs = append(refs, Ref{
						Key: a.Key.Text, Kind: "script_param", Path: path,
						Line:  li.PositionAt(a.Key.Range.Start).Line,
						Start: a.Key.Range.Start, End: a.Key.Range.End,
						OwnerKey: callKey,
					})
				}
				nextCall := callKey
				if owners[a.Key.Text] {
					nextCall = a.Key.Text
				}
				if b := jomini.BlockOf(a.Value); b != nil {
					walk(b.Statements, nextCall)
					continue
				}
			}
			if b := jomini.BlockOf(a.Value); b != nil {
				walk(b.Statements, callKey)
			}
		}
	}
	walk(root.Statements, "")
	return defs, refs
}

func scriptedMacroKeys(defs []Def) map[string]bool {
	out := map[string]bool{}
	for _, d := range defs {
		switch game.CanonicalKind(d.Kind) {
		case "scripted_trigger", "scripted_effect", "scripted_modifier", "script_value":
			out[d.Key] = true
		}
	}
	return out
}

func harvestDollarParams(
	b *jomini.Block, li *jomini.LineIndex, path, origin, ownerKey string,
) (defs []Def, refs []Ref, names map[string]bool) {
	names = map[string]bool{}
	seen := map[string]bool{}
	var record func(name string, start, end int)
	record = func(name string, start, end int) {
		names[name] = true
		if !seen[name] {
			seen[name] = true
			defs = append(defs, Def{
				Kind: "script_param", Key: name, Path: path, Origin: origin,
				Line: li.PositionAt(start).Line, Start: start, End: end,
				OwnerKey: ownerKey,
			})
			return
		}
		refs = append(refs, Ref{
			Key: name, Kind: "script_param", Path: path,
			Line: li.PositionAt(start).Line, Start: start, End: end,
			OwnerKey: ownerKey,
		})
	}
	var walk func(stmts []jomini.Statement)
	walk = func(stmts []jomini.Statement) {
		for _, st := range stmts {
			a, ok := st.(*jomini.Assignment)
			if !ok {
				if vs, ok := st.(*jomini.ValueStmt); ok {
					if inner := jomini.BlockOf(vs.Value); inner != nil {
						walk(inner.Statements)
					}
				}
				continue
			}
			if !a.Key.Quoted {
				for _, span := range dollarParamsIn(a.Key.Text, a.Key.Range.Start) {
					record(span.name, span.start, span.end)
				}
				if sc, ok := a.Value.(*jomini.Scalar); ok && !sc.Quoted {
					for _, span := range dollarParamsIn(sc.Text, sc.Range.Start) {
						record(span.name, span.start, span.end)
					}
				}
			}
			if inner := jomini.BlockOf(a.Value); inner != nil {
				walk(inner.Statements)
			}
		}
	}
	walk(b.Statements)
	return defs, refs, names
}

type dollarSpan struct {
	name       string
	start, end int
}

func dollarParamsIn(text string, base int) []dollarSpan {
	var out []dollarSpan
	for i := 0; i < len(text); {
		if text[i] != '$' {
			i++
			continue
		}
		j := i + 1
		nameStart := j
		for j < len(text) && isParamByte(text[j], j == nameStart) {
			j++
		}
		if j == nameStart || j >= len(text) || text[j] != '$' {
			i++
			continue
		}
		out = append(out, dollarSpan{
			name:  text[nameStart:j],
			start: base + i,
			end:   base + j + 1,
		})
		i = j + 1
	}
	return out
}

func isParamByte(c byte, first bool) bool {
	if first {
		return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	}
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9')
}

// extractGameRuleSettings harvests option keys inside game_rule bodies.
func extractGameRuleSettings(
	root *jomini.Root, li *jomini.LineIndex, path, origin string,
) []Def {
	if root == nil {
		return nil
	}
	var out []Def
	for _, st := range root.Statements {
		a, ok := st.(*jomini.Assignment)
		if !ok || a.Key.Quoted {
			continue
		}
		b := jomini.BlockOf(a.Value)
		if b == nil {
			continue
		}
		for _, inner := range b.Statements {
			ia, ok := inner.(*jomini.Assignment)
			if !ok || ia.Key.Quoted || jomini.BlockOf(ia.Value) == nil {
				continue
			}
			key := strings.ToLower(ia.Key.Text)
			if key == "default" || key == "categories" || key == "flag" {
				continue
			}
			out = append(out, makeDef("game_rule_setting", ia.Key.Text, path, origin,
				ia.Key.Range, li))
		}
	}
	return out
}

// scriptNameSpan returns the name token span for a ScriptName assignment.
// Scalar RHS uses the scalar; a block uses InnerKey (flag/name) when set.
func scriptNameSpan(a *jomini.Assignment, innerKey string) (name string, start, end int, ok bool) {
	if sc, is := a.Value.(*jomini.Scalar); is && !sc.Quoted && sc.Text != "" {
		return sc.Text, sc.Range.Start, sc.Range.End, true
	}
	if innerKey == "" {
		return "", 0, 0, false
	}
	b := jomini.BlockOf(a.Value)
	if b == nil {
		return "", 0, 0, false
	}
	for _, st := range b.Statements {
		ia, is := st.(*jomini.Assignment)
		if !is || ia.Key.Quoted || ia.Key.Text != innerKey {
			continue
		}
		sc, isS := ia.Value.(*jomini.Scalar)
		if !isS || sc.Quoted || sc.Text == "" {
			return "", 0, 0, false
		}
		return sc.Text, sc.Range.Start, sc.Range.End, true
	}
	return "", 0, 0, false
}

// scriptNameValue is the hover RHS for a ScriptName def. Scalar saves
// (save_scope_as) have no value; their hover body is the ancestor scope expr.
func scriptNameValue(a *jomini.Assignment, r game.ScriptNameRule) string {
	if r.Kind == "saved_scope" && r.InnerKey == "" {
		return ""
	}
	if sc, ok := a.Value.(*jomini.Scalar); ok && !sc.Quoted {
		return "yes"
	}
	b := jomini.BlockOf(a.Value)
	if b == nil {
		return "yes"
	}
	for _, st := range b.Statements {
		ia, ok := st.(*jomini.Assignment)
		if !ok || ia.Key.Quoted || ia.Key.Text != "value" {
			continue
		}
		if sc, ok := ia.Value.(*jomini.Scalar); ok && !sc.Quoted && sc.Text != "" {
			return sc.Text
		}
	}
	return "yes"
}

func conventionLocKind(kind string) bool {
	switch game.CanonicalKind(kind) {
	case "game_rules", "game_rule", "game_rule_category",
		"messages", "message_filter_types", "message_group_types", "message":
		return true
	default:
		return false
	}
}

func conventionLocRefs(
	root *jomini.Root, li *jomini.LineIndex, path string, defs []Def, kind string,
) []Ref {
	var out []Ref
	add := func(key string, start, end int) {
		if key == "" {
			return
		}
		out = append(out, Ref{
			Key: key, Kind: "loc-convention", Path: path,
			Line: li.PositionAt(start).Line, Start: start, End: end,
		})
	}
	ck := game.CanonicalKind(kind)
	for _, d := range defs {
		switch ck {
		case "game_rules", "game_rule":
			add("rule_"+d.Key, d.Start, d.End)
		case "game_rule_category":
			add("game_rule_category_"+d.Key, d.Start, d.End)
		case "message_filter_types":
			add("message_filter_"+d.Key, d.Start, d.End)
			add("message_filter_"+d.Key+"_desc", d.Start, d.End)
		}
	}
	if (ck != "game_rules" && ck != "game_rule") || root == nil {
		return out
	}
	for _, st := range root.Statements {
		a, ok := st.(*jomini.Assignment)
		if !ok {
			continue
		}
		b := jomini.BlockOf(a.Value)
		if b == nil {
			continue
		}
		for _, inner := range b.Statements {
			ia, ok := inner.(*jomini.Assignment)
			if !ok || ia.Key.Quoted || jomini.BlockOf(ia.Value) == nil {
				continue
			}
			key := ia.Key.Text
			add("setting_"+key, ia.Key.Range.Start, ia.Key.Range.End)
			add("setting_"+key+"_desc", ia.Key.Range.Start, ia.Key.Range.End)
		}
	}
	return out
}

// dropEphemeralDefs drops flags/variables/scopes from the install cache.
// script_param stays so $NAME$ in vanilla scripted_* files can resolve.
func dropEphemeralDefs(defs []Def) []Def {
	out := make([]Def, 0, len(defs))
	for _, d := range defs {
		k := game.CanonicalKind(d.Kind)
		if game.IsEphemeral(k) && k != "script_param" {
			continue
		}
		out = append(out, d)
	}
	return out
}

func scriptNameValueSlot(gameID string, stack []edgeFrame, key string) bool {
	if len(stack) == 0 {
		return false
	}
	_, ok := game.ScriptName(gameID, stack[len(stack)-1].key)
	return ok && (key == "value" || key == "name")
}
