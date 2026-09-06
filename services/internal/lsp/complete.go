// complete.go serves completion for script, loc, mod, and metadata files.
package lsp

import (
	"cmp"
	"slices"
	"strings"

	"paradox-modding-tools/services/internal/modfile"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"
	"paradox-modding-tools/services/internal/session"
)

const maxComplete = 80

// Complete returns completion items at (line, UTF-8 column).
func Complete(s *session.Session, path string, line, col int) []CompletionItem {
	src := s.FileText(path)
	stamp := func(items []CompletionItem) []CompletionItem {
		return stampRange(src, line, col, items)
	}
	if s.KindFor(path) == "mod" {
		return stamp(prefixFilter(modItems(), completePrefix(s, path, line, col)))
	}
	if isMetaFile(s, path) {
		return stamp(prefixFilter(metaItems(s), completePrefix(s, path, line, col)))
	}
	if s.KindFor(path) == "loc" || s.KindFor(path) == "gui" {
		if items := dataFnItems(s, path, line, col); len(items) > 0 {
			return stamp(items)
		}
		if s.KindFor(path) == "loc" {
			return stamp(capComplete(locItems(s, completePrefix(s, path, line, col))))
		}
	}
	at, ok := s.CursorAt(path, line, col)
	if !ok {
		return nil
	}
	prefix := ""
	if at.Word != "" && at.Start < at.Off {
		prefix = at.Src[at.Start:at.Off]
	}
	items := scriptItems(s, path, at, prefix, s.ScopeAt(path, line, col))
	Rank(s, at.Kind, items)
	return stamp(capComplete(items))
}

func stampRange(src string, line, col int, items []CompletionItem) []CompletionItem {
	if len(items) == 0 || src == "" {
		return items
	}
	li := jomini.NewLineIndex(src)
	off := li.OffsetAt(line, col)
	_, start, end := jomini.Result{Src: src}.TokenAt(off)
	if start == end {
		start, end = off, off
	}
	start = citeStart(src, start, items)
	rg := byteRange(li, start, end)
	for i := range items {
		items[i].Range = rg
	}
	return items
}

// citeStart widens the replaced span leftwards over a `name:` the offered
// citations already contain, so accepting one replaces the whole expression.
//
// The word scan behind TokenAt stops at `:` even though the lexer does not, so
// with `culture_group:tur` typed it reports only `tur`. Replacing just that and
// inserting `culture_group:turkic_group` produced
// `culture_group:culture_group:turkic_group`.
//
// The widening is exact rather than heuristic: it happens only when the text
// immediately before the span is literally the label's own prefix plus a colon,
// so the only characters ever swallowed are ones the label puts back.
func citeStart(src string, start int, items []CompletionItem) int {
	if len(items) == 0 || start <= 0 || src[start-1] != ':' {
		return start
	}
	cite, _, ok := strings.Cut(items[0].Label, ":")
	if !ok || cite == "" {
		return start
	}
	lead := start - 1 - len(cite)
	if lead < 0 || !strings.EqualFold(src[lead:start-1], cite) {
		return start
	}
	return lead
}

func completePrefix(s *session.Session, path string, line, col int) string {
	src := s.FileText(path)
	if src == "" {
		return ""
	}
	off := jomini.NewLineIndex(src).OffsetAt(line, col)
	word, start, _ := jomini.Result{Src: src}.TokenAt(off)
	if word != "" && start < off {
		return src[start:off]
	}
	return ""
}

func prefixFilter(items []CompletionItem, prefix string) []CompletionItem {
	if prefix == "" {
		return capComplete(items)
	}
	out := make([]CompletionItem, 0, min(len(items), maxComplete))
	for _, it := range items {
		if lowerPrefix(it.Label, prefix) {
			out = append(out, it)
			if len(out) >= maxComplete {
				break
			}
		}
	}
	return out
}

func capComplete(items []CompletionItem) []CompletionItem {
	if len(items) > maxComplete {
		return items[:maxComplete]
	}
	return items
}

