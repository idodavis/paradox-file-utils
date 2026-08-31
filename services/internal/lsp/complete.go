// complete.go serves completion for script, loc, mod, and metadata files.
package lsp

import (
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
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
	items := scriptItems(s, path, at.kind, prefix)
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
		out = append(out, item(k, "loc key"))
	}
	for _, k := range s.LocKeys(false) {
		add(k)
		if len(out) >= maxComplete {
			return out
		}
	}
	return out
}

func scriptItems(s *session.Session, path, kind, prefix string) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(label, detail string) {
		if len(out) >= maxComplete || label == "" || seen[label] || !lowerPrefix(label, prefix) {
			return
		}
		seen[label] = true
		out = append(out, item(label, detail))
	}
	for _, k := range s.Structures(kind) {
		add(k, kind)
	}
	for _, pair := range [][2]string{{"effect", "effect"}, {"trigger", "trigger"}, {"vocabulary", "script"}} {
		for _, k := range s.Vocab(pair[0]) {
			add(k, pair[1])
			if len(out) >= maxComplete {
				return out
			}
		}
	}
	if s.KindFor(path) == "gui" {
		for _, k := range s.Vocab("gui_type") {
			add(k, "gui type")
		}
		for _, k := range s.Vocab("gui_prop") {
			add(k, "gui")
			if len(out) >= maxComplete {
				return out
			}
		}
	}
	for _, d := range s.FindDefs(prefix, maxComplete-len(out), true, true) {
		add(d.Key, d.Type)
		if len(out) >= maxComplete {
			return out
		}
	}
	return out
}
