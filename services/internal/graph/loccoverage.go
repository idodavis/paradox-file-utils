// loccoverage.go reports missing, orphan, and untranslated loc keys per
// language (capped at 500 issues each) and looks up one key's english text.

package graph

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"paradox-modding-tools/services/internal/loc"
	"paradox-modding-tools/services/internal/model"
	"paradox-modding-tools/services/internal/session"
)

const issueCap = 500

type locSite struct {
	key, value, file string
	line             int
}

// Coverage returns per-language loc health for every workspace mod.
func Coverage(s *session.Session) []LocCoverage {
	idx := s.Index()
	if idx == nil {
		return nil
	}
	byLang := map[string]map[string]locSite{}
	for _, m := range s.Mods() {
		collectModLoc(m.Root, byLang)
	}
	if len(byLang) == 0 {
		return nil
	}

	referenced := map[string]locSite{}
	for _, r := range idx.Refs {
		if r.Kind != "loc" && r.Kind != "loc-broad" {
			continue
		}
		if _, _, ok := s.Locate(r.Path); !ok {
			continue
		}
		if _, ok := referenced[r.Key]; !ok {
			referenced[r.Key] = locSite{key: r.Key, file: r.Path, line: r.Line}
		}
	}

	inherited := func(key string) bool {
		if c := s.Cache(); c != nil && c.LocEnglish != nil {
			if _, ok := c.LocEnglish[key]; ok {
				return true
			}
		}
		for _, d := range idx.Defs {
			if d.Type == "loc_key" && d.Key == key && d.Origin == "" {
				return true
			}
		}
		if c := s.Cache(); c != nil {
			for _, d := range c.Defs {
				if d.Type == "loc_key" && d.Key == key {
					return true
				}
			}
		}
		return false
	}

	source := byLang["english"]
	if source == nil {
		source = map[string]locSite{}
	}

	langs := make([]string, 0, len(byLang))
	for lang := range byLang {
		langs = append(langs, lang)
	}
	sort.Strings(langs)

	out := make([]LocCoverage, 0, len(langs))
	for _, lang := range langs {
		entries := byLang[lang]
		row := LocCoverage{Language: lang, Defined: len(entries)}
		for key, site := range referenced {
			if _, ok := entries[key]; ok || inherited(key) {
				continue
			}
			if len(row.Missing) >= issueCap {
				break
			}
			row.Missing = append(row.Missing, LocIssue{Key: key, File: site.file, Line: site.line})
		}
		for key, site := range entries {
			if _, ok := referenced[key]; ok || inherited(key) {
				continue
			}
			if len(row.Orphaned) >= issueCap {
				break
			}
			row.Orphaned = append(row.Orphaned, LocIssue{Key: key, File: site.file, Line: site.line})
		}
		if lang != "english" {
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
				if len(row.Untranslated) >= issueCap {
					break
				}
				row.Untranslated = append(row.Untranslated, LocIssue{
					Key: key, File: site.file, Line: site.line, Value: src.value,
				})
			}
		}
		sortIssues(row.Missing)
		sortIssues(row.Orphaned)
		sortIssues(row.Untranslated)
		out = append(out, row)
	}
	return out
}

func sortIssues(is []LocIssue) {
	sort.Slice(is, func(i, j int) bool { return is[i].Key < is[j].Key })
}

func collectModLoc(root string, byLang map[string]map[string]locSite) {
	locDir := filepath.Join(root, "localization")
	_ = filepath.WalkDir(locDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		lower := strings.ToLower(d.Name())
		if !strings.HasSuffix(lower, ".yml") && !strings.HasSuffix(lower, ".yaml") {
			return nil
		}
		lang := loc.LanguageFromFilename(p)
		if lang == "" {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		r := loc.Parse(string(raw))
		entries := byLang[lang]
		if entries == nil {
			entries = map[string]locSite{}
			byLang[lang] = entries
		}
		for _, e := range r.Entries {
			entries[e.Key] = locSite{key: e.Key, value: e.Value, file: p, line: e.Line}
		}
		return nil
	})
}

// Lookup returns the english loc text and winning site for key, or nil.
func Lookup(s *session.Session, key string) *LocLookup {
	if key == "" {
		return nil
	}
	text := locValue(s, key)
	hit := &LocLookup{Key: key, Text: text}
	idx := s.Index()
	var defs []model.Def
	if idx != nil {
		for _, d := range idx.Defs {
			if d.Type == "loc_key" && d.Key == key {
				defs = append(defs, d)
			}
		}
	}
	order := map[string]int{}
	if idx != nil {
		for i, o := range idx.Order {
			order[o] = i
		}
	}
	if w := model.Winner(defs, order); w != nil {
		hit.File, hit.Line, hit.Origin = w.Path, w.Line, w.Origin
	}
	if hit.Text == "" && hit.File == "" {
		return nil
	}
	return hit
}
