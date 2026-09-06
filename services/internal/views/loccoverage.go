// loccoverage.go reports missing, orphan, and untranslated loc keys per
// language from the loc maps plus vanilla sidecar inherit. DefaultLocLang is
// not used here (IDE/LSP only). Untranslated compares to english.

package views

import (
	"cmp"
	"slices"
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/session"
)

// locSourceLang is the game inherit / authoring language for untranslated.
const locSourceLang = "english"

const healthLocCap = 500

type locSite struct {
	key, value, file string
	line             int
}

type locGID struct{ lang, kind, name, origin string }

// Coverage returns per-language loc KPIs and grouped Health rows.
func Coverage(s *session.Session) ([]HealthLang, []HealthRow) {
	byLang := map[string]map[string]locSite{}
	for lang, m := range s.LocByLang() {
		entries := map[string]locSite{}
		for k, v := range m {
			entries[k] = locSite{key: k, value: v.Value, file: v.Path, line: v.Line}
		}
		byLang[lang] = entries
	}
	if len(byLang) == 0 {
		return nil, nil
	}

	referenced := map[string][]locSite{}
	used := s.UsedLocKeys()
	for _, r := range s.LocRefs() {
		if r.Kind != "loc" {
			continue
		}
		if skipLocIssueFile(s, r.Path) {
			continue
		}
		referenced[r.Key] = append(referenced[r.Key], locSite{
			key: r.Key, file: r.Path, line: r.Line,
		})
	}

	source := byLang[locSourceLang]
	if source == nil {
		source = map[string]locSite{}
	}

	langs := make([]string, 0, len(byLang))
	for lang := range byLang {
		langs = append(langs, lang)
	}
	slices.Sort(langs)

	groups := map[locGID]*HealthRow{}
	var order []locGID
	addIssue := func(lang, kind string, site locSite) {
		origin, _, _ := s.Locate(site.file)
		origin = originID(origin)
		os := OverrideSite{
			Origin: origin, OriginName: s.OriginName(origin),
			Path: site.file, Rel: s.DisplayRel(site.file), Line: site.line,
		}
		fillSnippet(s, &os)
		id := locGID{lang, kind, site.key, origin}
		if g := groups[id]; g != nil {
			g.Sites = append(g.Sites, os)
			g.Refs++
			return
		}
		groups[id] = &HealthRow{
			Type: kind, Name: site.key, Language: lang,
			Origin: origin, OriginName: s.OriginName(origin),
			From: origin, FromName: s.OriginName(origin),
			Rel: os.Rel, Sites: []OverrideSite{os}, Refs: 1,
		}
		order = append(order, id)
	}

	outLangs := make([]HealthLang, 0, len(langs))
	for _, lang := range langs {
		inheritedKeys := inheritKeys(s, lang)
		entries := byLang[lang]
		outLangs = append(outLangs, HealthLang{Language: lang, Defined: len(entries)})
		for key, sites := range referenced {
			if _, ok := entries[key]; ok || inheritedKeys[key] {
				continue
			}
			for _, site := range sites {
				addIssue(lang, "missing", site)
			}
		}
		for key, site := range entries {
			if used[key] || inheritedKeys[key] {
				continue
			}
			addIssue(lang, "orphaned", site)
		}
		if lang == locSourceLang {
			continue
		}
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
			addIssue(lang, "untranslated", site)
		}
	}

	byKPI := map[string]*HealthLang{}
	for i := range outLangs {
		byKPI[outLangs[i].Language] = &outLangs[i]
	}
	for _, id := range order {
		hl := byKPI[id.lang]
		if hl == nil {
			continue
		}
		switch id.kind {
		case "missing":
			hl.Missing++
		case "orphaned":
			hl.Orphaned++
		case "untranslated":
			hl.Untranslated++
		}
	}
	slices.SortFunc(order, func(a, b locGID) int {
		if c := cmp.Compare(a.lang, b.lang); c != 0 {
			return c
		}
		if c := cmp.Compare(a.name, b.name); c != 0 {
			return c
		}
		return cmp.Compare(a.kind, b.kind)
	})
	rows := make([]HealthRow, 0, len(order))
	kept := map[string]int{}
	for _, id := range order {
		if kept[id.lang] >= healthLocCap {
			continue
		}
		kept[id.lang]++
		rows = append(rows, *groups[id])
	}
	slices.SortFunc(rows, healthRowLess)
	return outLangs, rows
}

func inheritKeys(s *session.Session, lang string) map[string]bool {
	out := map[string]bool{}
	for _, k := range s.InheritedLocKeys(lang) {
		out[k] = true
	}
	return out
}

func skipLocIssueFile(s *session.Session, path string) bool {
	if origin, _, ok := s.Locate(path); ok && origin == game.OriginVanilla {
		return true
	}
	switch s.KindFor(path) {
	case "mod", "meta":
		return true
	default:
		return false
	}
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
