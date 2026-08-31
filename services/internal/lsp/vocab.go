// vocab.go builds vanilla member sets for completion rank and hover kinds.
package lsp

import (
	"cmp"
	"slices"
	"strings"

	"paradox-modding-tools/services/internal/session"
)

type memberSets struct {
	structKeys, effects, triggers map[string]bool
}

func memberSetsFor(s *session.Session, kind string) memberSets {
	sk, ef, tr := s.MemberSets(kind)
	return memberSets{structKeys: sk, effects: ef, triggers: tr}
}

func (m memberSets) rank(label string) int {
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

func (m memberSets) catalogKind(word string) string {
	if m.effects[word] || m.effects[strings.ToLower(word)] {
		return "effect"
	}
	if m.triggers[word] || m.triggers[strings.ToLower(word)] {
		return "trigger"
	}
	return ""
}

// Rank reorders completion items in place.
func Rank(s *session.Session, kind string, items []CompletionItem) {
	m := memberSetsFor(s, kind)
	slices.SortStableFunc(items, func(a, b CompletionItem) int {
		return cmp.Compare(m.rank(a.Label), m.rank(b.Label))
	})
}