func item(label, detail string) CompletionItem {
	return CompletionItem{Label: label, Detail: detail, Kind: "property"}
}

func valItem(label, detail string) CompletionItem {
	return CompletionItem{Label: label, Detail: detail, Kind: "value"}
}

func documented(s *session.Session, label, detail, kind string) CompletionItem {
	it := CompletionItem{
		Label: label, Detail: detail, Kind: "property",
		Documentation: completeDoc(s, label, kind),
	}
	if kind != "" {
		if s.StructureBlock(kind, label) {
			it.InsertText = label + " = { $0 }"
		} else {
			it.InsertText = label + " = "
		}
	}
	return it
}

func documentedVal(s *session.Session, label, detail, kind string) CompletionItem {
	return CompletionItem{
		Label: label, Detail: detail, Kind: "value",
		Documentation: completeDoc(s, label, kind),
	}
}

func completeDoc(s *session.Session, label, kind string) string {
	doc := s.FieldDoc(label, kind)
	if u := s.TokenUsage(label); u != "" {
		if doc != "" {
			doc += "\n\n"
		}
		doc += "```\n" + u + "\n```"
	}
	return doc
}

func modItems() []CompletionItem {
	out := make([]CompletionItem, 0, len(modfile.DescriptorModKeys))
	for _, k := range modfile.DescriptorModKeys {
		out = append(out, item(k, "mod descriptor"))
	}
	return out
}

func metaItems(s *session.Session) []CompletionItem {
	keys := s.Vocab("meta")
	if len(keys) == 0 {
		return nil
	}
	out := make([]CompletionItem, 0, len(keys))
	for _, k := range keys {
		out = append(out, item(k, "metadata"))
	}
	return out
}

func dataFnItems(s *session.Session, path string, line, col int) []CompletionItem {
	src := s.FileText(path)
	if src == "" {
		return nil
	}
	off := jomini.NewLineIndex(src).OffsetAt(line, col)
	if off < 0 || off > len(src) {
		return nil
	}
	open := strings.LastIndex(src[:off], "[")
	if open < 0 {
		return nil
	}
	if strings.Contains(src[open:off], "]") {
		return nil
	}
	prefix := src[open+1 : off]
	var out []CompletionItem
	for _, k := range s.Vocab("datafunction") {
		if !lowerPrefix(k, prefix) {
			continue
		}
		out = append(out, documentedVal(s, k, "data function", ""))
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}

func locItems(s *session.Session, prefix string) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(k string) {
		if len(out) >= maxComplete || k == "" || seen[k] || !lowerPrefix(k, prefix) {
			return
		}
		seen[k] = true
		out = append(out, documentedVal(s, k, "loc key", ""))
	}
	for _, k := range s.LocKeys(prefix, maxComplete) {
		add(k)
	}
	return out
}

func scriptItems(
	s *session.Session, path string, at session.Cursor, prefix, scope string,
) []CompletionItem {
	if !at.InKey {
		if at.SlotKey == "" {
			return rootItems(s, at.Kind, prefix)
		}
		return valueItems(s, at.Kind, at.SlotKey, prefix)
	}
	// Prefer the inherited slot: inside `immediate = { capital_county = { … } }`
	// the effect context still holds even though capital_county is not a slot.
	slot := at.Slot
	if slot == "" {
		slot = jomini.ScriptSlot(at.SlotKey)
	}
	switch slot {
	case "trigger", "effect":
		return vocabAndCallDefs(s, prefix, slot, scope)
	default:
		return structureItems(s, path, at.Kind, prefix)
	}
}

func rootItems(s *session.Session, kind, prefix string) []CompletionItem {
	var out []CompletionItem
	ck := jomini.CanonicalKind(kind)
	if s.IsMacroKind(ck) && lowerPrefix(ck, prefix) {
		out = append(out, item(ck, kind))
	}
	out = append(out, defsOfType(s, prefix, ck)...)
	return out
}

