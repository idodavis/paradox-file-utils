// scopes.go ranks completion items; it never hides a candidate and never
// produces diagnostics. Rank prefers structure keys of the enclosing definition
// kind, then effects/triggers, then everything else.

package lsp

import (
	"sort"
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

// Rank reorders items in place. Unknown scope is a first-class outcome: items
// stay, they just sort later.
func Rank(s *session.Session, path, src string, offset int, items []CompletionItem) {
	kind := enclosingKind(s, path, src, offset)
	structSet := map[string]bool{}
	effectSet := map[string]bool{}
	if c := s.Cache(); c != nil {
		for _, k := range c.Structures[kind] {
			structSet[k] = true
		}
		for _, k := range c.Effects {
			effectSet[k] = true
		}
		for _, k := range c.Triggers {
			effectSet[k] = true
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return rankOf(items[i].Label, structSet, effectSet) < rankOf(items[j].Label, structSet, effectSet)
	})
}

func rankOf(label string, structs, effects map[string]bool) int {
	k := strings.ToLower(label)
	switch {
	case structs[k] || structs[label]:
		return 0
	case effects[k] || effects[label]:
		return 1
	default:
		return 2
	}
}

// enclosingKind is the definition kind of the outermost assignment containing
// offset, falling back to the file's extract kind.
func enclosingKind(s *session.Session, path, src string, offset int) string {
	_, rel, ok := s.Locate(path)
	if !ok {
		rel = path
	}
	rule := game.MatchExtract(s.GameID, rel).Kind
	if src == "" {
		return rule
	}
	res := parseOf(s, path, src)
	chain := parser.NodeAtOffset(res.Root, offset)
	if len(chain) == 0 {
		return rule
	}
	if a, ok := chain[0].(*parser.Assignment); ok && !a.Key.Quoted {
		if d := s.Resolve(a.Key.Text); d != nil {
			return d.Type
		}
	}
	return rule
}
