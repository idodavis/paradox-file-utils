// overrides.go maps catalog contests onto Wails OverrideRow DTOs with Rel/OriginName.

package views

import (
	"cmp"
	"slices"
	"strconv"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/session"
)

// OverrideSite is one place a key is defined, for the conflict monitor.
type OverrideSite struct {
	Origin     string `json:"origin"`
	OriginName string `json:"originName,omitempty"`
	Path       string `json:"path"`
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
		sites := overrideSites(s, c.Defs)
		out = append(out, OverrideRow{
			Kind:       c.Kind,
			Name:       c.Name,
			Rule:       c.Rule,
			Winner:     originID(c.Winner),
			WinnerName: s.OriginName(originID(c.Winner)),
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

// overrideSites is one OverrideSite per origin+path+line. Same-line twins from
// a DidOpen path-key miss are dropped; distinct lines stay (two real blocks).
func overrideSites(s *session.Session, defs []catalog.Def) []OverrideSite {
	seen := make(map[string]bool, len(defs))
	sites := make([]OverrideSite, 0, len(defs))
	for _, d := range defs {
		o := originID(d.Origin)
		id := o + "\x00" + session.CanonPath(d.Path) + "\x00" + strconv.Itoa(d.Line)
		if seen[id] {
			continue
		}
		seen[id] = true
		sites = append(sites, OverrideSite{
			Origin:     o,
			OriginName: s.OriginName(o),
			Path:       d.Path,
			Rel:        s.DisplayRel(d.Path),
			Line:       d.Line,
		})
	}
	return sites
}
