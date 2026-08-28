// override.go is the ONLY place that decides which definition of a key wins when
// several declare it. Most kinds use LIOS (last mod in load order wins); a few
// (gui_type) use FIOS (first mod wins). Vanilla always loses to any mod. Overrides
// reports every contested key for the conflict monitor.

package model

import "paradox-modding-tools/services/internal/game"

// OverrideSite is one place a key is defined, for the conflict monitor.
type OverrideSite struct {
	Origin string `json:"origin"` // "" = vanilla
	File   string `json:"file"`
	Line   int    `json:"line"`
}

// OverrideRow is one contested key: its kind/name, the resolution rule, the winning
// origin, and every site that defines it.
type OverrideRow struct {
	Kind   string         `json:"kind"`
	Name   string         `json:"name"`
	Rule   string         `json:"rule"` // "FIOS" | "LIOS"
	Winner string         `json:"winner"`
	Sites  []OverrideSite `json:"sites"`
}

// Winner picks the effective definition of a single key from all its declarations.
// order maps origin -> load rank (ascending = earlier). Vanilla ("") ranks below
// every mod. Returns nil for an empty slice.
func Winner(defs []Def, order map[string]int) *Def {
	if len(defs) == 0 {
		return nil
	}
	var vanilla *Def
	var mods []Def
	for i := range defs {
		if defs[i].Origin == "" {
			vanilla = &defs[i]
		} else {
			mods = append(mods, defs[i])
		}
	}
	if len(mods) == 0 {
		return vanilla
	}
	fios := game.IsFIOS(mods[0].Type)
	best := 0
	for i := 1; i < len(mods); i++ {
		ri, rb := order[mods[i].Origin], order[mods[best].Origin]
		if fios {
			if ri < rb {
				best = i
			}
		} else if ri >= rb { // LIOS: latest wins, ties -> later in slice
			best = i
		}
	}
	w := mods[best]
	return &w
}

// Overrides returns one row per key defined by more than one origin (mods, or a mod
// plus vanilla), each resolved through Winner.
func Overrides(idx *Index, cache *Cache) []OverrideRow {
	order := orderMap(idx.Order)
	byKey := map[string][]Def{}
	for _, d := range idx.Defs {
		byKey[d.Key] = append(byKey[d.Key], d)
	}
	vanilla := map[string][]Def{}
	if cache != nil {
		for _, d := range cache.Defs {
			if _, contested := byKey[d.Key]; contested {
				vanilla[d.Key] = append(vanilla[d.Key], d)
			}
		}
	}

	var rows []OverrideRow
	for key, defs := range byKey {
		all := append(append([]Def{}, defs...), vanilla[key]...)
		if len(all) < 2 {
			continue
		}
		w := Winner(all, order)
		if w == nil {
			continue
		}
		rule := "LIOS"
		if game.IsFIOS(w.Type) {
			rule = "FIOS"
		}
		sites := make([]OverrideSite, 0, len(all))
		for _, d := range all {
			sites = append(sites, OverrideSite{Origin: d.Origin, File: d.Path, Line: d.Line})
		}
		rows = append(rows, OverrideRow{Kind: w.Type, Name: key, Rule: rule, Winner: w.Origin, Sites: sites})
	}
	return rows
}

// orderMap turns a load-order slice into origin -> rank.
func orderMap(order []string) map[string]int {
	m := make(map[string]int, len(order))
	for i, o := range order {
		m[o] = i
	}
	return m
}
