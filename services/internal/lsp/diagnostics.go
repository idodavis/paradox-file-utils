// diagnostics.go publishes CST/loc structural errors, STRICT missing-loc, loc BOM
// and header issues, and a missing-descriptor warning for a mod root.

package lsp

import (
	"os"
	"path/filepath"
	"strings"

	"paradox-modding-tools/services/internal/loc"
	"paradox-modding-tools/services/internal/parser"
	"paradox-modding-tools/services/internal/session"
)

// Diagnose returns diagnostics for path. Suppressions (# pmt:ignore) are honored.
func Diagnose(s *session.Session, path string) []Diagnostic {
	src := fileText(s, path)
	sup := session.ScanSuppressions(src)
	var out []Diagnostic
	if isLocPath(path) {
		out = append(out, locDiags(path, src)...)
	} else if !isMetaJSON(path) {
		out = append(out, scriptDiags(s, path, src)...)
	}
	out = append(out, missingLocDiags(s, path)...)
	out = append(out, descriptorDiags(s, path)...)
	filtered := out[:0]
	for _, d := range out {
		if sup.Covers(d.Range.Start.Line, d.Code) {
			continue
		}
		filtered = append(filtered, d)
	}
	return filtered
}

func scriptDiags(s *session.Session, path, src string) []Diagnostic {
	res := parseOf(s, path, src)
	srcLabel := sourceFor(path)
	out := make([]Diagnostic, 0, len(res.Errors))
	for _, e := range res.Errors {
		out = append(out, Diagnostic{
			Range:    byteRange(src, e.Range.Start, e.Range.End),
			Severity: sevError,
			Message:  e.Message,
			Source:   srcLabel,
			Code:     string(e.Code),
		})
	}
	return out
}

func locDiags(path, src string) []Diagnostic {
	r := parseLoc(src)
	out := make([]Diagnostic, 0, len(r.Errors)+2)
	for _, e := range r.Errors {
		out = append(out, Diagnostic{
			Range:    byteRange(src, e.Range.Start, e.Range.End),
			Severity: locSeverity(e.Code),
			Message:  e.Message,
			Source:   srcLoc,
			Code:     "loc-" + string(e.Code),
		})
	}
	if raw, err := os.ReadFile(path); err == nil && !parser.HasUTF8BOM(raw) {
		out = append(out, Diagnostic{
			Range:    Range{},
			Severity: sevError,
			Message:  "This localization file has no UTF-8 BOM; the game ignores it. Save as UTF-8 with BOM.",
			Source:   srcLoc,
			Code:     "missing-bom",
		})
	}
	fileLang := loc.LanguageFromFilename(path)
	if r.Language != "" && fileLang != "" && !strings.EqualFold(r.Language, fileLang) {
		rg := Range{}
		if r.HeaderRange != nil {
			rg = byteRange(src, r.HeaderRange.Start, r.HeaderRange.End)
		}
		out = append(out, Diagnostic{
			Range:    rg,
			Severity: sevError,
			Message:  "Header l_" + r.Language + ": does not match filename _l_" + fileLang + ".yml.",
			Source:   srcLoc,
			Code:     "loc-header-mismatch",
		})
	}
	rel := strings.ReplaceAll(path, "\\", "/")
	if strings.Contains(rel, "/localisation/") {
		out = append(out, Diagnostic{
			Range:    Range{},
			Severity: sevError,
			Message:  "Folder is localisation/ — the game reads localization/.",
			Source:   srcLoc,
			Code:     "wrong-localization-folder",
		})
	}
	return out
}

func locSeverity(code loc.ErrorCode) int {
	switch code {
	case loc.ErrNoHeader, loc.ErrContentBeforeHeader, loc.ErrUnterminatedValue:
		return sevError
	default:
		return sevWarning
	}
}

func missingLocDiags(s *session.Session, path string) []Diagnostic {
	idx := s.Index()
	if idx == nil {
		return nil
	}
	var out []Diagnostic
	src := fileText(s, path)
	for _, r := range idx.Refs {
		if r.Path != path || r.Kind != "loc" {
			continue
		}
		if locDefined(s, r.Key) {
			continue
		}
		out = append(out, Diagnostic{
			Range:    byteRange(src, r.Start, r.End),
			Severity: sevWarning,
			Message:  "Missing localization key \"" + r.Key + "\".",
			Source:   sourceFor(path),
			Code:     "missing-required-loc",
		})
	}
	return out
}

func descriptorDiags(s *session.Session, path string) []Diagnostic {
	root := modRootOf(s, path)
	if root == "" {
		return nil
	}
	want := descriptorPath(s.GameID, root)
	if fileExists(want) {
		return nil
	}
	// Only attach to a file at the mod root (descriptor or any direct child)
	// so every nested script file is not spammed.
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil
	}
	if strings.Contains(rel, string(filepath.Separator)) && filepath.Base(path) != filepath.Base(want) {
		return nil
	}
	msg := "Mod is missing descriptor.mod."
	if s.GameID == "vic3" || s.GameID == "eu5" {
		msg = "Mod is missing .metadata/metadata.json."
	}
	return []Diagnostic{{
		Range:    Range{},
		Severity: sevWarning,
		Message:  msg,
		Source:   srcMod,
		Code:     "missing-descriptor",
	}}
}
