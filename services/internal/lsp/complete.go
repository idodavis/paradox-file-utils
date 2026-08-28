// complete.go prefixes-filters vocabulary and definitions. Items are never
// hidden; scopes.go only reorders them. Descriptor .mod and metadata.json have
// their own tiny key lists.

package lsp

import (
	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/session"
)

const maxComplete = 80

// Complete returns completion items at (line, UTF-8 column).
func Complete(s *session.Session, path string, line, col int) []CompletionItem {
	src := fileText(s, path)
	off := offsetOf(src, line, col)
	prefix := ""
	if _, start, _ := wordAt(src, off); start < off && start < len(src) {
		prefix = src[start:off]
	}

	if isModPath(path) {
		return prefixFilter(modItems(), prefix)
	}
	if isMetaJSON(path) {
		return prefixFilter(metaItems(s), prefix)
	}
	if isLocPath(path) {
		return prefixFilter(locItems(s), prefix)
	}

	items := scriptItems(s, path, src, off)
	Rank(s, path, src, off, items)
	return prefixFilter(items, prefix)
}

func prefixFilter(items []CompletionItem, prefix string) []CompletionItem {
	if prefix == "" {
		if len(items) > maxComplete {
			return items[:maxComplete]
		}
		return items
	}
	out := items[:0]
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

func modItems() []CompletionItem {
	out := make([]CompletionItem, 0, len(game.DescriptorModKeys))
	for _, k := range game.DescriptorModKeys {
		out = append(out, CompletionItem{Label: k, Kind: 5, Detail: "mod descriptor"})
	}
	return out
}

func metaItems(s *session.Session) []CompletionItem {
	c := s.Cache()
	if c == nil {
		return nil
	}
	out := make([]CompletionItem, 0, len(c.MetaKeys))
	for _, k := range c.MetaKeys {
		out = append(out, CompletionItem{Label: k, Kind: 5, Detail: "metadata"})
	}
	return out
}

func locItems(s *session.Session) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(k string) {
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		out = append(out, CompletionItem{Label: k, Kind: 12, Detail: "loc key"})
	}
	if idx := s.Index(); idx != nil {
		for k := range idx.Loc {
			add(k)
		}
		for _, d := range idx.Defs {
			if d.Type == "loc_key" {
				add(d.Key)
			}
		}
	}
	if c := s.Cache(); c != nil {
		for k := range c.LocEnglish {
			add(k)
		}
	}
	return out
}

func scriptItems(s *session.Session, path, src string, offset int) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(label, detail string, kind int) {
		if label == "" || seen[label] {
			return
		}
		seen[label] = true
		out = append(out, CompletionItem{Label: label, Kind: kind, Detail: detail})
	}
	c := s.Cache()
	kind := enclosingKind(s, path, src, offset)
	if c != nil {
		for _, k := range c.Structures[kind] {
			add(k, kind, 5)
		}
		for _, k := range c.Effects {
			add(k, "effect", 3)
		}
		for _, k := range c.Triggers {
			add(k, "trigger", 3)
		}
		for _, k := range c.Vocabulary {
			add(k, "script", 6)
		}
		if isGUIPath(path) {
			for _, k := range c.GUITypes {
				add(k, "gui type", 7)
			}
			for _, k := range c.GUIProps {
				add(k, "gui", 5)
			}
		}
	}
	if idx := s.Index(); idx != nil {
		for _, d := range idx.Defs {
			add(d.Key, d.Type, 12)
		}
	}
	if c != nil {
		for _, d := range c.Defs {
			add(d.Key, d.Type, 12)
		}
	}
	return out
}
