// override.go decides which definition of a key wins (LIOS, with per-game FIOS).
// Vanilla loses to any mod. Contests lists mod-vs-mod rows and vanilla overlays.

package catalog

import "paradox-modding-tools/services/internal/game"

// Contest is one contested (kind, key) without UI display fields.
type Contest struct {
	Kind     string
	Name     string
	Rule     string // "FIOS" | "LIOS"
	Winner   string
	Overlay  bool
	ModCount int
	Defs     []Def
}

// Winner picks the effective definition. order is origin → rank; vanilla always loses.
func Winner(gameID string, defs []Def, order map[string]int) *Def {
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
	fios := game.IsFIOS(gameID, mods[0].Kind)
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

func skipOverrideKind(t string) bool {
	return t == "loc_key" || t == "mod_descriptor" || t == "namespace" ||
		game.IsEphemeral(t)
}

// Contests returns override rows from workspace and vanilla defs.
func Contests(gameID string, workspace, vanilla []Def, order []string) []Contest {
	orderMap := OrderMap(order)
	by := map[string][]Def{}
	for _, d := range workspace {
		if skipOverrideKind(d.Kind) {
			continue
		}
		k := d.Kind + "\x00" + d.Key
		by[k] = append(by[k], d)
	}
	vanillaBy := map[string][]Def{}
	for _, d := range vanilla {
		if skipOverrideKind(d.Kind) {
			continue
		}
		k := d.Kind + "\x00" + d.Key
		if _, ok := by[k]; ok {
			vanillaBy[k] = append(vanillaBy[k], d)
		}
	}
	var rows []Contest
	for k, defs := range by {
		all := append(append([]Def{}, defs...), vanillaBy[k]...)
		modOrigins := map[string]bool{}
		hasVanilla := false
		for _, d := range all {
			if d.Origin == "" {
				hasVanilla = true
			} else {
				modOrigins[d.Origin] = true
			}
		}
		overlay := len(modOrigins) == 1 && hasVanilla
		if len(modOrigins) < 2 && !overlay {
			continue
		}
		w := Winner(gameID, all, orderMap)
		if w == nil {
			continue
		}
		rule := "LIOS"
		if game.IsFIOS(gameID, w.Kind) {
			rule = "FIOS"
		}
		rows = append(rows, Contest{
			Kind:     w.Kind,
			Name:     w.Key,
			Rule:     rule,
			Winner:   w.Origin,
			Overlay:  overlay,
			ModCount: len(modOrigins),
			Defs:     all,
		})
	}
	return rows
}

// OrderMap turns a load-order slice into origin -> rank.
func OrderMap(order []string) map[string]int {
	m := make(map[string]int, len(order))
	for i, o := range order {
		m[o] = i
	}
	return m
}
