// symbols.go lists top-level definitions in a file and prefix-searches the
// workspace index.

package lsp

import (
	"strings"

	"paradox-modding-tools/services/internal/session"
)

const maxWorkspaceSymbols = 200

// DocumentSymbols returns definitions declared in path.
func DocumentSymbols(s *session.Session, path string) []SymbolInformation {
	idx := s.Index()
	if idx == nil {
		return nil
	}
	var out []SymbolInformation
	for _, d := range idx.Defs {
		if d.Path != path {
			continue
		}
		out = append(out, SymbolInformation{
			Name:     d.Key,
			Kind:     13, // Variable
			Location: defLocation(d),
		})
	}
	return out
}

// WorkspaceSymbols searches definitions whose keys contain query.
func WorkspaceSymbols(s *session.Session, query string) []SymbolInformation {
	idx := s.Index()
	if idx == nil {
		return nil
	}
	q := strings.ToLower(query)
	var out []SymbolInformation
	for _, d := range idx.Defs {
		if q != "" && !strings.Contains(strings.ToLower(d.Key), q) {
			continue
		}
		out = append(out, SymbolInformation{
			Name:          d.Key,
			Kind:          13,
			Location:      defLocation(d),
			ContainerName: d.Type,
		})
		if len(out) >= maxWorkspaceSymbols {
			break
		}
	}
	return out
}