func valueItems(s *session.Session, kind, parentKey, prefix string) []CompletionItem {
	if fk := s.FireKind(parentKey); fk != "" {
		return defsOfType(s, prefix, fk)
	}
	if loc.Classify(parentKey) != loc.PropNone {
		return locItems(s, prefix)
	}
	if r, ok := jomini.ScriptName(parentKey); ok {
		return defsOfType(s, prefix, r.Kind)
	}
	// Declared beats derived. The game states the argument type of 1,694 CK3 /
	// 3,834 Vic3 / 1,742 EU5 tokens in Supported Targets, and until now nothing
	// read it here: `has_culture_group = ` fell through to install-derived field
	// typing and, when that had too little evidence to pass its coverage floor,
	// all the way to offering `yes` and `no`.
	if items := targetItems(s, parentKey, prefix); len(items) > 0 {
		return items
	}
	if k := s.FieldValueKind(parentKey); k != "" {
		return defsOfType(s, prefix, k)
	}
	if s.IsMacroDef(parentKey) || s.IsMacroKind(parentKey) {
		return scriptParamItems(s, parentKey, prefix)
	}
	if macroCallParent(s, parentKey) {
		return scriptParamItems(s, parentKey, prefix)
	}
	if enums := s.FieldEnums(kind, parentKey); len(enums) > 0 {
		return enumItems(enums, prefix)
	}
	return valueFallback(prefix)
}

// targetItems completes the value of a token whose argument type the game
// declares, e.g. `has_culture_group = ` where EU5 states
// "Supported Targets: culture_group".
//
// Which form to offer is itself declared, not guessed. A scope type that is a
// global link requiring data is written `prefix:id`, so the whole citation is
// offered and one accept produces a valid value. A type with no such link is
// written as a bare key of the database bound to it. A type with neither — a
// value, a boolean — yields nothing and the caller falls through.
func targetItems(s *session.Session, parentKey, prefix string) []CompletionItem {
	target := s.TokenTarget(parentKey)
	if target == "" {
		return nil
	}
	cite, kind := s.CiteFormOfScope(target)
	if kind == "" {
		return nil
	}
	if cite == "" {
		return defsOfType(s, prefix, kind)
	}
	return citedDefsOfType(s, prefix, cite, kind)
}

