// complete.go prefixes-filters vocabulary and definitions. Items are never
// hidden by Rank; scopes.go only reorders them. Descriptor .mod and
// metadata.json have their own tiny key lists.

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
	// Ident before the cursor only — an empty span is not prefix src[0:off].
	if word, start, _ := wordAt(src, off); word != "" && start < off {
		prefix = src[start:off]
	}

	if isModPath(path) {
		return prefixFilter(modItems(), prefix)
	}
	if isMetaJSON(path) {
		return prefixFilter(metaItems(s), prefix)
	}
	if isLocPath(path) {
		return capComplete(locItems(s, prefix))
	}

	kind := enclosingKind(s, path, src, off)
	items := scriptItems(s, path, kind, prefix)
	Rank(s, kind, items)
	return capComplete(items)
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
		out = append(out, CompletionItem{Label: k, Kind: 12, Detail: "loc key"})
	}
	if idx := s.Index(); idx != nil {
		for k := range idx.Loc {
			add(k)
			if len(out) >= maxComplete {
				return out
			}
		}
		for _, d := range idx.Defs {
			if d.Type == "loc_key" {
				add(d.Key)
				if len(out) >= maxComplete {
					return out
				}
			}
		}
	}
	if c := s.Cache(); c != nil {
		for k := range c.LocEnglish {
			add(k)
			if len(out) >= maxComplete {
				return out
			}
		}
	}
	return out
}

func scriptItems(s *session.Session, path, kind, prefix string) []CompletionItem {
	seen := map[string]bool{}
	var out []CompletionItem
	add := func(label, detail string, k int) {
		if len(out) >= maxComplete || label == "" || seen[label] ||
			!lowerPrefix(label, prefix) {
			return
		}
		seen[label] = true
		out = append(out, CompletionItem{Label: label, Kind: k, Detail: detail})
	}
	c := s.Cache()
	if c != nil {
		for _, k := range c.Structures[kind] {
			add(k, kind, 5)
		}
		for _, k := range c.Effects {
			add(k, "effect", 3)
			if len(out) >= maxComplete {
				return out
			}
		}
		for _, k := range c.Triggers {
			add(k, "trigger", 3)
			if len(out) >= maxComplete {
				return out
			}
		}
		for _, k := range c.Vocabulary {
			add(k, "script", 6)
			if len(out) >= maxComplete {
				return out
			}
		}
		if isGUIPath(path) {
			for _, k := range c.GUITypes {
				add(k, "gui type", 7)
			}
			for _, k := range c.GUIProps {
				add(k, "gui", 5)
				if len(out) >= maxComplete {
					return out
				}
			}
		}
	}
	if idx := s.Index(); idx != nil {
		for _, d := range idx.Defs {
			if d.Type == "saved_scope" {
				continue
			}
			add(d.Key, d.Type, 12)
			if len(out) >= maxComplete {
				return out
			}
		}
	}
	if c != nil {
		for _, d := range c.Defs {
			add(d.Key, d.Type, 12)
			if len(out) >= maxComplete {
				return out
			}
		}
	}
	return out
}
