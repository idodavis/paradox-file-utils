// inlay.go attaches loc-value previews, STRICT "missing loc" labels, and a
// scope hint after every_*/scope: keys.

package lsp

import (
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

const inlayKindType = 1

// InlayHints returns loc previews and scope labels for path.
func InlayHints(s *session.Session, path string) []InlayHint {
	if isLocPath(path) || isMetaJSON(path) {
		return nil
	}
	src := fileText(s, path)
	if src == "" {
		return nil
	}
	idx := s.Index()
	var out []InlayHint
	if idx != nil {
		for _, r := range idx.Refs {
			if r.Path != path {
				continue
			}
			label := ""
			if v, ok := locValue(s, r.Key); ok {
				if len(v) > 40 {
					v = v[:40] + "…"
				}
				label = v
			} else if r.Kind == "loc" {
				label = "missing loc"
			}
			if label == "" {
				continue
			}
			pos := parser.NewLineIndex(src).PositionAt(r.End)
			out = append(out, InlayHint{Line: pos.Line, Character: pos.Character, Label: label, Kind: inlayKindType})
		}
	}
	res := parseOf(s, path, src)
	parser.WalkStatements(res.Root, func(st parser.Statement) bool {
		a, ok := st.(*parser.Assignment)
		if !ok || a.Key.Quoted {
			return true
		}
		k := a.Key.Text
		if !(strings.HasPrefix(k, "every_") || strings.HasPrefix(k, "any_") ||
			strings.HasPrefix(k, "random_") || strings.HasPrefix(k, "ordered_") || k == "scope") {
			return true
		}
		scope := game.DefaultRootScope(s.GameID)
		if c := s.Cache(); c != nil && c.RootScopes != nil {
			if v := c.RootScopes[enclosingKind(s, path, src, a.Key.Range.Start)]; v != "" {
				scope = v
			}
		}
		if scope == "" {
			return true
		}
		pos := res.Lines().PositionAt(a.Key.Range.End)
		out = append(out, InlayHint{Line: pos.Line, Character: pos.Character, Label: scope, Kind: inlayKindType})
		return true
	})
	return out
}
