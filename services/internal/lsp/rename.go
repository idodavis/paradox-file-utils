// rename.go builds a workspace edit that replaces every indexed def/ref of the
// identifier at pos. Vanilla files are never edited.

package lsp

import "paradox-modding-tools/services/internal/session"

// Rename returns edits to rename the identifier at pos, or nil if none.
func Rename(s *session.Session, path string, line, col int, newName string) *WorkspaceEdit {
	if newName == "" {
		return nil
	}
	locs := References(s, path, line, col)
	if len(locs) == 0 {
		return nil
	}
	changes := map[string][]TextEdit{}
	for _, loc := range locs {
		if _, _, ok := s.Locate(loc.URI); !ok {
			continue // never edit vanilla / files outside workspace mods
		}
		changes[loc.URI] = append(changes[loc.URI], TextEdit{
			Range: loc.Range, NewText: newName,
		})
	}
	if len(changes) == 0 {
		return nil
	}
	return &WorkspaceEdit{Changes: changes}
}
