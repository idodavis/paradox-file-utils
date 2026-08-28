// codeactions.go offers create-loc-key (UTF-8 BOM) and add-BOM quick fixes.

package lsp

import (
	"os"
	"path/filepath"
	"strings"

	"paradox-modding-tools/services/internal/loc"
	"paradox-modding-tools/services/internal/session"
)

const bom = "\uFEFF"

// CodeActions returns quick-fixes for path.
func CodeActions(s *session.Session, path string) []CodeAction {
	var out []CodeAction
	if isLocPath(path) {
		if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 &&
			!(len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF) {
			out = append(out, CodeAction{
				Title: "Add UTF-8 BOM",
				Kind:  "quickfix",
				Edit: &WorkspaceEdit{Changes: map[string][]TextEdit{
					path: {{Range: Range{}, NewText: bom}},
				}},
			})
		}
	}
	idx := s.Index()
	if idx == nil {
		return out
	}
	seen := map[string]bool{}
	for _, r := range idx.Refs {
		if r.Path != path || r.Kind != "loc" || locDefined(s, r.Key) || seen[r.Key] {
			continue
		}
		seen[r.Key] = true
		if edit := locCreateEdit(s, path, r.Key); edit != nil {
			out = append(out, CodeAction{
				Title: "Create localization key \"" + r.Key + "\"",
				Kind:  "quickfix",
				Edit:  edit,
			})
		}
	}
	return out
}

func locCreateEdit(s *session.Session, fromPath, key string) *WorkspaceEdit {
	root := modRootOf(s, fromPath)
	if root == "" {
		return nil
	}
	lang := loc.LanguageFromFilename(fromPath)
	if lang == "" {
		lang = "english"
	}
	target, exists := locTarget(s, root, lang)
	entry := " " + key + ":0 \"\"\n"
	if exists {
		raw, err := os.ReadFile(target)
		if err != nil {
			return nil
		}
		text, _ := strings.CutPrefix(string(raw), bom)
		lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
		line := len(lines)
		if line > 0 && lines[line-1] == "" {
			line--
		}
		return &WorkspaceEdit{Changes: map[string][]TextEdit{
			target: {{
				Range:   Range{Start: Position{Line: line, Character: 0}, End: Position{Line: line, Character: 0}},
				NewText: entry,
			}},
		}}
	}
	body := bom + "l_" + lang + ":\n" + entry
	return &WorkspaceEdit{
		Create: []string{target},
		Changes: map[string][]TextEdit{
			target: {{Range: Range{}, NewText: body}},
		},
	}
}

func locTarget(s *session.Session, root, lang string) (path string, exists bool) {
	preferred := filepath.Join(root, "localization", lang, "zzz_pmt_edits_l_"+lang+".yml")
	if fileExists(preferred) {
		return preferred, true
	}
	if idx := s.Index(); idx != nil {
		origin := ""
		for _, m := range s.Mods() {
			if m.Root == root {
				origin = m.Origin
			}
		}
		marker := "_l_" + lang
		for _, d := range idx.Defs {
			if d.Type != "loc_key" || d.Origin != origin {
				continue
			}
			if strings.Contains(strings.ToLower(d.Path), marker) {
				return d.Path, true
			}
		}
	}
	return preferred, false
}
