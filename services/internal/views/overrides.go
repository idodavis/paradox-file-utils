// overrides.go maps catalog contests onto Wails OverrideRow DTOs with Rel/OriginName.

package views

import (
	"cmp"
	"slices"

	"paradox-modding-tools/services/internal/session"
)

// OverrideSite is one place a key is defined, for the conflict monitor.
type OverrideSite struct {
	Origin     string `json:"origin"`
	OriginName string `json:"originName,omitempty"`
	File       string `json:"file"`
	Rel        string `json:"rel,omitempty"`
	Line       int    `json:"line"`
}

// OverrideRow is one contested (kind, key): resolution rule, winning origin, and
// every site that defines it. Overlay is true when exactly one mod shadows vanilla.
type OverrideRow struct {
	Kind       string         `json:"kind"`
	Name       string         `json:"name"`
	Rule       string         `json:"rule"`
	Winner     string         `json:"winner"`
	WinnerName string         `json:"winnerName,omitempty"`
	Sites      []OverrideSite `json:"sites"`
	Overlay    bool           `json:"overlay,omitempty"`
}

// OverrideRows returns contested keys for the live session, sorted by kind then name.
func OverrideRows(s *session.Session) []OverrideRow {
	if s.DefCount() == 0 {
		return nil
	}
	order := make([]string, 0, len(s.Mods()))
	for _, m := range s.Mods() {
		order = append(order, m.Origin)
	}
	contests := s.Contests(order)
	out := make([]OverrideRow, 0, len(contests))
	for _, c := range contests {
		sites := make([]OverrideSite, 0, len(c.Defs))
		for _, d := range c.Defs {
			sites = append(sites, OverrideSite{
				Origin:     d.Origin,
				OriginName: s.OriginName(d.Origin),
				File:       d.Path,
				Rel:        s.DisplayRel(d.Path),
				Line:       d.Line,
			})
		}
		out = append(out, OverrideRow{
			Kind:       c.Kind,
			Name:       c.Name,
			Rule:       c.Rule,
			Winner:     c.Winner,
			WinnerName: s.OriginName(c.Winner),
			Sites:      sites,
			Overlay:    c.Overlay,
		})
	}
	slices.SortFunc(out, func(a, b OverrideRow) int {
		if c := cmp.Compare(a.Kind, b.Kind); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return out
}
