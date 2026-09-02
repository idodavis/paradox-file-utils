// complete.go serves completion for script, loc, mod, and metadata files.
package lsp

import (
	"strings"

	"paradox-modding-tools/services/internal/game"
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
	at, ok := resolveAt(s, path, line, col)
	if !ok {
		return nil
	}
	prefix := ""
	if at.word != "" && at.start < at.off {
		prefix = at.src[at.start:at.off]
	}
	items := scriptItems(s, path, at, prefix)
	Rank(s, at.kind, items)
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
	rg := byteRange(li, start, end)
	for i := range items {
		items[i].Range = rg
	}
	return items
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
	out := make([]CompletionItem, 0, len(game.DescriptorModKeys))
	for _, k := range game.DescriptorModKeys {
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
	for _, k := range s.LocKeys(false) {
		add(k)
		if len(out) >= maxComplete {
			return out
		}
	}
	return out
}

func scriptItems(s *session.Session, path string, at atPos, prefix string) []CompletionItem {
	if !at.inKey {
		if at.slotKey == "" {
			return rootItems(s, at.kind, prefix)
		}
		return valueItems(s, at.kind, at.slotKey, prefix)
	}
	slot := game.ScriptSlot(at.slotKey)
	switch slot {
	case "trigger":
		return vocabAndDefs(s, prefix, "trigger", "scripted_trigger")
	case "effect":
		return vocabAndDefs(s, prefix, "effect", "scripted_effect")
	default:
		return structureItems(s, path, at.kind, prefix)
	}
}

func rootItems(s *session.Session, kind, prefix string) []CompletionItem {
	var out []CompletionItem
	ck := game.CanonicalKind(kind)
	if game.IsCallKind(ck) && lowerPrefix(ck, prefix) {
		out = append(out, item(ck, kind))
	}
	out = append(out, defsOfType(s, prefix, ck)...)
	return out
}

func valueItems(s *session.Session, kind, parentKey, prefix string) []CompletionItem {
	if fk := game.FireKind(parentKey); fk != "" {
		return defsOfType(s, prefix, fk)
	}
	if loc.Classify(parentKey) != loc.PropNone {
		return locItems(s, prefix)
	}
	if k := s.FieldValueKind(parentKey); k != "" {
		return defsOfType(s, prefix, k)
	}
	if k := game.RefFieldKind(parentKey); k != "" {
		return defsOfType(s, prefix, k)
	}
	if enums := s.FieldEnums(kind, parentKey); len(enums) > 0 {
		return enumItems(enums, prefix)
	}
	return valueFallback(prefix)
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

func vocabAndDefs(s *session.Session, prefix, vocabKind, defType string) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(label, detail string) {
		if len(out) >= maxComplete || label == "" || seen[label] || !lowerPrefix(label, prefix) {
			return
		}
		seen[label] = true
		out = append(out, documented(s, label, detail, ""))
	}
	for _, k := range s.Vocab(vocabKind) {
		add(k, vocabKind)
		if len(out) >= maxComplete {
			return out
		}
	}
	for _, d := range s.FindDefs(prefix, maxComplete, true, true) {
		if game.CanonicalKind(d.Type) != defType {
			continue
		}
		add(d.Key, d.Type)
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}

func defsOfType(s *session.Session, prefix, defType string) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	for _, d := range s.FindDefs(prefix, maxComplete, true, true) {
		if game.CanonicalKind(d.Type) != defType || d.Key == "" || seen[d.Key] {
			continue
		}
		if !lowerPrefix(d.Key, prefix) {
			continue
		}
		seen[d.Key] = true
		out = append(out, documentedVal(s, d.Key, d.Type, d.Type))
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}
