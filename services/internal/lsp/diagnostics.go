// diagnostics.go publishes parse/loc/descriptor diagnostics and pmt:ignore.
package lsp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/parser/jomini"
	"paradox-modding-tools/services/internal/parser/loc"
	"paradox-modding-tools/services/internal/session"
)

func diag(rg Range, sev int, msg, code string) Diagnostic {
	return Diagnostic{Range: rg, Severity: sev, Message: msg, Code: code}
}

// Diagnose returns diagnostics for path. Suppressions (# pmt:ignore) are honored.
// Install (vanilla) files are silent — harvest stays in the scan cache, not IDE lint.
func Diagnose(s *session.Session, path string) []Diagnostic {
	if origin, _, ok := s.Locate(path); ok && origin == game.OriginVanilla {
		return nil
	}
	src := s.FileText(path)
	sup := scanSuppressions(src)
	var out []Diagnostic
	if s.KindFor(path) == "loc" {
		out = append(out, locDiags(path, src)...)
	} else if !isMetaFile(s, path) {
		out = append(out, scriptDiags(s, path)...)
	}
	out = append(out, missingLocDiags(s, path)...)
	out = append(out, requiredLocDiags(s, path)...)
	out = append(out, unknownEventDiags(s, path)...)
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

func scriptDiags(s *session.Session, path string) []Diagnostic {
	res := s.Parsed(path)
	li := res.Lines()
	out := make([]Diagnostic, 0, len(res.Errors))
	for _, e := range res.Errors {
		out = append(out, diag(byteRange(li, e.Range.Start, e.Range.End),
			sevError, e.Message, string(e.Code)))
	}
	return out
}

func locDiags(path, src string) []Diagnostic {
	r := loc.Parse(src)
	li := jomini.NewLineIndex(src)
	out := make([]Diagnostic, 0, len(r.Errors)+2)
	for _, e := range r.Errors {
		out = append(out, diag(byteRange(li, e.Range.Start, e.Range.End),
			locSeverity(e.Code), e.Message, "loc-"+string(e.Code)))
	}
	if !locHasBOM(path, src) {
		out = append(out, diag(Range{}, sevError,
			"This localization file has no UTF-8 BOM; the game ignores it. Save as UTF-8 with BOM.",
			"missing-bom"))
	}
	fileLang := loc.LanguageFromFilename(path)
	if r.Language != "" && fileLang != "" && !strings.EqualFold(r.Language, fileLang) {
		rg := Range{}
		if r.HeaderRange != nil {
			rg = byteRange(li, r.HeaderRange.Start, r.HeaderRange.End)
		}
		out = append(out, diag(rg, sevError,
			"Header l_"+r.Language+": does not match filename _l_"+fileLang+".yml.",
			"loc-header-mismatch"))
	}
	if strings.Contains(strings.ReplaceAll(path, "\\", "/"), "/localisation/") {
		out = append(out, diag(Range{}, sevError,
			"Folder is localisation/ — the game reads localization/.",
			"wrong-localization-folder"))
	}
	return out
}

// locHasBOM reports a UTF-8 BOM in the buffer or on disk. FileText/Monaco
// strip the BOM, so vanilla loc (all BOM on disk) would false-positive otherwise.
func locHasBOM(path, src string) bool {
	if strings.HasPrefix(src, "\uFEFF") {
		return true
	}
	raw, err := os.ReadFile(path)
	return err == nil && jomini.HasUTF8BOM(raw)
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
	var out []Diagnostic
	src := s.FileText(path)
	var li *jomini.LineIndex
	if s.KindFor(path) == "loc" {
		li = jomini.NewLineIndex(src)
	} else {
		li = s.Parsed(path).Lines()
	}
	for _, r := range s.RefsInFile(path) {
		if r.Kind != "loc" || locDefined(s, r.Key) {
			continue
		}
		out = append(out, diag(byteRange(li, r.Start, r.End), sevWarning,
			"Missing localization key \""+r.Key+"\".", "missing-required-loc"))
	}
	return out
}

func requiredLocDiags(s *session.Session, path string) []Diagnostic {
	src := s.FileText(path)
	if src == "" {
		return nil
	}
	li := s.Parsed(path).Lines()
	var out []Diagnostic
	for _, d := range s.DefsInFile(path) {
		for _, key := range game.RequiredLocKeys(d.Kind, d.Key) {
			if locDefined(s, key) {
				continue
			}
			out = append(out, diag(byteRange(li, d.Start, d.End), sevWarning,
				"Missing localization key \""+key+"\".", "required-loc"))
		}
	}
	return out
}

func unknownEventDiags(s *session.Session, path string) []Diagnostic {
	src := s.FileText(path)
	if src == "" {
		return nil
	}
	li := s.Parsed(path).Lines()
	var out []Diagnostic
	for _, r := range s.RefsInFile(path) {
		if r.Kind != "event" || s.Resolve(r.Key) != nil {
			continue
		}
		ns, rest, ok := strings.Cut(r.Key, ".")
		if !ok || rest == "" || !modDeclaresNamespace(s, ns) {
			continue
		}
		out = append(out, diag(byteRange(li, r.Start, r.End), sevWarning,
			"Unknown event \""+r.Key+"\".", "unknown-event"))
	}
	return out
}

func modDeclaresNamespace(s *session.Session, ns string) bool {
	if ns == "" {
		return false
	}
	for _, d := range s.FindDefs(ns+".", 8, true, false) {
		if game.CanonicalKind(d.Kind) == "event" &&
			strings.HasPrefix(d.Key, ns+".") {
			return true
		}
	}
	return false
}

func descriptorDiags(s *session.Session, path string) []Diagnostic {
	root := modRootOf(s, path)
	if root == "" {
		return nil
	}
	want := game.DescriptorPath(s.GameID, root)
	if fileExists(want) {
		return nil
	}
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
	return []Diagnostic{diag(Range{}, sevWarning, msg, "missing-descriptor")}
}

var ignoreRe = regexp.MustCompile(`(?i)#\s*pmt:ignore(-next-line)?\b(.*)`)

type lineSuppression struct {
	all   bool
	codes map[string]bool
}

// Suppressions maps a 0-based line to what diagnostics are suppressed on it.
type Suppressions map[int]lineSuppression

func scanSuppressions(text string) Suppressions {
	sup := Suppressions{}
	if !strings.Contains(text, "pmt:ignore") {
		return sup
	}
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	for line, lineText := range strings.Split(text, "\n") {
		sup.scanLine(lineText, line)
	}
	return sup
}

func (s Suppressions) scanLine(lineText string, line int) {
	hash := strings.IndexByte(lineText, '#')
	if hash < 0 {
		return
	}
	m := ignoreRe.FindStringSubmatch(lineText[hash:])
	if m == nil {
		return
	}
	target := line
	if m[1] != "" {
		target = line + 1
	}
	s.merge(target, parseCodes(m[2]))
}

func (s Suppressions) Covers(line int, code string) bool {
	ls, ok := s[line]
	if !ok {
		return false
	}
	return ls.all || (code != "" && ls.codes[code])
}

func (s Suppressions) merge(line int, codes []string) {
	ls := s[line]
	if len(codes) == 0 {
		ls.all = true
	} else if !ls.all {
		if ls.codes == nil {
			ls.codes = map[string]bool{}
		}
		for _, c := range codes {
			ls.codes[c] = true
		}
	}
	s[line] = ls
}

func parseCodes(rest string) []string {
	var out []string
	for _, tok := range strings.Fields(rest) {
		if strings.HasPrefix(tok, "-") {
			break
		}
		out = append(out, tok)
	}
	return out
}
