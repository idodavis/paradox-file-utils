// complete.go serves completion for script, loc, mod, and metadata files.
package lsp

import (
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"
	"paradox-modding-tools/services/internal/session"
)

const maxComplete = 80

// Complete returns completion items at (line, UTF-8 column).
func Complete(s *session.Session, path string, line, col int) []CompletionItem {
	if s.KindFor(path) == "mod" {
		return prefixFilter(modItems(), completePrefix(s, path, line, col))
	}
	if isMetaFile(s, path) {
		return prefixFilter(metaItems(s), completePrefix(s, path, line, col))
	}
	if s.KindFor(path) == "loc" {
		return capComplete(locItems(s, completePrefix(s, path, line, col)))
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
	return capComplete(items)
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
	return CompletionItem{Label: label, Detail: detail}
}

func documented(s *session.Session, label, detail, kind string) CompletionItem {
	return CompletionItem{
		Label: label, Detail: detail,
		Documentation: s.FieldDoc(label, kind),
	}
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

func locItems(s *session.Session, prefix string) []CompletionItem {
	if prefix == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(k string) {
		if len(out) >= maxComplete || k == "" || seen[k] || !lowerPrefix(k, prefix) {
			return
		}
		seen[k] = true
		out = append(out, documented(s, k, "loc key", ""))
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
		return valueItems(s, at.slotKey, prefix)
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

func valueItems(s *session.Session, parentKey, prefix string) []CompletionItem {
	if fk := game.FireKind(parentKey); fk != "" {
		return defsOfType(s, prefix, fk)
	}
	if loc.Classify(parentKey) != loc.PropNone {
		return locItems(s, prefix)
	}
	return nil
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
		out = append(out, documented(s, d.Key, d.Type, d.Type))
		if len(out) >= maxComplete {
			break
		}
	}
	return out
}