// citedDefsOfType offers `prefix:id` values. The user may be part-way through
// either half, so the typed text is matched against the whole citation and
// against the bare id — typing `turkic` still finds
// `culture_group:turkic_group`.
func citedDefsOfType(s *session.Session, typed, cite, kind string) []CompletionItem {
	idPrefix := typed
	if i := strings.IndexByte(typed, ':'); i >= 0 {
		if !lowerPrefix(cite, typed[:i]) {
			return nil
		}
		idPrefix = typed[i+1:]
	}
	seen := map[string]bool{}
	var out []CompletionItem
	for _, d := range s.FindDefs(idPrefix, maxComplete, true, true) {
		if jomini.CanonicalKind(d.Kind) != kind || d.Key == "" {
			continue
		}
		label := cite + ":" + d.Key
		if seen[label] || !lowerPrefix(label, typed) && !lowerPrefix(d.Key, idPrefix) {
			continue
		}
		seen[label] = true
		out = append(out, documentedVal(s, label, d.Kind, d.Kind))
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}

// scriptParamItems offers the parameters of the macro being called.
//
// The owner filter must happen before the limit. Asking for the first
// maxComplete script params and then keeping this macro's returned whichever
// params map iteration reached first — across 2,536 different CK3 macros — so
// in practice the list was empty for the macro under the cursor.
func scriptParamItems(s *session.Session, macroKey, prefix string) []CompletionItem {
	var out []CompletionItem
	for _, name := range s.FindParamsOf(macroKey, maxComplete) {
		if !lowerPrefix(name, prefix) {
			continue
		}
		out = append(out, valItem(name, "script parameter"))
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}

func macroCallParent(s *session.Session, key string) bool {
	d := s.Resolve(key)
	if d == nil {
		return false
	}
	return s.IsMacroKind(d.Kind)
}

func enumItems(vals []string, prefix string) []CompletionItem {
	var out []CompletionItem
	for _, v := range vals {
		if !lowerPrefix(v, prefix) {
			continue
		}
		out = append(out, valItem(v, "value"))
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}

func valueFallback(prefix string) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(label, detail string) {
		if len(out) >= maxComplete || label == "" || seen[label] ||
			!lowerPrefix(label, prefix) {
			return
		}
		seen[label] = true
		out = append(out, valItem(label, detail))
	}
	add("yes", "bool")
	add("no", "bool")
	return out
}

func structureItems(s *session.Session, path, kind, prefix string) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(label, detail string) {
		if len(out) >= maxComplete || label == "" || seen[label] || !lowerPrefix(label, prefix) {
			return
		}
		seen[label] = true
		out = append(out, documented(s, label, detail, kind))
	}
	for _, k := range s.Structures(kind) {
		add(k, kind)
	}
	if s.KindFor(path) != "gui" {
		return out
	}
	for _, k := range s.Vocab("gui_type") {
		add(k, "gui type")
	}
	for _, k := range s.Vocab("gui_prop") {
		add(k, "gui")
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}

// vocabAndCallDefs offers the engine tokens legal in the current block, then
// the mod's own scripted_* macros. When the game's schema is loaded and the
// scope is known, tokens are filtered to those the game declares usable there:
// inside a landed_title block, character-only effects are not offered. An
// unknown scope falls back to the whole vocabulary rather than hiding anything.
func vocabAndCallDefs(s *session.Session, prefix, vocabKind, scope string) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(label, detail string) {
		if len(out) >= maxComplete || label == "" || seen[label] || !lowerPrefix(label, prefix) {
			return
		}
		seen[label] = true
		out = append(out, documented(s, label, detail, ""))
	}
	tokens := s.Vocab(vocabKind)
	if len(tokens) == 0 {
		// The effect and trigger lists come from script_docs. Without it there
		// is no engine API to narrow to, so offer what the walk harvested from
		// the corpus instead of nothing at all.
		tokens = s.Vocab("vocabulary")
	}
	if scope != "" && s.HasSchema() {
		if scoped := s.TokensInScope(scope, vocabKind, prefix, maxComplete); len(scoped) > 0 {
			tokens = scoped
		}
	}
	for _, k := range tokens {
		add(k, vocabKind)
		if len(out) >= maxComplete {
			return out
		}
	}
	for _, d := range s.FindMacroDefs(prefix, maxComplete) {
		add(d.Key, d.Kind)
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}

func defsOfType(s *session.Session, prefix, defType string) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(key, kind string) {
		if key == "" || seen[key] || !lowerPrefix(key, prefix) {
			return
		}
		seen[key] = true
		out = append(out, documentedVal(s, key, kind, kind))
	}
	if jomini.IsEphemeral(defType) {
		for _, d := range s.FindDefsOf(prefix, defType, maxComplete, true, true) {
			add(d.Key, d.Kind)
			if len(out) >= maxComplete {
				break
			}
		}
		return out
	}
	for _, d := range s.FindDefs(prefix, maxComplete, true, true) {
		if jomini.CanonicalKind(d.Kind) != defType {
			continue
		}
		add(d.Key, d.Kind)
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}

// rankSets orders completion items: the kind's own structure keys first, then
// the engine tokens legal here, then everything else.
type rankSets struct {
	structKeys, effects, triggers map[string]bool
}

func (m rankSets) rank(label string) int {
	k := strings.ToLower(label)
	switch {
	case m.structKeys[k] || m.structKeys[label]:
		return 0
	case m.effects[k] || m.effects[label] || m.triggers[k] || m.triggers[label]:
		return 1
	default:
		return 2
	}
}

// Rank reorders completion items in place.
func Rank(s *session.Session, kind string, items []CompletionItem) {
	sk, ef, tr := s.MemberSets(kind)
	m := rankSets{structKeys: sk, effects: ef, triggers: tr}
	slices.SortStableFunc(items, func(a, b CompletionItem) int {
		return cmp.Compare(m.rank(a.Label), m.rank(b.Label))
	})
}
