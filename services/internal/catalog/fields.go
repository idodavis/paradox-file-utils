// fields.go types a field from the values it takes: `field = value` where every
// value names a def of one kind makes the field that kind, and a small closed
// set of scalars makes it an enum. Both are install-derived — no dump states
// which field holds which type — and both merge mods over vanilla.

package catalog

import (
	"maps"
	"paradox-modding-tools/services/internal/parser/jomini"
	"slices"
	"strings"
)

// fieldKindCoverage is the share of a field's values that must resolve to the
// kind before the field is typed as it.
//
// Unanimity alone is not evidence: values that resolve to nothing were skipped,
// so one coincidental hit typed the field and every other use of it then became
// a reference of that kind which could never resolve. Measured on vanilla,
// `has_cultural_parameter` was typed `ep_3` on 1 of 432 values,
// `remove_variable` was typed `achievements` on 2 of 392, and `has_game_rule`
// was typed `modifiers` on 4 of 212 — between them the largest source of false
// "dangling reference" rows in Workspace Health.
//
// The distribution is strongly bimodal, so the exact threshold is not delicate:
// CK3 types 861 fields of which 632 sit at ≥90% coverage and 18 below 5%; Vic3
// types 357 with 194 at ≥90%. Half is the honest reading of "most of what this
// field holds is a thing of that kind".
const fieldKindCoverage = 50

// deriveFieldValueKinds types a field from the definitions its values name: the
// values that resolve must agree on one kind, and enough of them must resolve.
func deriveFieldValueKinds(rhs map[string]map[string]bool, defs []Def) map[string]string {
	kindsByKey := map[string][]string{}
	for _, d := range defs {
		k := jomini.CanonicalKind(d.Kind)
		if k == "" || d.Key == "" || jomini.IsEphemeral(k) {
			continue
		}
		seen := false
		for _, have := range kindsByKey[d.Key] {
			if have == k {
				seen = true
				break
			}
		}
		if !seen {
			kindsByKey[d.Key] = append(kindsByKey[d.Key], k)
		}
	}
	out := map[string]string{}
	for field, vals := range rhs {
		// A bare number is a weight, not a field name. Typing `20` as `event`
		// made `20 = empty` inside a Vic3 gene block read as a reference to a
		// missing event.
		if isDigitKey(field) {
			continue
		}
		var kind string
		ok, conflict, hits := false, false, 0
		for v := range vals {
			kinds := kindsByKey[v]
			if len(kinds) == 0 {
				continue
			}
			if len(kinds) > 1 {
				conflict = true
				break
			}
			if !ok {
				kind, ok = kinds[0], true
			} else if kinds[0] != kind {
				conflict = true
				break
			}
			hits++
		}
		if ok && !conflict && hits*100 >= len(vals)*fieldKindCoverage {
			out[field] = kind
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// MergeFieldValueKinds overlays mods on vanilla; a type conflict drops the field.
func MergeFieldValueKinds(vanilla, mods map[string]string) map[string]string {
	if len(vanilla) == 0 && len(mods) == 0 {
		return nil
	}
	out := maps.Clone(vanilla)
	if out == nil {
		out = map[string]string{}
	}
	for k, v := range mods {
		if prev, ok := out[k]; ok && prev != v {
			delete(out, k)
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

const fieldEnumMin, fieldEnumMax = 2, 64

// deriveFieldEnums keeps unique scalar RHS per kind+field when the field did not
// resolve to a def type and the unique count is a small enum (2–64).
func deriveFieldEnums(
	byKind map[string]map[string]map[string]bool,
	typed map[string]string,
) map[string]map[string][]string {
	typedL := map[string]bool{}
	for k := range typed {
		typedL[strings.ToLower(k)] = true
	}
	var out map[string]map[string][]string
	for kind, fields := range byKind {
		for field, vals := range fields {
			if typedL[strings.ToLower(field)] {
				continue
			}
			if n := len(vals); n < fieldEnumMin || n > fieldEnumMax {
				continue
			}
			keys := make([]string, 0, len(vals))
			for v := range vals {
				keys = append(keys, v)
			}
			slices.Sort(keys)
			if out == nil {
				out = map[string]map[string][]string{}
			}
			m := out[kind]
			if m == nil {
				m = map[string][]string{}
				out[kind] = m
			}
			m[strings.ToLower(field)] = keys
		}
	}
	return out
}

// MergeFieldEnums overlays mod enums on vanilla per kind+field.
func MergeFieldEnums(
	vanilla, mods map[string]map[string][]string,
) map[string]map[string][]string {
	if len(vanilla) == 0 && len(mods) == 0 {
		return nil
	}
	out := cloneFieldEnums(vanilla)
	if out == nil {
		out = map[string]map[string][]string{}
	}
	for kind, fields := range mods {
		dst := out[kind]
		if dst == nil {
			dst = map[string][]string{}
			out[kind] = dst
		}
		for field, vals := range fields {
			dst[strings.ToLower(field)] = slices.Clone(vals)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneFieldEnums(
	in map[string]map[string][]string,
) map[string]map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]map[string][]string, len(in))
	for kind, fields := range in {
		m := make(map[string][]string, len(fields))
		for field, vals := range fields {
			m[field] = slices.Clone(vals)
		}
		out[kind] = m
	}
	return out
}
