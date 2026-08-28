// overrides.go exposes FIOS/LIOS conflict rows. Winner selection stays in
// model.Overrides; this file only sorts for a stable UI.

package graph

import (
	"sort"

	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/session"
)

// OverrideRows returns contested keys for the live session, sorted by kind then name.
func OverrideRows(s *session.Session) []model.OverrideRow {
	idx := s.Index()
	if idx == nil {
		return nil
	}
	rows := model.Overrides(idx, s.Cache())
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}
