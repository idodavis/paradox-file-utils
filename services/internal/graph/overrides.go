// overrides.go exposes FIOS/LIOS conflict rows. Winner selection stays in
// model.Overrides; this file only sorts for a stable UI.

package graph

import (
	"cmp"
	"slices"

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
	for i := range rows {
		for j := range rows[i].Sites {
			rows[i].Sites[j].Rel = relOf(s, rows[i].Sites[j].File)
		}
	}
	slices.SortFunc(rows, func(a, b model.OverrideRow) int {
		if c := cmp.Compare(a.Kind, b.Kind); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return rows
}
