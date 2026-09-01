// loccoverage.go reports missing, orphan, and untranslated loc keys per
// language from the Index loc maps plus vanilla sidecar inherit (capped at 500).

package views

import (
	"cmp"
	"slices"
	"strings"

	"paradox-modding-tools/services/internal/catalog"
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
			entries[k] = locSite{key: k, value: v.Value, file: v.File, line: v.Line}
		}
		byLang[lang] = entries
	}
	if len(byLang) == 0 {
		return nil
	}

	referenced := map[string]locSite{}
	for _, r := range s.LocRefs() {
		if _, _, ok := s.Locate(r.Path); !ok {
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

	_, _, ver, installID := s.CacheInfo()
	if ver == "" {
		ver = "latest"
	}

	out := make([]LocCoverage, 0, len(langs))
	for _, lang := range langs {
		inheritedKeys := sidecarKeys(s, installID, ver, lang)
		entries := byLang[lang]
		row := LocCoverage{Language: lang, Defined: len(entries)}
		addIssue := func(kind string, site locSite, value string) {
			if len(row.Issues) >= issueCap {
				return
			}
			origin, _, _ := s.Locate(site.file)
			row.Issues = append(row.Issues, LocIssueRow{
				Language: lang, Kind: kind, Key: site.key,
				File: site.file, Rel: s.DisplayRel(site.file),
				Line: site.line, Value: value, Origin: origin,
				OriginName: s.OriginName(origin),
			})
		}
		for key, site := range referenced {
			if _, ok := entries[key]; ok || inheritedKeys[key] {
				continue
			}
			addIssue("missing", site, "")
		}
		for key, site := range entries {
			if _, ok := referenced[key]; ok || inheritedKeys[key] {
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

func sidecarKeys(s *session.Session, installID, ver, lang string) map[string]bool {
	out := map[string]bool{}
	if lang == s.DefaultLang() {
		for _, k := range s.LocKeys(true) {
			out[k] = true
		}
	}
	if installID == "" {
		return out
	}
	vl, err := catalog.LoadVanillaLoc(installID, ver, lang)
	if err != nil || vl == nil {
		return out
	}
	for k := range vl.Sites {
		out[k] = true
	}
	return out
}

// Lookup returns the default-lang loc text and winning site for key, or nil.
func Lookup(s *session.Session, key string) *LocLookup {
	if key == "" {
		return nil
	}
	text := locValue(s, key)
	hit := &LocLookup{Key: key, Text: text}
	if file, line, origin, ok := s.LocSite(key); ok {
		hit.File, hit.Line, hit.Origin = file, line, origin
		hit.OriginName = s.OriginName(origin)
	}
	if hit.Text == "" && hit.File == "" {
		return nil
	}
	return hit
}
