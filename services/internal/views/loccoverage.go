// loccoverage.go reports missing, orphan, and untranslated loc keys per
// language from the Index loc maps plus vanilla sidecar inherit (capped at 500).

package views

import (
	"cmp"
	"slices"
	"strings"

	"paradox-modding-tools/services/internal/session"
)

const issueCap = 500

type locSite struct {
	key, value, file string
	line             int
}

// Coverage returns per-language loc health for every workspace mod.
func Coverage(s *session.Session) []LocCoverage {
	byLang := map[string]map[string]locSite{}
	for lang, m := range s.LocByLang() {
		entries := map[string]locSite{}
		for k, v := range m {
			entries[k] = locSite{key: k, value: v.Value, file: v.Path, line: v.Line}
		}
		byLang[lang] = entries
	}
	if len(byLang) == 0 {
		return nil
	}

	referenced := map[string]locSite{}
	used := s.UsedLocKeys()
	for _, r := range s.LocRefs() {
		if r.Kind != "loc" {
			continue
		}
		if skipLocIssueFile(s, r.Path) {
			continue
		}
		if _, ok := referenced[r.Key]; !ok {
			referenced[r.Key] = locSite{key: r.Key, file: r.Path, line: r.Line}
		}
	}

	defaultLang := s.DefaultLang()
	source := byLang[defaultLang]
	if source == nil {
		source = map[string]locSite{}
	}

	langs := make([]string, 0, len(byLang))
	for lang := range byLang {
		langs = append(langs, lang)
	}
	slices.Sort(langs)

	out := make([]LocCoverage, 0, len(langs))
	for _, lang := range langs {
		inheritedKeys := inheritKeys(s, lang)
		entries := byLang[lang]
		row := LocCoverage{Language: lang, Defined: len(entries)}
		addIssue := func(kind string, site locSite, value string) {
			if len(row.Issues) >= issueCap {
				return
			}
			origin, _, _ := s.Locate(site.file)
			origin = originID(origin)
			row.Issues = append(row.Issues, LocIssueRow{
				Language: lang, Kind: kind, Key: site.key,
				Path: site.file, Rel: s.DisplayRel(site.file),
				Line: site.line, Value: value, Origin: origin,
				OriginName: s.OriginName(origin),
			})
		}
		for key, site := range referenced {
			if locPresent(s, key, entries, inheritedKeys) {
				continue
			}
			addIssue("missing", site, "")
		}
		for key, site := range entries {
			if used[key] || inheritedKeys[key] || skipOrphanLocFile(site.file) {
				continue
			}
			addIssue("orphaned", site, site.value)
		}
		if lang != defaultLang {
			for key, site := range entries {
				src, ok := source[key]
				if !ok {
					continue
				}
				isCopy := src.value == site.value && strings.TrimSpace(site.value) != ""
				blank := strings.TrimSpace(site.value) == ""
				if !isCopy && !blank {
					continue
				}
				addIssue("untranslated", site, src.value)
			}
		}
		slices.SortFunc(row.Issues, func(a, b LocIssueRow) int {
			return cmp.Compare(a.Key, b.Key)
		})
		out = append(out, row)
	}
	return out
}

func locPresent(
	s *session.Session, key string, entries map[string]locSite, inherited map[string]bool,
) bool {
	if _, ok := entries[key]; ok || inherited[key] {
		return true
	}
	if _, ok := s.DefaultLoc(key); ok {
		return true
	}
	_, _, _, ok := s.LocSite(key)
	return ok
}

func inheritKeys(s *session.Session, lang string) map[string]bool {
	out := map[string]bool{}
	for _, k := range s.InheritedLocKeys(lang) {
		out[k] = true
	}
	return out
}

func skipLocIssueFile(s *session.Session, path string) bool {
	switch s.KindFor(path) {
	case "mod", "meta":
		return true
	default:
		return false
	}
}

func skipOrphanLocFile(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.Contains(p, "game_rules") ||
		strings.Contains(p, "game_rule") ||
		strings.Contains(p, "message_filter") ||
		strings.Contains(p, "/messages/") ||
		strings.Contains(p, "messages_l_")
}

// Lookup returns the default-lang loc text and winning site for key, or nil.
func Lookup(s *session.Session, key string) *LocLookup {
	if key == "" {
		return nil
	}
	text := locValue(s, key)
	hit := &LocLookup{Key: key, Text: text}
	if file, line, origin, ok := s.LocSite(key); ok {
		hit.Path, hit.Line, hit.Origin = file, line, originID(origin)
		hit.OriginName = s.OriginName(hit.Origin)
	}
	if hit.Text == "" && hit.Path == "" {
		return nil
	}
	return hit
}
