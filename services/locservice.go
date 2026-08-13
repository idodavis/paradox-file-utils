// Package services: LocService provides missing-loc lint and auto-localization.
package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/loc"
	parser "paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/repos"
	"paradox-modding-tools/services/internal/semantics"

	"github.com/jmoiron/sqlx"
)

// LocDiagnostic is one missing localization finding.
type LocDiagnostic struct {
	Key      string `json:"key"`
	Language string `json:"language"`
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
	Source   string `json:"source"`
}

// AutoLocResult describes where auto-loc wrote stub keys.
type AutoLocResult struct {
	Count   int    `json:"count"`
	Path    string `json:"path"`
	Preview string `json:"preview"`
}

// LocService lint and auto-loc for workspace mods.
type LocService struct {
	DB   *sqlx.DB
	repo *repos.IndexRepository
}

func (s *LocService) getRepo() *repos.IndexRepository {
	if s.repo == nil {
		s.repo = repos.NewIndexRepository(s.DB)
	}
	return s.repo
}

var locKeyShape = regexp.MustCompile(`^[A-Za-z_][\w.:-]*$`)

// LintMissingLoc finds script loc refs missing from the target language.
func (s *LocService) LintMissingLoc(workspaceID, language string) ([]LocDiagnostic, error) {
	if language == "" {
		language = "l_english"
	}
	repo := s.getRepo()
	gameID, installID, err := repo.GetWorkspaceInfo(workspaceID)
	if err != nil {
		return nil, err
	}

	modPaths, err := repo.ListModPaths(workspaceID)
	if err != nil {
		return nil, err
	}

	var locRoots []string
	for _, mp := range modPaths {
		locRoots = append(locRoots, filepath.Join(mp, "localization"))
	}
	if installID != "" {
		instPath, _, err := repo.GetInstallPath(installID)
		if err == nil {
			info := game.Get(gameID)
			if info != nil {
				for _, lr := range info.LocRoots {
					locRoots = append(locRoots, filepath.Join(instPath, lr))
				}
			}
		}
	}

	byLang, err := loc.CollectKeys(locRoots)
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for k := range byLang[language] {
		known[k] = true
	}

	boot := semantics.BootstrapFor(gameID)
	attrSeed := boot.LocAttrs()
	attrSet := map[string]bool{}
	for _, a := range attrSeed {
		attrSet[a] = true
	}

	var diags []LocDiagnostic
	seen := map[string]bool{}
	for _, mp := range modPaths {
		_ = filepath.WalkDir(mp, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".txt" {
				return nil
			}
			f, err := parser.ParseFile(path)
			if err != nil {
				return nil
			}
			collectLocRefs(f, path, attrSet, func(key string, line int, source string) {
				if !locKeyShape.MatchString(key) || known[key] {
					return
				}
				id := key + "@" + path
				if seen[id] {
					return
				}
				seen[id] = true
				diags = append(diags, LocDiagnostic{
					Key: key, Language: language, FilePath: path, Line: line, Source: source,
				})
			})
			return nil
		})
	}
	return diags, nil
}

func collectLocRefs(
	f *parser.ParadoxFile,
	path string,
	attrSet map[string]bool,
	emit func(key string, line int, source string),
) {
	var walk func(obj *parser.Object, parentKey string)
	walk = func(obj *parser.Object, parentKey string) {
		if obj == nil {
			return
		}
		for _, e := range obj.Entries {
			if e.Expression == nil {
				continue
			}
			ex := e.Expression
			if attrSet[ex.Key] && ex.Literal != nil && ex.Literal.String != nil {
				val := strings.Trim(*ex.Literal.String, `"`)
				if val != "" {
					emit(val, ex.Pos.Line, ex.Key)
				}
			}
			if ex.Object != nil {
				walk(ex.Object, ex.Key)
			}
		}
	}
	for _, e := range f.Entries {
		if e.Expression != nil && e.Expression.Object != nil {
			walk(e.Expression.Object, e.Expression.Key)
		}
	}
	_ = path
}

// AutoLocalize copies missing keys from sourceLang into targetLang under the first mod.
func (s *LocService) AutoLocalize(workspaceID, sourceLang, targetLang, modID string) (*AutoLocResult, error) {
	if sourceLang == "" {
		sourceLang = "l_english"
	}
	if targetLang == "" {
		return nil, fmt.Errorf("target language required")
	}
	repo := s.getRepo()
	mods, err := repo.ListModPaths(workspaceID)
	if err != nil || len(mods) == 0 {
		return nil, fmt.Errorf("no mods in workspace")
	}
	modPath := mods[0]
	if modID != "" {
		// prefer matching path suffix
		for _, p := range mods {
			if strings.Contains(p, modID) {
				modPath = p
				break
			}
		}
	}

	diags, err := s.LintMissingLoc(workspaceID, targetLang)
	if err != nil {
		return nil, err
	}
	langDir := strings.TrimPrefix(targetLang, "l_")
	outPath := filepath.Join(modPath, "localization", langDir, "pmt_autoloc_"+langDir+".yml")
	if len(diags) == 0 {
		return &AutoLocResult{Count: 0, Path: outPath, Preview: "No missing keys."}, nil
	}

	// Build source key values from install+mods
	gameID, installID, _ := repo.GetWorkspaceInfo(workspaceID)
	var locRoots []string
	locRoots = append(locRoots, filepath.Join(modPath, "localization"))
	if installID != "" {
		instPath, _, err := repo.GetInstallPath(installID)
		if err == nil {
			info := game.Get(gameID)
			if info != nil {
				for _, lr := range info.LocRoots {
					locRoots = append(locRoots, filepath.Join(instPath, lr))
				}
			}
		}
	}
	byLang, err := loc.CollectKeys(locRoots)
	if err != nil {
		return nil, err
	}
	src := byLang[sourceLang]

	if err := loc.EnsureLanguageFile(outPath, targetLang); err != nil {
		return nil, err
	}

	var entries []loc.Entry
	seen := map[string]bool{}
	for _, d := range diags {
		if seen[d.Key] {
			continue
		}
		seen[d.Key] = true
		val := ""
		if se, ok := src[d.Key]; ok {
			val = se.Value
		}
		entries = append(entries, loc.Entry{Key: d.Key, Value: val, Version: "0"})
	}
	if err := loc.AppendEntries(outPath, entries); err != nil {
		return nil, err
	}
	preview := ""
	for i, e := range entries {
		if i >= 12 {
			preview += fmt.Sprintf("…and %d more\n", len(entries)-i)
			break
		}
		preview += loc.FormatEntry(e.Key, e.Value, 0) + "\n"
	}
	return &AutoLocResult{Count: len(entries), Path: outPath, Preview: preview}, nil
}
